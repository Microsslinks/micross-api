package model

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 「我的客户」的用例。库与夹具复用 agent_code_test.go 里的
// setupAgentCodeTest / seedCodeTestUser / seedCodeTestPlan / seedCodeTestCode。

// assignOwner 把一位客户挂到某位经销商名下（绑号时落的那个归属，这里手工造）。
func assignOwner(t *testing.T, customerId int, agentId int) {
	t.Helper()
	require.NoError(t, DB.Model(&User{}).Where("id = ?", customerId).
		UpdateColumn("parent_agent_id", agentId).Error)
}

func setTestQuota(t *testing.T, userId int, quota int) {
	t.Helper()
	require.NoError(t, DB.Model(&User{}).Where("id = ?", userId).
		UpdateColumn("quota", quota).Error)
}

func testQuotaOf(t *testing.T, userId int) int {
	t.Helper()
	var user User
	require.NoError(t, DB.Select("id", "quota").First(&user, userId).Error)
	return user.Quota
}

// seedTestPlanWithDiscount 签一个基础折扣指定的方案。
// 在任何人读过折扣之前就改好，避免走到解析缓存上去。
func seedTestPlanWithDiscount(t *testing.T, discount string) *DiscountPlan {
	t.Helper()
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)
	require.NoError(t, DB.Model(&DiscountPlan{}).Where("id = ?", plan.Id).
		Update("base_discount", discount).Error)
	plan.BaseDiscount = discount
	return plan
}

// 名单只认归属：A 名下的人归 A，B 的人不该出现在 A 的名单里，
// 而且每个人此刻生效的价与余额都要摊开给 A 看。
func TestListAgentCustomersShowsOnlyOwnCustomers(t *testing.T) {
	setupAgentCodeTest(t)
	agentA := seedCodeTestUser(t, true)
	agentB := seedCodeTestUser(t, true)
	priced := seedCodeTestUser(t, false)
	unpriced := seedCodeTestUser(t, false)
	outsider := seedCodeTestUser(t, false)
	plan := seedTestPlanWithDiscount(t, "0.800000")

	assignOwner(t, priced.Id, agentA.Id)
	assignOwner(t, unpriced.Id, agentA.Id)
	assignOwner(t, outsider.Id, agentB.Id)
	setTestQuota(t, priced.Id, 500)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", priced.Id).
		UpdateColumn("used_quota", 120).Error)
	require.NoError(t, SetAgentCustomerDiscount(agentA.Id, priced.Id, plan.Id))

	rows, total, err := ListAgentCustomers(agentA.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	require.Len(t, rows, 2)

	// 最新挂进来的排前面，经销商打开先看到新客户
	assert.Equal(t, unpriced.Id, rows[0].UserId)
	assert.Equal(t, priced.Id, rows[1].UserId)

	// 定过价的那位：价、来源、余额都摆在明面上
	assert.Equal(t, plan.Id, rows[1].PlanId)
	assert.Equal(t, plan.Name, rows[1].PlanName)
	assert.Equal(t, "0.800000", rows[1].PlanDiscount)
	assert.Equal(t, DiscountSourceAgent, rows[1].BindingSource)
	assert.True(t, rows[1].PricedByMe)
	assert.Equal(t, 500, rows[1].Quota)
	assert.Equal(t, 120, rows[1].UsedQuota)

	// 没定过价的那位：没有生效方案，界面按官方标价显示
	assert.Equal(t, 0, rows[0].PlanId)
	assert.Empty(t, rows[0].BindingSource)
	assert.False(t, rows[0].PricedByMe)

	// 单行读法（改价/发额度之后回给界面的那一行）口径一致，且不认别人名下的客户
	single, err := GetAgentCustomer(agentA.Id, priced.Id)
	require.NoError(t, err)
	assert.Equal(t, rows[1].PlanId, single.PlanId)
	_, err = GetAgentCustomer(agentA.Id, outsider.Id)
	require.ErrorIs(t, err, ErrAgentCustomerNotFound)
}

// 改价要真的改到计费上，撤价要真的回到客户号那份价：
// 后者是客户号的意义所在——经销商当初答应客户的价，不能因为他后来改过价就没了。
func TestSetAgentCustomerDiscountAppliesAndRevertsToCodePrice(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	codePlan := seedTestPlanWithDiscount(t, "0.900000")
	dealerPlan := seedTestPlanWithDiscount(t, "0.800000")
	assignOwner(t, customer.Id, agent.Id)

	code := seedCodeTestCode(t, agent.Id, codePlan.Id)
	_, err := BindCustomerCode(customer.Id, code.Code)
	require.NoError(t, err)
	assertDiscount(t, customer.Id, "0.9")

	// 经销商定下自己的价
	require.NoError(t, SetAgentCustomerDiscount(agent.Id, customer.Id, dealerPlan.Id))
	assertDiscount(t, customer.Id, "0.8")
	var reloaded User
	require.NoError(t, DB.First(&reloaded, customer.Id).Error)
	assert.Equal(t, dealerPlan.Id, reloaded.DiscountPlanId, "快路径要跟着改")

	// 客户号那条绑定原样留着，只是此刻不生效；经销商定价是新插的一条
	var enabled []DiscountBinding
	require.NoError(t, DB.Where("subject_id = ? AND status = ?", customer.Id, DiscountStatusEnabled).
		Order("id").Find(&enabled).Error)
	require.Len(t, enabled, 2)
	assert.Equal(t, DiscountSourceCustomerCode, enabled[0].Source)
	assert.Equal(t, DiscountSourceAgent, enabled[1].Source)

	row, err := GetAgentCustomer(agent.Id, customer.Id)
	require.NoError(t, err)
	assert.True(t, row.PricedByMe)
	assert.Equal(t, dealerPlan.Name, row.PlanName)

	// 撤掉自己的价：回到客户号给的 0.9，而不是掉到官方标价
	require.NoError(t, SetAgentCustomerDiscount(agent.Id, customer.Id, 0))
	assertDiscount(t, customer.Id, "0.9")
	require.NoError(t, DB.First(&reloaded, customer.Id).Error)
	assert.Equal(t, codePlan.Id, reloaded.DiscountPlanId)

	row, err = GetAgentCustomer(agent.Id, customer.Id)
	require.NoError(t, err)
	assert.False(t, row.PricedByMe)
	assert.Equal(t, DiscountSourceCustomerCode, row.BindingSource)
	assert.Equal(t, codePlan.Id, row.PlanId)
}

// assertDiscount 走一遍真正的计价口径：经销商改的价必须落到客户实际计费上。
func assertDiscount(t *testing.T, userId int, want string) {
	t.Helper()
	resolution, err := ResolveUserDiscount(userId, "claude-3-5-sonnet")
	require.NoError(t, err)
	assert.True(t,
		decimal.RequireFromString(want).Equal(decimal.RequireFromString(resolution.Discount)),
		"期望 %s，实际 %s", want, resolution.Discount)
}

// 改价的四条线：只能改自己名下的人、平台定的价不能碰、只能卖货架上的方案。
func TestSetAgentCustomerDiscountRejectsOutOfBounds(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	agentB := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	outsider := seedCodeTestUser(t, false)
	disabledPlan := seedCodeTestPlan(t, DiscountStatusDisabled)
	sellable := seedTestPlanWithDiscount(t, "0.800000")
	assignOwner(t, customer.Id, agent.Id)
	assignOwner(t, outsider.Id, agentB.Id)

	// 别人名下的客户：一律按"不在你名下"回报，不确认他是否存在
	require.ErrorIs(t, SetAgentCustomerDiscount(agent.Id, outsider.Id, sellable.Id), ErrAgentCustomerNotFound)
	require.ErrorIs(t, SetAgentCustomerDiscount(agent.Id, 999999, sellable.Id), ErrAgentCustomerNotFound)

	// 自己不是经销商
	require.ErrorIs(t, SetAgentCustomerDiscount(outsider.Id, outsider.Id, sellable.Id), ErrAgentProfileNotFound)

	// 停用的方案不在货架上
	require.ErrorIs(t, SetAgentCustomerDiscount(agent.Id, customer.Id, disabledPlan.Id), ErrAgentPlanNotSellable)
	require.ErrorIs(t, SetAgentCustomerDiscount(agent.Id, customer.Id, 999999), ErrAgentPlanNotSellable)

	// 低于自己最低售价折扣的方案也不在货架上：那个价卖出去是亏的
	floor := "0.950000"
	_, err := UpdateAgentProfileSettings(agent.Id, AgentProfileSettings{MinDiscount: &floor})
	require.NoError(t, err)
	require.ErrorIs(t, SetAgentCustomerDiscount(agent.Id, customer.Id, sellable.Id), ErrAgentPlanNotSellable)

	// 从头到尾没落下任何绑定
	var bindingCount int64
	require.NoError(t, DB.Model(&DiscountBinding{}).Where("subject_id = ?", customer.Id).
		Count(&bindingCount).Error)
	assert.Equal(t, int64(0), bindingCount)

	// 平台单独定过价的客户，经销商改不动——那是平台的决定
	manualPlan := seedCodeTestPlan(t, DiscountStatusEnabled)
	require.NoError(t, BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   customer.Id,
		PlanId:      manualPlan.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	}))
	require.ErrorIs(t, SetAgentCustomerDiscount(agent.Id, customer.Id, sellable.Id), ErrCustomerPricedByPlatform)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, customer.Id).Error)
	assert.Equal(t, manualPlan.Id, reloaded.DiscountPlanId)
}

// 发额度是自己余额的转移：一边减一边加，同一笔账。
func TestIssueQuotaToCustomerMovesBalance(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	outsider := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)
	setTestQuota(t, agent.Id, 1000)
	setTestQuota(t, customer.Id, 10)

	require.NoError(t, IssueQuotaToCustomer(agent.Id, customer.Id, 300))
	assert.Equal(t, 700, testQuotaOf(t, agent.Id))
	assert.Equal(t, 310, testQuotaOf(t, customer.Id))

	// 额度必须是正数
	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, customer.Id, 0), ErrAgentQuotaInvalid)
	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, customer.Id, -5), ErrAgentQuotaInvalid)

	// 自己不够就一分都不动
	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, customer.Id, 100000), ErrAgentQuotaNotEnough)
	assert.Equal(t, 700, testQuotaOf(t, agent.Id))
	assert.Equal(t, 310, testQuotaOf(t, customer.Id))

	// 名下没有这个人
	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, outsider.Id, 1), ErrAgentCustomerNotFound)
	assert.Equal(t, 700, testQuotaOf(t, agent.Id))
	assert.Equal(t, 0, testQuotaOf(t, outsider.Id))
}

// 平台可以关掉"给下属发额度"这个开关；不是经销商的人没有这个开关，也没有名下的客户。
func TestIssueQuotaToCustomerRespectsSwitch(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)
	setTestQuota(t, agent.Id, 1000)

	disabled := CustomerCodeStatusDisabled
	_, err := UpdateAgentProfileSettings(agent.Id, AgentProfileSettings{IssueQuotaEnabled: &disabled})
	require.NoError(t, err)

	require.ErrorIs(t, IssueQuotaToCustomer(agent.Id, customer.Id, 10), ErrAgentQuotaIssuingDisabled)
	assert.Equal(t, 1000, testQuotaOf(t, agent.Id))
	assert.Equal(t, 0, testQuotaOf(t, customer.Id))

	plainCustomer := seedCodeTestUser(t, false)
	require.ErrorIs(t, IssueQuotaToCustomer(plainCustomer.Id, customer.Id, 10), ErrAgentProfileNotFound)
}

// 被设成经销商的人要从名单上消失：他是同行的对手，不再是"我的客户"。
func TestListAgentCustomersDropsPromotedCustomers(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	assignOwner(t, customer.Id, agent.Id)

	rows, total, err := ListAgentCustomers(agent.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	require.Len(t, rows, 1)

	require.NoError(t, PromoteUserToAgentWithMarkup(customer.Id, "1.200000", CustomerCodeStatusEnabled, ""))
	rows, total, err = ListAgentCustomers(agent.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, rows)
}
