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

// 客户档案核算的验收：一个客户 + 一份模型清单进去，一张赚亏表出来。
// 折扣必须走多方案裁决（与计费同一套代码），成本口径必须与试算一致。

// auditModelRow 从结果里取出指定模型的行，缺行时让测试直接失败并说明缺谁。
func auditModelRow(t *testing.T, result *CustomerPricingAuditResult, modelName string) *CustomerPricingAuditModel {
	t.Helper()
	for _, row := range result.Models {
		if row.Model == modelName {
			return row
		}
	}
	t.Fatalf("核算结果里没有模型 %s", modelName)
	return nil
}

func TestAuditCustomerPricing(t *testing.T) {
	t.Run("多方案各管一片模型，核算表按裁决结果算赚亏", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		// task-17 §17.1 (c) 修复：v0.32.0 规则退化为纯范围标记（commit 003c2755），
		// 计费用 discount 等于 plan.BaseDiscount 而非 rule.Discount。原来测试期望
		// 命中的 rule.Discount=0.8（gpt）/ 0.85（claude），实际只会用 plan.BaseDiscount。
		// 这里把 platformPlan / agentPlan 的 BaseDiscount 分别设为 0.8 / 0.85，
		// 让两个方案仍能体现"不同模型不同价"但走 base discount 路径。
		platformPlan := bindSimulatePlan(t, user.Id, "0.800000",
			&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.800000", Status: model.DiscountStatusEnabled},
		)
		// 经销商来源的第二套方案：claude 模型他更便宜（BaseDiscount 0.85 → 进货 0.9 会亏）。
		agentPlan := &model.DiscountPlan{
			Name:         fmt.Sprintf("audit-agent-plan-%d", time.Now().UnixNano()),
			OwnerType:    model.DiscountOwnerAgent,
			BaseDiscount: "0.850000",
			MinDiscount:  "0",
			BillingMode:  model.DiscountBillingUsage,
			Status:       model.DiscountStatusEnabled,
		}
		require.NoError(t, agentPlan.Insert())
		require.NoError(t, (&model.DiscountRule{
			PlanId: agentPlan.Id, ScopeType: model.DiscountScopeModel, ScopeValue: "claude-*",
			Discount: "0.850000", Status: model.DiscountStatusEnabled,
		}).Insert())
		require.NoError(t, model.BindDiscountPlan(&model.DiscountBinding{
			SubjectType: model.DiscountSubjectUser,
			SubjectId:   user.Id,
			PlanId:      agentPlan.Id,
			Source:      model.DiscountSourceAgent,
			Status:      model.DiscountStatusEnabled,
		}))

		seedSimulateChannel(t, "gpt 便宜线", costRatioPtr("0.700000"), []string{"default"}, "gpt-4o")
		seedSimulateChannel(t, "claude 贵线", costRatioPtr("0.900000"), []string{"default"}, "claude-3-5-sonnet")

		result, err := AuditCustomerPricing(user.Id, []string{"gpt-4o", "claude-3-5-sonnet"})
		require.NoError(t, err)

		// gpt-4o：platformPlan BaseDiscount=0.8（v0.32.0 规则退化为纯范围标记后，
		// discount 用 plan.BaseDiscount 而不是 rule.Discount）。
		// 进货 0.7 → 保本，毛利 0.8/0.7-1 = 0.142857。
		gptRow := auditModelRow(t, result, "gpt-4o")
		assert.Equal(t, "0.800000", gptRow.Discount)
		assert.Equal(t, model.DiscountResolvedFromModel, gptRow.Source)
		assert.Equal(t, platformPlan.Id, gptRow.Plan.Id)
		assert.Equal(t, CustomerAuditVerdictOK, gptRow.Verdict)
		assert.Equal(t, 1, gptRow.UsableCount)
		require.NotNil(t, gptRow.CheapestCost)
		assert.Equal(t, "0.700000", *gptRow.CheapestCost)
		require.NotNil(t, gptRow.GrossMargin)
		assert.Equal(t, "0.142857", *gptRow.GrossMargin)

		// claude：agentPlan BaseDiscount=0.85，进货 0.9 → 亏（毛利 0.85/0.9-1 = -0.055556）。
		claudeRow := auditModelRow(t, result, "claude-3-5-sonnet")
		assert.Equal(t, "0.850000", claudeRow.Discount)
		assert.Equal(t, agentPlan.Id, claudeRow.Plan.Id)
		assert.Equal(t, CustomerAuditVerdictLoss, claudeRow.Verdict)
		require.NotNil(t, claudeRow.GrossMargin)
		assert.Equal(t, "-0.055556", *claudeRow.GrossMargin)

		// 汇总：一个保本一个亏 → 不能签。
		assert.False(t, result.Summary.Signable)
		assert.Equal(t, 1, result.Summary.OK)
		assert.Equal(t, 1, result.Summary.Loss)
		assert.Contains(t, result.Summary.Conclusion, "1 个会亏")
		assert.Empty(t, result.Warnings)
	})

	t.Run("四种结论齐全：保本、会亏、成本未知、没有线路", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		// 毛利底线 0.1：进货折扣要比客户折扣低 0.1 以上才算保本。
		operation_setting.GetDiscountSetting().MinMarginRatio = "0.100000"
		user := seedSimulateCustomer(t, "", "")
		bindSimulatePlan(t, user.Id, "0.900000")

		seedSimulateChannel(t, "ok 线", costRatioPtr("0.700000"), []string{"default"}, "ok-model")
		seedSimulateChannel(t, "loss 线", costRatioPtr("0.850000"), []string{"default"}, "loss-model")
		seedSimulateChannel(t, "unknown 线", nil, []string{"default"}, "unknown-cost-model")

		result, err := AuditCustomerPricing(user.Id, []string{"ok-model", "loss-model", "unknown-cost-model", "no-channel-model"})
		require.NoError(t, err)

		assert.Equal(t, "0.900000", auditModelRow(t, result, "ok-model").Discount)
		assert.Equal(t, CustomerAuditVerdictOK, auditModelRow(t, result, "ok-model").Verdict)
		assert.Equal(t, CustomerAuditVerdictLoss, auditModelRow(t, result, "loss-model").Verdict)
		assert.Contains(t, auditModelRow(t, result, "loss-model").VerdictDetail, "毛利底线")

		unknownRow := auditModelRow(t, result, "unknown-cost-model")
		assert.Equal(t, CustomerAuditVerdictUnknownCost, unknownRow.Verdict)
		assert.Nil(t, unknownRow.CheapestCost)
		assert.Nil(t, unknownRow.GrossMargin)

		noChannelRow := auditModelRow(t, result, "no-channel-model")
		assert.Equal(t, CustomerAuditVerdictNoChannel, noChannelRow.Verdict)
		assert.Equal(t, 0, noChannelRow.ChannelCount)

		summary := result.Summary
		assert.Equal(t, 4, summary.Total)
		assert.Equal(t, 1, summary.OK)
		assert.Equal(t, 1, summary.Loss)
		assert.Equal(t, 1, summary.UnknownCost)
		assert.Equal(t, 1, summary.NoChannel)
		assert.False(t, summary.Signable)
		assert.Contains(t, summary.Conclusion, "1 个会亏")
		assert.Contains(t, summary.Conclusion, "1 个没有可用线路")
		assert.Contains(t, summary.Conclusion, "1 个成本未知")
	})

	t.Run("全部保本时结论是可以签", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		bindSimulatePlan(t, user.Id, "0.900000")
		seedSimulateChannel(t, "便宜线", costRatioPtr("0.700000"), []string{"default"}, "gpt-4o")

		result, err := AuditCustomerPricing(user.Id, []string{"gpt-4o"})
		require.NoError(t, err)
		assert.True(t, result.Summary.Signable)
		assert.Contains(t, result.Summary.Conclusion, "可以签")
	})

	t.Run("清单修剪去重并保留输入顺序", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		bindSimulatePlan(t, user.Id, "0.900000")
		seedSimulateChannel(t, "gpt 线", costRatioPtr("0.700000"), []string{"default"}, "gpt-4o")
		seedSimulateChannel(t, "claude 线", costRatioPtr("0.700000"), []string{"default"}, "claude-3-5-sonnet")

		result, err := AuditCustomerPricing(user.Id, []string{
			"  claude-3-5-sonnet，gpt-4o  ", "gpt-4o", "", "claude-3-5-sonnet",
		})
		require.NoError(t, err)
		require.Len(t, result.Models, 2)
		assert.Equal(t, "claude-3-5-sonnet", result.Models[0].Model)
		assert.Equal(t, "gpt-4o", result.Models[1].Model)
		assert.Equal(t, 2, result.Summary.Total)
	})

	t.Run("没绑方案的客户按官方标价并给出提示", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")
		seedSimulateChannel(t, "gpt 线", costRatioPtr("0.700000"), []string{"default"}, "gpt-4o")

		result, err := AuditCustomerPricing(user.Id, []string{"gpt-4o"})
		require.NoError(t, err)
		row := auditModelRow(t, result, "gpt-4o")
		assert.Equal(t, model.DiscountNone, row.Discount)
		assert.Equal(t, model.DiscountResolvedFromDefault, row.Source)
		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "未绑定任何折扣方案")
	})

	t.Run("经销商按拿货价核算并给出提示", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		// 毛利底线是全局设置，上个用例改成了 0.1——这里要的是默认的 0，显式设回。
		operation_setting.GetDiscountSetting().MinMarginRatio = "0"
		require.NoError(t, model.DB.AutoMigrate(&model.AgentProfile{}))
		user := seedSimulateCustomer(t, "", "")
		require.NoError(t, model.DB.Model(&model.User{}).Where("id = ?", user.Id).
			Update("subject_type", model.SubjectTypeAgent).Error)
		require.NoError(t, model.DB.Create(&model.AgentProfile{
			UserId: user.Id, MarkupRatio: "1.100000", WholesaleDiscount: "0", MinDiscount: "0",
		}).Error)
		// 经销商拿货价 = 最便宜线路进货折扣 0.5 × 加价率 1.1 = 0.55。
		seedSimulateChannel(t, "拿货线", costRatioPtr("0.500000"), []string{"default"}, "gpt-4o")
		seedSimulateChannel(t, "没目录的线", costRatioPtr("0.500000"), []string{"default"}, "claude-3-5-sonnet")

		result, err := AuditCustomerPricing(user.Id, []string{"gpt-4o", "claude-3-5-sonnet"})
		require.NoError(t, err)

		gptRow := auditModelRow(t, result, "gpt-4o")
		assert.Equal(t, "0.550000", gptRow.Discount)
		assert.Equal(t, model.DiscountResolvedFromAgentWholesale, gptRow.Source)
		assert.Equal(t, CustomerAuditVerdictOK, gptRow.Verdict)
		require.NotNil(t, gptRow.GrossMargin)
		assert.Equal(t, "0.100000", *gptRow.GrossMargin)

		require.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "经销商")
	})

	t.Run("输入不对时把话说清楚", func(t *testing.T) {
		setupDiscountSimulateTest(t)
		user := seedSimulateCustomer(t, "", "")

		_, err := AuditCustomerPricing(user.Id, []string{"", "  "})
		assert.ErrorIs(t, err, ErrCustomerAuditNoModels)

		tooMany := make([]string, CustomerAuditMaxModels+1)
		for i := range tooMany {
			tooMany[i] = fmt.Sprintf("model-%d", i)
		}
		_, err = AuditCustomerPricing(user.Id, tooMany)
		assert.ErrorIs(t, err, ErrCustomerAuditTooManyModels)

		_, err = AuditCustomerPricing(999999, []string{"gpt-4o"})
		assert.ErrorIs(t, err, ErrCustomerAuditUserNotFound)
	})
}
