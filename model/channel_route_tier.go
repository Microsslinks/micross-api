package model

import (
	"fmt"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// channelRouteTier 是「本次挑线路的一层候选」：同层内按权重随机分摊，层与层之间的顺序
// 就是重试顺序（retry=0 用第一层，失败后逐层往下）。
//
// 毛利优先策略下，层由（进货折扣, 上游供应商优先级）两级键决定，于是「毛利相同的算一层、
// 层内仍走原有的优先级分层 + 权重随机」——既优先毛利高的线路，又不会把流量全压到一条线上
// （压一条的话它一挂全断，口径见 .docs/task-02-business-goals/04-cost-aware-routing.md §6）。
// 稳定优先策略下只有优先级一级键，等同于改造前的行为。
type channelRouteTier struct {
	costRatio decimal.Decimal // 这一层的进货折扣；costKnown 为 false 时是零值，没有意义
	costKnown bool
	priority  int64
	channels  []int
}

// sameRouteTier 判断一条线路能否并进当前这一层。
//
// 上游优先级永远参与判断；进货折扣与"成本是否已知"**只在毛利优先时**参与——稳定优先
// （以及根本没有折扣的老客户，过滤器为 nil）下，同一个优先级就是一层、层内按权重随机，
// 与改造前完全一致（口径见 buildChannelRouteTiers 的说明与 02-work-queue.md §八 L4）。
//
// 注意：稳定优先时 tier.costRatio／costKnown 只是层里第一条线路带出来的值，不构成分层键，
// 不要拿它当"这一层的成本"用。
func sameRouteTier(tier channelRouteTier, priority int64, costRatio decimal.Decimal, costKnown bool, prioritizeMargin bool) bool {
	if tier.priority != priority {
		return false
	}
	if !prioritizeMargin {
		return true
	}
	if tier.costKnown != costKnown {
		return false
	}
	return !costKnown || tier.costRatio.Equal(costRatio)
}

// buildChannelRouteTiers 把候选线路切成有序的若干层。返回的层序就是重试顺序。
//
// 分层键跟着策略走：**毛利优先**时是（进货折扣, 上游优先级）两级键；**稳定优先**时只有
// 优先级一级键——后者等同于改造前的行为。
//
// 因此**非「毛利优先」时绝不能按进货折扣拆层**：那会把"同一个优先级按权重分摊"变成
// "固定只走排在前面的那一条"（流量压偏，一条挂掉就是一批请求失败）；更要紧的是，
// **根本没有折扣的老客户**（costFilter 为 nil）也走这条路径，拆层就等于改了老客户的行为，
// 破坏"未绑方案的用户行为与改造前逐笔一致"这条底线（详见 02-work-queue.md §八 L4）。
//
// 调用方必须持有 channelSyncLock（读锁）：本函数只读 channelsIDM，不做任何数据库访问。
// 传入的 channels 切片不会被改动（排序在副本上做）。
func buildChannelRouteTiers(channels []int, costFilter *ChannelCostFilter) ([]channelRouteTier, error) {
	if len(channels) == 0 {
		return nil, nil
	}
	prioritizeMargin := costFilter != nil && costFilter.PrioritizeMargin

	type candidate struct {
		channel   *Channel
		costRatio decimal.Decimal
		costKnown bool
		priority  int64
	}
	candidates := make([]candidate, 0, len(channels))
	for _, channelId := range channels {
		channel, ok := channelsIDM[channelId]
		if !ok {
			return nil, fmt.Errorf("数据库一致性错误，渠道# %d 不存在，请联系管理员修复", channelId)
		}
		costRatio, costKnown := channelCostRatio(channelId)
		candidates = append(candidates, candidate{
			channel:   channel,
			costRatio: costRatio,
			costKnown: costKnown,
			priority:  channel.GetPriority(),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if prioritizeMargin {
			// 没录进货折扣的线路排最后：它成本未知，不能因为「读不到」就当它最便宜。
			// 正常路径上这种线路已被 filterChannelsByCost 剔掉，只有放行模式才会走到这儿。
			if candidates[i].costKnown != candidates[j].costKnown {
				return candidates[i].costKnown
			}
			if cmp := candidates[i].costRatio.Cmp(candidates[j].costRatio); cmp != 0 {
				return cmp < 0
			}
		}
		return candidates[i].priority > candidates[j].priority
	})

	tiers := make([]channelRouteTier, 0, len(candidates))
	for _, c := range candidates {
		last := len(tiers) - 1
		if last >= 0 && sameRouteTier(tiers[last], c.priority, c.costRatio, c.costKnown, prioritizeMargin) {
			tiers[last].channels = append(tiers[last].channels, c.channel.Id)
			continue
		}
		tiers = append(tiers, channelRouteTier{
			costRatio: c.costRatio,
			costKnown: c.costKnown,
			priority:  c.priority,
			channels:  []int{c.channel.Id},
		})
	}
	return tiers, nil
}

// describeCostBreach 组装「当前折扣下没有不亏本的线路」的报错。
//
// 业务方要求这条错误里说清三件事：① 为什么失败；② 还有哪条线路能走；③ 走它平台会亏多少
// （.docs/task-02-business-goals/04-cost-aware-routing.md §7）。三件事都在这一条错误里给出，
// 客户和运营都不用再去翻别的地方。
//
// 调用方必须持有 channelSyncLock（读锁）。
func describeCostBreach(group, model string, channels []int, costFilter *ChannelCostFilter) error {
	sellRatio := decimal.NewFromFloat(costFilter.SellRatio)

	// 按进货折扣升序排：先列亏得最少的，运营真要补进货价也先补这一条。
	type losingLine struct {
		name      string
		costRatio decimal.Decimal
	}
	lines := make([]losingLine, 0, len(channels))
	unknownCostCount := 0
	for _, channelId := range channels {
		channel, ok := channelsIDM[channelId]
		if !ok {
			continue
		}
		costRatio, costKnown := channelCostRatio(channelId)
		if !costKnown {
			// 未录进货折扣的线路对折扣客户本就不可用（文档 §3 的 S1 口径）。这种要单独说一句，
			// 否则运营看到「明明有线路却报没有」会以为是系统坏了。
			unknownCostCount++
			continue
		}
		lines = append(lines, losingLine{name: channel.Name, costRatio: costRatio})
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

// formatRatioAsDiscount 把 0.55 这样的比例写成折扣口径的「5.5」（即中文习惯的 5.5 折）。
//
// 乘 10 而不是乘 100：折扣的「折」是十等分，0.55 是 5.5 折、不是 55 折。
// 末尾无意义的零要去掉（0.60 → 6），但不能用 TrimRight 砍字符串——那样「2.00」
// 会被砍成「2.」再砍成「2」，把 2 折写成 2 折碰巧对，但「10.00」会被砍成「1」。
// decimal 的 Round 后再 String 正好就是要的形态；SellRatio 来自 float64，
// Round(4) 是先把浮点尾巴（5.500000000000001 这种）收掉，免得漏进给客户看的文案里。
func formatRatioAsDiscount(ratio decimal.Decimal) string {
	return ratio.Mul(decimal.NewFromInt(10)).Round(4).String()
}

// formatLossRatio 把亏损比例写成「3.0 个百分点」这种直白说法。
// 用差额口径（进货折扣 − 售价折扣），与成本过滤用的口径一致，不与试算页的相对毛利混用。
func formatLossRatio(loss decimal.Decimal) string {
	if loss.LessThan(decimal.Zero) {
		loss = decimal.Zero
	}
	return loss.Mul(decimal.NewFromInt(100)).Round(1).StringFixed(1) + " 个百分点"
}
