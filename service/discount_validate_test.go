package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 注：本套测试里规则的 Discount 字段不再决定校验结果——rule.Discount 已废弃，
// 命中时按 plan.BaseDiscount 出价，校验也按 plan.BaseDiscount 与进货折扣比较毛利底线。
// 测试数据里的 Discount 字段保留下来只是为了让结构体可正常 Insert，写什么值都无所谓，
// 真正算账的是 plan.BaseDiscount。所以下面「该报击穿」的用例会用「进货价 > 方案基础折扣」来构造，
// 而不是「进货价 > 规则折扣」。

// seedValidatePlan 建一个方案（可指定最低折扣）加它的规则。
// 校验是方案级的，不需要绑定客户，所以这里不建绑定。
func seedValidatePlan(t *testing.T, baseDiscount string, minDiscount string, rules ...*model.DiscountRule) *model.DiscountPlan {
	t.Helper()
	plan := &model.DiscountPlan{
		Name:         fmt.Sprintf("validate-plan-%d", time.Now().UnixNano()),
		OwnerType:    model.DiscountOwnerPlatform,
		BaseDiscount: baseDiscount,
		MinDiscount:  minDiscount,
		BillingMode:  model.DiscountBillingUsage,
		Status:       model.DiscountStatusEnabled,
	}
	require.NoError(t, plan.Insert())
	for _, rule := range rules {
		rule.PlanId = plan.Id
		require.NoError(t, rule.Insert())
	}
	return plan
}

// findValidateViolation 按原因码取一条提示项：多数用例只关心「报了哪一类」。
func findValidateViolation(result *DiscountValidateResult, reason string) *DiscountValidateViolation {
	for _, violation := range result.Violations {
		if violation.Reason == reason {
			return violation
		}
	}
	return nil
}

// 「规则折扣低于方案最低折扣」这条校验口径已随 rule.Discount 一起废弃——
// 规则改为纯范围标记，命中时一律按 plan.BaseDiscount 出价；最低折扣（plan.MinDiscount）
// 只能与基础折扣比较（由 ValidateDiscountFloor 在写入口把关），校验阶段不再逐条规则比一遍。
// 保留这段注释便于以后回看 spec 演化。
//
// 方案最低折扣高于基础折扣：写入口已经拦过，这里是复核，出一个方案级的提示。
func TestValidateDiscountMinAboveBase(t *testing.T) {
	setupDiscountSimulateTest(t)
	plan := seedValidatePlan(t, "0.300000", "0.500000")

	result, err := ValidateDiscountPlan(plan.Id, 0, 0)
	require.NoError(t, err)

	assert.False(t, result.Passed)
	require.Len(t, result.Violations, 1)
	violation := result.Violations[0]
	assert.Equal(t, "plan", violation.ScopeType)
	assert.Equal(t, plan.Name, violation.ScopeValue)
	assert.Equal(t, DiscountViolationMinAboveBase, violation.Reason)
	assert.Contains(t, violation.Detail, "0.500000")
	assert.Contains(t, violation.Detail, "0.300000")
}

// 验收口径 7：方案基础折扣低于进货成本 → 报 cost_breach 并把最低进货折扣与毛利底线都摆出来。
// rule.Discount 已废弃，校验按 plan.BaseDiscount 与进货折扣比较。
func TestValidateDiscountCostBreach(t *testing.T) {
	setupDiscountSimulateTest(t)
	seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	// 进货 0.95 高于方案基础折扣 0.9，会击穿毛利底线（min_margin=0）。
	seedSimulateChannel(t, "偏贵线路", costRatioPtr("0.950000"), []string{"default"}, "gpt-4o")
	plan := seedValidatePlan(t, "0.900000", "0",
		&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: model.DiscountStatusEnabled},
	)

	result, err := ValidateDiscountPlan(plan.Id, 0, 0)
	require.NoError(t, err)

	assert.False(t, result.Passed)
	require.Len(t, result.Violations, 1)
	violation := result.Violations[0]
	assert.Equal(t, model.DiscountScopeModel, violation.ScopeType)
	assert.Equal(t, "gpt-4o", violation.ScopeValue)
	// violation.Discount 现在是方案基础折扣，不再是规则折扣。
	assert.Equal(t, "0.900000", violation.Discount)
	assert.Equal(t, DiscountViolationCostBreach, violation.Reason)
	assert.Contains(t, violation.Detail, "0.950000", "要让运营看到最低进货折扣")
	assert.Contains(t, violation.Detail, "0.900000", "也要看到他按什么底线判的")
	assert.Equal(t, []string{"偏贵线路"}, violation.AvailableChannels)
}

// 厂商规则按「该厂商名下所有可用线路里最便宜的那条」判：有一条顶得住就不算亏。
func TestValidateDiscountPassesWhenCheapestLineCoversCost(t *testing.T) {
	setupDiscountSimulateTest(t)
	seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	var vendor model.Vendor
	require.NoError(t, model.DB.Where("name = ?", "OpenAI").First(&vendor).Error)
	require.NoError(t, model.DB.Create(&model.Model{ModelName: "gpt-4o-mini", VendorID: vendor.Id, Status: 1}).Error)
	seedSimulateChannel(t, "便宜线路", costRatioPtr("0.27"), []string{"default"}, "gpt-4o")
	seedSimulateChannel(t, "偏贵线路", costRatioPtr("0.45"), []string{"default"}, "gpt-4o-mini")
	plan := seedValidatePlan(t, "0.900000", "0",
		&model.DiscountRule{ScopeType: model.DiscountScopeVendor, ScopeValue: "OpenAI", Discount: "0.300000", Status: model.DiscountStatusEnabled},
	)

	result, err := ValidateDiscountPlan(plan.Id, 0, 0)
	require.NoError(t, err)

	assert.True(t, result.Passed, "0.27 ≤ 0.90，至少有一条线路顶得住")
	assert.Empty(t, result.Violations)
	assert.Empty(t, result.Warnings)
	assert.Equal(t, "0.000000", result.MinMarginRatio)
}

// 规则覆盖不到线路时说清楚是哪一种「打不到」：模型没线路，还是目录里压根没这个厂商。
func TestValidateDiscountWithoutAvailableChannel(t *testing.T) {
	t.Run("模型在目录里但它没有可用线路", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		seedSimulateCustomer(t, "OpenAI", "gpt-4o")
		plan := seedValidatePlan(t, "0.900000", "0",
			&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: model.DiscountStatusEnabled},
		)

		result, err := ValidateDiscountPlan(plan.Id, 0, 0)
		require.NoError(t, err)

		assert.False(t, result.Passed)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, DiscountViolationNoChannel, result.Violations[0].Reason)
		assert.Contains(t, result.Violations[0].Detail, "没有可用线路")
		assert.Equal(t, "gpt-4o", result.Violations[0].ScopeValue)
	})

	t.Run("厂商在模型目录里没有模型", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		plan := seedValidatePlan(t, "0.900000", "0",
			&model.DiscountRule{ScopeType: model.DiscountScopeVendor, ScopeValue: "不存在的厂商", Discount: "0.300000", Status: model.DiscountStatusEnabled},
		)

		result, err := ValidateDiscountPlan(plan.Id, 0, 0)
		require.NoError(t, err)

		assert.False(t, result.Passed)
		require.Len(t, result.Violations, 1)
		assert.Equal(t, DiscountViolationNoChannel, result.Violations[0].Reason)
		assert.Contains(t, result.Violations[0].Detail, "模型目录里没有")
		assert.Equal(t, "不存在的厂商", result.Violations[0].ScopeValue)
	})
}

// 规则里的 `claude-*` 要展开成目录里真实存在的模型，再按这些模型的线路判成本。
func TestValidateDiscountWithWildcardScope(t *testing.T) {
	t.Run("匹配到的模型里最便宜那条线路顶得住就通过", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		seedSimulateCustomer(t, "Anthropic", "claude-3-5-sonnet")
		var vendor model.Vendor
		require.NoError(t, model.DB.Where("name = ?", "Anthropic").First(&vendor).Error)
		require.NoError(t, model.DB.Create(&model.Model{ModelName: "claude-3-opus", VendorID: vendor.Id, Status: 1}).Error)
		seedSimulateChannel(t, "Anthropic 便宜线路", costRatioPtr("0.27"), []string{"default"}, "claude-3-5-sonnet")
		seedSimulateChannel(t, "Anthropic 偏贵线路", costRatioPtr("0.45"), []string{"default"}, "claude-3-opus")
		plan := seedValidatePlan(t, "0.900000", "0",
			&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "claude-*", Discount: "0.300000", Status: model.DiscountStatusEnabled},
		)

		result, err := ValidateDiscountPlan(plan.Id, 0, 0)
		require.NoError(t, err)

		assert.True(t, result.Passed, "0.27 ≤ 0.30")
		assert.Empty(t, result.Violations)
	})

	t.Run("匹配到的模型都亏就报出来并列出参与判断的线路", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		seedSimulateCustomer(t, "Anthropic", "claude-3-5-sonnet")
		var vendor model.Vendor
		require.NoError(t, model.DB.Where("name = ?", "Anthropic").First(&vendor).Error)
		require.NoError(t, model.DB.Create(&model.Model{ModelName: "claude-3-opus", VendorID: vendor.Id, Status: 1}).Error)
		// 两条线路的进货折扣都高于方案基础折扣 0.9，必然击穿毛利底线（min_margin=0）。
		seedSimulateChannel(t, "Anthropic 偏贵线路", costRatioPtr("0.950000"), []string{"default"}, "claude-3-5-sonnet")
		seedSimulateChannel(t, "Anthropic 更贵线路", costRatioPtr("0.970000"), []string{"default"}, "claude-3-opus")
		plan := seedValidatePlan(t, "0.900000", "0",
			&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "claude-*", Discount: "0.300000", Status: model.DiscountStatusEnabled},
		)

		result, err := ValidateDiscountPlan(plan.Id, 0, 0)
		require.NoError(t, err)

		assert.False(t, result.Passed)
		violation := findValidateViolation(result, DiscountViolationCostBreach)
		require.NotNil(t, violation)
		assert.Equal(t, "claude-*", violation.ScopeValue)
		assert.Contains(t, violation.Detail, "0.950000", "要让运营看到最低进货折扣")
		assert.Equal(t, []string{"Anthropic 偏贵线路", "Anthropic 更贵线路"}, violation.AvailableChannels)
	})
}

// 线路没录进货折扣时不下结论：不报击穿，但要讲清「这几条线没成本，判不了」。
func TestValidateDiscountWithoutCostRatio(t *testing.T) {
	setupDiscountSimulateTest(t)
	seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	seedSimulateChannel(t, "没录成本的线路", nil, []string{"default"}, "gpt-4o")
	plan := seedValidatePlan(t, "0.900000", "0",
		&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: model.DiscountStatusEnabled},
	)

	result, err := ValidateDiscountPlan(plan.Id, 0, 0)
	require.NoError(t, err)

	assert.True(t, result.Passed, "不知道成本就不该报击穿")
	assert.Empty(t, result.Violations)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "都没录进货折扣")
}

// 指定线路时就只看它：「改完这条线路的进货价先验一验」是这个参数的用途。
func TestValidateDiscountForSpecifiedChannel(t *testing.T) {
	setupDiscountSimulateTest(t)
	seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	seedSimulateChannel(t, "便宜线路", costRatioPtr("0.27"), []string{"default"}, "gpt-4o")
	// 偏贵线路的进货折扣高于方案基础折扣 0.9；onlyPricey 这条用例依赖这个差额。
	pricey := seedSimulateChannel(t, "偏贵线路", costRatioPtr("0.950000"), []string{"default"}, "gpt-4o")
	otherModelChannel := seedSimulateChannel(t, "别的模型的线路", costRatioPtr("0.50"), []string{"default"}, "gpt-4o-mini")
	plan := seedValidatePlan(t, "0.900000", "0",
		&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: model.DiscountStatusEnabled},
	)

	all, err := ValidateDiscountPlan(plan.Id, 0, 0)
	require.NoError(t, err)
	assert.True(t, all.Passed, "看全部线路时最低进货 0.27 ≤ 0.90")

	onlyPricey, err := ValidateDiscountPlan(plan.Id, 0, pricey.Id)
	require.NoError(t, err)
	assert.False(t, onlyPricey.Passed, "只看偏贵那条就该报出来")
	violation := findValidateViolation(onlyPricey, DiscountViolationCostBreach)
	require.NotNil(t, violation)
	assert.Equal(t, []string{"偏贵线路"}, violation.AvailableChannels)

	unrelated, err := ValidateDiscountPlan(plan.Id, 0, otherModelChannel.Id)
	require.NoError(t, err)
	assert.True(t, unrelated.Passed)
	assert.Contains(t, unrelated.Warnings, "指定的线路不在任何规则的可用范围内，本次没有参与判断")
}

// 停用的规则不生效，不该出现在提示里。
func TestValidateDiscountIgnoresDisabledRule(t *testing.T) {
	setupDiscountSimulateTest(t)
	rule := &model.DiscountRule{
		ScopeType:  model.DiscountScopeModel,
		ScopeValue: "gpt-4o",
		Discount:   "0.100000",
		Status:     model.DiscountStatusEnabled,
	}
	plan := seedValidatePlan(t, "0.900000", "0.500000", rule)
	// 规则表 status 列带 default:1，新建时传 0 会被数据库默认值盖掉，
	// 所以「停用」只能走更新路径——运营侧改停用走的也是这条路。
	rule.Status = model.DiscountStatusDisabled
	require.NoError(t, rule.Update())

	result, err := ValidateDiscountPlan(plan.Id, 0, 0)
	require.NoError(t, err)

	assert.True(t, result.Passed, "停用的规则折扣再低也不该报出来")
	assert.Empty(t, result.Violations)
}

// 输入不存在要能区分出来，controller 据此回 400 而不是 500。
func TestValidateDiscountForMissingTargets(t *testing.T) {
	setupDiscountSimulateTest(t)

	_, err := ValidateDiscountPlan(999999, 0, 0)
	assert.ErrorIs(t, err, ErrDiscountValidatePlanNotFound)

	plan := seedValidatePlan(t, "0.900000", "0")
	_, err = ValidateDiscountPlan(plan.Id, 999999, 0)
	assert.ErrorIs(t, err, ErrDiscountValidateUserNotFound)
}

// 不指定客户时按 default 分组算，指定客户就按他所属分组算——分组口径要和路由侧一致。
func TestValidateDiscountUsesCustomerGroup(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).
		Updates(map[string]interface{}{"group": "vip"}).Error)
	// VIP 线路进货折扣高于方案基础折扣 0.9：会击穿毛利底线。
	seedSimulateChannel(t, "只给 VIP 的线路", costRatioPtr("0.950000"), []string{"vip"}, "gpt-4o")
	plan := seedValidatePlan(t, "0.900000", "0",
		&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: model.DiscountStatusEnabled},
	)

	withoutUser, err := ValidateDiscountPlan(plan.Id, 0, 0)
	require.NoError(t, err)
	assert.False(t, withoutUser.Passed)
	require.NotNil(t, findValidateViolation(withoutUser, DiscountViolationNoChannel), "default 分组下没有这条线路")

	withUser, err := ValidateDiscountPlan(plan.Id, user.Id, 0)
	require.NoError(t, err)
	assert.False(t, withUser.Passed)
	violation := findValidateViolation(withUser, DiscountViolationCostBreach)
	require.NotNil(t, violation, "vip 分组下这条线路 0.95 > 0.9，会亏")
	assert.Equal(t, []string{"只给 VIP 的线路"}, violation.AvailableChannels)
}
