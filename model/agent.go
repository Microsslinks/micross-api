package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	// 用户主体类型。普通客户是绝大多数，行为与既有完全一致；
	// 经销商多一个"管理他人"的身份，靠 AgentProfile 记录表达。
	SubjectTypeIndividual = "individual"
	SubjectTypeAgent      = "agent"

	// 客户类型筛选口径，给运营在用户列表里区分人群用。
	// individual / agent 直接对应 users.subject_type；enterprise 不是库里的值，
	// 而是"绑了折扣方案但不是经销商"推导出来的。判定顺序与前端
	// web/src/features/users/constants.ts 的 resolveCustomerType 必须一致。
	CustomerTypeIndividual = "individual"
	CustomerTypeEnterprise = "enterprise"
	CustomerTypeAgent      = "agent"

	// 客户号状态
	CustomerCodeStatusDisabled = 0
	CustomerCodeStatusEnabled  = 1

	// 客户号不限使用次数
	CustomerCodeUnlimitedUses = 0
)

// ErrAgentProfileNotFound 该用户不是经销商（没有经营档案）。
var ErrAgentProfileNotFound = errors.New("该用户不是经销商")

// ErrAgentHasCustomerCodes 经销商名下还挂着客户号，不能直接取消身份。
// 客户号记录着"哪些客户归他"，先把号作废才不会让这些客户的归属断链。
var ErrAgentHasCustomerCodes = errors.New("该经销商名下还有客户号")

// AgentProfile 是经销商的经营档案：一个用户被平台设为经销商后才会有一行。
//
// 两个折扣的分工：
//   - WholesaleDiscount 批发折扣，平台卖给经销商的价格，只能由平台设定，经销商不可自改；
//   - MinDiscount 最低折扣，平台给这个经销商的地板价，他给下属客户定的价不得低于它（D3）。
//
// 价格铁律（见 .docs/task-02-business-goals/03-agent-and-commission.md）：
// 渠道成本折扣 ≤ 批发折扣 ≤ 零售折扣。
type AgentProfile struct {
	Id                int    `json:"id"`
	UserId            int    `json:"user_id" gorm:"type:int;not null;uniqueIndex:uk_agent_profile_user"`
	WholesaleDiscount string `json:"wholesale_discount" gorm:"type:decimal(10,6);not null"`
	MinDiscount       string `json:"min_discount" gorm:"type:decimal(10,6);not null"`
	IssueQuotaEnabled int    `json:"issue_quota_enabled" gorm:"type:int;not null;default:1"` // 1 允许给下属发额度，0 停发（已发额度不受影响）
	Remark            string `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt         int64  `json:"updated_at" gorm:"bigint"`
}

func (AgentProfile) TableName() string {
	return "agent_profiles"
}

func (p *AgentProfile) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *AgentProfile) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

// NormalizeDefaults 补全折扣的零值口径。与折扣方案同理，列定义里刻意不写 DEFAULT：
// SQLite 对 decimal 列默认值的判定不稳定，写在列上会导致每次启动重建整张表。
func (p *AgentProfile) NormalizeDefaults() {
	if p.WholesaleDiscount == "" {
		p.WholesaleDiscount = DiscountNone
	}
	if p.MinDiscount == "" {
		p.MinDiscount = "0"
	}
}

// CustomerCode 是经销商发给客户的兑换式客户号：一码同时承载
// 「归属哪个经销商」与「按哪个折扣方案计价」，客户注册时带号或在登录后绑号。
type CustomerCode struct {
	Id        int    `json:"id"`
	Code      string `json:"code" gorm:"type:varchar(32);not null;uniqueIndex:uk_customer_code"`
	AgentId   int    `json:"agent_id" gorm:"type:int;not null;index:idx_customer_code_agent"` // 经销商用户 id
	PlanId    int    `json:"plan_id" gorm:"type:int;not null;default:0"`                      // 客户用此号后绑定的折扣方案
	MaxUses   int    `json:"max_uses" gorm:"type:int;not null;default:0"`                     // 0 表示不限次数
	UsedCount int    `json:"used_count" gorm:"type:int;not null;default:0"`
	ExpiredAt int64  `json:"expired_at" gorm:"type:bigint;not null;default:0"` // 0 表示不过期
	Status    int    `json:"status" gorm:"type:int;not null;default:1"`
	Remark    string `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint"`
}

func (CustomerCode) TableName() string {
	return "customer_codes"
}

func (c *CustomerCode) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	c.CreatedAt = now
	c.UpdatedAt = now
	return nil
}

func (c *CustomerCode) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = common.GetTimestamp()
	return nil
}

// IsUsable 客户号此刻还能不能被使用：未作废、未过期、未用满次数。
func (c *CustomerCode) IsUsable(now int64) bool {
	if c.Status != CustomerCodeStatusEnabled {
		return false
	}
	if c.ExpiredAt > 0 && now > c.ExpiredAt {
		return false
	}
	if c.MaxUses > 0 && c.UsedCount >= c.MaxUses {
		return false
	}
	return true
}

// GetAgentProfileByUserId 取某用户的经销商档案。没被设为经销商时返回 ErrAgentProfileNotFound。
func GetAgentProfileByUserId(userId int) (*AgentProfile, error) {
	if userId <= 0 {
		return nil, ErrAgentProfileNotFound
	}
	var profile AgentProfile
	if err := DB.Where("user_id = ?", userId).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrAgentProfileNotFound
		}
		return nil, err
	}
	return &profile, nil
}

// GetCustomerCodeByCode 按客户号取值，供绑定与校验共用。
func GetCustomerCodeByCode(code string) (*CustomerCode, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("客户号不能为空")
	}
	var customerCode CustomerCode
	if err := DB.Where("code = ?", code).First(&customerCode).Error; err != nil {
		return nil, err
	}
	return &customerCode, nil
}

// PromoteUserToAgent 把一个用户设为经销商：改主体类型 + 建经营档案，同一事务完成。
//
// 只动 users.subject_type 与 agent_profiles，**不碰 users.role**：
// 经销商是业务身份而不是权限等级，他仍然是普通用户（见 .docs/task-05-roles/README.md §二）。
// 档案已存在时按传入的值覆盖，方便平台调整批发折扣与地板价。
func PromoteUserToAgent(userId int, wholesaleDiscount string, minDiscount string, issueQuotaEnabled int, remark string) error {
	if userId <= 0 {
		return errors.New("无效的用户 ID")
	}
	profile := AgentProfile{
		UserId:            userId,
		WholesaleDiscount: wholesaleDiscount,
		MinDiscount:       minDiscount,
		IssueQuotaEnabled: issueQuotaEnabled,
		Remark:            remark,
	}
	profile.NormalizeDefaults()
	return DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).Where("id = ?", userId).
			Update("subject_type", SubjectTypeAgent).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userId).
			Assign(map[string]interface{}{
				"wholesale_discount":  profile.WholesaleDiscount,
				"min_discount":        profile.MinDiscount,
				"issue_quota_enabled": profile.IssueQuotaEnabled,
				"remark":              profile.Remark,
				"updated_at":          common.GetTimestamp(),
			}).
			FirstOrCreate(&profile).Error
	})
}

// DemoteAgentToUser 取消经销商身份：删经营档案 + 改回普通客户，同一事务完成。
//
// 名下还挂着客户号的经销商不允许取消，否则那些客户号会指向一个不再是经销商的人。
// 用户的余额、分组、已绑折扣方案都不动——取消身份不等于清账。
func DemoteAgentToUser(userId int) error {
	if userId <= 0 {
		return errors.New("无效的用户 ID")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var codeCount int64
		if err := tx.Model(&CustomerCode{}).Where("agent_id = ?", userId).Count(&codeCount).Error; err != nil {
			return err
		}
		if codeCount > 0 {
			return ErrAgentHasCustomerCodes
		}
		if err := tx.Where("user_id = ?", userId).Delete(&AgentProfile{}).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", userId).
			Update("subject_type", SubjectTypeIndividual).Error
	})
}
