package model

// 客户价目本（task-06 第四步「一键查价」的读侧零件）。
//
// 计费只关心「最终按几折」，运营关心的是「这个客户身上到底挂了哪些价」：
// 谁给他挂的、什么时候到期、这套价管哪些模型。这里把绑定摊平成人能读的一行行。
//
// 读法与计价完全一致——复用同一个 loadDiscountMultiContext 读绑定与生效窗口，
// 不另写一套读法，免得界面上的价目本与扣费时读到的不是一回事。
//
// 范围：只列**生效中**的绑定（与计价同一个过滤条件 status = 启用），已解绑的历史记录
// 不出现；不在生效时间窗内的绑定照样列出并写明原因，用来回答「我上周谈的那套价怎么没生效」。

// CustomerPriceBookRule 方案里的一条规则，写清这套价对谁改价。
type CustomerPriceBookRule struct {
	ScopeType  string `json:"scope_type"`
	ScopeValue string `json:"scope_value"`
	Discount   string `json:"discount"`
	Priority   int    `json:"priority"`
	Status     int    `json:"status"`
}

// CustomerPriceBookEntry 这个客户身上的一套价：一条绑定 + 它指向的方案与规则。
// 方案已被删除时 PlanName 为空、PlanStatus 为 0——绑定还在，价没了。
type CustomerPriceBookEntry struct {
	BindingId     int                      `json:"binding_id"`
	PlanId        int                      `json:"plan_id"`
	PlanName      string                   `json:"plan_name"`
	PlanStatus    int                      `json:"plan_status"`
	Source        string                   `json:"source"`
	BillingMode   string                   `json:"billing_mode"`
	BaseDiscount  string                   `json:"base_discount"`
	EffectiveFrom int64                    `json:"effective_from"`
	EffectiveTo   int64                    `json:"effective_to"`
	InWindow      bool                     `json:"in_window"`
	WindowReason  string                   `json:"window_reason"`
	Rules         []*CustomerPriceBookRule `json:"rules"`
}

// GetCustomerPriceBook 读一个客户身上全部的价，按绑定 id 从小到大（挂上去的先后）。
func GetCustomerPriceBook(userId int) ([]*CustomerPriceBookEntry, error) {
	if userId <= 0 {
		return make([]*CustomerPriceBookEntry, 0), nil
	}
	ctx, err := loadDiscountMultiContext(userId)
	if err != nil {
		return nil, err
	}
	if len(ctx.allBindings) == 0 {
		return make([]*CustomerPriceBookEntry, 0), nil
	}

	// ctx 里只装了窗口内绑定指向的方案与规则（计费用得到的就这些）。价目本要连窗口外
	// 的那几条一起说清楚，所以把它们指向的方案与规则也批量补上——同样是两次 IN 查询。
	planIds := make([]int, 0, len(ctx.allBindings))
	for i := range ctx.allBindings {
		planIds = append(planIds, ctx.allBindings[i].PlanId)
	}
	plans, err := getDiscountPlansByIds(planIds)
	if err != nil {
		return nil, err
	}
	rulesByPlan, err := getDiscountRulesByPlanIds(planIds)
	if err != nil {
		return nil, err
	}

	entries := make([]*CustomerPriceBookEntry, 0, len(ctx.allBindings))
	for i := range ctx.allBindings {
		binding := &ctx.allBindings[i]
		entry := &CustomerPriceBookEntry{
			BindingId:     binding.Id,
			PlanId:        binding.PlanId,
			Source:        binding.Source,
			EffectiveFrom: binding.EffectiveFrom,
			EffectiveTo:   binding.EffectiveTo,
			InWindow:      ctx.windowReasons[i] == "",
			WindowReason:  ctx.windowReasons[i],
			Rules:         make([]*CustomerPriceBookRule, 0),
		}
		if plan, ok := plans[binding.PlanId]; ok {
			entry.PlanName = plan.Name
			entry.PlanStatus = plan.Status
			entry.BillingMode = plan.BillingMode
			entry.BaseDiscount = formatResolvedDiscount(plan.BaseDiscount)
		}
		for _, rule := range rulesByPlan[binding.PlanId] {
			entry.Rules = append(entry.Rules, &CustomerPriceBookRule{
				ScopeType:  rule.ScopeType,
				ScopeValue: rule.ScopeValue,
				Discount:   formatResolvedDiscount(rule.Discount),
				Priority:   rule.Priority,
				Status:     rule.Status,
			})
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
