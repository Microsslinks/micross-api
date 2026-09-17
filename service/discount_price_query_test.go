package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 客户查价的验收：一个客户 + 一个（或一串）模型进去，一张「现价 + 为什么是这个价」出来。
// 三条不能破：钱的数字与核算完全一致（同一份口径）、落选者必须写明原因、价目本必须
// 与计价读到的绑定一致（窗口外的价也要列出来）。

// priceQueryModelRow 从结果里取出指定模型的行，缺行时让测试直接失败并说明缺谁。
func priceQueryModelRow(t *testing.T, result *CustomerPriceQueryResult, modelName string) *CustomerPriceQueryModel {
	t.Helper()
	for _, row := range result.Models {
		if row.Model == modelName {
			return row
		}
	}
	t.Fatalf("查价结果里没有模型 %s", modelName)
	return nil
}

// seedPriceQueryTwoPlans 造一个挂了两套价的客户：平台方案（BaseDiscount 0.8）
// 与经销商方案（BaseDiscount 0.85），返回两套方案，供断言按方案 id 取价目本条目。
// task-17 §17.1 (c) 修复：v0.32.0 规则退化为纯范围标记，platformPlan / agentPlan
// 的 BaseDiscount 决定 gpt / claude 实际折扣（不是 rule.Discount）。
func seedPriceQueryTwoPlans(t *testing.T, userId int) (*model.DiscountPlan, *model.DiscountPlan) {
	t.Helper()
	platformPlan := bindSimulatePlan(t, userId, "0.800000",
		&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.800000", Status: model.DiscountStatusEnabled},
	)
	agentPlan := &model.DiscountPlan{
		Name:         fmt.Sprintf("price-query-agent-plan-%d", time.Now().UnixNano()),
		OwnerType:    model.DiscountOwnerAgent,
		BaseDiscount: "0.850000",
		MinDiscount:  "0",
		BillingMode:  model.DiscountBillingUsage,
		Status:       model.DiscountStatusEnabled,
	}
	require.NoError(t, agentPlan.Insert())
	require.NoError(t, (&model.DiscountRule{
		PlanId: agentPlan.Id, ScopeType: model.DiscountScopeModel, ScopeValue: "claude-3-5-sonnet",
		Discount: "0.850000", Status: model.DiscountStatusEnabled,
	}).Insert())
	require.NoError(t, model.BindDiscountPlan(&model.DiscountBinding{
		SubjectType: model.DiscountSubjectUser,
		SubjectId:   userId,
		PlanId:      agentPlan.Id,
		Source:      model.DiscountSourceAgent,
		Status:      model.DiscountStatusEnabled,
	}))
	return platformPlan, agentPlan
}

func TestQueryCustomerPricing(t *testing.T) {
	t.Run("价目本摊出身上全部方案，逐模型给出胜者与落选原因", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		platformPlan, agentPlan := seedPriceQueryTwoPlans(t, user.Id)
		seedSimulateChannel(t, "gpt 便宜线", costRatioPtr("0.700000"), []string{"default"}, "gpt-4o")
		seedSimulateChannel(t, "claude 贵线", costRatioPtr("0.900000"), []string{"default"}, "claude-3-5-sonnet")

		result, err := QueryCustomerPricing(user.Id, []string{"gpt-4o", "claude-3-5-sonnet"})
		require.NoError(t, err)

		// 价目本：两套价都在，各自写清谁挂的、基础折扣、管哪些模型。
		require.Len(t, result.PriceBook, 2)
		entriesByPlan := make(map[int]*model.CustomerPriceBookEntry, len(result.PriceBook))
		for _, entry := range result.PriceBook {
			entriesByPlan[entry.PlanId] = entry
		}
		platformEntry := entriesByPlan[platformPlan.Id]
		require.NotNil(t, platformEntry)
		assert.Equal(t, model.DiscountSourceManual, platformEntry.Source)
		assert.True(t, platformEntry.InWindow)
		assert.Empty(t, platformEntry.WindowReason)
		// task-17 §17.1 (c)：seedPriceQueryTwoPlans 已把 BaseDiscount 改成 0.8。
		assert.Equal(t, "0.800000", platformEntry.BaseDiscount)
		require.Len(t, platformEntry.Rules, 1)
		assert.Equal(t, "gpt-4o", platformEntry.Rules[0].ScopeValue)
		assert.Equal(t, "0.800000", platformEntry.Rules[0].Discount)

		agentEntry := entriesByPlan[agentPlan.Id]
		require.NotNil(t, agentEntry)
		assert.Equal(t, model.DiscountSourceAgent, agentEntry.Source)

		// 逐模型：钱与核算一致（同一份口径），候选明细里胜者一条、落选者也带原因。
		require.Len(t, result.Models, 2)
		gptRow := priceQueryModelRow(t, result, "gpt-4o")
		assert.Equal(t, "0.800000", gptRow.Discount)
		assert.Equal(t, model.DiscountResolvedFromModel, gptRow.Source)
		assert.Equal(t, platformPlan.Id, gptRow.Plan.Id)
		require.NotNil(t, gptRow.GrossMargin)
		assert.Equal(t, "0.142857", *gptRow.GrossMargin)
		require.Len(t, gptRow.Candidates, 2)
		applied := 0
		for _, candidate := range gptRow.Candidates {
			if candidate.Applied {
				applied++
				assert.Equal(t, platformPlan.Id, candidate.PlanId)
				assert.Empty(t, candidate.RejectReason)
				continue
			}
			assert.NotEmpty(t, candidate.RejectReason)
		}
		assert.Equal(t, 1, applied)

		claudeRow := priceQueryModelRow(t, result, "claude-3-5-sonnet")
		assert.Equal(t, "0.850000", claudeRow.Discount)
		assert.Equal(t, agentPlan.Id, claudeRow.Plan.Id)

		// 汇总沿用核算的口径：一个保本一个亏 → 不能签。
		assert.Equal(t, 2, result.Summary.Total)
		assert.False(t, result.Summary.Signable)
	})

	t.Run("查价的钱与核算逐行一致", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		seedPriceQueryTwoPlans(t, user.Id)
		seedSimulateChannel(t, "gpt 便宜线", costRatioPtr("0.700000"), []string{"default"}, "gpt-4o")
		seedSimulateChannel(t, "claude 贵线", costRatioPtr("0.900000"), []string{"default"}, "claude-3-5-sonnet")

		audit, err := AuditCustomerPricing(user.Id, []string{"gpt-4o", "claude-3-5-sonnet"})
		require.NoError(t, err)
		queried, err := QueryCustomerPricing(user.Id, []string{"gpt-4o", "claude-3-5-sonnet"})
		require.NoError(t, err)

		require.Len(t, queried.Models, len(audit.Models))
		assert.Equal(t, audit.MinMarginRatio, queried.MinMarginRatio)
		assert.Equal(t, audit.Summary, queried.Summary)
		for index, auditRow := range audit.Models {
			queryRow := queried.Models[index]
			assert.Equal(t, auditRow.Model, queryRow.Model)
			assert.Equal(t, auditRow.Discount, queryRow.Discount)
			assert.Equal(t, auditRow.Verdict, queryRow.Verdict)
			assert.Equal(t, auditRow.ChannelCount, queryRow.ChannelCount)
			assert.Equal(t, auditRow.CheapestCost, queryRow.CheapestCost)
			assert.Equal(t, auditRow.GrossMargin, queryRow.GrossMargin)
		}
	})

	t.Run("不填模型名时只回价目本", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		bindSimulatePlan(t, user.Id, "0.900000")

		result, err := QueryCustomerPricing(user.Id, []string{"", "   "})
		require.NoError(t, err)
		require.Len(t, result.PriceBook, 1)
		assert.Empty(t, result.Models)
		assert.Equal(t, 0, result.Summary.Total)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "还没填模型名")
	})

	t.Run("生效时间窗外的价也列进价目本并写明原因", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		futurePlan := &model.DiscountPlan{
			Name:         fmt.Sprintf("price-query-future-plan-%d", time.Now().UnixNano()),
			OwnerType:    model.DiscountOwnerPlatform,
			BaseDiscount: "0.880000",
			MinDiscount:  "0",
			BillingMode:  model.DiscountBillingUsage,
			Status:       model.DiscountStatusEnabled,
		}
		require.NoError(t, futurePlan.Insert())
		require.NoError(t, model.BindDiscountPlan(&model.DiscountBinding{
			SubjectType:   model.DiscountSubjectUser,
			SubjectId:     user.Id,
			PlanId:        futurePlan.Id,
			Source:        model.DiscountSourceManual,
			EffectiveFrom: time.Now().Unix() + 86400,
			Status:        model.DiscountStatusEnabled,
		}))

		result, err := QueryCustomerPricing(user.Id, nil)
		require.NoError(t, err)
		require.Len(t, result.PriceBook, 1)
		entry := result.PriceBook[0]
		assert.False(t, entry.InWindow)
		assert.Equal(t, "尚未到生效时间", entry.WindowReason)
		// 没生效的价也要能被解释：方案名与规则照样给出来。
		assert.Equal(t, futurePlan.Name, entry.PlanName)
		assert.Equal(t, "0.880000", entry.BaseDiscount)
	})

	t.Run("客户不存在时把话说清楚", func(t *testing.T) {
		setupDiscountSimulateTest(t)

		_, err := QueryCustomerPricing(999999, []string{"gpt-4o"})
		assert.ErrorIs(t, err, ErrCustomerAuditUserNotFound)

		tooMany := make([]string, CustomerAuditMaxModels+1)
		for i := range tooMany {
			tooMany[i] = fmt.Sprintf("model-%d", i)
		}
		user := seedSimulateCustomer(t, "", "")
		_, err = QueryCustomerPricing(user.Id, tooMany)
		assert.ErrorIs(t, err, ErrCustomerAuditTooManyModels)
	})
}
