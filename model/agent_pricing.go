package model

import (
	"errors"

	"github.com/shopspring/decimal"
)

// 经销商的货架：他能拿去卖给客户的折扣方案。
//
// 平台方案（owner_type = platform）就是他的货；他自己名下的方案（owner_type = agent）
// 是他自己组价用的，不在这个列表里重复出现。
//
// 最要紧的一条：平台给他设了零售折扣下限（agent_profiles.min_discount）时，
// 低于下限的方案不给挑——那个价他卖出去就是亏的，所以干脆不出现在货架上，
// 而不是等他选完了再报错。

// AgentSellablePlan 是货架上的一件货：只带经销商挑方案时看得懂的字段。
type AgentSellablePlan struct {
	Id           int    `json:"id"`
	Name         string `json:"name"`
	BaseDiscount string `json:"base_discount"`
	BillingMode  string `json:"billing_mode"`
	Remark       string `json:"remark"`
}

// ListSellablePlansForAgent 列出这位经销商此刻能卖的方案，并把他自己的零售折扣下限一并返回，
// 好让界面把「为什么少了几个方案」讲清楚。
func ListSellablePlansForAgent(agentId int) (string, []*AgentSellablePlan, error) {
	if agentId <= 0 {
		return "0", nil, errors.New("无效的经销商 ID")
	}

	// 没有档案也能拿这个列表（下限按 0 即不限），因为能不能发号另有一道身份判断。
	floor := "0"
	profile, err := GetAgentProfileByUserId(agentId)
	if err != nil && !errors.Is(err, ErrAgentProfileNotFound) {
		return floor, nil, err
	}
	if profile != nil {
		profile.NormalizeDefaults()
		floor = profile.MinDiscount
	}

	var plans []*DiscountPlan
	if err := DB.Where("owner_type = ? AND status = ?", DiscountOwnerPlatform, DiscountStatusEnabled).
		Order("id DESC").Find(&plans).Error; err != nil {
		return floor, nil, err
	}

	// 折扣比较放在 Go 里做：三个数据库对 decimal 的类型转换各不相同，
	// 写进 SQL 会一边能跑一边报错。
	floorValue, err := decimal.NewFromString(floor)
	if err != nil {
		floorValue = decimal.Zero
	}

	sellable := make([]*AgentSellablePlan, 0, len(plans))
	for _, plan := range plans {
		plan.NormalizeDefaults()
		base, err := decimal.NewFromString(plan.BaseDiscount)
		if err != nil {
			continue
		}
		if floorValue.GreaterThan(decimal.Zero) && base.LessThan(floorValue) {
			continue
		}
		sellable = append(sellable, &AgentSellablePlan{
			Id:           plan.Id,
			Name:         plan.Name,
			BaseDiscount: plan.BaseDiscount,
			BillingMode:  plan.BillingMode,
			Remark:       plan.Remark,
		})
	}
	return floor, sellable, nil
}
