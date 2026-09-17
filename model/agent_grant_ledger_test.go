package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIssueQuotaToCustomerWritesAccountLedger 守住 task-20 §20.3：
// 经销商 grant 客户额度时，必须同时写两行 ledger：
//   - 经销商（agent）：amount = -agentCost（出账，余额减少）
//   - 客户（customer）：amount = +quota（入账，余额增加）
//
// amount 符号约定：正数=入账，负数=出账；refType=agent_quota_grant；operator_id=经销商 id
// （区别于系统自动的 topup/refund）。
func TestIssueQuotaToCustomerWritesAccountLedger(t *testing.T) {
	setupAgentCodeTest(t)

	// 1. 建经销商：profile.IssueQuotaEnabled 必须是 enabled，否则 IssueQuotaToCustomer 拒绝
	agentID := 8001
	customerID := 8002
	agent := &User{
		Id:            agentID,
		Username:      "grant-agent",
		AffCode:       "grant-agent-aff",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		SubjectType:   SubjectTypeAgent,
		Group:         "default",
		Quota:         100_000,
	}
	customer := &User{
		Id:              customerID,
		Username:        "grant-customer",
		AffCode:         "grant-customer-aff",
		Role:            common.RoleCommonUser,
		Status:          common.UserStatusEnabled,
		SubjectType:     "individual",
		ParentAgentId:   agentID,
		Group:           "default",
		Quota:           0,
		DiscountPlanId:  0,
	}
	require.NoError(t, DB.Create(agent).Error)
	require.NoError(t, DB.Create(customer).Error)
	require.NoError(t, DB.Create(&AgentProfile{
		UserId:             agentID,
		IssueQuotaEnabled:  CustomerCodeStatusEnabled,
	}).Error)

	// 2. 触发 grant：face=10_000，rate=1.0 → agentCost=10_000
	const faceQuota = 10_000
	require.NoError(t, IssueQuotaToCustomer(agentID, customerID, faceQuota),
		"rate=1.0 应该让 grant 成功")

	// 3. ledger 必须有两行：agent 出账 -10_000 + customer 入账 +10_000
	var ledgers []AccountLedger
	require.NoError(t, DB.Where("event_type = ?", AccountEventAgentQuotaGrant).
		Order("id ASC").Find(&ledgers).Error)
	require.Len(t, ledgers, 2, "grant 必须写两行 ledger（agent 出 + customer 入）")

	agentLedger := ledgers[0]
	customerLedger := ledgers[1]

	// 4. agent ledger：amount=-agentCost，subject_id=agent
	assert.Equal(t, "user", agentLedger.SubjectType)
	assert.Equal(t, agentID, agentLedger.SubjectId)
	assert.Equal(t, int64(-faceQuota), agentLedger.Amount)
	assert.Equal(t, "agent_quota_grant", agentLedger.RefType)
	assert.Equal(t, agentID, agentLedger.OperatorId, "operator_id 应是经销商自己")
	// 重读：rate=1.0 → agentCost=10_000；agent initial=100_000 → after=90_000
	assert.Equal(t, int64(100_000-faceQuota), agentLedger.BalanceAfter,
		"agent ledger.balance_after = 100_000 - 10_000 = 90_000")

	// 5. customer ledger：amount=+quota，subject_id=customer
	assert.Equal(t, "user", customerLedger.SubjectType)
	assert.Equal(t, customerID, customerLedger.SubjectId)
	assert.Equal(t, int64(faceQuota), customerLedger.Amount)
	assert.Equal(t, "agent_quota_grant", customerLedger.RefType)
	assert.Equal(t, agentID, customerLedger.OperatorId)
	assert.Equal(t, int64(faceQuota), customerLedger.BalanceAfter)

	// 6. 实际余额也对账
	var reloadedAgent, reloadedCustomer User
	require.NoError(t, DB.First(&reloadedAgent, agentID).Error)
	require.NoError(t, DB.First(&reloadedCustomer, customerID).Error)
	assert.Equal(t, 100_000-faceQuota, reloadedAgent.Quota)
	assert.Equal(t, faceQuota, reloadedCustomer.Quota)
	assert.Equal(t, int64(reloadedAgent.Quota), agentLedger.BalanceAfter)
	assert.Equal(t, int64(reloadedCustomer.Quota), customerLedger.BalanceAfter)
}

// TestIssueQuotaToCustomerRollsBackLedger 守住：grant 事务失败时 ledger 必须回滚，
// 不可能"钱扣了但 ledger 没记"或反之。
//
// 触发：客户被另一个经销商抢占绑定（ParentAgentId != agentId），IssueQuotaToCustomer
// 返回 ErrAgentCustomerNotFound，事务回滚。
func TestIssueQuotaToCustomerRollsBackLedger(t *testing.T) {
	setupAgentCodeTest(t)

	agentID := 8101
	customerID := 8102
	agent := &User{
		Id:            agentID,
		Username:      "rollback-agent",
		AffCode:       "rollback-agent-aff",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		SubjectType:   SubjectTypeAgent,
		Quota:         50_000,
	}
	customer := &User{
		Id:            customerID,
		Username:      "rollback-customer",
		AffCode:       "rollback-customer-aff",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		SubjectType:   "individual",
		ParentAgentId: 9999, // 假装绑了别的经销商
		Quota:         0,
	}
	require.NoError(t, DB.Create(agent).Error)
	require.NoError(t, DB.Create(customer).Error)
	require.NoError(t, DB.Create(&AgentProfile{
		UserId:            agentID,
		IssueQuotaEnabled: CustomerCodeStatusEnabled,
	}).Error)

	err := IssueQuotaToCustomer(agentID, customerID, 5000)
	require.Error(t, err, "ParentAgentId 不匹配必须失败")

	// ledger 必须 0 行（事务整体回滚）
	var count int64
	require.NoError(t, DB.Model(&AccountLedger{}).
		Where("event_type = ?", AccountEventAgentQuotaGrant).
		Count(&count).Error)
	assert.Equal(t, int64(0), count, "grant 失败事务必须回滚 ledger")

	// 余额也不动
	var reloadAgent User
	require.NoError(t, DB.First(&reloadAgent, agentID).Error)
	assert.Equal(t, 50_000, reloadAgent.Quota)
}