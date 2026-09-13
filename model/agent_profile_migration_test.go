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

// 这一组用例守着一次真实故障：agent_profiles 在 SQLite 上是「建过就不再 AutoMigrate」的表
// （它带 decimal 列，走 AutoMigrate 会把整张表每启动重建一遍），所以给它加 markup_ratio
// 必须自己在迁移里补 ALTER TABLE。漏了这步，老库就缺这一列，而新库看不出问题——
// 报错是读档案时的 "no such column: markup_ratio"，用户能看到的是：
// 「设为经销商」保存失败、经销商拿货价解析不到（按方案折扣兜底）、经销商台账打不开。

// setupLegacyAgentProfileTest 给「老库升级」用例一份干净的 SQLite，但不建任何表：
// 表结构由用例自己按改造前那份建，才叫老库。
func setupLegacyAgentProfileTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:agent-legacy-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// createLegacyAgentProfiles 建出改造前的 agent_profiles（没有 markup_ratio 列），
// 并放入一行当年写下的经营档案。
func createLegacyAgentProfiles(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("CREATE TABLE `agent_profiles` ("+
		"`id` integer,"+
		"`user_id` integer NOT NULL,"+
		"`wholesale_discount` decimal(10,6) NOT NULL,"+
		"`min_discount` decimal(10,6) NOT NULL,"+
		"`issue_quota_enabled` integer NOT NULL DEFAULT 1,"+
		"`remark` varchar(255) DEFAULT '',"+
		"`created_at` integer,"+
		"`updated_at` integer,"+
		"PRIMARY KEY (`id`))").Error)
	require.NoError(t, DB.Exec("INSERT INTO `agent_profiles` "+
		"(`id`,`user_id`,`wholesale_discount`,`min_discount`,`issue_quota_enabled`) "+
		"VALUES (1,7,'0.7','0.6',1)").Error)
}

func TestMigrateAgentTablesAddsMarkupRatioToExistingTable(t *testing.T) {
	setupLegacyAgentProfileTest(t)
	createLegacyAgentProfiles(t)

	hasMarkupRatio := func() bool {
		var cols []struct {
			Name string `gorm:"column:name"`
		}
		require.NoError(t, DB.Raw("PRAGMA table_info(`agent_profiles`)").Scan(&cols).Error)
		for _, c := range cols {
			if c.Name == "markup_ratio" {
				return true
			}
		}
		return false
	}

	require.False(t, hasMarkupRatio(), "改造前的老库本来就没有这一列")

	require.NoError(t, migrateAgentTables(DB))
	assert.True(t, hasMarkupRatio(), "老库升级后必须有平台加价率这一列")

	// 再跑一次不能出错：迁移每次启动都会跑。
	require.NoError(t, migrateAgentTables(DB))

	// 补上列之后，当年那行档案要读得出来；行里没存过加价率，按缺省档 +10% 处理。
	profile, err := GetAgentProfileByUserId(7)
	require.NoError(t, err)
	assert.Equal(t, AgentWholesaleMarkupDefault, profile.EffectiveMarkupRatio())
	assert.Equal(t, 1, profile.IssueQuotaEnabled)
}
