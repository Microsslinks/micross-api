package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 客户挂了多套方案时，试算要把"没用上的那几套"一并说清楚，并且与计费算出同一个折扣。
// 老实现只读快路径上那一条绑定：界面显示的价可能根本不是这一单真扣的价。
// rule.Discount 已废弃：胜出方案的命中规则（gpt-4o）按方案基础折扣（plan.BaseDiscount=0.9）出价。
func TestSimulateDiscountWithSeveralBoundPlans(t *testing.T) {
	setupDiscountSimulateTest(t)
	setting := operation_setting.GetDiscountSetting()
	previousSwitch := setting.EnableBillingDiscount
	setting.EnableBillingDiscount = true
	t.Cleanup(func() {
		setting.EnableBillingDiscount = previousSwitch
	})

	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	winner := bindSimulatePlan(t, user.Id, "0.900000", &model.DiscountRule{
		ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: model.DiscountStatusEnabled,
	})
	// 第二条更便宜，但只有方案基础折扣：具体度上先输给上面那条模型级规则。
	// 它必须出现在候选里，而不是被静默丢掉。
	loserPlan := &model.DiscountPlan{
		Name:         fmt.Sprintf("simulate-loser-%d", time.Now().UnixNano()),
		OwnerType:    model.DiscountOwnerPlatform,
		BaseDiscount: "0.500000",
		MinDiscount:  "0",
		BillingMode:  model.DiscountBillingUsage,
		Status:       model.DiscountStatusEnabled,
	}
	require.NoError(t, loserPlan.Insert())
	require.NoError(t, model.BindDiscountPlan(&model.DiscountBinding{
		SubjectType: model.DiscountSubjectUser,
		SubjectId:   user.Id,
		PlanId:      loserPlan.Id,
		Source:      model.DiscountSourceSubscription,
		Status:      model.DiscountStatusEnabled,
	}))
	seedSimulateChannel(t, "微观互联-GPT", costRatioPtr("0.27"), []string{"default"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	// 与计费同一口径：胜出方案命中模型级规则，按方案基础折扣 0.9 出价。
	assert.Equal(t, "0.900000", result.Resolution.Discount)
	billing := model.ResolveBillingDiscount(user.Id, "gpt-4o")
	assert.InDelta(t, 0.9, billing.Ratio, 1e-9)
	require.NotNil(t, result.Plan)
	assert.Equal(t, winner.Id, result.Plan.Id)
	assert.Equal(t, winner.Id, billing.PlanId)

	// 两条绑定都摊出来，且正好一条是这次生效的。
	require.Len(t, result.Candidates, 2)
	applied := 0
	for _, candidate := range result.Candidates {
		switch candidate.PlanId {
		case winner.Id:
			assert.True(t, candidate.Applied)
			assert.False(t, candidate.Rejected)
			// rule.Discount 已废弃：胜者命中规则按方案基础折扣（0.9）出价。
			assert.Equal(t, "0.900000", candidate.Discount)
			assert.Equal(t, model.DiscountSourceManual, candidate.Source)
			assert.Equal(t, model.DiscountResolvedFromModel, candidate.Specificity)
			applied++
		case loserPlan.Id:
			assert.False(t, candidate.Applied)
			assert.True(t, candidate.Rejected)
			assert.Equal(t, "0.500000", candidate.Discount)
			// 原因来自裁决原文：界面直接把这句话显示给运营，留空就只剩一个"未生效"。
			assert.Contains(t, candidate.RejectReason, "具体度")
		default:
			t.Fatalf("出现了没预料到的候选方案 %d", candidate.PlanId)
		}
	}
	assert.Equal(t, 1, applied)

	// 挂了几套就要说出来，不能让运营以为试算只认一套。
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "2 套折扣方案")
}
