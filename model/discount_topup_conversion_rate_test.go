package model

// 这一组用例守着 task-09：discount_plans 在 SQLite 上是"建过就不再 AutoMigrate"的表，
// 给它加 topup_conversion_rate 必须自己在迁移里补 ALTER TABLE。漏了它的后果具体到新代码上：
// 发额度路径报 "no such column: topup_conversion_rate"，Phase 2 的"按比例折算"会直接 500。
//
// 与 agent_profile_migration_test.go 同款模式：搭一份"改造前"的老库 → 跑迁移 → 验证补列成功
// → 二次迁移幂等 → 老行被 DEFAULT 填进。

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupLegacyDiscountPlanTest 给"老库升级"用例一份干净的 SQLite，但不建任何表：
// 表结构由用例自己按改造前那份建，才叫老库。
func setupLegacyDiscountPlanTest(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	dsn := fmt.Sprintf("file:discount-legacy-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		_ = sqlDB.Close()
	})
	return db
}

// createLegacyDiscountPlans 建出改造前的 discount_plans（没有 topup_conversion_rate 列），
// 并放入一行当年写下的方案。
func createLegacyDiscountPlans(t *testing.T) {
	t.Helper()
	require.NoError(t, DB.Exec("CREATE TABLE `discount_plans` ("+
		"`id` integer,"+
		"`name` varchar(64) NOT NULL,"+
		"`owner_type` varchar(16) NOT NULL DEFAULT 'platform',"+
		"`owner_id` integer NOT NULL DEFAULT 0,"+
		"`base_discount` decimal(10,6) NOT NULL,"+
		"`min_discount` decimal(10,6) NOT NULL,"+
		"`billing_mode` varchar(16) NOT NULL DEFAULT 'usage',"+
		"`commission_ratio` decimal(10,6) NOT NULL,"+
		"`status` integer NOT NULL DEFAULT 1,"+
		"`remark` varchar(255) DEFAULT '',"+
		"`created_at` integer,"+
		"`updated_at` integer,"+
		"PRIMARY KEY (`id`))").Error)
	require.NoError(t, DB.Exec("INSERT INTO `discount_plans` "+
		"(`id`,`name`,`owner_type`,`owner_id`,`base_discount`,`min_discount`,`billing_mode`,`commission_ratio`,`status`) "+
		"VALUES (1,'legacy-plan','platform',0,'0.900000','0.005000','usage','0.150000',1)").Error)
}

// hasTopupConversionRate 用 PRAGMA 查 discount_plans 里有没有这一列；列比对走 SQLite 的元数据，
// 不依赖 gorm 的 Migrator.HasColumn（后者在表走 AutoMigrate 时才稳定）。
func hasTopupConversionRate(t *testing.T, db *gorm.DB) bool {
	t.Helper()
	var cols []struct {
		Name string `gorm:"column:name"`
	}
	require.NoError(t, db.Raw("PRAGMA table_info(`discount_plans`)").Scan(&cols).Error)
	for _, c := range cols {
		if c.Name == "topup_conversion_rate" {
			return true
		}
	}
	return false
}

func TestMigrateDiscountTablesAddsTopupConversionRateToExistingTable(t *testing.T) {
	db := setupLegacyDiscountPlanTest(t)
	createLegacyDiscountPlans(t)

	require.False(t, hasTopupConversionRate(t, db),
		"改造前的老库本来就没有 topup_conversion_rate 这一列")

	require.NoError(t, migrateDiscountTables(db))
	assert.True(t, hasTopupConversionRate(t, db),
		"老库升级后必须有 topup_conversion_rate 列")

	// 二次迁移必须幂等：每启动都会跑一次。
	require.NoError(t, migrateDiscountTables(db))

	// 老行被 ALTER ADD COLUMN 的 DEFAULT 1.0 自动填上：
	//   NOT NULL + DEFAULT 是 SQLite 补列的标准模式，不会让"老行没值"失败。
	var plan DiscountPlan
	require.NoError(t, db.First(&plan, 1).Error)
	assertDiscountValue(t, DiscountTopupConversionDefault, plan.TopupConversionRate)
}

// TopupConversionRate 必须在 NormalizeDefaults 里兜底：直接从内存构造 DiscountPlan 时
// 没走 AutoMigrate 默认值，老代码里也出现过"没调 Insert 拿默认值就 Insert"的写法。
func TestNormalizeDefaultsFillsTopupConversionRate(t *testing.T) {
	plan := &DiscountPlan{Name: "normalize-test"}
	plan.NormalizeDefaults()
	assert.Equal(t, DiscountTopupConversionDefault, plan.TopupConversionRate,
		"空字符串必须兜底为 1.0（保持 1:1），否则下游解析会拿空串 panic")
}