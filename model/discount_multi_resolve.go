package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 多方案解析（task-06 第一步：纯读取，不动任何写入路径）。
//
// 单方案解析（discount_resolve.go）只认 users.discount_plan_id 指向的那一条绑定，
// 同一个客户身上其余生效绑定谈好的价全部静默失效。这里换一种读法：
// 把这个人所有生效绑定逐个对当前模型解析出一个折扣，再按五层裁决出一个答案——
//
//	1. 具体度：模型级规则 > 厂商级规则 > 方案基础折扣（谁把话说得更具体就用谁的价）
//	2. 来源：manual > agent > subscription > customer_code > migration（一样具体看是谁定的）
//	3. 价格：更便宜的赢（同具体度同来源才走到这一层）
//	4. 时间：更新的绑定赢（绑定 id 更大）
//	5. 最后仍与经销商自己消费的拿货价比一次便宜（沿用 applyAgentWholesaleDiscount，保持现状）
//
// 计费入口已指向 ResolveUserDiscountMulti；试算与保存前校验跟进迁移后，
// 「折扣口径只能有一份」的原则落在这一族函数上。
//
// 现状（每人至多一条生效绑定）下两种读法答案一致；界面放开多挂之后
// 这里的裁决就是唯一计价口径，落选候选与原因保留在返回值里，
// 供后续「一键查价」和消费日志解释"为什么是这个价"。

// DiscountCandidate 是一条绑定对「当前模型」的解析结果。
// Rejected 为 true 时 RejectReason 写明它输在哪一层（或根本没进赛场的原因）。
type DiscountCandidate struct {
	Binding      *DiscountBinding // 这条价是谁、通过什么渠道挂上来的
	Plan         *DiscountPlan    // 方案停用或已删除时为空
	Rule         *DiscountRule    // 命中模型级 / 厂商级规则时非空
	Specificity  string           // DiscountResolvedFromModel / Vendor / PlanBase，即落到哪一档
	Discount     string           // 这条绑定对当前模型的折扣，6 位小数
	Rejected     bool
	RejectReason string
}

// ResolveUserDiscountMulti 算出「这个客户 + 这个模型」该按几折（多方案版）。
// 返回形态与 ResolveUserDiscount 完全一致，计费链路可以无缝换用。
func ResolveUserDiscountMulti(userId int, modelName string) (*DiscountResolution, error) {
	resolution, _, err := ResolveUserDiscountDetailed(userId, modelName)
	return resolution, err
}

// ResolveUserDiscountDetailed 在多方案答案之外把全部候选（含落选者与落选原因）一并返回，
// 供查价 / 试算解释"这个价怎么来的"。candidates 可能为空（没有任何绑定）。
func ResolveUserDiscountDetailed(userId int, modelName string) (*DiscountResolution, []*DiscountCandidate, error) {
	resolution, candidates, err := resolveMultiPlanDiscount(userId, modelName)
	if err != nil {
		return nil, nil, err
	}
	resolution, err = applyAgentWholesaleDiscount(resolution, userId, modelName)
	if err != nil {
		return nil, nil, err
	}
	return resolution, candidates, nil
}

// resolveMultiPlanDiscount 只看折扣方案这一层（多方案版的 resolvePlanDiscount）：
// 读全部生效绑定 → 批量取方案与规则 → 逐条解析 → 五层裁决（前四层，拿货价在外层）。
func resolveMultiPlanDiscount(userId int, modelName string) (*DiscountResolution, []*DiscountCandidate, error) {
	resolution := &DiscountResolution{
		Discount: DiscountNone,
		Source:   DiscountResolvedFromDefault,
	}
	if userId <= 0 {
		return resolution, nil, nil
	}

	// 用户不存在视同没有绑定方案；数据库真出错必须报上去，不能悄悄按 1.0 计费。
	// 与单方案解析同一口径。
	var user User
	if err := DB.Select("id").First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resolution, nil, nil
		}
		return nil, nil, err
	}

	// 绑定读取与 syncUserDiscountPlanId 同一口径：只按 subject_id + 生效状态过滤，
	// 不区分 subject_type（user / agent 两种主体都落在 users.id 上，快路径向来一视同仁）。
	allBindings := make([]DiscountBinding, 0)
	if err := DB.Where("subject_id = ? AND status = ?", userId, DiscountStatusEnabled).
		Order("id ASC").Find(&allBindings).Error; err != nil {
		return nil, nil, err
	}

	now := common.GetTimestamp()
	candidates := make([]*DiscountCandidate, 0, len(allBindings))
	planIds := make([]int, 0, len(allBindings))
	for i := range allBindings {
		binding := &allBindings[i]
		candidate := &DiscountCandidate{Binding: binding}
		// 生效时间窗口的判定与 pickActiveDiscountBinding 完全一致；
		// 窗口外的绑定仍出现在候选里（标明原因），方便回答"我上周谈的那套价怎么没生效"。
		if binding.EffectiveFrom > now {
			candidate.Rejected = true
			candidate.RejectReason = "尚未到生效时间"
			candidates = append(candidates, candidate)
			continue
		}
		if binding.EffectiveTo > 0 && binding.EffectiveTo <= now {
			candidate.Rejected = true
			candidate.RejectReason = "已过生效时间"
			candidates = append(candidates, candidate)
			continue
		}
		planIds = append(planIds, binding.PlanId)
		candidates = append(candidates, candidate)
	}
	if len(planIds) == 0 {
		return resolution, candidates, nil
	}

	// 方案与规则各一次 IN 查询批量取回：计费热路径绝不能逐条查。
	plans, err := getDiscountPlansByIds(planIds)
	if err != nil {
		return nil, nil, err
	}
	rulesByPlan, err := getDiscountRulesByPlanIds(planIds)
	if err != nil {
		return nil, nil, err
	}

	// 逐条解析：方案内仍是 模型级规则 → 厂商级规则 → 基础折扣，
	// 与单方案解析（resolvePlanDiscount 第 4-6 步）同一套优先级。
	// 厂商名只在有候选真需要它（模型级规则没命中）时才解析——查的是模型目录表，
	// 单方案解析同样在这一步才碰它，模型级命中时压根不查，这里保持同一节奏。
	vendorName := ""
	vendorResolved := false
	for _, candidate := range candidates {
		if candidate.Rejected {
			continue
		}
		plan, ok := plans[candidate.Binding.PlanId]
		if !ok || plan.Status != DiscountStatusEnabled {
			// 方案停用或已删除：这条绑定对当前模型没有发言权。
			candidate.Rejected = true
			candidate.RejectReason = "方案已停用或已删除"
			continue
		}
		candidate.Plan = plan
		rules := rulesByPlan[plan.Id]
		if rule := pickDiscountRule(rules, DiscountScopeModel, modelName); rule != nil {
			candidate.Specificity = DiscountResolvedFromModel
			candidate.Rule = rule
			candidate.Discount = formatResolvedDiscount(rule.Discount)
			continue
		}
		if !vendorResolved {
			vendorName, err = resolveVendorName(modelName)
			if err != nil {
				return nil, nil, err
			}
			vendorResolved = true
		}
		if rule := pickDiscountRule(rules, DiscountScopeVendor, vendorName); rule != nil {
			candidate.Specificity = DiscountResolvedFromVendor
			candidate.Rule = rule
			candidate.Discount = formatResolvedDiscount(rule.Discount)
			continue
		}
		candidate.Specificity = DiscountResolvedFromPlanBase
		candidate.Discount = formatResolvedDiscount(plan.BaseDiscount)
	}

	winner := pickWinningDiscountCandidate(candidates)
	if winner == nil {
		// 所有绑定都被停用方案拖下水：按未绑定处理。快路径口径保持
		// "仍指向 pickActiveDiscountBinding 选中的那条"，与单方案解析一致，
		// 便于排查为什么没生效。
		if binding := pickActiveDiscountBinding(allBindings, now); binding != nil {
			resolution.PlanId = binding.PlanId
		}
		return resolution, candidates, nil
	}
	markDiscountCandidateRejections(candidates, winner)

	resolution.Discount = winner.Discount
	resolution.Source = winner.Specificity
	resolution.Rule = winner.Rule
	resolution.Plan = winner.Plan
	resolution.PlanId = winner.Binding.PlanId
	return resolution, candidates, nil
}

// pickWinningDiscountCandidate 按前四层裁决挑出唯一胜者。
// 落选原因的标注在 markDiscountCandidateRejections 里另算，两处用同一套层。
func pickWinningDiscountCandidate(candidates []*DiscountCandidate) *DiscountCandidate {
	var winner *DiscountCandidate
	for _, candidate := range candidates {
		if candidate.Rejected {
			continue
		}
		if winner == nil || compareDiscountCandidates(candidate, winner) < 0 {
			winner = candidate
		}
	}
	return winner
}

// compareDiscountCandidates 五层裁决的前四层：负数表示 a 赢。
// 具体度 → 来源 → 价格（便宜赢）→ 绑定新旧（新赢）。
func compareDiscountCandidates(a *DiscountCandidate, b *DiscountCandidate) int {
	if r := discountSpecificityRank(a.Specificity) - discountSpecificityRank(b.Specificity); r != 0 {
		return r
	}
	if r := discountSourceRank(a.Binding.Source) - discountSourceRank(b.Binding.Source); r != 0 {
		return r
	}
	if c := compareDiscountStrings(a.Discount, b.Discount); c != 0 {
		return c
	}
	return b.Binding.Id - a.Binding.Id
}

// markDiscountCandidateRejections 给落选候选写明输在哪一层，
// 判层顺序与 compareDiscountCandidates 完全一致，保证原因和结果对得上。
func markDiscountCandidateRejections(candidates []*DiscountCandidate, winner *DiscountCandidate) {
	for _, candidate := range candidates {
		if candidate == winner {
			continue
		}
		switch {
		case candidate.Rejected:
			// 没进赛场的原因（窗口外 / 方案停用）已经写好。
		case discountSpecificityRank(candidate.Specificity) > discountSpecificityRank(winner.Specificity):
			candidate.Rejected = true
			candidate.RejectReason = "具体度不如胜者：胜者命中了更具体的规则"
		case discountSourceRank(candidate.Binding.Source) > discountSourceRank(winner.Binding.Source):
			candidate.Rejected = true
			candidate.RejectReason = "同具体度下定价方优先级低于胜者"
		case compareDiscountStrings(candidate.Discount, winner.Discount) > 0:
			candidate.Rejected = true
			candidate.RejectReason = "同具体度同来源下不如胜者便宜"
		default:
			candidate.Rejected = true
			candidate.RejectReason = "同具体度同来源同价格下绑定早于胜者"
		}
	}
}

// discountSpecificityRank 越小越具体；模型级 > 厂商级 > 方案基础折扣。
func discountSpecificityRank(specificity string) int {
	switch specificity {
	case DiscountResolvedFromModel:
		return 0
	case DiscountResolvedFromVendor:
		return 1
	default:
		return 2
	}
}

// compareDiscountStrings 比较两个折扣字符串，负数表示 a 更便宜。
// 解析不出来的折扣没有资格赢（让它落选），两个都坏算平手交给下一层。
func compareDiscountStrings(a string, b string) int {
	valueA, errA := decimal.NewFromString(strings.TrimSpace(a))
	valueB, errB := decimal.NewFromString(strings.TrimSpace(b))
	switch {
	case errA != nil && errB != nil:
		return 0
	case errA != nil:
		return 1
	case errB != nil:
		return -1
	}
	return valueA.Compare(valueB)
}

// getDiscountPlansByIds 批量取方案，一次 IN 查询；结果里没有的 id 即方案已删除。
func getDiscountPlansByIds(ids []int) (map[int]*DiscountPlan, error) {
	plans := make([]*DiscountPlan, 0, len(ids))
	if err := DB.Where("id IN ?", ids).Find(&plans).Error; err != nil {
		return nil, err
	}
	result := make(map[int]*DiscountPlan, len(plans))
	for _, plan := range plans {
		plan.NormalizeDefaults()
		result[plan.Id] = plan
	}
	return result, nil
}

// getDiscountRulesByPlanIds 批量取规则，一次 IN 查询。
// 排序与 GetDiscountRulesByPlanId 相同（priority DESC, id ASC），
// 同一层级里第一条命中的就是该用的那条。
func getDiscountRulesByPlanIds(planIds []int) (map[int][]*DiscountRule, error) {
	rules := make([]*DiscountRule, 0)
	if err := DB.Where("plan_id IN ?", planIds).Order("priority DESC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	result := make(map[int][]*DiscountRule, len(planIds))
	for _, rule := range rules {
		result[rule.PlanId] = append(result[rule.PlanId], rule)
	}
	return result, nil
}
