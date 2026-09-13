package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 设为经销商只改身份与档案，**不能碰 role**：经销商是业务身份，不是权限等级，
// 设完他仍然是普通用户（见 .docs/task-05-roles/README.md §二）。
func TestPromoteUserToAgentKeepsRoleAndWritesProfile(t *testing.T) {
	setupAgentTest(t)
	user := seedAgentTestUser(t, SubjectTypeIndividual, 0)
	require.Equal(t, common.RoleCommonUser, user.Role)

	require.NoError(t, PromoteUserToAgent(user.Id, "0.700000", "0.750000", 1, "第一批经销商"))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, SubjectTypeAgent, reloaded.SubjectType)
	assert.Equal(t, common.RoleCommonUser, reloaded.Role, "设成经销商不该动角色")

	profile, err := GetAgentProfileByUserId(user.Id)
	require.NoError(t, err)
	// SQLite 读回 decimal 不带尾零（0.7 而非 0.700000），比较前一律按折扣口径归一。
	wholesale, err := NormalizeDiscount(profile.WholesaleDiscount)
	require.NoError(t, err)
	assert.Equal(t, "0.700000", wholesale)
	floor, err := NormalizeDiscountRatio(profile.MinDiscount)
	require.NoError(t, err)
	assert.Equal(t, "0.750000", floor)
	assert.Equal(t, 1, profile.IssueQuotaEnabled)
	assert.Equal(t, "第一批经销商", profile.Remark)
}

// 已经是经销商时再次设置，按新值覆盖而不是建出第二份档案——平台要能调整批发折扣。
func TestPromoteUserToAgentUpdatesExistingProfile(t *testing.T) {
	setupAgentTest(t)
	user := seedAgentTestUser(t, SubjectTypeAgent, 0)
	require.NoError(t, DB.Create(&AgentProfile{
		UserId:            user.Id,
		WholesaleDiscount: "0.900000",
		MinDiscount:       "0.950000",
		IssueQuotaEnabled: 1,
	}).Error)

	require.NoError(t, PromoteUserToAgent(user.Id, "0.600000", "0.650000", 0, ""))

	var count int64
	require.NoError(t, DB.Model(&AgentProfile{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count, "重复设置不应建出第二份档案")

	profile, err := GetAgentProfileByUserId(user.Id)
	require.NoError(t, err)
	wholesale, err := NormalizeDiscount(profile.WholesaleDiscount)
	require.NoError(t, err)
	assert.Equal(t, "0.600000", wholesale)
	floor, err := NormalizeDiscountRatio(profile.MinDiscount)
	require.NoError(t, err)
	assert.Equal(t, "0.650000", floor)
	assert.Equal(t, 0, profile.IssueQuotaEnabled)
}

// 取消经销商要回到普通客户，并把档案清掉。
func TestDemoteAgentToUserRestoresIndividual(t *testing.T) {
	setupAgentTest(t)
	user := seedAgentTestUser(t, SubjectTypeAgent, 0)
	require.NoError(t, DB.Create(&AgentProfile{
		UserId:            user.Id,
		WholesaleDiscount: "0.700000",
		MinDiscount:       "0.750000",
	}).Error)

	require.NoError(t, DemoteAgentToUser(user.Id))

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, SubjectTypeIndividual, reloaded.SubjectType)

	var count int64
	require.NoError(t, DB.Model(&AgentProfile{}).Where("user_id = ?", user.Id).Count(&count).Error)
	assert.Equal(t, int64(0), count, "取消后不该留下档案")
}

// 名下还挂着客户号时不许取消，否则那些客户号会指向一个不再是经销商的人。
// 拒绝必须是"整笔回滚"，不能出现身份没改档案却删了这类半截状态。
func TestDemoteAgentToUserRefusesWhenCustomerCodesExist(t *testing.T) {
	setupAgentTest(t)
	agent := seedAgentTestUser(t, SubjectTypeAgent, 0)
	require.NoError(t, DB.Create(&AgentProfile{
		UserId:            agent.Id,
		WholesaleDiscount: "0.700000",
		MinDiscount:       "0.750000",
	}).Error)
	require.NoError(t, DB.Create(&CustomerCode{
		Code:    "KEEP-ME-01",
		AgentId: agent.Id,
		Status:  CustomerCodeStatusEnabled,
	}).Error)

	err := DemoteAgentToUser(agent.Id)
	require.ErrorIs(t, err, ErrAgentHasCustomerCodes)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, agent.Id).Error)
	assert.Equal(t, SubjectTypeAgent, reloaded.SubjectType, "拒绝时身份必须原样保留")
	_, err = GetAgentProfileByUserId(agent.Id)
	assert.NoError(t, err, "拒绝时档案也必须原样保留")
}
