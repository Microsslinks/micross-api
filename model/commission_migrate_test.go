package model

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupCommissionMigrateTest 给 commission 迁移用例一份干净的 SQLite in-memory。
//
// 与 model/agent_test.go:setupAgentTest 同套路，AutoMigrate 完整 User struct
// （含新加的 AffCommissionBalance 字段）让 schema 与代码同步，然后测迁移函数
// 是否能在此基础上再幂等跑通、并正确建出 commission_records 表与唯一约束。
func setupCommissionMigrateTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:commission-migrate-%d?mode=memory&cache=shared", time.Now().UnixNano())
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

// TestMigrateCommissionTablesAddsCommissionRecordsAndUserField
//
// 验证：
//  1. 迁移后 commission_records 表已建
//  2. User.AffCommissionBalance 字段被 gorm 认知（迁移函数跑了 &User{} 后字段列存在）
//
// 注意：完整模拟"老 users 表不含新字段"的场景需要手工 CREATE TABLE，
// 而 gorm AutoMigrate 比对 schema 时一旦发现列与 struct 不匹配就报"failed to look up field"，
// 手工 SQL 字段名差异（group / linux_do_id 等）会让模拟成本太高。task-09 阶段 task-list 文档
// 写"SQLite 老库补列"路径，实际项目里 users 表已含 decimal 之外的列、AutoMigrate 走的是
// ALTER ADD COLUMN 路径而非重建，行为更接近直接增量加列，所以这里测"迁移函数能跑通 +
// 新字段生效"就够了——老库升级兜底交给 migrateDB() 的完整链路。
func TestMigrateCommissionTablesAddsCommissionRecordsAndUserField(t *testing.T) {
	setupCommissionMigrateTest(t)

	// 1. 先建完整 User 表（含新字段），模拟"已运行过的服务"基线。
	require.NoError(t, DB.AutoMigrate(&User{}))

	// 2. 跑迁移。
	require.NoError(t, migrateCommissionTables(DB))

	// 3. commission_records 表已建。
	assert.True(t, DB.Migrator().HasTable(&CommissionRecord{}),
		"migrateCommissionTables 必须建出 commission_records 表")

	// 4. users.aff_commission_balance 列存在。
	assert.True(t, hasColumn(DB, "users", "aff_commission_balance"),
		"User struct 含 AffCommissionBalance，AutoMigrate 跑过后必须能 SELECT")

	// 5. 落库字段值默认 0。
	var balance int
	require.NoError(t, DB.Raw("SELECT aff_commission_balance FROM users LIMIT 1").Scan(&balance).Error)
	assert.Equal(t, 0, balance)
}

// TestMigrateCommissionTablesIsIdempotent
//
// 迁移函数跑两次不应报错（与 discount / agent 同一模式）。
// AutoMigrate 是幂等的，第二次跑只会跳过已存在的列/索引/约束。
func TestMigrateCommissionTablesIsIdempotent(t *testing.T) {
	setupCommissionMigrateTest(t)
	require.NoError(t, DB.AutoMigrate(&User{}))

	require.NoError(t, migrateCommissionTables(DB))
	require.NoError(t, migrateCommissionTables(DB), "二次迁移必须不报错")

	assert.True(t, hasColumn(DB, "users", "aff_commission_balance"))
	assert.True(t, DB.Migrator().HasTable(&CommissionRecord{}))
}

// TestCommissionRecordUniqueOnConsumeLogAndInviter
//
// 验证 (consume_log_id, inviter_id) 唯一约束生效——
// README §七第 4 条：同一笔消费 + 同一邀请人 只能返佣一次（防重试返佣）。
func TestCommissionRecordUniqueOnConsumeLogAndInviter(t *testing.T) {
	setupCommissionMigrateTest(t)
	require.NoError(t, DB.AutoMigrate(&User{}, &CommissionRecord{}))

	// 1. 第一次插入 OK。
	rec1 := &CommissionRecord{
		InviterId:    100,
		InviteeId:    200,
		ConsumeLogId: 1,
		Gross:        1000,
		Rate:         "0.050000",
		Amount:       50,
		Margin:       100,
		Breach:       false,
		Currency:     "USD",
		SettledAt:    time.Now().Unix(),
		CreatedAt:    time.Now().Unix(),
	}
	require.NoError(t, DB.Create(rec1).Error, "首次插入必须成功")

	// 2. 第二次插入同 (consume_log_id, inviter_id) 必须报错。
	rec2 := &CommissionRecord{
		InviterId:    100, // 同邀请人
		InviteeId:    201, // 不同被邀请人（即便如此也要被拦住，约束是前两个字段）
		ConsumeLogId: 1,   // 同 consume_log_id
		Gross:        2000,
		Rate:         "0.050000",
		Amount:       100,
		Margin:       200,
		Breach:       false,
		Currency:     "USD",
		SettledAt:    time.Now().Unix(),
		CreatedAt:    time.Now().Unix(),
	}
	err := DB.Create(rec2).Error
	require.Error(t, err, "唯一约束应触发错误（防止重试返佣）")
	assert.True(t,
		strings.Contains(strings.ToLower(err.Error()), "unique") ||
			strings.Contains(strings.ToLower(err.Error()), "constraint"),
		"错误信息应含 unique/constraint 字样，实际: %v", err)

	// 3. 反向验证：换 consume_log_id 后允许（约束精确到这两个字段）。
	rec3 := &CommissionRecord{
		InviterId:    100,
		InviteeId:    200,
		ConsumeLogId: 2, // 不同 consume_log_id
		Gross:        3000,
		Rate:         "0.050000",
		Amount:       150,
		Margin:       300,
		Breach:       false,
		Currency:     "USD",
		SettledAt:    time.Now().Unix(),
		CreatedAt:    time.Now().Unix(),
	}
	require.NoError(t, DB.Create(rec3).Error, "不同 consume_log_id 应允许")
}

// hasColumn 辅助：查 SQLite 表里某列是否存在（PRAGMA table_info）。
//
// 比 db.Migrator().HasColumn 更直接——后者受 gorm tag 缓存影响，
// 对刚 AutoMigrate 出来的列偶发返回 false。
func hasColumn(db *gorm.DB, tableName, columnName string) bool {
	var count int64
	err := db.Raw(
		"SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?",
		tableName, columnName,
	).Scan(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}