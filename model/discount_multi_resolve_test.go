package model

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 多方案解析测试。环境与 discount_resolve_test.go 共用一套搭法
// （setupResolveTest / seedResolveCustomer），这里只补「按指定来源建方案并绑定」的夹具。

// multiPlanSeq 兜底唯一性：Windows 时钟分辨率下连续建方案会撞出同一个纳秒时间戳，
// 光靠 UnixNano 撞过 discount_plans 的 (owner, name) 唯一索引。
var multiPlanSeq int64

// seedMultiPlan 建一个方案（可选规则）但不绑定；绑定由 bindMultiPlan 按来源单独做，
// 因为多方案测试的核心变量就是「同一客户身上不同来源的多条绑定」。
func seedMultiPlan(t *testing.T, baseDiscount string, rules ...*DiscountRule) *DiscountPlan {
	t.Helper()
	plan := &DiscountPlan{
		Name:         fmt.Sprintf("multi-plan-%d-%d", time.Now().UnixNano(), atomic.AddInt64(&multiPlanSeq, 1)),
		OwnerType:    DiscountOwnerPlatform,
		BaseDiscount: baseDiscount,
		MinDiscount:  "0",
		BillingMode:  DiscountBillingUsage,
		Status:       DiscountStatusEnabled,
	}
	require.NoError(t, plan.Insert())
	for _, rule := range rules {
		rule.PlanId = plan.Id
		require.NoError(t, rule.Insert())
	}
	return plan
}

// bindMultiPlan 把方案按指定来源绑到客户身上。走 BindDiscountPlan，快路径由它同步。
func bindMultiPlan(t *testing.T, userId int, plan *DiscountPlan, source string) *DiscountBinding {
	t.Helper()
	binding := &DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   userId,
		PlanId:      plan.Id,
		Source:      source,
		Status:      DiscountStatusEnabled,
	}
	require.NoError(t, BindDiscountPlan(binding))
	return binding
}

// TestResolveUserDiscountMulti 盯的是多方案裁决的层序，以及「现状下与单方案解析逐字一致」。
func TestResolveUserDiscountMulti(t *testing.T) {
	// 现在存量客户每人至多一条生效绑定：两种读法必须给出同一个答案，
	// 这是计费入口敢换过去的全部前提。
	t.Run("只有一条绑定时与单方案解析完全一致", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.800000", Status: DiscountStatusEnabled},
		)

		single, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		multi, err := ResolveUserDiscountMulti(user.Id, "gpt-4o")
		require.NoError(t, err)

		assert.Equal(t, single.Discount, multi.Discount)
		assert.Equal(t, single.Source, multi.Source)
		assert.Equal(t, single.PlanId, multi.PlanId)
		assert.Equal(t, "0.800000", multi.Discount)
		assert.Equal(t, DiscountResolvedFromModel, multi.Source)
	})

	// 01-plan 的旗舰场景：平台企业VIP（基础 95 折）、经销商谈的 Claude 9 折（模型级）、
	// 客户号签的 95 折。调 Claude 时必须是经销商那条模型级规则赢，而不是被
	// 来源更高的平台方案的基础折扣静默顶掉。
	t.Run("模型级规则压过其他方案的基础折扣", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "Anthropic", "claude-3-5-sonnet")
		platformPlan := seedMultiPlan(t, "0.950000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.800000", Status: DiscountStatusEnabled},
		)
		agentPlan := seedMultiPlan(t, "0.950000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "claude-*", Discount: "0.900000", Status: DiscountStatusEnabled},
		)
		codePlan := seedMultiPlan(t, "0.950000")
		bindMultiPlan(t, user.Id, platformPlan, DiscountSourceManual)
		bindMultiPlan(t, user.Id, agentPlan, DiscountSourceAgent)
		bindMultiPlan(t, user.Id, codePlan, DiscountSourceCustomerCode)

		resolution, candidates, err := ResolveUserDiscountDetailed(user.Id, "claude-3-5-sonnet")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromModel, resolution.Source)
		assert.Equal(t, agentPlan.Id, resolution.PlanId)
		require.NotNil(t, resolution.Rule)
		assert.Equal(t, "claude-*", resolution.Rule.ScopeValue)

		// 平台方案对 GPT 有模型级规则，调 gpt-4o 时换它赢——各管一片模型。
		resolution, _, err = ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.800000", resolution.Discount)
		assert.Equal(t, platformPlan.Id, resolution.PlanId)

		// 客户号方案只有基础折扣，两头都轮不到它。
		for _, candidate := range candidates {
			if candidate.Plan != nil && candidate.Plan.Id == codePlan.Id {
				assert.True(t, candidate.Rejected)
				assert.Contains(t, candidate.RejectReason, "具体度")
			}
		}
	})

	// 具体度相同看定价方：平台的 95 折压过经销商的 85 折——
	// 平台明确拍板的价，经销商不能靠便宜顶掉。
	t.Run("同具体度时平台定价压过经销商定价", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		platformPlan := seedMultiPlan(t, "1.000000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.950000", Status: DiscountStatusEnabled},
		)
		agentPlan := seedMultiPlan(t, "1.000000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-*", Discount: "0.850000", Status: DiscountStatusEnabled},
		)
		bindMultiPlan(t, user.Id, platformPlan, DiscountSourceManual)
		bindMultiPlan(t, user.Id, agentPlan, DiscountSourceAgent)

		resolution, candidates, err := ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.950000", resolution.Discount)
		assert.Equal(t, platformPlan.Id, resolution.PlanId)

		for _, candidate := range candidates {
			if candidate.Plan != nil && candidate.Plan.Id == agentPlan.Id {
				assert.True(t, candidate.Rejected)
				assert.Contains(t, candidate.RejectReason, "定价方")
			}
		}
	})

	// 同一个人定的两套价：取便宜的（对客户有利）。
	t.Run("同具体度同来源时取更便宜的那套", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		expensive := seedMultiPlan(t, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.900000", Status: DiscountStatusEnabled},
		)
		cheap := seedMultiPlan(t, "0.950000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.800000", Status: DiscountStatusEnabled},
		)
		bindMultiPlan(t, user.Id, expensive, DiscountSourceManual)
		bindMultiPlan(t, user.Id, cheap, DiscountSourceManual)

		resolution, candidates, err := ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.800000", resolution.Discount)
		assert.Equal(t, cheap.Id, resolution.PlanId)

		for _, candidate := range candidates {
			if candidate.Plan != nil && candidate.Plan.Id == expensive.Id {
				assert.True(t, candidate.Rejected)
				assert.Contains(t, candidate.RejectReason, "便宜")
			}
		}
	})

	// 价格也打平时取更新的绑定：谁后挂的算谁的。
	t.Run("价格也相同时取更新的绑定", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		older := seedMultiPlan(t, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.800000", Status: DiscountStatusEnabled},
		)
		newer := seedMultiPlan(t, "0.950000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.800000", Status: DiscountStatusEnabled},
		)
		olderBinding := bindMultiPlan(t, user.Id, older, DiscountSourceManual)
		newerBinding := bindMultiPlan(t, user.Id, newer, DiscountSourceManual)
		require.Greater(t, newerBinding.Id, olderBinding.Id, "自增主键应保证后建的绑定 id 更大")

		resolution, _, err := ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.800000", resolution.Discount)
		assert.Equal(t, newer.Id, resolution.PlanId)
	})

	// 窗口外的绑定不参与裁决，但候选里要能看见原因——
	// 这是"我上周谈的那套价怎么没生效"这类问题的答案来源。
	t.Run("未到生效时间的绑定不参与并标注原因", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := seedMultiPlan(t, "0.900000")
		binding := &DiscountBinding{
			SubjectType:   DiscountSubjectUser,
			SubjectId:     user.Id,
			PlanId:        plan.Id,
			EffectiveFrom: common.GetTimestamp() + 3600,
			Source:        DiscountSourceManual,
			Status:        DiscountStatusEnabled,
		}
		require.NoError(t, BindDiscountPlan(binding))

		resolution, candidates, err := ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, DiscountNone, resolution.Discount)
		assert.Equal(t, DiscountResolvedFromDefault, resolution.Source)
		require.Len(t, candidates, 1)
		assert.True(t, candidates[0].Rejected)
		assert.Contains(t, candidates[0].RejectReason, "尚未到生效时间")
	})

	// 全部绑定都指向停用方案时按官方标价，快路径口径（PlanId 仍指向该方案）不变。
	t.Run("方案停用后回落官方标价且快路径口径不变", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := seedMultiPlan(t, "0.900000")
		bindMultiPlan(t, user.Id, plan, DiscountSourceManual)

		plan.Status = DiscountStatusDisabled
		require.NoError(t, plan.Update())

		resolution, candidates, err := ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, DiscountNone, resolution.Discount)
		assert.Equal(t, DiscountResolvedFromDefault, resolution.Source)
		assert.Equal(t, plan.Id, resolution.PlanId, "快路径仍指向该方案，便于排查为什么没生效")
		require.Len(t, candidates, 1)
		assert.True(t, candidates[0].Rejected)
		assert.Contains(t, candidates[0].RejectReason, "停用")
	})

	t.Run("没有绑定方案的客户按官方标价", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")

		resolution, candidates, err := ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, DiscountNone, resolution.Discount)
		assert.Equal(t, DiscountResolvedFromDefault, resolution.Source)
		assert.Equal(t, 0, resolution.PlanId)
		assert.Empty(t, candidates)
	})

	// 落选原因是解释口径的一部分：每个落选者都必须有一句话说明输在哪。
	t.Run("每个落选候选都写明原因", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		platformPlan := seedMultiPlan(t, "0.950000")
		agentPlan := seedMultiPlan(t, "0.900000")
		codePlan := seedMultiPlan(t, "0.990000")
		bindMultiPlan(t, user.Id, platformPlan, DiscountSourceManual)
		bindMultiPlan(t, user.Id, agentPlan, DiscountSourceAgent)
		bindMultiPlan(t, user.Id, codePlan, DiscountSourceCustomerCode)

		_, candidates, err := ResolveUserDiscountDetailed(user.Id, "gpt-4o")
		require.NoError(t, err)
		require.Len(t, candidates, 3)

		rejected := 0
		winner := 0
		for _, candidate := range candidates {
			if candidate.Rejected {
				rejected++
				assert.NotEmpty(t, candidate.RejectReason)
			} else {
				winner++
			}
		}
		assert.Equal(t, 2, rejected)
		assert.Equal(t, 1, winner, "有且只有一个胜者")
	})
}

// TestResolveBillingDiscountMulti 计费入口换到多方案解析后的端到端校验：
// 总开关、金额乘数、命中的方案 id 都要和单方案时代一致。
func TestResolveBillingDiscountMulti(t *testing.T) {
	t.Run("多条绑定同时生效时计费按裁决结果乘", func(t *testing.T) {
		setupResolveTest(t)
		setBillingDiscountSwitch(t, true)
		user := seedResolveCustomer(t, "Anthropic", "claude-3-5-sonnet")
		platformPlan := seedMultiPlan(t, "0.950000")
		agentPlan := seedMultiPlan(t, "0.950000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "claude-*", Discount: "0.900000", Status: DiscountStatusEnabled},
		)
		bindMultiPlan(t, user.Id, platformPlan, DiscountSourceManual)
		bindMultiPlan(t, user.Id, agentPlan, DiscountSourceAgent)

		discount := ResolveBillingDiscount(user.Id, "claude-3-5-sonnet")

		assert.True(t, discount.Applied())
		assert.InDelta(t, 0.9, discount.Ratio, 1e-9)
		assert.Equal(t, DiscountResolvedFromModel, discount.Source)
		assert.Equal(t, agentPlan.Id, discount.PlanId)
	})
}

// TestFixDiscountRuleUniqueIndex 保护的是「两个方案可以各有同名规则」这个多方案的地基：
// 最初的 uk_rule_plan_scope 漏了 plan_id，建成了全表 (scope_type, scope_value) 唯一，
// 平台方案和经销商方案都写不出各自对 gpt-4o 的报价。启动迁移必须能把老索引换掉。
func TestFixDiscountRuleUniqueIndex(t *testing.T) {
	setupResolveTest(t)

	// 把索引换回建错的老样子：全表唯一、不含 plan_id。
	require.NoError(t, DB.Exec("DROP INDEX uk_rule_plan_scope").Error)
	require.NoError(t, DB.Exec("CREATE UNIQUE INDEX uk_rule_plan_scope ON discount_rules(scope_type, scope_value)").Error)

	// 老索引下两个方案写同名规则必撞，先钉死这个事实，防止测试空转。
	planA := seedMultiPlan(t, "1.000000")
	require.NoError(t, (&DiscountRule{
		PlanId: planA.Id, ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o",
		Discount: "0.950000", Status: DiscountStatusEnabled,
	}).Insert())
	planB := seedMultiPlan(t, "1.000000")
	require.Error(t, (&DiscountRule{
		PlanId: planB.Id, ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o",
		Discount: "0.900000", Status: DiscountStatusEnabled,
	}).Insert(), "换索引前同名规则应该撞全表唯一约束")

	// 跑启动迁移里的修正：索引换成带 plan_id 的新定义。
	require.NoError(t, fixDiscountRuleUniqueIndex(DB))
	columns, err := discountRuleUniqueIndexColumns(DB)
	require.NoError(t, err)
	assert.Contains(t, columns, "plan_id")

	// 换完后第二个方案的同名规则能写进去，方案内重复仍然被拦。
	require.NoError(t, (&DiscountRule{
		PlanId: planB.Id, ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o",
		Discount: "0.900000", Status: DiscountStatusEnabled,
	}).Insert())
	require.Error(t, (&DiscountRule{
		PlanId: planB.Id, ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o",
		Discount: "0.850000", Status: DiscountStatusEnabled,
	}).Insert(), "同一方案内的重复规则仍然要被拦")

	// 修正过的索引再跑一次应该是幂等的（不重建、不报错）。
	require.NoError(t, fixDiscountRuleUniqueIndex(DB))
}
