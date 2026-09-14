package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 保存前提示的原因码。全部是提示，不是否决——运营看到「这单会亏」后确认，仍然可以保存。
const (
	// DiscountViolationMinAboveBase 方案最低折扣高于基础折扣（写入口已拦，这里复核）。
	DiscountViolationMinAboveBase = "min_above_base"
	// DiscountViolationCostBreach 规则范围内最便宜的进货价都高于毛利底线，这一档会亏。
	DiscountViolationCostBreach = "cost_breach"
	// DiscountViolationNoChannel 规则覆盖不到任何可用线路。
	DiscountViolationNoChannel = "no_channel"
)

// 方案级提示的 scope_type：它不属于某条规则，而是整个方案的自洽性。
const discountValidateScopePlan = "plan"

var (
	// ErrDiscountValidatePlanNotFound 要校验的折扣方案不存在。
	ErrDiscountValidatePlanNotFound = errors.New("折扣方案不存在")
	// ErrDiscountValidateUserNotFound 校验时指定的客户不存在。
	ErrDiscountValidateUserNotFound = errors.New("客户不存在")
)

// DiscountValidateViolation 是一条提示项。scope_type + scope_value 必须明确指出
// 是哪个模型 / 哪个厂商 / 整个方案，前端照着展示，不能只回一句「校验失败」。
type DiscountValidateViolation struct {
	ScopeType         string   `json:"scope_type"`
	ScopeValue        string   `json:"scope_value"`
	Discount          string   `json:"discount"`
	Reason            string   `json:"reason"`
	Detail            string   `json:"detail"`
	AvailableChannels []string `json:"available_channels"`
}

type DiscountValidateResult struct {
	Passed         bool                         `json:"passed"`
	MinMarginRatio string                       `json:"min_margin_ratio"`
	Violations     []*DiscountValidateViolation `json:"violations"`
	Warnings       []string                     `json:"warnings"`
}

// ValidateDiscountPlan 把「这个方案会不会亏」算一遍，列出提示项。
//
// 它是试算的轻量版：不改任何数据，也不拦保存——亏本促销、大客户见面礼、引流试样
// 都是真实业务，系统只负责先把话说清楚（01-simulate-api.md §5.2）。
// userId 大于 0 时按该客户的分组挑可用线路，否则按 default 分组；
// channelId 大于 0 时只看这一条线路，用于「改完这条线路的进货价，先验一验」。
//
// 注意：规则（rule.Discount）已废弃，本函数按方案基础折扣（plan.BaseDiscount）出价；
// 仍按规则遍历是因为不同规则覆盖的模型对应不同进货价，毛利判断要按范围分做。
func ValidateDiscountPlan(planId int, userId int, channelId int) (*DiscountValidateResult, error) {
	plan, err := model.GetDiscountPlanById(planId)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiscountValidatePlanNotFound
		}
		return nil, err
	}
	rules, err := model.GetDiscountRulesByPlanId(planId)
	if err != nil {
		return nil, err
	}

	// 不指定客户时按 default 分组算：运营常想「没选客户先看看这方案本身亏不亏」。
	groups := []string{"default"}
	if userId > 0 {
		user, err := model.GetUserById(userId, false)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrDiscountValidateUserNotFound
			}
			return nil, err
		}
		groups = candidateGroups(user.Group)
	}

	minMarginRatio := normalizeDiscountRatioText(operation_setting.GetDiscountSetting().MinMarginRatio)
	minMargin, err := decimal.NewFromString(minMarginRatio)
	if err != nil {
		minMargin, minMarginRatio = decimal.Zero, "0.000000"
	}

	result := &DiscountValidateResult{
		MinMarginRatio: minMarginRatio,
		Violations:     make([]*DiscountValidateViolation, 0),
		Warnings:       make([]string, 0),
	}

	minDiscount, err := decimal.NewFromString(strings.TrimSpace(plan.MinDiscount))
	if err != nil {
		minDiscount = decimal.Zero
		result.Warnings = append(result.Warnings, "方案最低折扣无法解析，已按 0 处理")
	}
	baseDiscount, err := decimal.NewFromString(strings.TrimSpace(plan.BaseDiscount))
	if err != nil {
		baseDiscount = decimal.Zero
		result.Warnings = append(result.Warnings, "方案基础折扣无法解析，已按 0 处理")
	}
	if minDiscount.GreaterThan(baseDiscount) {
		result.Violations = append(result.Violations, &DiscountValidateViolation{
			ScopeType:  discountValidateScopePlan,
			ScopeValue: plan.Name,
			Discount:   baseDiscount.StringFixed(6),
			Reason:     DiscountViolationMinAboveBase,
			Detail: fmt.Sprintf("方案最低折扣 %s 高于基础折扣 %s",
				minDiscount.StringFixed(6), baseDiscount.StringFixed(6)),
			AvailableChannels: []string{},
		})
	}

	channelUsed := false
	// rule.Discount 已废弃——规则改为纯范围标记，命中时按方案基础折扣（plan.BaseDiscount）出价。
	// 校验也不再读 rule.Discount：方案基础折扣若低于最低折扣，写入时已被 ValidateDiscountFloor 拦下。
	// 这里只针对「每条规则的覆盖范围」判断可买性与毛利底线，不同规则的覆盖范围仍可能顶到不同
	// 的进货价，所以仍是按规则遍历，只是每条规则都套用同一个 plan.BaseDiscount。
	for _, rule := range rules {
		if rule.Status != model.DiscountStatusEnabled {
			continue
		}
		scopeValue := strings.TrimSpace(rule.ScopeValue)

		// 成本层：规则作用范围内最便宜的那条线路都顶不住，这一档就是会亏。
		// 取「最低进货折扣」与「至少有一条线路满足 cost_ratio ≤ 折扣 − 底线」是同一件事。
		modelNames, err := model.GetDiscountScopeModelNames(rule.ScopeType, scopeValue)
		if err != nil {
			return nil, err
		}
		if len(modelNames) == 0 {
			result.Violations = append(result.Violations, &DiscountValidateViolation{
				ScopeType:         rule.ScopeType,
				ScopeValue:        scopeValue,
				Discount:          baseDiscount.StringFixed(6),
				Reason:            DiscountViolationNoChannel,
				Detail:            "模型目录里没有这条规则覆盖的模型，这条规则打不到任何真实模型",
				AvailableChannels: []string{},
			})
			continue
		}

		byModel, err := model.GetDiscountChannelCandidatesByModels(groups, modelNames)
		if err != nil {
			return nil, err
		}
		channelNames := make([]string, 0)
		lowestCost := decimal.Decimal{}
		costLines := 0
		for _, modelName := range modelNames {
			for _, candidate := range byModel[modelName] {
				if channelId > 0 && candidate.ChannelId != channelId {
					continue
				}
				channelNames = append(channelNames, candidate.ChannelName)
				if channelId > 0 {
					channelUsed = true
				}
				if candidate.CostRatio == nil {
					continue
				}
				cost, err := decimal.NewFromString(*candidate.CostRatio)
				if err != nil || cost.LessThanOrEqual(decimal.Zero) {
					// 脏数据或 0：与试算同一口径，一律按「没录进货折扣」处理，
					// 不能拿它算出一个荒唐的结论。
					continue
				}
				costLines++
				if lowestCost.IsZero() || cost.LessThan(lowestCost) {
					lowestCost = cost
				}
			}
		}

		if len(channelNames) == 0 {
			if channelId > 0 {
				// 指定了一条线路，但它不在这条规则的可用范围内：这不构成问题，
				// 但也别拿别的线路替它下结论。
				continue
			}
			result.Violations = append(result.Violations, &DiscountValidateViolation{
				ScopeType:         rule.ScopeType,
				ScopeValue:        scopeValue,
				Discount:          baseDiscount.StringFixed(6),
				Reason:            DiscountViolationNoChannel,
				Detail:            "这条规则覆盖的模型在当前分组下没有可用线路",
				AvailableChannels: []string{},
			})
			continue
		}

		if costLines == 0 {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("规则「%s」范围内的可用线路都没录进货折扣，无法判断是否赔本", scopeValue))
			continue
		}

		floor := baseDiscount.Sub(minMargin)
		if lowestCost.GreaterThan(floor) {
			result.Violations = append(result.Violations, &DiscountValidateViolation{
				ScopeType:  rule.ScopeType,
				ScopeValue: scopeValue,
				Discount:   baseDiscount.StringFixed(6),
				Reason:     DiscountViolationCostBreach,
				Detail: fmt.Sprintf("最低可用进货折扣 %s，高于毛利底线 %s",
					lowestCost.StringFixed(6), floor.StringFixed(6)),
				AvailableChannels: dedupeSortedStrings(channelNames),
			})
		}
	}

	if channelId > 0 && !channelUsed {
		result.Warnings = append(result.Warnings, "指定的线路不在任何规则的可用范围内，本次没有参与判断")
	}
	result.Passed = len(result.Violations) == 0
	return result, nil
}

// dedupeSortedStrings 给前端一份稳定的线路清单：去重后按名字排序，
// 顺序不随查询结果变化，前端展示和人工对照都不会跳。
func dedupeSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	deduped := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		deduped = append(deduped, value)
	}
	sort.Strings(deduped)
	return deduped
}
