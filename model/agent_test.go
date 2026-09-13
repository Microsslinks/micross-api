package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupAgentTest 给经销商相关用例一份干净的 SQLite。用的是全局 DB，
// 子测试之间必须隔离，收尾还原（与 discount_resolve_test.go 同一套做法）。
func setupAgentTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:agent-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&User{}, &AgentProfile{}, &CustomerCode{}, &DiscountPlan{}, &DiscountBinding{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// agentTestSeq 保证同一进程内 username / aff_code 不重复。不能用时间戳：
// Windows 上 time.Now() 的分辨率不足以让同一测试里的两次创建取到不同的值，
// 而 users.username 与 users.aff_code 都是唯一索引。
var agentTestSeq int

func uniqueAgentTestSuffix() string {
	agentTestSeq++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano()%1000000, agentTestSeq)
}

func seedAgentTestUser(t *testing.T, subjectType string, parentAgentId int) *User {
	t.Helper()
	suffix := uniqueAgentTestSuffix()
	user := &User{
		Username:      "agent-test-" + suffix,
		Password:      "unused-password-hash",
		Role:          common.RoleCommonUser,
		Status:        common.UserStatusEnabled,
		Group:         "default",
		AffCode:       "aff-" + suffix,
		SubjectType:   subjectType,
		ParentAgentId: parentAgentId,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

// 归属两列必须真的落库：它决定"这个客户算谁的人"，读错等于把别人的客户算到自己头上。
func TestUserOwnershipColumnsPersist(t *testing.T) {
	setupAgentTest(t)
	agent := seedAgentTestUser(t, SubjectTypeAgent, 0)
	customer := seedAgentTestUser(t, SubjectTypeIndividual, agent.Id)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, customer.Id).Error)
	assert.Equal(t, SubjectTypeIndividual, reloaded.SubjectType)
	assert.Equal(t, agent.Id, reloaded.ParentAgentId)

	var reloadedAgent User
	require.NoError(t, DB.First(&reloadedAgent, agent.Id).Error)
	assert.Equal(t, SubjectTypeAgent, reloadedAgent.SubjectType)
	assert.Equal(t, 0, reloadedAgent.ParentAgentId, "经销商自己直属平台")
}

// 没有设置主体类型的存量用户读出来必须是 individual，不能是空串——
// 否则"是不是经销商"的判定会退化成一个说不清的分支。
func TestLegacyUserWithoutSubjectTypeReadsAsIndividual(t *testing.T) {
	setupAgentTest(t)
	user := seedAgentTestUser(t, "", 0)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, user.Id).Error)
	assert.Equal(t, SubjectTypeIndividual, reloaded.SubjectType)
}

func TestGetAgentProfileByUserId(t *testing.T) {
	setupAgentTest(t)
	agent := seedAgentTestUser(t, SubjectTypeAgent, 0)
	require.NoError(t, DB.Create(&AgentProfile{
		UserId:            agent.Id,
		WholesaleDiscount: "0.700000",
		MinDiscount:       "0.750000",
		IssueQuotaEnabled: 1,
	}).Error)

	profile, err := GetAgentProfileByUserId(agent.Id)
	require.NoError(t, err)
	// SQLite 读回 decimal 不带尾零（0.7 而非 0.700000），比较前一律按折扣口径归一。
	wholesale, err := NormalizeDiscount(profile.WholesaleDiscount)
	require.NoError(t, err)
	assert.Equal(t, "0.700000", wholesale)
	floor, err := NormalizeDiscountRatio(profile.MinDiscount)
	require.NoError(t, err)
	assert.Equal(t, "0.750000", floor)
	assert.Equal(t, 1, profile.IssueQuotaEnabled)
	assert.True(t, profile.CreatedAt > 0)

	ordinary := seedAgentTestUser(t, SubjectTypeIndividual, 0)
	_, err = GetAgentProfileByUserId(ordinary.Id)
	assert.ErrorIs(t, err, ErrAgentProfileNotFound)

	_, err = GetAgentProfileByUserId(0)
	assert.ErrorIs(t, err, ErrAgentProfileNotFound)
}

func TestGetCustomerCodeByCode(t *testing.T) {
	setupAgentTest(t)
	agent := seedAgentTestUser(t, SubjectTypeAgent, 0)
	require.NoError(t, DB.Create(&CustomerCode{
		Code:      "VIP8F3K2",
		AgentId:   agent.Id,
		PlanId:    7,
		MaxUses:   10,
		ExpiredAt: 1893456000,
		Status:    CustomerCodeStatusEnabled,
	}).Error)

	found, err := GetCustomerCodeByCode("  VIP8F3K2  ")
	require.NoError(t, err)
	assert.Equal(t, agent.Id, found.AgentId)
	assert.Equal(t, 7, found.PlanId)

	_, err = GetCustomerCodeByCode("NOT-EXIST")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	_, err = GetCustomerCodeByCode("   ")
	assert.Error(t, err)
}

func TestCustomerCodeIsUsable(t *testing.T) {
	const now int64 = 1800000000
	cases := []struct {
		name string
		code CustomerCode
		want bool
	}{
		{"未作废、未过期、不限次数", CustomerCode{Status: CustomerCodeStatusEnabled}, true},
		{"已作废", CustomerCode{Status: CustomerCodeStatusDisabled}, false},
		{"已过期", CustomerCode{Status: CustomerCodeStatusEnabled, ExpiredAt: now - 1}, false},
		{"过期时刻当刻仍可用", CustomerCode{Status: CustomerCodeStatusEnabled, ExpiredAt: now}, true},
		{"次数用满", CustomerCode{Status: CustomerCodeStatusEnabled, MaxUses: 3, UsedCount: 3}, false},
		{"次数未用满", CustomerCode{Status: CustomerCodeStatusEnabled, MaxUses: 3, UsedCount: 2}, true},
		{"不限次数用多少次都可用", CustomerCode{Status: CustomerCodeStatusEnabled, MaxUses: 0, UsedCount: 999}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.code.IsUsable(now))
		})
	}
}

func TestAgentProfileNormalizeDefaults(t *testing.T) {
	empty := &AgentProfile{}
	empty.NormalizeDefaults()
	assert.Equal(t, DiscountNone, empty.WholesaleDiscount)
	assert.Equal(t, "0", empty.MinDiscount)

	kept := &AgentProfile{WholesaleDiscount: "0.700000", MinDiscount: "0.600000"}
	kept.NormalizeDefaults()
	assert.Equal(t, "0.700000", kept.WholesaleDiscount)
	assert.Equal(t, "0.600000", kept.MinDiscount)
}
