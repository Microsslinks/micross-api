package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 消费日志要回答"为什么不是按他另一套方案算"，所以计费解析除了乘数还得把落选的绑定带出来。
// 这里盯的就是那几个字段：落选的是哪一套、什么价、输在哪一层。
func TestResolveBillingDiscountCarriesRejectedBindings(t *testing.T) {
	t.Run("挂了两套：胜者生效，落选者带原因", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		winner := seedMultiPlan(t, "0.900000", &DiscountRule{
			ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: DiscountStatusEnabled,
		})
		// 这一套更便宜，但只有方案基础折扣：具体度上先输给上面那条模型级规则。
		loser := seedMultiPlan(t, "0.500000")
		bindMultiPlan(t, user.Id, winner, DiscountSourceManual)
		bindMultiPlan(t, user.Id, loser, DiscountSourceSubscription)

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		assert.InDelta(t, 0.3, discount.Ratio, 1e-9)
		assert.Equal(t, winner.Id, discount.PlanId)
		require.Len(t, discount.Rejected, 1)
		assert.Equal(t, loser.Id, discount.Rejected[0].PlanId)
		assert.Equal(t, DiscountSourceSubscription, discount.Rejected[0].Source)
		assert.Equal(t, "0.500000", discount.Rejected[0].Discount)
		assert.Contains(t, discount.Rejected[0].Reason, "具体度")
	})

	t.Run("只挂一套：没有落选者", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.800000")

		discount := ResolveBillingDiscount(user.Id, "gpt-4o")

		assert.True(t, discount.Applied())
		assert.Empty(t, discount.Rejected, "没人跟它抢，日志里不该多出落选字段")
	})
}
