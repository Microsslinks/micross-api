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

// 台账要连 logs 一起读，所以这里比 setupAgentTest 多迁两张表（tokens / logs）。
// 同样用的是全局 DB 与 LOG_DB，子测试之间必须隔离，收尾要还原回去。
func setupAgentLedgerTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:agent-ledger-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&User{}, &AgentProfile{}, &Token{}, &Log{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// seedLedgerAgent 建一个经销商，并把这几个总数写进他的账户行——
// 台账里的余额/累计花费/请求数读的就是这三个字段，不是现算的。
func seedLedgerAgent(t *testing.T, markupRatio string, quota int, usedQuota int, requestCount int) *User {
	t.Helper()
	suffix := uniqueAgentTestSuffix()
	user := &User{
		Username: "ledger-" + suffix,
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  "ledger-aff-" + suffix,
	}
	require.NoError(t, DB.Create(user).Error)
	require.NoError(t, PromoteUserToAgentWithMarkup(user.Id, markupRatio, CustomerCodeStatusEnabled, ""))
	require.NoError(t, DB.Model(&User{}).Where("id = ?", user.Id).Updates(map[string]interface{}{
		"quota":         quota,
		"used_quota":    usedQuota,
		"request_count": requestCount,
	}).Error)
	require.NoError(t, DB.First(user, user.Id).Error)
	return user
}

func seedLedgerToken(t *testing.T, userId int, name string, status int) *Token {
	t.Helper()
	token := &Token{
		UserId:      userId,
		Key:         "sk-ledger-" + uniqueAgentTestSuffix(),
		Name:        name,
		Status:      status,
		ExpiredTime: -1,
	}
	require.NoError(t, DB.Create(token).Error)
	return token
}

func seedLedgerLog(t *testing.T, userId int, tokenId int, logType int, quota int, promptTokens int, completionTokens int, createdAt int64) {
	t.Helper()
	require.NoError(t, LOG_DB.Create(&Log{
		UserId:           userId,
		TokenId:          tokenId,
		Type:             logType,
		Quota:            quota,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		CreatedAt:        createdAt,
	}).Error)
}

// TestGetAgentLedgerSumsEachKey 台账按 Key 把消费拆开：
// 每个 Key 的额度、请求数、token 数、最后使用时间各自归并，
// 充值流水与没有 Key 的消费不能混进来（否则「这个 Key 花了多少」就是假的）。
func TestGetAgentLedgerSumsEachKey(t *testing.T) {
	setupAgentLedgerTest(t)
	agent := seedLedgerAgent(t, "1.100000", 5000, 600, 4)

	first := seedLedgerToken(t, agent.Id, "客户甲", common.TokenStatusEnabled)
	second := seedLedgerToken(t, agent.Id, "客户乙", common.TokenStatusDisabled)

	seedLedgerLog(t, agent.Id, first.Id, LogTypeConsume, 100, 10, 5, 1000)
	seedLedgerLog(t, agent.Id, first.Id, LogTypeConsume, 200, 20, 10, 2000)
	seedLedgerLog(t, agent.Id, first.Id, LogTypeConsume, 300, 30, 15, 3000)
	seedLedgerLog(t, agent.Id, second.Id, LogTypeConsume, 400, 40, 20, 4000)
	// 充值是账户流水，不属于任何一个 Key 的消费。
	seedLedgerLog(t, agent.Id, first.Id, LogTypeTopup, 9999, 0, 0, 5000)
	// token_id = 0 的调用不是从 Key 发出来的，挂到任何一个 Key 头上都是错的。
	seedLedgerLog(t, agent.Id, 0, LogTypeConsume, 777, 7, 7, 6000)

	ledger, err := GetAgentLedger(agent.Id)
	require.NoError(t, err)

	assert.Equal(t, 5000, ledger.Quota)
	assert.Equal(t, 600, ledger.UsedQuota)
	assert.Equal(t, 4, ledger.RequestCount)
	require.Len(t, ledger.Keys, 2)

	// 花得多的排前面。
	assert.Equal(t, "客户甲", ledger.Keys[0].Name)
	assert.Equal(t, first.Id, ledger.Keys[0].TokenId)
	assert.EqualValues(t, 600, ledger.Keys[0].UsedQuota)
	assert.EqualValues(t, 3, ledger.Keys[0].RequestCount)
	assert.EqualValues(t, 90, ledger.Keys[0].TotalTokens)
	assert.EqualValues(t, 3000, ledger.Keys[0].LastUsedAt)

	assert.Equal(t, "客户乙", ledger.Keys[1].Name)
	assert.Equal(t, common.TokenStatusDisabled, ledger.Keys[1].Status)
	assert.EqualValues(t, 400, ledger.Keys[1].UsedQuota)
	assert.EqualValues(t, 1, ledger.Keys[1].RequestCount)
	assert.EqualValues(t, 60, ledger.Keys[1].TotalTokens)
	assert.EqualValues(t, 4000, ledger.Keys[1].LastUsedAt)
}

// TestGetAgentLedgerKeepsRemovedKey 删掉的 Key 仍要出现在台账里：
// 客户那个 Key 用废了被删掉，那笔账还在，抹掉会让各 Key 之和对不上他的累计花费。
func TestGetAgentLedgerKeepsRemovedKey(t *testing.T) {
	setupAgentLedgerTest(t)
	agent := seedLedgerAgent(t, "1.100000", 0, 250, 1)
	removed := seedLedgerToken(t, agent.Id, "废掉的 Key", common.TokenStatusEnabled)
	seedLedgerLog(t, agent.Id, removed.Id, LogTypeConsume, 250, 25, 25, 1000)
	require.NoError(t, DB.Delete(&Token{}, removed.Id).Error)

	kept := seedLedgerToken(t, agent.Id, "还在用的 Key", common.TokenStatusEnabled)

	ledger, err := GetAgentLedger(agent.Id)
	require.NoError(t, err)
	require.Len(t, ledger.Keys, 2)

	assert.True(t, ledger.Keys[0].Removed)
	assert.Equal(t, "废掉的 Key", ledger.Keys[0].Name)
	assert.EqualValues(t, 250, ledger.Keys[0].UsedQuota)
	assert.False(t, ledger.Keys[1].Removed)
	assert.Equal(t, kept.Id, ledger.Keys[1].TokenId)
	assert.EqualValues(t, 0, ledger.Keys[1].UsedQuota)
}

// TestGetAgentLedgerMarkupRatio 台账里带出平台给这位经销商定的那档毛利，
// 让他看得见自己的拿货价是按什么算出来的；档案缺了（身份是经销商但没档案行）
// 就按缺省档给，不能因此让他看不到自己的账。
func TestGetAgentLedgerMarkupRatio(t *testing.T) {
	setupAgentLedgerTest(t)
	agent := seedLedgerAgent(t, "1.250000", 0, 0, 0)

	ledger, err := GetAgentLedger(agent.Id)
	require.NoError(t, err)
	normalized, err := NormalizeAgentMarkup(ledger.MarkupRatio)
	require.NoError(t, err)
	assert.Equal(t, "1.250000", normalized)

	// 身份是经销商、档案行却被抹掉：按缺省档继续，不报错。
	require.NoError(t, DB.Where("user_id = ?", agent.Id).Delete(&AgentProfile{}).Error)
	ledger, err = GetAgentLedger(agent.Id)
	require.NoError(t, err)
	normalized, err = NormalizeAgentMarkup(ledger.MarkupRatio)
	require.NoError(t, err)
	// 缺省常量写的是 "1.1"，读回来按折扣口径归一后是六位小数，比较前先归一。
	expectedDefault, err := NormalizeAgentMarkup(AgentWholesaleMarkupDefault)
	require.NoError(t, err)
	assert.Equal(t, expectedDefault, normalized)
	assert.Empty(t, ledger.Keys)
}

// TestGetAgentLedgerRejectsNonAgent 普通客户不该看到台账这张表，
// 哪怕他手里有别的用户的消费记录。
func TestGetAgentLedgerRejectsNonAgent(t *testing.T) {
	setupAgentLedgerTest(t)
	owner := seedLedgerAgent(t, "1.100000", 0, 0, 0)
	token := seedLedgerToken(t, owner.Id, "客户甲", common.TokenStatusEnabled)
	seedLedgerLog(t, owner.Id, token.Id, LogTypeConsume, 100, 10, 5, 1000)

	ordinary := seedAgentTestUser(t, SubjectTypeIndividual, 0)

	_, err := GetAgentLedger(ordinary.Id)
	assert.ErrorIs(t, err, ErrAgentProfileNotFound)

	_, err = GetAgentLedger(0)
	assert.ErrorIs(t, err, ErrAgentProfileNotFound)
}
