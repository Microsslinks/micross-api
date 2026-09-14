package model

import (
	"testing"

	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setBillingDiscountSwitch 拨动「折扣是否参与计费」总开关，并在用例收尾还原。
// 它是全局设置，同一文件里的用例靠它互相隔离（数据 fixture 见 discount_resolve_test.go）。
func setBillingDiscountSwitch(t *testing.T, enabled bool) {
	t.Helper()
	setting := operation_setting.GetDiscountSetting()
	previous := setting.EnableBillingDiscount
	setting.EnableBillingDiscount = enabled
	t.Cleanup(func() {
		setting.EnableBillingDiscount = previous
	})
}

// TestResolveBillingDiscount 盯的是「什么情况下真的打折」这一个判断。
// 金额怎么乘由 relay/helper 的测试和人工逐笔比对负责，
// 这里只保证喂给计费的乘数是对的——它是预扣、结算、任务重算三处共用的那一份。
func TestResolveBillingDiscount(t *testing.T) {
	// 总开关是唯一能一票否决的开关：关着时哪怕客户绑了 5 折方案，也必须原价。
	t.Run("总开关关着：绑了方案的客户也按官方标价", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, false)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.800000")

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		assert.Equal(t, 1.0, discount.Ratio)
		assert.False(t, discount.Applied())
	})

	t.Run("没有客户身份：不打折", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)

		discount := ResolveBillingDiscount(0, "gpt-4o")

		assert.Equal(t, 1.0, discount.Ratio)
		assert.False(t, discount.Applied())
	})

	t.Run("没绑方案：不打折", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		assert.Equal(t, 1.0, discount.Ratio)
		assert.False(t, discount.Applied())
	})

	t.Run("方案停用：不打折", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := bindResolvePlan(t, user.Id, "0.800000")

		plan.Status = DiscountStatusDisabled
		require.NoError(t, plan.Update())

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		assert.Equal(t, 1.0, discount.Ratio)
		assert.False(t, discount.Applied())
	})

	t.Run("绑了启用中的方案：用方案基础折扣", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := bindResolvePlan(t, user.Id, "0.800000")

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		assert.InDelta(t, 0.8, discount.Ratio, 1e-9)
		assert.True(t, discount.Applied())
		assert.Equal(t, DiscountResolvedFromPlanBase, discount.Source)
		assert.Equal(t, plan.Id, discount.PlanId)
	})

	t.Run("命中模型级规则：按方案基础折扣（rule.Discount 已废弃）", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := bindResolvePlan(t, user.Id, "0.900000", &DiscountRule{
			ScopeType:  DiscountScopeModel,
			ScopeValue: "gpt-4o",
			Discount:   "0.700000",
			Status:     DiscountStatusEnabled,
		})

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		// rule.Discount 已废弃：命中规则时按方案基础折扣出价（plan.BaseDiscount = 0.9）。
		// rule.Discount 字段保留在这里仅为让结构体可正常 Insert，写什么值都不影响计费。
		assert.InDelta(t, 0.9, discount.Ratio, 1e-9)
		assert.True(t, discount.Applied())
		assert.Equal(t, DiscountResolvedFromModel, discount.Source)
		assert.Equal(t, plan.Id, discount.PlanId)
	})

	// 折扣读不出数字时不能把整条计费链路带停：记一条错误日志，这一单按官方标价算。
	t.Run("折扣字段坏掉：回退官方标价而不是报错", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := bindResolvePlan(t, user.Id, "0.800000")

		// 绕过写入时的规范化，模拟库里已经躺着一条脏数据。
		require.NoError(t, DB.Exec("UPDATE discount_plans SET base_discount = ? WHERE id = ?", "not-a-number", plan.Id).Error)

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		assert.Equal(t, 1.0, discount.Ratio)
		assert.False(t, discount.Applied())
	})
}
