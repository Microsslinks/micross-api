package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	// ErrDiscountSimulateUserNotFound 试算指定的客户不存在。
	ErrDiscountSimulateUserNotFound = errors.New("客户不存在")
	// ErrDiscountSimulateChannelNotAvailable 指定的线路不在该客户的可用范围内。
	ErrDiscountSimulateChannelNotAvailable = errors.New("该线路不在该客户的可用范围内")
)

// ---- 试算出参：字段与 01-simulate-api.md §4.2 一一对应 ----

type DiscountSimulateUser struct {
	Id       int    `json:"id"`
	Username string `json:"username"`
	Group    string `json:"group"`
}

type DiscountSimulatePlan struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Status int    `json:"status"`
}

type DiscountSimulateRule struct {
	Id         int    `json:"id"`
	ScopeType  string `json:"scope_type"`
	ScopeValue string `json:"scope_value"`
	Discount   string `json:"discount"`
	Priority   int    `json:"priority"`
}

type DiscountSimulateResolution struct {
	Discount    string                `json:"discount"`
	Source      string                `json:"source"`
	MatchedRule *DiscountSimulateRule `json:"matched_rule"`
}

// DiscountSimulateChannel 是一条候选线路的试算结果。
// 没录进货折扣时三个值字段都是 null——「不知道」不能显示成「赚 0 元」。
type DiscountSimulateChannel struct {
	ChannelId   int     `json:"channel_id"`
	ChannelName string  `json:"channel_name"`
	CostRatio   *string `json:"cost_ratio"`
	GrossMargin *string `json:"gross_margin"`
	PassesFloor *bool   `json:"passes_floor"`
}

type DiscountSimulateResult struct {
	User           DiscountSimulateUser       `json:"user"`
	Model          string                     `json:"model"`
	Vendor         string                     `json:"vendor"`
	Plan           *DiscountSimulatePlan      `json:"plan"`
	Resolution     DiscountSimulateResolution `json:"resolution"`
	CostKnown      bool                       `json:"cost_known"`
	MinMarginRatio string                     `json:"min_margin_ratio"`
	Channels       []*DiscountSimulateChannel `json:"channels"`
	Warnings       []string                   `json:"warnings"`
}

// SimulateDiscount 算「这个客户 + 这个模型」按几折、会走哪几条线路、每条赚多少。
// 只读不落库：报价侧不存静态成本基准。
// channelId 大于 0 时只算这一条线路，且它必须在这个客户的候选里，否则报错。
func SimulateDiscount(userId int, modelName string, channelId int) (*DiscountSimulateResult, error) {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return nil, errors.New("模型名不能为空")
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiscountSimulateUserNotFound
		}
		return nil, err
	}

	candidates, err := model.GetDiscountChannelCandidates(candidateGroups(user.Group), modelName)
	if err != nil {
		return nil, err
	}
	if channelId > 0 {
		picked := make([]*model.DiscountChannelCandidate, 0, 1)
		for _, candidate := range candidates {
			if candidate.ChannelId == channelId {
				picked = append(picked, candidate)
			}
		}
		if len(picked) == 0 {
			return nil, ErrDiscountSimulateChannelNotAvailable
		}
		candidates = picked
	}

	resolution, err := model.ResolveUserDiscount(userId, modelName)
	if err != nil {
		return nil, err
	}
	discount, err := decimal.NewFromString(resolution.Discount)
	if err != nil {
		// 折扣列只由 NormalizeDiscount 写入，解析不出来说明库里被改坏了。
		// 不能兜底成 1.0 悄悄算：那会让试算结果和实际折扣对不上。
		return nil, errors.New("折扣数据异常：" + resolution.Discount)
	}
	vendorName, err := model.GetVendorNameByModelName(modelName)
	if err != nil {
		return nil, err
	}

	minMarginRatio := normalizeDiscountRatioText(operation_setting.GetDiscountSetting().MinMarginRatio)
	minMargin, err := decimal.NewFromString(minMarginRatio)
	if err != nil {
		minMargin, minMarginRatio = decimal.Zero, "0.000000"
	}
	// 毛利底线是「客户折扣 − 底线」：进货折扣不高于这个数才算不赔本。
	floor := discount.Sub(minMargin)

	result := &DiscountSimulateResult{
		User:   DiscountSimulateUser{Id: user.Id, Username: user.Username, Group: user.Group},
		Model:  modelName,
		Vendor: vendorName,
		Plan:   buildSimulatePlan(resolution.Plan),
		Resolution: DiscountSimulateResolution{
			Discount:    resolution.Discount,
			Source:      resolution.Source,
			MatchedRule: buildSimulateRule(resolution),
		},
		MinMarginRatio: minMarginRatio,
		Channels:       make([]*DiscountSimulateChannel, 0, len(candidates)),
		Warnings:       make([]string, 0),
	}

	switch {
	case resolution.Source == model.DiscountResolvedFromAgentWholesale:
		// 经销商自己消费按拿货价算，跟他绑没绑方案无关，不加"未绑方案"的提示。
	case resolution.PlanId == 0:
		result.Warnings = append(result.Warnings, "该客户未绑定折扣方案，按官方标价试算")
	case resolution.Plan == nil:
		result.Warnings = append(result.Warnings, "该客户绑定的折扣方案已停用，按官方标价试算")
	}
	if len(candidates) == 0 {
		result.Warnings = append(result.Warnings, "该客户分组下此模型无可用线路")
	}

	// 只要有一条候选没录进货折扣，就不知道这个模型的成本——不能报 true。
	costKnown := len(candidates) > 0
	for _, candidate := range candidates {
		channel := &DiscountSimulateChannel{ChannelId: candidate.ChannelId, ChannelName: candidate.ChannelName}
		if candidate.CostRatio == nil {
			costKnown = false
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("线路「%s」未录入进货折扣，无法判断是否赔本", candidate.ChannelName))
			result.Channels = append(result.Channels, channel)
			continue
		}
		costRatio, err := decimal.NewFromString(*candidate.CostRatio)
		if err != nil || costRatio.LessThanOrEqual(decimal.Zero) {
			// 脏数据或 0：0 会让毛利算成无穷大，一律按未录入处理并把话说清楚。
			costKnown = false
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("线路「%s」的进货折扣无法解析，已按未录入处理", candidate.ChannelName))
			result.Channels = append(result.Channels, channel)
			continue
		}
		normalizedCost := costRatio.StringFixed(6)
		// 毛利 = 客户折扣 ÷ 进货折扣 − 1：客户 0.30、进货 0.27 → 0.111111。
		grossMargin := discount.DivRound(costRatio, 6).Sub(decimal.NewFromInt(1)).StringFixed(6)
		passesFloor := costRatio.LessThanOrEqual(floor)
		channel.CostRatio = &normalizedCost
		channel.GrossMargin = &grossMargin
		channel.PassesFloor = &passesFloor
		result.Channels = append(result.Channels, channel)
	}
	result.CostKnown = costKnown
	return result, nil
}

// candidateGroups 算出「这个客户的哪些分组可能走这个模型」。
// 分组为空按 default；配成 auto 的客户展开成它实际可用的分组，与路由侧同一套口径。
func candidateGroups(userGroup string) []string {
	group := strings.TrimSpace(userGroup)
	switch {
	case group == "":
		return []string{"default"}
	case group == "auto":
		return GetUserAutoGroup(group)
	default:
		return []string{group}
	}
}

// normalizeDiscountRatioText 把比例类配置整成固定 6 位小数字符串；空值或解析不出按 0。
func normalizeDiscountRatioText(raw string) string {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return "0.000000"
	}
	return value.StringFixed(6)
}

// buildSimulatePlan 把方案转成试算出参；未绑定或已停用时返回 nil。
func buildSimulatePlan(plan *model.DiscountPlan) *DiscountSimulatePlan {
	if plan == nil {
		return nil
	}
	return &DiscountSimulatePlan{Id: plan.Id, Name: plan.Name, Status: plan.Status}
}

// buildSimulateRule 指出这个折扣是哪条规则给的；回落到方案基础折扣时为 nil。
// 命中规则时规则折扣就是最终执行的那个折扣，直接用 resolution.Discount（已规范成 6 位）。
func buildSimulateRule(resolution *model.DiscountResolution) *DiscountSimulateRule {
	if resolution.Rule == nil {
		return nil
	}
	return &DiscountSimulateRule{
		Id:         resolution.Rule.Id,
		ScopeType:  resolution.Rule.ScopeType,
		ScopeValue: resolution.Rule.ScopeValue,
		Discount:   resolution.Discount,
		Priority:   resolution.Rule.Priority,
	}
}
