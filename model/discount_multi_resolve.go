package model

import (
	"errors"
	"fmt"
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
// 模型名为空串时返回 (nil, nil, nil)，计费侧据此按官方标价收——空名字没有可解析的价。
func ResolveUserDiscountDetailed(userId int, modelName string) (*DiscountResolution, []*DiscountCandidate, error) {
	modelName = strings.TrimSpace(modelName)
	resolutions, candidates, err := ResolveUserDiscountDetailedForModels(userId, []string{modelName})
	if err != nil {
		return nil, nil, err
	}
	return resolutions[modelName], candidates[modelName], nil
}

// ResolveUserDiscountDetailedForModels 批量版：一个客户 + 一份模型清单一次算完，
// 绑定 / 方案 / 规则 / 厂商 / 拿货价各读一次（不是每个模型来一遍），
// 逐模型给出与单模型版完全一致的答案与候选——两条路共用同一套解析与裁决代码，
// 不允许出现「核算一张价、扣费另一张价」。
// 给「客户档案核算」这类管理端一次性报表用；计费热路径继续走单模型版。
// 返回的两个 map 按修剪去重后的模型名索引，输入顺序不保留。
func ResolveUserDiscountDetailedForModels(userId int, modelNames []string) (map[string]*DiscountResolution, map[string][]*DiscountCandidate, error) {
	resolutions, candidates, err := resolveMultiPlanDiscountForModels(userId, modelNames)
	if err != nil {
		return nil, nil, err
	}
	// 拿货价批量算一次，再逐模型合入——合入规则与单模型版完全一致（applyWholesaleToResolution）。
	wholesales, err := ResolveAgentWholesaleForModels(userId, modelNames)
	if err != nil {
		// 拿货价是叠在方案折扣之上的一层，它自己出错不该把整份清单拖回原价：
		// 记一笔日志，全部照方案折扣算，与单模型版同一处置。
		common.SysError(fmt.Sprintf("解析经销商拿货价失败，客户 %d 本次核算按方案折扣：%v", userId, err))
		wholesales = nil
	}
	for modelName, resolution := range resolutions {
		resolutions[modelName] = applyWholesaleToResolution(resolution, wholesales[modelName])
	}
	return resolutions, candidates, nil
}

// resolveMultiPlanDiscount 单模型版（多方案版的 resolvePlanDiscount）。
// 就是批量核心套一个模型名，防止有人在这里另写一套顺序。
func resolveMultiPlanDiscount(userId int, modelName string) (*DiscountResolution, []*DiscountCandidate, error) {
	modelName = strings.TrimSpace(modelName)
	resolutions, candidates, err := resolveMultiPlanDiscountForModels(userId, []string{modelName})
	if err != nil {
		return nil, nil, err
	}
	return resolutions[modelName], candidates[modelName], nil
}

// discountMultiContext 一个客户 + 一份模型清单共用的读取结果。
// 批量与单模型两条路从同一批数据出发解析，保证答案永远一致。
type discountMultiContext struct {
	allBindings   []DiscountBinding // 这个客户的全部生效绑定（含窗口外的）
	windowReasons []string          // 与 allBindings 平行：非空表示该绑定不在生效窗口内（写明原因）
	plans         map[int]*DiscountPlan
	rulesByPlan   map[int][]*DiscountRule
	now           int64
}

// resolveMultiPlanDiscountForModels 批量核心（方案层）：读全部生效绑定 → 批量取方案与规则
// → 逐模型逐条解析 → 五层裁决的前四层（拿货价在外层 ResolveUserDiscountDetailedForModels）。
func resolveMultiPlanDiscountForModels(userId int, modelNames []string) (map[string]*DiscountResolution, map[string][]*DiscountCandidate, error) {
	nameList := normalizeLookupValues(modelNames)
	resolutions := make(map[string]*DiscountResolution, len(nameList))
	candidatesByModel := make(map[string][]*DiscountCandidate, len(nameList))
	if len(nameList) == 0 {
		return resolutions, candidatesByModel, nil
	}
	if userId <= 0 {
		for _, name := range nameList {
			resolutions[name] = &DiscountResolution{Discount: DiscountNone, Source: DiscountResolvedFromDefault}
			candidatesByModel[name] = nil
		}
		return resolutions, candidatesByModel, nil
	}

	ctx, err := loadDiscountMultiContext(userId)
	if err != nil {
		return nil, nil, err
	}

	// 厂商名只在「这批方案里真有启用中的厂商级规则」时才查模型目录——单模型版的老规矩：
	// 模型级命中压根不碰目录表（有些环境的目录表还没建，提前查会把整单拖回原价）。
	vendorNames := make(map[string]string, len(nameList))
	if ctx.hasVendorScopeRules() {
		vendorNames, err = GetVendorNamesByModelNames(nameList)
		if err != nil {
			return nil, nil, err
		}
	}

	for _, modelName := range nameList {
		resolutions[modelName], candidatesByModel[modelName] = ctx.resolveModel(modelName, vendorNames[modelName])
	}
	return resolutions, candidatesByModel, nil
}

// loadDiscountMultiContext 读一个客户的绑定（含生效窗口判定）与关联的方案、规则。
// 用户不存在视同没有绑定方案；数据库真出错必须报上去，不能悄悄按 1.0 计费。
// 与单方案解析同一口径。
func loadDiscountMultiContext(userId int) (*discountMultiContext, error) {
	ctx := &discountMultiContext{
		plans:       make(map[int]*DiscountPlan),
		rulesByPlan: make(map[int][]*DiscountRule),
		now:         common.GetTimestamp(),
	}
	var user User
	if err := DB.Select("id").First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ctx, nil
		}
		return nil, err
	}

	// 绑定读取与 syncUserDiscountPlanId 同一口径：只按 subject_id + 生效状态过滤，
	// 不区分 subject_type（user / agent 两种主体都落在 users.id 上，快路径向来一视同仁）。
	allBindings := make([]DiscountBinding, 0)
	if err := DB.Where("subject_id = ? AND status = ?", userId, DiscountStatusEnabled).
		Order("id ASC").Find(&allBindings).Error; err != nil {
		return nil, err
	}
	ctx.allBindings = allBindings

	// 生效时间窗口的判定与 pickActiveDiscountBinding 完全一致，对所有模型都一样，算一次就够；
	// 窗口外的绑定仍出现在候选里（标明原因），方便回答"我上周谈的那套价怎么没生效"。
	ctx.windowReasons = make([]string, len(allBindings))
	planIds := make([]int, 0, len(allBindings))
	for i := range allBindings {
		switch {
		case allBindings[i].EffectiveFrom > ctx.now:
			ctx.windowReasons[i] = "尚未到生效时间"
		case allBindings[i].EffectiveTo > 0 && allBindings[i].EffectiveTo <= ctx.now:
			ctx.windowReasons[i] = "已过生效时间"
		default:
			planIds = append(planIds, allBindings[i].PlanId)
		}
	}
	if len(planIds) == 0 {
		return ctx, nil
	}

	// 方案与规则各一次 IN 查询批量取回：计费热路径绝不能逐条查。
	var err error
	ctx.plans, err = getDiscountPlansByIds(planIds)
	if err != nil {
		return nil, err
	}
	ctx.rulesByPlan, err = getDiscountRulesByPlanIds(planIds)
	if err != nil {
		return nil, err
	}
	return ctx, nil
}

// resolveModel 把 context 里的绑定对一个模型逐条解析并裁决出唯一答案。
// 逐条解析的优先级与单方案解析（resolvePlanDiscount 第 4-6 步）完全一致：
// 模型级规则 → 厂商级规则 → 基础折扣。vendorName 为空串时跳过厂商级匹配
// （查不到厂商与没有厂商级规则，效果一样都是跳过）。
func (ctx *discountMultiContext) resolveModel(modelName string, vendorName string) (*DiscountResolution, []*DiscountCandidate) {
	resolution := &DiscountResolution{
		Discount: DiscountNone,
		Source:   DiscountResolvedFromDefault,
	}
	candidates := make([]*DiscountCandidate, 0, len(ctx.allBindings))
	for i := range ctx.allBindings {
		candidate := &DiscountCandidate{Binding: &ctx.allBindings[i]}
		if ctx.windowReasons[i] != "" {
			candidate.Rejected = true
			candidate.RejectReason = ctx.windowReasons[i]
		}
		candidates = append(candidates, candidate)
	}

	for _, candidate := range candidates {
		if candidate.Rejected {
			continue
		}
		plan, ok := ctx.plans[candidate.Binding.PlanId]
		if !ok || plan.Status != DiscountStatusEnabled {
			// 方案停用或已删除：这条绑定对当前模型没有发言权。
			candidate.Rejected = true
			candidate.RejectReason = "方案已停用或已删除"
			continue
		}
		candidate.Plan = plan
		rules := ctx.rulesByPlan[plan.Id]
		if rule := pickDiscountRule(rules, DiscountScopeModel, modelName); rule != nil {
			candidate.Specificity = DiscountResolvedFromModel
			candidate.Rule = rule
			candidate.Discount = formatResolvedDiscount(rule.Discount)
			continue
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
		if binding := pickActiveDiscountBinding(ctx.allBindings, ctx.now); binding != nil {
			resolution.PlanId = binding.PlanId
		}
		return resolution, candidates
	}
	markDiscountCandidateRejections(candidates, winner)

	resolution.Discount = winner.Discount
	resolution.Source = winner.Specificity
	resolution.Rule = winner.Rule
	resolution.Plan = winner.Plan
	resolution.PlanId = winner.Binding.PlanId
	return resolution, candidates
}

// hasVendorScopeRules 判断这批方案里有没有启用中的厂商级规则，
// 决定整份清单要不要查模型目录表。
func (ctx *discountMultiContext) hasVendorScopeRules() bool {
	for _, rules := range ctx.rulesByPlan {
		for _, rule := range rules {
			if rule.Status == DiscountStatusEnabled && rule.ScopeType == DiscountScopeVendor {
				return true
			}
		}
	}
	return false
}

// GetVendorNamesByModelNames 批量查「模型名 → 厂商名」。
// 与单条版 resolveVendorName 同一口径：目录里没有、没挂厂商、厂商被删的模型都不出现在
// 结果里（调用方拿到空串，跳过厂商级匹配）。
func GetVendorNamesByModelNames(modelNames []string) (map[string]string, error) {
	nameList := normalizeLookupValues(modelNames)
	result := make(map[string]string, len(nameList))
	if len(nameList) == 0 {
		return result, nil
	}
	rows := make([]Model, 0, len(nameList))
	if err := DB.Where("model_name IN ?", nameList).Find(&rows).Error; err != nil {
		return nil, err
	}
	vendorIds := make([]int, 0, len(rows))
	for i := range rows {
		if rows[i].VendorID > 0 {
			vendorIds = append(vendorIds, rows[i].VendorID)
		}
	}
	vendorNames := make(map[int]string, len(vendorIds))
	if len(vendorIds) > 0 {
		vendors := make([]Vendor, 0, len(vendorIds))
		if err := DB.Where("id IN ?", vendorIds).Find(&vendors).Error; err != nil {
			return nil, err
		}
		for i := range vendors {
			vendorNames[vendors[i].Id] = vendors[i].Name
		}
	}
	for i := range rows {
		if name, ok := vendorNames[rows[i].VendorID]; ok {
			result[rows[i].ModelName] = name
		}
	}
	return result, nil
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
