package model

import (
	"fmt"
	"sort"
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
	// PrioritizeMargin 表示保本线路有多条时按毛利从高到低挑（默认）。
	// 为 false 时按上游供应商优先级挑（「稳定／质量优先」），即改造前的行为。
	PrioritizeMargin bool
	// AllowCostBreach 表示这个客户被允许走亏损线路：一条保本线路都没有时放行，
	// 而不是报错。默认 false——平台不能自己决定「亏就亏吧」（文档 §7）。
	AllowCostBreach bool
	// Breach 是放行时记下的击穿事实，供消费日志留痕；没放行过就是 nil。
	Breach *CostBreach
}

// CostBreach 是一次「客户已同意、平台认赔」的留痕：这一单实际走了哪条亏本线路、亏多少。
// 它以 other.admin_info.cost_breach 进消费日志，而 admin_info 对非管理员整块不可见，
// 所以客户看不到自己的「平台亏损」标记，管理员对账时看得到。
type CostBreach struct {
	ChannelId   int    `json:"channel_id"`
	ChannelName string `json:"channel_name"`
	// SellRatio / CostRatio 都是 6 位小数字符串，与库里的折扣口径一致。
	SellRatio string `json:"sell_ratio"`
	CostRatio string `json:"cost_ratio"`
	// LossRatio 是每单亏损比例（进货折扣 − 售价折扣，正数表示亏）。
	LossRatio string `json:"loss_ratio"`
}

// Enabled 表示这道筛子是否真的动手。
func (f *ChannelCostFilter) Enabled() bool {
	return f != nil && f.SellRatio > 0 && f.SellRatio < 1
}

// NewChannelCostFilter 组成成本过滤器。毛利底线读运营设置，空串或读不出来按 0
// （即「不赔本就行」，与试算／保存前校验同一口径）。
//
// policy 是这个客户的路由策略，没有配置行时传 nil，按默认口径处理：毛利优先、
// 不允许走亏损线路。
func NewChannelCostFilter(sellRatio float64, policy *DiscountRoutingPolicy) *ChannelCostFilter {
	filter := &ChannelCostFilter{
		SellRatio:        sellRatio,
		PrioritizeMargin: policy.PrioritizesMargin(),
		AllowCostBreach:  policy.AllowsCostBreach(),
	}
	if setting := operation_setting.GetDiscountSetting(); setting != nil {
		parsed, err := decimal.NewFromString(strings.TrimSpace(setting.MinMarginRatio))
		if err == nil && parsed.GreaterThan(decimal.Zero) {
			filter.MinMarginRatio = parsed.InexactFloat64()
		}
	}
	return filter
}

// recordBreach 记下「这一单走了亏本线路」。
//
// 只在「一条保本线路都没有 + 这个客户被允许走亏损线路」的放行分支里调用。记在过滤器上
// 而不是直接写日志，是因为选线路逻辑拿不到 gin.Context，也不该知道日志长什么样；由调用方
// 转交给消费日志的 other.admin_info.cost_breach。
//
// 调用方必须持有 channelSyncLock（读锁）：本函数只读传入 channel 上的 CostRatio，不
// 再访问 channelsIDM；缓存关闭场景下 channelsIDM 为空也能用（task-14.2）。
func (f *ChannelCostFilter) recordBreach(channel *Channel) {
	if f == nil || channel == nil || channel.CostRatio == nil {
		return
	}
	raw := strings.TrimSpace(*channel.CostRatio)
	if raw == "" {
		return
	}
	costRatio, err := decimal.NewFromString(raw)
	if err != nil || costRatio.LessThanOrEqual(decimal.Zero) {
		// 未录进货折扣的线路本来就被剔掉，放行模式里仍可能出现；算不出亏损额就不记。
		return
	}
	sellRatio := decimal.NewFromFloat(f.SellRatio)
	f.Breach = &CostBreach{
		ChannelId:   channel.Id,
		ChannelName: channel.Name,
		SellRatio:   sellRatio.StringFixed(6),
		CostRatio:   costRatio.StringFixed(6),
		LossRatio:   costRatio.Sub(sellRatio).StringFixed(6),
	}
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

// filterChannelsByCostDB 与 filterChannelsByCost 同语义，但成本数据从 DB 读，
// 不依赖 channelsIDM / channelSyncLock——缓存关闭（MemoryCacheEnabled=false）
// 场景下也能用（task-14.2）。
//
// 调用方无需持锁。本函数只读，不会改候选顺序；过滤口径与内存版完全一致：
// 未录、空串、非数字、≤ 0 一律按「没有成本信息」处理并排除。
func filterChannelsByCostDB(channels []int, costFilter *ChannelCostFilter) ([]int, error) {
	if !costFilter.Enabled() || len(channels) == 0 {
		return channels, nil
	}
	var rows []*Channel
	if err := DB.Select("id", "cost_ratio").Where("id IN ?", channels).Find(&rows).Error; err != nil {
		return nil, err
	}
	costMap := make(map[int]*string, len(rows))
	for _, c := range rows {
		costMap[c.Id] = c.CostRatio
	}
	budget := costFilter.budget()
	kept := make([]int, 0, len(channels))
	for _, id := range channels {
		ptr, ok := costMap[id]
		if !ok || ptr == nil {
			continue
		}
		raw := strings.TrimSpace(*ptr)
		if raw == "" {
			continue
		}
		parsed, err := decimal.NewFromString(raw)
		if err != nil || parsed.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if parsed.LessThanOrEqual(budget) {
			kept = append(kept, id)
		}
	}
	return kept, nil
}

// describeCostBreachDB 与 describeCostBreach 同语义，但数据从 DB 读，给缓存关闭
// 场景下「有线路但不保本」时的报错信息用（task-14.2）。
//
// 列线路名时按进货折扣升序——先列亏得最少的，运营补进货价时优先看这条。
// 未录进货折扣的条数单独统计并明示，避免运营误以为是「系统坏了」。
func describeCostBreachDB(group, model string, channels []int, costFilter *ChannelCostFilter) error {
	sellRatio := decimal.NewFromFloat(costFilter.SellRatio)

	var rows []*Channel
	if err := DB.Select("id", "name", "cost_ratio").Where("id IN ?", channels).Find(&rows).Error; err != nil {
		return err
	}
	type losingLine struct {
		name      string
		costRatio decimal.Decimal
	}
	lines := make([]losingLine, 0, len(rows))
	unknownCostCount := 0
	for _, c := range rows {
		if c.CostRatio == nil {
			unknownCostCount++
			continue
		}
		raw := strings.TrimSpace(*c.CostRatio)
		if raw == "" {
			unknownCostCount++
			continue
		}
		costRatio, err := decimal.NewFromString(raw)
		if err != nil || costRatio.LessThanOrEqual(decimal.Zero) {
			unknownCostCount++
			continue
		}
		lines = append(lines, losingLine{name: c.Name, costRatio: costRatio})
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].costRatio.LessThan(lines[j].costRatio) })

	const maxListedLines = 3
	parts := make([]string, 0, maxListedLines+2)
	for i, line := range lines {
		if i >= maxListedLines {
			parts = append(parts, fmt.Sprintf("另有 %d 条线路同样亏本", len(lines)-maxListedLines))
			break
		}
		parts = append(parts, fmt.Sprintf("走「%s」（进货 %s 折）每单亏 %s",
			line.name, formatRatioAsDiscount(line.costRatio), formatLossRatio(line.costRatio.Sub(sellRatio))))
	}
	if unknownCostCount > 0 {
		parts = append(parts, fmt.Sprintf("另有 %d 条线路未录进货折扣，无法判断是否保本", unknownCostCount))
	}
	if len(parts) == 0 {
		parts = append(parts, "该模型下没有可用的线路")
	}

	return fmt.Errorf("当前折扣（%s 折）下没有不亏本的上游供应商线路（模型 %s，分组 %s）：%s。如需继续使用，请联系管理员为这个账号开通「允许走亏损线路」",
		formatRatioAsDiscount(sellRatio), model, group, strings.Join(parts, "、"))
}
