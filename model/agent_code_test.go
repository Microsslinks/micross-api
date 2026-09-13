package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 客户号与经销商档案的用例。
//
// 用全局 DB，所以每个栈顶用例给自己一份干净的内存库，收尾还原
// （与 agent_wholesale_test.go 同一套做法）。

func setupAgentCodeTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:agent-code-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&User{}, &AgentProfile{}, &CustomerCode{},
		&DiscountPlan{}, &DiscountRule{}, &DiscountBinding{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// seedCodeTestUser 建一个用户；isAgent 为真时顺手把他设成经销商。
func seedCodeTestUser(t *testing.T, isAgent bool) *User {
	t.Helper()
	suffix := uniqueAgentTestSuffix()
	user := &User{
		Username: "code-" + suffix,
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "code-aff-" + suffix,
	}
	require.NoError(t, DB.Create(user).Error)
	if isAgent {
		require.NoError(t, PromoteUserToAgent(user.Id, DiscountNone, "0", CustomerCodeStatusEnabled, ""))
	}
	return user
}

func seedCodeTestPlan(t *testing.T, status int) *DiscountPlan {
	t.Helper()
	plan := &DiscountPlan{
		Name:            "code-plan-" + uniqueAgentTestSuffix(),
		OwnerType:       DiscountOwnerPlatform,
		BaseDiscount:    "0.900000",
		MinDiscount:     "0",
		BillingMode:     DiscountBillingUsage,
		CommissionRatio: "0",
		Status:          DiscountStatusEnabled,
	}
	require.NoError(t, plan.Insert())
	if status != DiscountStatusEnabled {
		// Status 这一列带 default:1，插入时传 0 会被 GORM 当成零值跳过、落到默认的"启用"上，
		// 所以停用要单独写一次（与方案编辑接口 Update() 的做法一致）。
		require.NoError(t, DB.Model(&DiscountPlan{}).Where("id = ?", plan.Id).
			Update("status", status).Error)
		plan.Status = status
	}
	return plan
}

// 只有经销商能签号：号的全部意义是把客户归到签发人名下。
func TestCreateCustomerCodesRequiresAgent(t *testing.T) {
	setupAgentCodeTest(t)
	customer := seedCodeTestUser(t, false)

	_, err := CreateCustomerCodes(CustomerCodeIssue{AgentId: customer.Id, Count: 1})
	require.ErrorIs(t, err, ErrAgentProfileNotFound)
}

// 号上带了不存在或已停用的方案，必须当场拒绝：否则客户拿到号以为自己有折扣，
// 实际按原价计费，事后没人说得清这张号到底算不算数。
func TestCreateCustomerCodesRejectsUnusablePlan(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	disabled := seedCodeTestPlan(t, DiscountStatusDisabled)

	_, err := CreateCustomerCodes(CustomerCodeIssue{AgentId: agent.Id, PlanId: disabled.Id, Count: 1})
	require.ErrorIs(t, err, ErrCustomerCodePlanUnavailable)

	_, err = CreateCustomerCodes(CustomerCodeIssue{AgentId: agent.Id, PlanId: 999999, Count: 1})
	require.ErrorIs(t, err, ErrCustomerCodePlanUnavailable)
}

// 一批签出来的号要能用：前缀统一、互不重复、状态可用、归属正确，列表能按签署人查回来。
func TestCreateCustomerCodesGeneratesUsableBatch(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)

	codes, err := CreateCustomerCodes(CustomerCodeIssue{
		AgentId:   agent.Id,
		PlanId:    plan.Id,
		Count:     3,
		MaxUses:   2,
		ExpiredAt: common.GetTimestamp() + 86400,
		Remark:    "  给张老板  ",
	})
	require.NoError(t, err)
	require.Len(t, codes, 3)

	seen := make(map[string]bool, len(codes))
	now := common.GetTimestamp()
	for _, code := range codes {
		assert.True(t, strings.HasPrefix(code.Code, CustomerCodePrefix), code.Code)
		assert.False(t, seen[code.Code], "号不能重复：%s", code.Code)
		seen[code.Code] = true
		assert.Equal(t, agent.Id, code.AgentId)
		assert.Equal(t, plan.Id, code.PlanId)
		assert.Equal(t, 2, code.MaxUses)
		assert.Equal(t, "给张老板", code.Remark, "备注要去掉首尾空白")
		assert.Equal(t, CustomerCodeStatusEnabled, code.Status)
		assert.True(t, code.IsUsable(now))
	}

	listed, total, err := ListCustomerCodes(agent.Id, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, int64(3), total)
	require.Len(t, listed, 3)
	// 最新签的排在最前面：经销商打开列表先看到刚发出去的那一批。
	assert.Equal(t, codes[2].Id, listed[0].Id)
}

// 作废只能动作自己的人：别人的号连存在都不该被确认。
func TestRevokeCustomerCodeOnlyOwnCodes(t *testing.T) {
	setupAgentCodeTest(t)
	agentA := seedCodeTestUser(t, true)
	agentB := seedCodeTestUser(t, true)

	codes, err := CreateCustomerCodes(CustomerCodeIssue{AgentId: agentA.Id, Count: 1})
	require.NoError(t, err)
	codeId := codes[0].Id

	require.ErrorIs(t, RevokeCustomerCode(agentB.Id, codeId), ErrCustomerCodeNotFound)

	require.NoError(t, RevokeCustomerCode(agentA.Id, codeId))
	var reloaded CustomerCode
	require.NoError(t, DB.First(&reloaded, codeId).Error)
	assert.Equal(t, CustomerCodeStatusDisabled, reloaded.Status)
	assert.False(t, reloaded.IsUsable(common.GetTimestamp()))
}

// 入参校验：数量上下限、有效期不能是过去时、备注长度、使用次数上限。
func TestNormalizeCustomerCodeIssue(t *testing.T) {
	future := common.GetTimestamp() + 3600
	cases := []struct {
		name  string
		issue CustomerCodeIssue
		ok    bool
	}{
		{"正常", CustomerCodeIssue{AgentId: 1, Count: 1, MaxUses: 1, ExpiredAt: future}, true},
		{"不限次数不限有效期", CustomerCodeIssue{AgentId: 1, Count: 50}, true},
		{"没给经销商", CustomerCodeIssue{AgentId: 0, Count: 1}, false},
		{"数量为 0", CustomerCodeIssue{AgentId: 1, Count: 0}, false},
		{"超过一次上限", CustomerCodeIssue{AgentId: 1, Count: CustomerCodeMaxBatch + 1}, false},
		{"使用次数为负", CustomerCodeIssue{AgentId: 1, Count: 1, MaxUses: -1}, false},
		{"使用次数过大", CustomerCodeIssue{AgentId: 1, Count: 1, MaxUses: CustomerCodeMaxUsesCap + 1}, false},
		{"有效期是过去时", CustomerCodeIssue{AgentId: 1, Count: 1, ExpiredAt: common.GetTimestamp() - 1}, false},
		{"备注过长", CustomerCodeIssue{AgentId: 1, Count: 1, Remark: strings.Repeat("字", 256)}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NormalizeCustomerCodeIssue(tc.issue)
			if tc.ok {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}

// 改档案是「传哪个改哪个」：只改加价率时，最低折扣、发额度开关、备注都得原样留着。
func TestUpdateAgentProfileSettingsKeepsUntouchedFields(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	require.NoError(t, DB.Model(&AgentProfile{}).Where("user_id = ?", agent.Id).
		Updates(map[string]interface{}{
			"markup_ratio":        "1.200000",
			"min_discount":        "0.500000",
			"issue_quota_enabled": CustomerCodeStatusEnabled,
			"remark":              "老备注",
		}).Error)

	markup := "1.500000"
	profile, err := UpdateAgentProfileSettings(agent.Id, AgentProfileSettings{MarkupRatio: &markup})
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("1.5").Equal(decimal.RequireFromString(profile.MarkupRatio)))
	// 没传的字段不能被清掉（decimal 列在 SQLite 上会归一化，所以按数值比）
	assert.True(t, decimal.RequireFromString("0.5").Equal(decimal.RequireFromString(profile.MinDiscount)))
	assert.Equal(t, CustomerCodeStatusEnabled, profile.IssueQuotaEnabled)
	assert.Equal(t, "老备注", profile.Remark)
}

// 停发额度取 0 时必须真的落库：这一列是 int，GORM 默认会跳过零值。
func TestUpdateAgentProfileSettingsPersistsZeroValue(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)

	disabled := CustomerCodeStatusDisabled
	_, err := UpdateAgentProfileSettings(agent.Id, AgentProfileSettings{IssueQuotaEnabled: &disabled})
	require.NoError(t, err)

	var reloaded AgentProfile
	require.NoError(t, DB.Where("user_id = ?", agent.Id).First(&reloaded).Error)
	assert.Equal(t, CustomerCodeStatusDisabled, reloaded.IssueQuotaEnabled)
}

// 最低折扣的下限口径：0 = 不限，其余必须落在 (0, 1]。
func TestNormalizeAgentMinDiscount(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		ok   bool
	}{
		{"0", "0.000000", true},
		{"0.5", "0.500000", true},
		{"1", "1.000000", true},
		{"1.2", "", false},
		{"-0.1", "", false},
		{"abc", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := NormalizeAgentMinDiscount(tc.raw)
			if !tc.ok {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// 不是经销商时改档案要返回"这个人不是经销商"，而不是悄悄建一行档案出来。
func TestUpdateAgentProfileSettingsRequiresProfile(t *testing.T) {
	setupAgentCodeTest(t)
	customer := seedCodeTestUser(t, false)

	markup := "1.100000"
	_, err := UpdateAgentProfileSettings(customer.Id, AgentProfileSettings{MarkupRatio: &markup})
	require.True(t, errors.Is(err, ErrAgentProfileNotFound))
}
