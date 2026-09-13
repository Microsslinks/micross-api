package model

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// 客户号的签发、列举与作废。
//
// 客户号是经销商把「归属」和「折扣」一次交给客户的办法：一码同时写着归属哪个经销商、
// 按哪套折扣方案计价。客户拿到号之后，绑一下这两件事就自动生效（绑号见 agent_customer.go）。
//
// 一张号对应一位客户：归属落在客户的 users.parent_agent_id 上，号这边只记下"谁用掉的"
// （见 CustomerCode.BoundUserId）。号用完即废，但记录一直留着——将来要回溯某位客户是谁
// 带来的，只有这张号说得清。

const (
	// CustomerCodePrefix 写在号前面，让它在聊天窗口里一眼能看出是客户号而不是随机串。
	CustomerCodePrefix = "AG"
	// CustomerCodeRandomLength 随机部分的长度。36 进制 10 位，撞号概率可以忽略；
	// 库里还有唯一索引兜底，生成时也会先查一次。
	CustomerCodeRandomLength = 10
	// CustomerCodeMaxBatch 一次最多签多少个：发号是给人拿去分发的，一次上百个既不实用也容易点错。
	CustomerCodeMaxBatch = 50
	// CustomerCodeMaxUsesPerCode 一张号能让几位客户用：恒为 1。
	// 号是"把某个人拉进来"的一次性凭证，不是可以转发的邀请码——放开这个数等于造出一张
	// 谁拿到都能绑的公开号，泄露一次就收不住。所以它不做成可配的。
	CustomerCodeMaxUsesPerCode = 1
)

var (
	// ErrCustomerCodeNotFound 号不存在，或不属于这位经销商。
	ErrCustomerCodeNotFound = errors.New("客户号不存在")
	// ErrCustomerCodePlanUnavailable 号上指定的折扣方案不存在或已停用。
	ErrCustomerCodePlanUnavailable = errors.New("客户号指定的折扣方案不存在或已停用")
)

// CustomerCodeIssue 是签一批客户号之前要定下来的几件事。
// 这里没有"使用次数"：一张号只拉一位客户，是写死的规则，不给调用方留旋钮——
// 留了就会有人填 0（不限次数），那等于发出去一张谁拿到都能绑的公开号。
type CustomerCodeIssue struct {
	AgentId   int
	PlanId    int
	Count     int
	ExpiredAt int64
	Remark    string
}

// NormalizeCustomerCodeIssue 校验签号入参并规范 remark。
// 纯校验、不碰数据库，好让调用方在落库前就能把话说明白。
func NormalizeCustomerCodeIssue(issue CustomerCodeIssue) (CustomerCodeIssue, error) {
	if issue.AgentId <= 0 {
		return issue, errors.New("无效的经销商 ID")
	}
	if issue.Count <= 0 {
		return issue, errors.New("生成数量至少为 1")
	}
	if issue.Count > CustomerCodeMaxBatch {
		return issue, fmt.Errorf("一次最多生成 %d 个客户号", CustomerCodeMaxBatch)
	}
	if issue.PlanId < 0 {
		return issue, errors.New("折扣方案不正确")
	}
	if issue.ExpiredAt < 0 {
		return issue, errors.New("有效期不正确")
	}
	// 有效的号才有发出去的价值：过期时间已经是过去时的话，客户拿到就是废号。
	if issue.ExpiredAt > 0 && issue.ExpiredAt <= common.GetTimestamp() {
		return issue, errors.New("有效期已经过去了")
	}
	issue.Remark = strings.TrimSpace(issue.Remark)
	if utf8.RuneCountInString(issue.Remark) > 255 {
		return issue, errors.New("备注不能超过 255 个字符")
	}
	return issue, nil
}

// IsUserAgent 判断这个用户此刻是不是经销商。
// 只看 users 上的快路径字段，与计价那条链同源（见 agent_wholesale.go）。
func IsUserAgent(userId int) (bool, error) {
	if userId <= 0 {
		return false, nil
	}
	var user User
	if err := DB.Select("id", "subject_type").First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}
	return user.SubjectType == SubjectTypeAgent, nil
}

// CreateCustomerCodes 签一批客户号。
//
// 签发人必须是经销商：客户号的全部意义就是「把客户归到这个人名下」，
// 一个不是经销商的人签出来的号绑上去只会得到一个指向普通客户的归属。
//
// 每一张都写死 MaxUses = CustomerCodeMaxUsesPerCode：调用方没有地方能改这个数，
// 所以"签出一张能绑多次的号"从入参这一层就不成立。
func CreateCustomerCodes(issue CustomerCodeIssue) ([]*CustomerCode, error) {
	normalized, err := NormalizeCustomerCodeIssue(issue)
	if err != nil {
		return nil, err
	}
	isAgent, err := IsUserAgent(normalized.AgentId)
	if err != nil {
		return nil, err
	}
	if !isAgent {
		return nil, ErrAgentProfileNotFound
	}
	if err := ensureCustomerCodePlanUsable(normalized.PlanId); err != nil {
		return nil, err
	}

	codes := make([]*CustomerCode, 0, normalized.Count)
	err = DB.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < normalized.Count; i++ {
			code, err := generateUniqueCustomerCode(tx)
			if err != nil {
				return err
			}
			record := &CustomerCode{
				Code:      code,
				AgentId:   normalized.AgentId,
				PlanId:    normalized.PlanId,
				MaxUses:   CustomerCodeMaxUsesPerCode,
				ExpiredAt: normalized.ExpiredAt,
				Status:    CustomerCodeStatusEnabled,
				Remark:    normalized.Remark,
			}
			if err := tx.Create(record).Error; err != nil {
				return err
			}
			codes = append(codes, record)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return codes, nil
}

// ListCustomerCodes 列出某位经销商签出去的号，最新的在前。
//
// onlyUsable 为 true 时只给"还能用的号"：经销商打开这一页，想先看的是"手上还有哪些号
// 能发出去"，而不是一屏旧号。用完的、作废的、过期的都还在库里，切回全部就能翻到。
//
// 顺手把「用掉这张号的人」补上：那两列不落库（见 CustomerCode 的 BoundUsername），
// 只有在列表这一趟查得出来。
func ListCustomerCodes(agentId int, offset int, limit int, onlyUsable bool) ([]*CustomerCode, int64, error) {
	if agentId <= 0 {
		return nil, 0, errors.New("无效的经销商 ID")
	}
	query := DB.Model(&CustomerCode{}).Where("agent_id = ?", agentId)
	if onlyUsable {
		// 口径与 customerCodeRejectReason 一致：没作废、没过期、没被用掉。
		// MaxUses 为 0 是早于「一张号一位客户」的旧数据，仍按不限次数算。
		now := common.GetTimestamp()
		query = query.Where("status = ?", CustomerCodeStatusEnabled).
			Where("(expired_at = 0 OR expired_at > ?)", now).
			Where("(max_uses = 0 OR used_count < max_uses)")
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	codes := make([]*CustomerCode, 0)
	if err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&codes).Error; err != nil {
		return nil, 0, err
	}
	if err := fillCustomerCodeBoundUsers(codes); err != nil {
		return nil, 0, err
	}
	return codes, total, nil
}

// fillCustomerCodeBoundUsers 给列表里的号补上"用掉它的人"的名字。
//
// 用 Unscoped 跳过软删除过滤：那个人后来可能被禁用、注销，但"这个号给了谁"是已经发生的
// 事，查得出来比查不出来重要。真查不到（行都没了）就留空，不让整个列表跟着失败。
func fillCustomerCodeBoundUsers(codes []*CustomerCode) error {
	ids := make([]int, 0, len(codes))
	seen := make(map[int]bool, len(codes))
	for _, code := range codes {
		if code.BoundUserId > 0 && !seen[code.BoundUserId] {
			seen[code.BoundUserId] = true
			ids = append(ids, code.BoundUserId)
		}
	}
	if len(ids) == 0 {
		return nil
	}

	users := make([]User, 0, len(ids))
	if err := DB.Unscoped().Select("id", "username", "display_name").
		Where("id IN ?", ids).Find(&users).Error; err != nil {
		return err
	}
	byId := make(map[int]User, len(users))
	for _, user := range users {
		byId[user.Id] = user
	}
	for _, code := range codes {
		if user, ok := byId[code.BoundUserId]; ok {
			code.BoundUsername = user.Username
			code.BoundDisplayName = user.DisplayName
		}
	}
	return nil
}

// RevokeCustomerCode 作废一个号。
//
// 不删行：作废是业务动作，号曾经发出去过就得留痕（谁签的、什么时候签的、
// 已经被用了几次）。作废后 IsUsable 返回 false，绑号那条路自然就不通了。
func RevokeCustomerCode(agentId int, codeId int) error {
	if agentId <= 0 || codeId <= 0 {
		return ErrCustomerCodeNotFound
	}
	result := DB.Model(&CustomerCode{}).
		Where("id = ? AND agent_id = ?", codeId, agentId).
		Updates(map[string]interface{}{
			"status":     CustomerCodeStatusDisabled,
			"updated_at": common.GetTimestamp(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrCustomerCodeNotFound
	}
	return nil
}

// ensureCustomerCodePlanUsable 号上带的折扣方案必须真的能用：不存在或已停用的方案
// 绑上去只会让客户以为自己有折扣、实际按原价计费。
// PlanId 为 0 表示这张号只落归属、不绑折扣。
func ensureCustomerCodePlanUsable(planId int) error {
	if planId <= 0 {
		return nil
	}
	plan, err := GetDiscountPlanById(planId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrCustomerCodePlanUnavailable
		}
		return err
	}
	if plan.Status != DiscountStatusEnabled {
		return ErrCustomerCodePlanUnavailable
	}
	return nil
}

// generateUniqueCustomerCode 生成一个库里还没有的号。
//
// 随机部分撞号概率可以忽略，但仍然查一次库：发号是一次性动作，
// 多一次查询换「发出去的号绝对不重复」是划算的。
func generateUniqueCustomerCode(tx *gorm.DB) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate := CustomerCodePrefix + strings.ToUpper(common.GetRandomString(CustomerCodeRandomLength))
		var count int64
		if err := tx.Model(&CustomerCode{}).Where("code = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", errors.New("生成客户号失败，请重试")
}
