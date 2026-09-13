package model

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

// ChannelCostFilter 是路由侧「成本过滤」的入参：把这一单的售价折扣与平台
// 毛利底线交给选线路逻辑，用来剔掉会亏本的候选线路。
//
// 只在客户真的享受折扣（0 < SellRatio < 1）时才动手：折扣为 1 的客户
// （没绑方案、方案停用、总开关关闭）走原逻辑，候选集一条不动——这样
// 「没绑方案的用户行为与改造前逐笔一致」这条底线在路由侧同样成立。
type ChannelCostFilter struct {
	// SellRatio 是这一单的售价折扣（客户折扣），来自 ResolveBillingDiscount。
	SellRatio float64
	// MinMarginRatio 是平台毛利底线，来自运营设置 discount_setting.min_margin_ratio。
	MinMarginRatio float64
}

// Enabled 表示这道筛子是否真的动手。
func (f *ChannelCostFilter) Enabled() bool {
	return f != nil && f.SellRatio > 0 && f.SellRatio < 1
}

// NewChannelCostFilter 组成成本过滤器。毛利底线读运营设置，空串或读不出来按 0
// （即「不赔本就行」，与试算／保存前校验同一口径）。
func NewChannelCostFilter(sellRatio float64) *ChannelCostFilter {
	filter := &ChannelCostFilter{SellRatio: sellRatio}
	if setting := operation_setting.GetDiscountSetting(); setting != nil {
		parsed, err := decimal.NewFromString(strings.TrimSpace(setting.MinMarginRatio))
		if err == nil && parsed.GreaterThan(decimal.Zero) {
			filter.MinMarginRatio = parsed.InexactFloat64()
		}
	}
	return filter
}

// budget 是这一单允许的最高进货折扣：售价折扣减掉毛利底线。
func (f *ChannelCostFilter) budget() decimal.Decimal {
	return decimal.NewFromFloat(f.SellRatio).Sub(decimal.NewFromFloat(f.MinMarginRatio))
}

// ChannelPassesCostBudget 判断某一条线路在给定成本约束下是否可用。
//
// 给「旁路」复用主选线路的筛子用：粘连（channel affinity）会绕过候选集挑选，
// 命中后必须回头补一次同样的成本校验，否则折扣客户会被粘连钉在亏损线路上
// （口径见 .docs/task-02-business-goals/04-cost-aware-routing.md §4.3）。
// 过滤器未启用时一律放行，行为与改造前一致。
func ChannelPassesCostBudget(channelId int, costFilter *ChannelCostFilter) bool {
	if !costFilter.Enabled() {
		return true
	}
	channelSyncLock.RLock()
	defer channelSyncLock.RUnlock()
	cost, ok := channelCostRatio(channelId)
	return ok && cost.LessThanOrEqual(costFilter.budget())
}

// filterChannelsByCost 只保留「进货折扣 ≤ 售价折扣 − 毛利底线」的候选线路。
//
// 未录进货折扣（列是 NULL 或空串）、写成非数字、或 ≤ 0 的线路，一律按
// 「没有成本信息」处理并排除——口径见 .docs/task-02-business-goals/
// 04-cost-aware-routing.md §3，三种策略里选的是保守的 S1。这意味着运营没把
// 进货价录全之前，折扣客户只能走「已录进货价且保本」的线路。
//
// 调用方必须持有 channelSyncLock（读锁）：本函数只读 channelsIDM，
// 不做任何数据库访问，因此在锁内也是安全的。
func filterChannelsByCost(channels []int, costFilter *ChannelCostFilter) []int {
	if !costFilter.Enabled() || len(channels) == 0 {
		return channels
	}
	budget := costFilter.budget()
	kept := make([]int, 0, len(channels))
	for _, channelId := range channels {
		cost, ok := channelCostRatio(channelId)
		if !ok {
			continue
		}
		if cost.LessThanOrEqual(budget) {
			kept = append(kept, channelId)
		}
	}
	return kept
}

// channelCostRatio 读出某条线路录的进货折扣。未录、读不出或非正数时返回 false。
func channelCostRatio(channelId int) (decimal.Decimal, bool) {
	channel, ok := channelsIDM[channelId]
	if !ok || channel == nil || channel.CostRatio == nil {
		return decimal.Zero, false
	}
	raw := strings.TrimSpace(*channel.CostRatio)
	if raw == "" {
		return decimal.Zero, false
	}
	parsed, err := decimal.NewFromString(raw)
	if err != nil || parsed.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero, false
	}
	return parsed, true
}
