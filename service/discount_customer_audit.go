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

// 客户档案核算（task-06 第二步，零件三）：签客户之前把「这个客户 + 他要的整份模型清单」
// 一次算完——每个模型按几折、命中哪条价、还有几条线路、最便宜线路成本多少、每单毛利多少，
// 最后一句「这份报价能不能签」。纯读取：不改任何价，不动任何单。
// 逐模型的折扣走多方案解析（与计费同一套代码），线路成本口径与试算完全一致。

var (
	// ErrCustomerAuditUserNotFound 核算指定的客户不存在。
	ErrCustomerAuditUserNotFound = errors.New("客户不存在")
	// ErrCustomerAuditNoModels 模型清单是空的（或全是空白行）。
	ErrCustomerAuditNoModels = errors.New("模型清单不能为空")
	// ErrCustomerAuditTooManyModels 一份清单太大。管理端手工粘贴的场景，
	// 上限是防呆不是业务规则——真有几百个模型的客户应该按厂商分几次算。
	ErrCustomerAuditTooManyModels = fmt.Errorf("一次最多核算 %d 个模型，请分批", CustomerAuditMaxModels)
)

// CustomerAuditMaxModels 一份清单最多算多少个模型。
const CustomerAuditMaxModels = 100

// 每个模型的结论（verdict 取值），前端照此染色。
const (
	// CustomerAuditVerdictOK 有过毛利底线的线路：路由能正常挑到赚钱的线，这个模型可以签。
	CustomerAuditVerdictOK = "ok"
	// CustomerAuditVerdictLoss 已录成本的线路全过不了毛利底线：这单会亏，签之前要处理。
	CustomerAuditVerdictLoss = "loss"
	// CustomerAuditVerdictUnknownCost 有线路但没录进货折扣：成本无从谈起，先补数据。
	CustomerAuditVerdictUnknownCost = "unknown_cost"
	// CustomerAuditVerdictNoChannel 该客户分组下没有可用线路：客户一下单就报错。
	CustomerAuditVerdictNoChannel = "no_channel"
)

// CustomerPricingAuditModel 清单里一个模型的核算结果。
// 三个值字段的「不知道」一律用 null 表达，不能显示成「赚 0 元」。
type CustomerPricingAuditModel struct {
	Model         string                `json:"model"`
	Vendor        string                `json:"vendor"`
	Discount      string                `json:"discount"`
	Source        string                `json:"source"`
	Plan          *DiscountSimulatePlan `json:"plan"`
	MatchedRule   *DiscountSimulateRule `json:"matched_rule"`
	ChannelCount  int                   `json:"channel_count"`
	UsableCount   int                   `json:"usable_count"`
	CheapestCost  *string               `json:"cheapest_cost"`
	GrossMargin   *string               `json:"gross_margin"`
	Verdict       string                `json:"verdict"`
	VerdictDetail string                `json:"verdict_detail"`
}

// CustomerPricingAuditSummary 整份清单的汇总：各结论多少个，最后一句「能不能签」。
type CustomerPricingAuditSummary struct {
	Total       int    `json:"total"`
	OK          int    `json:"ok"`
	Loss        int    `json:"loss"`
	UnknownCost int    `json:"unknown_cost"`
	NoChannel   int    `json:"no_channel"`
	Signable    bool   `json:"signable"`
	Conclusion  string `json:"conclusion"`
}

// CustomerPricingAuditResult 客户档案核算的完整答案。
type CustomerPricingAuditResult struct {
	User           DiscountSimulateUser        `json:"user"`
	MinMarginRatio string                      `json:"min_margin_ratio"`
	Models         []*CustomerPricingAuditModel `json:"models"`
	Summary        CustomerPricingAuditSummary  `json:"summary"`
	Warnings       []string                    `json:"warnings"`
}

// AuditCustomerPricing 把「这个客户 + 这份模型清单」一次算完。
// modelNames 里重复、空白的名字会被修剪去重；顺序按输入保留（运营对着客户的单子核对）。
func AuditCustomerPricing(userId int, modelNames []string) (*CustomerPricingAuditResult, error) {
	nameList := auditModelNames(modelNames)
	if len(nameList) == 0 {
		return nil, ErrCustomerAuditNoModels
	}
	if len(nameList) > CustomerAuditMaxModels {
		return nil, ErrCustomerAuditTooManyModels
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerAuditUserNotFound
		}
		return nil, err
	}

	// 折扣一次批量解析（与计费同一套代码），线路候选一次批量取回，厂商名一次批量查。
	resolutions, _, err := model.ResolveUserDiscountDetailedForModels(userId, nameList)
	if err != nil {
		return nil, err
	}
	candidatesByModel, err := model.GetDiscountChannelCandidatesByModels(candidateGroups(user.Group), nameList)
	if err != nil {
		return nil, err
	}
	vendorNames, err := model.GetVendorNamesByModelNames(nameList)
	if err != nil {
		return nil, err
	}

	minMarginRatio := normalizeDiscountRatioText(operation_setting.GetDiscountSetting().MinMarginRatio)
	minMargin, err := decimal.NewFromString(minMarginRatio)
	if err != nil {
		minMargin, minMarginRatio = decimal.Zero, "0.000000"
	}

	result := &CustomerPricingAuditResult{
		User:           DiscountSimulateUser{Id: user.Id, Username: user.Username, Group: user.Group},
		MinMarginRatio: minMarginRatio,
		Models:         make([]*CustomerPricingAuditModel, 0, len(nameList)),
		Warnings:       make([]string, 0),
	}

	// 整份清单的「没绑任何方案」提示：有一个模型命中方案就不算没绑，
	// 单个模型的特殊情况（方案停用）逐行自己说明。
	anyPlanResolved := false
	anyWholesale := false
	for _, modelName := range nameList {
		resolution := resolutions[modelName]
		if resolution == nil {
			// 批量解析对每个有效模型名都有答案，这里是数据异常，不猜。
			return nil, fmt.Errorf("模型 %s 的折扣解析结果缺失", modelName)
		}
		if resolution.PlanId > 0 {
			anyPlanResolved = true
		}
		if resolution.Source == model.DiscountResolvedFromAgentWholesale {
			anyWholesale = true
		}

		discount, err := decimal.NewFromString(resolution.Discount)
		if err != nil {
			// 折扣列只由 NormalizeDiscount 写入，解析不出来说明库里被改坏了，
			// 与试算同一处置：报错，不兜底成 1.0 悄悄算。
			return nil, errors.New("折扣数据异常：" + resolution.Discount)
		}
		// 毛利底线是「客户折扣 − 底线」：进货折扣不高于这个数才算不赔本（与试算同式）。
		floor := discount.Sub(minMargin)

		audited := auditOneModel(modelName, resolution, discount, floor, candidatesByModel[modelName])
		audited.Vendor = vendorNames[modelName]
		audited.Plan = buildSimulatePlan(resolution.Plan)
		audited.MatchedRule = buildSimulateRule(resolution)
		result.Models = append(result.Models, audited)
	}

	if !anyPlanResolved && !anyWholesale {
		result.Warnings = append(result.Warnings, "该客户未绑定任何折扣方案，整份清单按官方标价核算")
	}
	if anyWholesale {
		result.Warnings = append(result.Warnings, "该客户是经销商，部分模型按其拿货价核算")
	}
	result.Summary = summarizeCustomerAudit(result.Models)
	return result, nil
}

// auditOneModel 算一个模型的行：折扣怎么来的、线路情况、最便宜成本、毛利、结论。
// 线路成本的判定口径与 SimulateDiscount 完全一致：没录进货折扣、录的数字解析不出来（含 0）
// 一律当「不知道」；毛利 = 客户折扣 ÷ 进货折扣 − 1，按最便宜那条线路算（能签的前提）。
func auditOneModel(modelName string, resolution *model.DiscountResolution, discount decimal.Decimal, floor decimal.Decimal, candidates []*model.DiscountChannelCandidate) *CustomerPricingAuditModel {
	audited := &CustomerPricingAuditModel{
		Model:    modelName,
		Discount: resolution.Discount,
		Source:   resolution.Source,
	}

	if len(candidates) == 0 {
		audited.Verdict = CustomerAuditVerdictNoChannel
		audited.VerdictDetail = "该客户分组下没有可用线路，客户一下单就报错"
		return audited
	}
	audited.ChannelCount = len(candidates)

	unknownCostCount := 0
	for _, candidate := range candidates {
		if candidate.CostRatio == nil {
			unknownCostCount++
			continue
		}
		costRatio, err := decimal.NewFromString(*candidate.CostRatio)
		if err != nil || costRatio.LessThanOrEqual(decimal.Zero) {
			// 脏数据或 0：0 会让毛利算成无穷大，与试算同一处置按未录入处理。
			unknownCostCount++
			continue
		}
		if audited.CheapestCost == nil || costRatio.LessThan(parseAuditDecimal(*audited.CheapestCost)) {
			cheapest := costRatio.StringFixed(6)
			audited.CheapestCost = &cheapest
			margin := discount.DivRound(costRatio, 6).Sub(decimal.NewFromInt(1)).StringFixed(6)
			audited.GrossMargin = &margin
		}
		if costRatio.LessThanOrEqual(floor) {
			audited.UsableCount++
		}
	}

	switch {
	case audited.UsableCount > 0:
		audited.Verdict = CustomerAuditVerdictOK
		audited.VerdictDetail = fmt.Sprintf("%d/%d 条线路保本，按最便宜线路（进货折扣 %s）毛利 %s",
			audited.UsableCount, audited.ChannelCount, *audited.CheapestCost, *audited.GrossMargin)
	case audited.CheapestCost == nil:
		audited.Verdict = CustomerAuditVerdictUnknownCost
		audited.VerdictDetail = fmt.Sprintf("%d 条线路都没录进货折扣，先补数据再签", audited.ChannelCount)
	default:
		audited.Verdict = CustomerAuditVerdictLoss
		detail := fmt.Sprintf("已录成本的线路全部过不了毛利底线（最便宜的 %s 也高于 %.6f）",
			*audited.CheapestCost, floor.InexactFloat64())
		if unknownCostCount > 0 {
			detail += fmt.Sprintf("，另有 %d 条线路未录进货折扣无法一并判断", unknownCostCount)
		}
		audited.VerdictDetail = detail
	}
	return audited
}

// summarizeCustomerAudit 汇总整份清单：各结论计数 + 一句「能不能签」。
// 能签的标准是苛刻的：没有任何会亏、没有任何没线路、没有任何成本未知的模型。
func summarizeCustomerAudit(models []*CustomerPricingAuditModel) CustomerPricingAuditSummary {
	summary := CustomerPricingAuditSummary{Total: len(models)}
	for _, audited := range models {
		switch audited.Verdict {
		case CustomerAuditVerdictOK:
			summary.OK++
		case CustomerAuditVerdictLoss:
			summary.Loss++
		case CustomerAuditVerdictUnknownCost:
			summary.UnknownCost++
		case CustomerAuditVerdictNoChannel:
			summary.NoChannel++
		}
	}
	summary.Signable = summary.Loss == 0 && summary.UnknownCost == 0 && summary.NoChannel == 0

	switch {
	case summary.Total == 0:
		summary.Conclusion = "清单是空的"
	case summary.Signable:
		summary.Conclusion = fmt.Sprintf("这份报价可以签：%d 个模型全部保本", summary.Total)
	default:
		parts := make([]string, 0, 3)
		if summary.Loss > 0 {
			parts = append(parts, fmt.Sprintf("%d 个会亏", summary.Loss))
		}
		if summary.NoChannel > 0 {
			parts = append(parts, fmt.Sprintf("%d 个没有可用线路", summary.NoChannel))
		}
		if summary.UnknownCost > 0 {
			parts = append(parts, fmt.Sprintf("%d 个成本未知", summary.UnknownCost))
		}
		summary.Conclusion = "签之前先处理这几行：" + strings.Join(parts, "、")
	}
	return summary
}

// auditModelNames 修剪去重模型清单并保留输入顺序。
// 前端给的是逐行粘贴的文本拆成的数组；逗号（中英文）、顿号、换行都容得下，
// 但空格不当分隔符——模型名里允许有空格，拆碎就对不上线路了。
func auditModelNames(modelNames []string) []string {
	seen := make(map[string]struct{}, len(modelNames))
	nameList := make([]string, 0, len(modelNames))
	for _, raw := range modelNames {
		for _, name := range strings.FieldsFunc(raw, func(r rune) bool {
			return r == ',' || r == '\n' || r == '\r' || r == '\t' || r == '，' || r == '、'
		}) {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			nameList = append(nameList, name)
		}
	}
	return nameList
}

// parseAuditDecimal 解析核算行里已规范成 6 位小数的成本字符串。
// 它来自上一步自己的 StringFixed，解析失败只能是代码写错——按 1 处理
// （比正常进货折扣都高，不会把已有的最便宜值替换掉）。
func parseAuditDecimal(value string) decimal.Decimal {
	parsed, err := decimal.NewFromString(value)
	if err != nil {
		return decimal.NewFromInt(1)
	}
	return parsed
}
