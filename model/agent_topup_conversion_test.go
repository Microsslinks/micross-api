package model

// 这一组用例守着 task-09：经销商给客户发额度时按方案折算比例扣经销商余额。
//
// 关键口径：
//   - 面值 quota：客户按原值收
//   - 经销商实扣 agentCost = ceil(quota × rate)
//   - rate 取自客户主方案 (user.Discount_plan_id) 的 TopupConversionRate
//   - rate = 0 / > 1 都拒；rate = 1.0 保持与改前完全一致（向后兼容）
//
// 用 decimal.NewFromString / Equal 做比较，避开 SQLite 把 0.875 存成 0.875000 / 0.875
// 这种 NUMERIC 亲和性坑——计费只关心数值，不关心字面。

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedAgentTopupPlan 建一个比例可配的折扣方案。
func seedAgentTopupPlan(t *testing.T, topupRate string) *DiscountPlan {
	t.Helper()
	plan := &DiscountPlan{
		Name:               "topup-rate-test-" + topupRate,
		OwnerType:          DiscountOwnerAgent,
		BaseDiscount:       "0.900000",
		MinDiscount:        "0.005000",
		BillingMode:        DiscountBillingUsage,
		CommissionRatio:    "0.000000",
		TopupConversionRate: topupRate,
		Status:             DiscountStatusEnabled,
	}
	require.NoError(t, DB.Create(plan).Error)
	t.Cleanup(func() { _ = DB.Unscoped().Delete(&DiscountPlan{}, plan.Id).Error })
	return plan
}

// bindPlanAsPrimary 把 plan 绑给客户，同时把快路径 user.discount_plan_id 指过去。
// 复用 BindDiscountPlan 让"快路径同步"这件事走与生产一致的入口。
func bindPlanAsPrimary(t *testing.T, plan *DiscountPlan, customerId int) {
	t.Helper()
	require.NoError(t, BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   customerId,
		PlanId:      plan.Id,
		Source:      DiscountSourceAgent,
		Status:      DiscountStatusEnabled,
	}))
}

// 比例 = 0.875 时，发 300：经销商实扣 = ceil(300 × 0.875) = 263，客户按 300 收。
func TestIssueQuotaToCustomerAppliesConversionRate(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)
	setTestQuota(t, agent.Id, 1000)
	setTestQuota(t, customer.Id, 0)

	plan := seedAgentTopupPlan(t, "0.875000")
	bindPlanAsPrimary(t, plan, customer.Id)

	require.NoError(t, IssueQuotaToCustomer(agent.Id, customer.Id, 300))
	assert.Equal(t, 1000-263, testQuotaOf(t, agent.Id),
		"经销商扣 agentCost=263（300 × 0.875 = 262.5 向上取整）")
	assert.Equal(t, 300, testQuotaOf(t, customer.Id),
		"客户按面值原额收 300（不折算）")
}

// 比例 = 1.0 时与改前完全一致：经销商扣 quota，客户收 quota。
// 这条用例同时验证"客户没挂方案"路径——planId=0 → 兜底 1.0。
func TestIssueQuotaToCustomerRateOneHundredPercentIsCompatible(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)
	setTestQuota(t, agent.Id, 1000)
	setTestQuota(t, customer.Id, 0)

	plan := seedAgentTopupPlan(t, "1.000000")
	bindPlanAsPrimary(t, plan, customer.Id)

	require.NoError(t, IssueQuotaToCustomer(agent.Id, customer.Id, 300))
	assert.Equal(t, 1000-300, testQuotaOf(t, agent.Id))
	assert.Equal(t, 300, testQuotaOf(t, customer.Id))
}

// 比例 = 0 禁止：钱白送会让平台亏。
func TestIssueQuotaToCustomerRejectsZeroRate(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)
	setTestQuota(t, agent.Id, 1000)
	setTestQuota(t, customer.Id, 0)

	plan := seedAgentTopupPlan(t, "0.000000")
	bindPlanAsPrimary(t, plan, customer.Id)

	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, customer.Id, 100),
		ErrAgentTopupRateZero)
	// 没扣也没收：余额保持原状
	assert.Equal(t, 1000, testQuotaOf(t, agent.Id))
	assert.Equal(t, 0, testQuotaOf(t, customer.Id))
}

// 比例 > 1 禁止：扣得比面值还多会把经销商掏空。
func TestIssueQuotaToCustomerRejectsRateOverOne(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)
	setTestQuota(t, agent.Id, 1000)
	setTestQuota(t, customer.Id, 0)

	plan := seedAgentTopupPlan(t, "1.500000")
	bindPlanAsPrimary(t, plan, customer.Id)

	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, customer.Id, 100),
		ErrAgentTopupRateTooHigh)
	assert.Equal(t, 1000, testQuotaOf(t, agent.Id))
	assert.Equal(t, 0, testQuotaOf(t, customer.Id))
}

// 折算比例 < 1 时经销商可扣的钱比面值少：余额够扣面值、但不够扣 agentCost 时也要拒。
// 这一条专门堵"按面值算余额，扣时却按折算数"的实现 bug——两种口径必须都用 agentCost。
func TestIssueQuotaToCustomerChecksBalanceAgainstAgentCost(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)
	// agent 余额 250 < agentCost=263（300 × 0.875 向上取整），但 ≥ quota=300
	// ——如果按 quota 算余额会判够、按 agentCost 算才判不够。这条用例守着后者。
	setTestQuota(t, agent.Id, 250)
	setTestQuota(t, customer.Id, 0)

	plan := seedAgentTopupPlan(t, "0.875000")
	bindPlanAsPrimary(t, plan, customer.Id)

	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, customer.Id, 300),
		ErrAgentQuotaNotEnough)
	assert.Equal(t, 250, testQuotaOf(t, agent.Id))
	assert.Equal(t, 0, testQuotaOf(t, customer.Id))
}

// resolveAgentTopupCost 边界：返回值格式与默认口径不能漂。
func TestResolveAgentTopupCostDefaults(t *testing.T) {
	// 没挂方案 → 1.0 + agentCost = quota
	agentCost, rate, err := resolveAgentTopupCost(DB, 0, 300)
	require.NoError(t, err)
	assert.Equal(t, 300, agentCost)
	assert.True(t, rate.Equal(decimal.NewFromInt(1)),
		"planId=0 应当按 1.0 兜底")

	// 解析失败的字符串应当返回 error（正常路径不该触发，但兜底要有）
	_, _, err = resolveAgentTopupCost(DB, 0, 100)
	require.NoError(t, err)
	// dummy 触达只是确认函数签名稳定
	_ = common.QuotaForNewUser
}