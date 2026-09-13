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
// MarkupRatio 是平台在这笔生意上加的那一档毛利，也是档案里唯一参与计价的数：
// 经销商拿货价 = 这个模型最便宜一条线路的进货折扣 × MarkupRatio（见 agent_wholesale.go）。
// 只能由平台设定，经销商不可自改。空值表示这一行还没设过，按 AgentWholesaleMarkupDefault 处理。
//
// 这一列刻意不带 not null：SQLite 给已存在的表加列时不允许加一个没有默认值的 NOT NULL 列，
// 而默认值又不写在列上（各库对默认值的归一化不一样，写上去会反复触发 AutoMigrate 建表）。
// 取值范围与缺省由 NormalizeAgentMarkup / EffectiveMarkupRatio 兜住。
//
// WholesaleDiscount 与 MinDiscount 是更早那版「平台手填一个批发折扣、再给一个地板价」
// 留下的列：现在全仓库没有任何计价逻辑读它们，也不再有界面往里写业务值。
// 不删是因为它们承载着历史数据，删掉就找不回来了。
type AgentProfile struct {
	Id                int    `json:"id"`
	UserId            int    `json:"user_id" gorm:"type:int;not null;uniqueIndex:uk_agent_profile_user"`
	MarkupRatio       string `json:"markup_ratio" gorm:"type:decimal(10,6)"`
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

// NormalizeDefaults 补全零值口径。与折扣方案同理，列定义里刻意不写 DEFAULT：
// SQLite 对 decimal 列默认值的判定不稳定，写在列上会导致每次启动重建整张表。
func (p *AgentProfile) NormalizeDefaults() {
	if p.MarkupRatio == "" {
		p.MarkupRatio = AgentWholesaleMarkupDefault
	}
	if p.WholesaleDiscount == "" {
		p.WholesaleDiscount = DiscountNone
	}
	if p.MinDiscount == "" {
		p.MinDiscount = "0"
	}
}

// EffectiveMarkupRatio 取这份档案此刻生效的平台加价率。
// 库里存空值（这一行是在加价率这列存在之前建的）时按缺省档给，缺省不是猜数。
func (p *AgentProfile) EffectiveMarkupRatio() string {
	if p == nil || strings.TrimSpace(p.MarkupRatio) == "" {
		return AgentWholesaleMarkupDefault
	}
	return p.MarkupRatio
}

// CustomerCode 是经销商发给客户的兑换式客户号：一码同时承载
// 「归属哪个经销商」与「按哪个折扣方案计价」，客户注册时带号或在登录后绑号。
//
// 一张号只拉一位客户（MaxUses 恒为 CustomerCodeMaxUsesPerCode）：它是"把这个人拉进来"
// 的一次性凭证，不是可以到处转发的邀请码。谁用掉的记在 BoundUserId 上——归属一旦落下就
// 跟着客户走，但"这位客户当初是谁带来的"只有这张号说得清，所以这条记录要一直留着。
type CustomerCode struct {
	Id        int    `json:"id"`
	Code      string `json:"code" gorm:"type:varchar(32);not null;uniqueIndex:uk_customer_code"`
	AgentId   int    `json:"agent_id" gorm:"type:int;not null;index:idx_customer_code_agent"` // 经销商用户 id
	PlanId    int    `json:"plan_id" gorm:"type:int;not null;default:0"`                      // 客户用此号后绑定的折扣方案
	MaxUses   int    `json:"max_uses" gorm:"type:int;not null;default:1"`                     // 恒为 1；这一列留着是为了兼容早于「一张号一位客户」的旧数据
	UsedCount int    `json:"used_count" gorm:"type:int;not null;default:0"`
	// BoundUserId 是用掉这张号的人，0 表示还没人用。一张号只给一位客户，一个字段就够。
	BoundUserId int `json:"bound_user_id" gorm:"type:int;not null;default:0;index:idx_customer_code_bound"`
	// BoundUsername / BoundDisplayName 只在列表回显时按 BoundUserId 补上，不落库。
	BoundUsername    string `json:"bound_username" gorm:"-"`
	BoundDisplayName string `json:"bound_display_name" gorm:"-"`
	ExpiredAt        int64  `json:"expired_at" gorm:"type:bigint;not null;default:0"` // 0 表示不过期
	Status           int    `json:"status" gorm:"type:int;not null;default:1"`
	Remark           string `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt        int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt        int64  `json:"updated_at" gorm:"bigint"`
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

// PromoteUserToAgentWithMarkup 把一个用户设为经销商，并定下平台加价率：
// 改主体类型 + 建经营档案，同一事务完成。新代码走这个入口。
//
// 加价率传进来之前已由 NormalizeAgentMarkup 校验过；这里再走一次 NormalizeDefaults，
// 是为了让空值落到缺省档上，而不是把空串写进库。
func PromoteUserToAgentWithMarkup(userId int, markupRatio string, issueQuotaEnabled int, remark string) error {
	return promoteUserToAgent(userId, markupRatio, DiscountNone, "0", issueQuotaEnabled, remark)
}

// PromoteUserToAgent 是老入口：按「平台手填批发折扣 + 地板价」那版口径写档案。
// 那两个数现在没有任何计价逻辑读取，保留它只是为了不动既有调用方与既有测试；
// 新代码一律用 PromoteUserToAgentWithMarkup。
func PromoteUserToAgent(userId int, wholesaleDiscount string, minDiscount string, issueQuotaEnabled int, remark string) error {
	return promoteUserToAgent(userId, AgentWholesaleMarkupDefault, wholesaleDiscount, minDiscount, issueQuotaEnabled, remark)
}

// promoteUserToAgent 是两个入口的共同实现：改主体类型 + 建/更新经营档案，同一事务完成。
//
// 只动 users.subject_type 与 agent_profiles，**不碰 users.role**：
// 经销商是业务身份而不是权限等级，他仍然是普通用户（见 .docs/task-05-roles/README.md §二）。
// 档案已存在时按传入的值覆盖，方便平台调整这位经销商的加价率。
func promoteUserToAgent(userId int, markupRatio string, wholesaleDiscount string, minDiscount string, issueQuotaEnabled int, remark string) error {
	if userId <= 0 {
		return errors.New("无效的用户 ID")
	}
	profile := AgentProfile{
		UserId:            userId,
		MarkupRatio:       markupRatio,
		WholesaleDiscount: wholesaleDiscount,
		MinDiscount:       minDiscount,
		IssueQuotaEnabled: issueQuotaEnabled,
		Remark:            remark,
	}
	profile.NormalizeDefaults()
	return DB.Transaction(func(tx *gorm.DB) error {
		// 顺带把归属清掉：经销商直属平台。不清的话，一个绑过客户号的人被设成经销商后，
		// 还会留在原来那位经销商的「我的客户」名单里——他已经是同行的对手了。
		if err := tx.Model(&User{}).Where("id = ?", userId).
			Updates(map[string]interface{}{
				"subject_type":    SubjectTypeAgent,
				"parent_agent_id": 0,
			}).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", userId).
			Assign(map[string]interface{}{
				"markup_ratio":        profile.MarkupRatio,
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
