package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// 客户路由策略的取值。库里存字符串，认不出的值一律按默认（毛利优先）处理。
const (
	// RoutingStrategyMargin 按毛利优先挑线路（默认）：能多赚就多赚。
	RoutingStrategyMargin = "margin"
	// RoutingStrategyPriority 按上游供应商优先级挑线路：客户体验优先，更稳的线路先走。
	RoutingStrategyPriority = "priority"
)

// DiscountRoutingPolicy 是「这个客户怎么挑线路」的例外配置。
//
// 它跟折扣方案是两件事：折扣方案回答「这个客户按几折收钱」，方案是共享的、可以绑给很多
// 客户；这张表回答「这个客户的钱花在哪条线路上」，按单个客户配，所以不能挂在方案上。
//
// 绝大多数客户没有这一行——没有这一行就是默认口径：毛利优先、不允许走亏损线路。
// 只有运营针对某个客户单独改过，才会有一行。
type DiscountRoutingPolicy struct {
	Id     int `json:"id"`
	UserId int `json:"user_id" gorm:"type:int;not null;uniqueIndex:uk_routing_user"`
	// RoutingStrategy 见 RoutingStrategyMargin / RoutingStrategyPriority。
	RoutingStrategy string `json:"routing_strategy" gorm:"type:varchar(16);not null"`
	// AllowCostBreach 表示允许这个客户走会亏本的线路（0 不允许／1 允许）。
	// 刻意用 int 而不是 bool：三种数据库对布尔列的零值、默认值与列比对口径不一致，
	// 会让 AutoMigrate 反复重建整张表（AGENTS.md 有明确警告）。
	AllowCostBreach int `json:"allow_cost_breach" gorm:"type:int;not null"`
	// UpdatedBy 是最后改这一行的人（管理员 id），出事时要说得清是谁开的。
	UpdatedBy int    `json:"updated_by" gorm:"type:int;not null"`
	Remark    string `json:"remark" gorm:"type:varchar(255);not null"`
	CreatedAt int64  `json:"created_at" gorm:"type:bigint;not null"`
	UpdatedAt int64  `json:"updated_at" gorm:"type:bigint;not null"`
}

func (DiscountRoutingPolicy) TableName() string {
	return "discount_routing_policies"
}

func (p *DiscountRoutingPolicy) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	p.NormalizeDefaults()
	return nil
}

func (p *DiscountRoutingPolicy) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	p.NormalizeDefaults()
	return nil
}

// NormalizeDefaults 把认不出的取值收进默认口径。列定义里刻意不写 DEFAULT：
// 默认值属于业务规则，由代码兜住比让数据库兜住更可控（与 discount_plans 的做法一致）。
func (p *DiscountRoutingPolicy) NormalizeDefaults() {
	if p.RoutingStrategy != RoutingStrategyPriority {
		p.RoutingStrategy = RoutingStrategyMargin
	}
	if p.AllowCostBreach != 0 {
		p.AllowCostBreach = 1
	}
}

// PrioritizesMargin 表示按毛利优先挑线路。没有配置行（nil）时也走这个默认口径。
func (p *DiscountRoutingPolicy) PrioritizesMargin() bool {
	return p == nil || p.RoutingStrategy != RoutingStrategyPriority
}

// AllowsCostBreach 表示这个客户被允许走会亏本的线路。没有配置行（nil）时不允许。
func (p *DiscountRoutingPolicy) AllowsCostBreach() bool {
	return p != nil && p.AllowCostBreach == 1
}

// GetDiscountRoutingPolicy 按客户读路由策略。这个客户没有配置过时返回 (nil, nil)——
// 「没配过」不是错误，是绝大多数客户的常态，调用方据此走默认口径。
func GetDiscountRoutingPolicy(userId int) (*DiscountRoutingPolicy, error) {
	if userId <= 0 {
		return nil, nil
	}
	// 用 Find 而不是 First：没配过的客户占绝大多数，用 First 会给这种正常情况
	// 刷一屏 record not found 的错误日志（与 GetModelByName 同一个考虑）。
	var policies []DiscountRoutingPolicy
	if err := DB.Where("user_id = ?", userId).Limit(1).Find(&policies).Error; err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, nil
	}
	return &policies[0], nil
}

// ResolveCustomerRouting 读这个客户当前生效的路由策略。
//
// 没有配置行、客户 id 非法、查库出错都返回 nil，调用方一律按默认策略处理——这与
// ResolveBillingDiscount「读不到就按不打折」是同一个回退口径：一条例外配置读不出来，
// 不能把正常请求打挂。查库出错会记 SysError，不静默发生。
//
// 这个函数在选线路的热路径上，但只在客户真的享受折扣时才会被调到
// （BuildChannelCostFilter 先看 discount.Applied()），所以全价客户不会有这次查询。
func ResolveCustomerRouting(userId int) *DiscountRoutingPolicy {
	if userId <= 0 {
		return nil
	}
	policy, err := GetDiscountRoutingPolicy(userId)
	if err != nil {
		common.SysError(fmt.Sprintf("resolve customer routing policy failed, userId=%d: %s", userId, err.Error()))
		return nil
	}
	return policy
}

// SaveDiscountRoutingPolicy 写入（或更新）这个客户的路由策略。
// 已经有配置行时只改策略、开启状态、备注和操作人，不动创建时间。
func SaveDiscountRoutingPolicy(policy *DiscountRoutingPolicy) error {
	if policy == nil || policy.UserId <= 0 {
		return errors.New("user_id 必须是正整数")
	}
	existing, err := GetDiscountRoutingPolicy(policy.UserId)
	if err != nil {
		return err
	}
	if existing == nil {
		policy.NormalizeDefaults()
		return DB.Create(policy).Error
	}
	existing.RoutingStrategy = policy.RoutingStrategy
	existing.AllowCostBreach = policy.AllowCostBreach
	existing.Remark = policy.Remark
	existing.UpdatedBy = policy.UpdatedBy
	return DB.Save(existing).Error
}

// DeleteDiscountRoutingPolicy 删掉这个客户的例外配置，回到默认口径（毛利优先、不允许击穿）。
func DeleteDiscountRoutingPolicy(userId int) error {
	if userId <= 0 {
		return errors.New("user_id 必须是正整数")
	}
	return DB.Where("user_id = ?", userId).Delete(&DiscountRoutingPolicy{}).Error
}
