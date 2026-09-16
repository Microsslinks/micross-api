package service

// 经销商代注册客户 — P3 任务-08。
//
// 服务层专注于"业务规则"，不感知 HTTP：
//   - 鉴权 = 调用方必须是 enabled 的经销商（subject_type=agent）
//   - 资源归属 = 客户必须 parent_agent_id == 调用方经销商
//   - 重置密码要走过 token 失效闭环：increment auth_version + update + publish + revoke
//   - 停用要级联把客户的 token.status 设为 disabled
//
// HTTP 层（controller）从这里拿结果，包成统一响应。

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

// ===================== 鉴权 =====================

// requireAgent 校验调用方是启用中的经销商。返回错误即可，不需要把 user 返回给上层。
//
// 经销商是业务身份不是权限等级，subject_type=agent 才是判据（见 model/agent.go 的 promoteUserToAgent）。
// 同时 status 必须为 enabled，否则连读自己客户列表都该被拒。
func requireAgent(agentId int) error {
	if agentId <= 0 {
		return errors.New("无效的用户 ID")
	}
	var user model.User
	if err := model.DB.Select("id", "subject_type", "status").First(&user, agentId).Error; err != nil {
		return err
	}
	if user.SubjectType != model.SubjectTypeAgent {
		return model.ErrAgentProfileNotFound
	}
	if user.Status != common.UserStatusEnabled {
		return errors.New("你的账号已被停用")
	}
	return nil
}

// loadCustomerUnderAgent 校验客户确实归这位经销商名下。model 包已经有 ensureCustomerBelongsToAgent，
// 但那里是私有的，这里直接 SELECT 一行做归属校验 + 读出 customer 实例，供后续字段更新。
//
// 资源归属错时一律返回 ErrAgentCustomerNotFound（与"查无此人"合并处理），不确认别人名下的客户是否存在。
func loadCustomerUnderAgent(tx *gorm.DB, agentId int, customerId int) (*model.User, error) {
	if customerId <= 0 {
		return nil, model.ErrAgentCustomerNotFound
	}
	var customer model.User
	if err := tx.Select("id", "subject_type", "parent_agent_id", "status").
		First(&customer, customerId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, model.ErrAgentCustomerNotFound
		}
		return nil, err
	}
	// 客户已经被设成经销商：归属早已在 promoteUserToAgent 里清掉，这里也算"不在你名下"。
	if customer.SubjectType == model.SubjectTypeAgent || customer.ParentAgentId != agentId {
		return nil, model.ErrAgentCustomerNotFound
	}
	return &customer, nil
}

// ===================== 代注册 =====================

// RegisterCustomerByAgentParams 代注册参数。Username 必填，DisplayName 可选。
// Password 在 controller 层做格式校验（必须由调用方传入明文；service 层只校验非空）。
type RegisterCustomerByAgentParams struct {
	Username    string
	Password    string
	DisplayName string
}

// RegisterCustomerByAgent 经销商代注册客户：复用 model/user.go 的 InsertByAgent，事务内写
// users 行 + 设置默认 quota + 默认 setting + 写 ParentAgentId。
//
// 不发邀请码奖励（finishInsert(0)），但代注册的客户本人可以拿 QuotaForNewUser。
func RegisterCustomerByAgent(agentId int, params RegisterCustomerByAgentParams) (*model.User, error) {
	if err := requireAgent(agentId); err != nil {
		return nil, err
	}
	username := strings.TrimSpace(params.Username)
	if username == "" {
		return nil, errors.New("用户名不能为空")
	}
	if len(params.Password) < 8 {
		return nil, errors.New("密码至少 8 位")
	}

	newUser := &model.User{
		Username:    username,
		Password:    params.Password, // 仍为明文，InsertByAgent 里的 prepareForInsert 会调 Password2Hash
		DisplayName: strings.TrimSpace(params.DisplayName),
	}
	if err := newUser.InsertByAgent(agentId); err != nil {
		return nil, err
	}
	common.SysLog(fmt.Sprintf("经销商 %d 代注册客户 %s (id=%d)", agentId, username, newUser.Id))
	return newUser, nil
}

// ===================== 重置密码 =====================

// ResetCustomerPasswordByAgent 经销商给名下客户重置密码，返回新的明文密码（仅此一次）。
//
// 完整闭环（与 model/user.go 的 ResetUserPasswordByEmail 一致）：
//   1. IncrementUserAuthVersionWithTx — 让旧 token/session 失效
//   2. Update password — 用新密码落库
//   3. PublishUserAuthCache — 推缓存
//   4. RevokeAllUserSessions — 撤销所有 session
//
// 明文密码不写日志、不入库、不再次返回。
func ResetCustomerPasswordByAgent(agentId int, customerId int) (string, error) {
	if err := requireAgent(agentId); err != nil {
		return "", err
	}

	newPassword := common.GetRandomString(8)
	hashed, err := common.Password2Hash(newPassword)
	if err != nil {
		return "", err
	}

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if _, err := loadCustomerUnderAgent(tx, agentId, customerId); err != nil {
			return err
		}
		if _, err := model.IncrementUserAuthVersionWithTx(tx, customerId); err != nil {
			return err
		}
		return tx.Model(&model.User{}).Where("id = ?", customerId).
			Update("password", hashed).Error
	}); err != nil {
		return "", err
	}

	if err := model.PublishUserAuthCache(customerId); err != nil {
		// 缓存没跟上不影响账号状态；这里只记一下不返回失败（与 ResetUserPasswordByEmail 同样口径）
		common.SysLog(fmt.Sprintf("经销商 %d 重置客户 %d 密码后推送 auth 缓存失败: %s", agentId, customerId, err.Error()))
	}
	if _, err := model.RevokeAllUserSessions(customerId, "agent_password_reset"); err != nil {
		common.SysLog(fmt.Sprintf("经销商 %d 重置客户 %d 密码后撤销 session 失败: %s", agentId, customerId, err.Error()))
	}
	return newPassword, nil
}

// ===================== 停用 =====================

// DisableCustomerByAgent 经销商停用名下客户：users.status = disabled，级联把所有 api_tokens 设为 disabled。
//
// 幂等：客户已经是 disabled 时直接返回 nil，不重复写。
// 与 ResetCustomerPasswordByAgent 不同，这里不清 auth_version——停用已经够狠，再清一次没意义。
func DisableCustomerByAgent(agentId int, customerId int) error {
	if err := requireAgent(agentId); err != nil {
		return err
	}

	err := model.DB.Transaction(func(tx *gorm.DB) error {
		customer, err := loadCustomerUnderAgent(tx, agentId, customerId)
		if err != nil {
			return err
		}
		if customer.Status == common.UserStatusDisabled {
			return nil // 幂等
		}
		if err := tx.Model(&model.User{}).Where("id = ?", customerId).
			Update("status", common.UserStatusDisabled).Error; err != nil {
			return err
		}
		// 级联：把这个客户的所有 token 一起停用。
		// where 条件里加 status <> disabled 保持幂等，且避免 UpdatedAt 不必要变更。
		if err := tx.Model(&model.Token{}).
			Where("user_id = ? AND status <> ?", customerId, common.UserStatusDisabled).
			Update("status", common.UserStatusDisabled).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	common.SysLog(fmt.Sprintf("经销商 %d 停用客户 %d", agentId, customerId))
	// 推一次 token 缓存，下游对 token.status 的判读立刻可见
	_ = model.InvalidateUserTokensCache(customerId)
	return nil
}