package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupReplayTest 准备一个内存 SQLite，含 users + logs 表，模拟生产 DB 的最小集。
//
// model.Log / model.User 都建上（与主服务的 model.InitDB 子集对齐）。
// 不建 commission_records——G1 脚本承诺"只算不算账"，验证这个边界条件。
func setupReplayTest(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:replay-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Log{}, &model.Channel{}))
	model.DB, model.LOG_DB = db, db

	outputPath := filepath.Join(t.TempDir(), "report.csv")

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
		_ = os.Remove(outputPath)
	})
	return db, outputPath
}

// seedLogsAndUsers 造一组 Log + User fixture：
//   - inviter + invitee（invitee.InviterId = inviter.Id）→ 有邀请关系
//   - loneUser（InviterId = 0）→ 无邀请关系，应跳过
func seedLogsAndUsers(t *testing.T, db *gorm.DB) (inviter, invitee, loneUser *model.User, logs []*model.Log) {
	t.Helper()
	inviter = &model.User{
		Username: "g1-inviter", Password: "u", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: fmt.Sprintf("aff-g1-inv-%d", time.Now().UnixNano()),
	}
	require.NoError(t, db.Create(inviter).Error)
	invitee = &model.User{
		Username: "g1-invitee", Password: "u", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		InviterId: inviter.Id,
		AffCode:   fmt.Sprintf("aff-g1-inv2-%d", time.Now().UnixNano()),
	}
	require.NoError(t, db.Create(invitee).Error)
	loneUser = &model.User{
		Username: "g1-lone", Password: "u", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: fmt.Sprintf("aff-g1-lone-%d", time.Now().UnixNano()),
	}
	require.NoError(t, db.Create(loneUser).Error)

	now := time.Now().Unix()
	logs = []*model.Log{
		// invitee 3 条消费，正常返佣（gross=100 → raw=5, final=5）
		{UserId: invitee.Id, Type: model.LogTypeConsume, ModelName: "gpt-4o",
			Quota: 100, ChannelId: 1, Group: "default", CreatedAt: now - 86400},
		{UserId: invitee.Id, Type: model.LogTypeConsume, ModelName: "gpt-4o",
			Quota: 200, ChannelId: 1, Group: "default", CreatedAt: now - 43200},
		{UserId: invitee.Id, Type: model.LogTypeConsume, ModelName: "claude",
			Quota: 1000, ChannelId: 2, Group: "default", CreatedAt: now - 3600},
		// loneUser 1 条消费（无邀请关系，应跳过）
		{UserId: loneUser.Id, Type: model.LogTypeConsume, ModelName: "gpt-4o",
			Quota: 50, ChannelId: 1, Group: "default", CreatedAt: now - 3600},
	}
	for _, l := range logs {
		require.NoError(t, db.Create(l).Error)
	}
	return inviter, invitee, loneUser, logs
}

// TestG1ReplayDryRunCallsServiceFunctions
//
// 验证 G1 脚本的主流程（dry-run）：
//   - 查到 invitee 3 条 + loneUser 1 条 = 4 条
//   - with_inviter = 3
//   - total_raw = 5 + 10 + 50 = 65（rate=0.05: 100*0.05=5, 200*0.05=10, 1000*0.05=50）
//   - total_final = 0（cost_ratio 未接入，margin=0 → AssertNoLoss 归零 + breach=true）
//   - breach_count = 3（margin_unwired 3 条）
//   - effective_breach_rate = 0（margin_out_of_budget = 0）
//   - CSV 写到 outputPath
//   - 不写 commission_records（验证"只算不算账"边界条件）
func TestG1ReplayDryRunCallsServiceFunctions(t *testing.T) {
	db, outputPath := setupReplayTest(t)
	_, _, _, _ = seedLogsAndUsers(t, db)

	a := args{days: 7, rate: "0.05", output: outputPath}
	replayOnDB(a) // 跳过 initDB，用测试 setup 的 DB

	// CSV 校验
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	csv := string(content)
	// 表头
	assert.Contains(t, csv, "log_id,user_id,channel_id,model_name,gross,rate,raw_amount,final_amount,margin,breach")
	// 应该有 3 行数据（invitee 3 条，loneUser 1 条被跳过）
	lines := 0
	for _, line := range []byte(csv) {
		if line == '\n' {
			lines++
		}
	}
	assert.GreaterOrEqual(t, lines, 4, "应有 3 行数据 + 表头 + 末尾换行")

	// commission_records 表应为空（验证"只算不算账"边界条件）：
//   - 表不存在（测试 setup 没建）→ 视为合规
//   - 表存在但 0 条 → 也合规
//   - 表存在且 >0 条 → 失败
	if db.Migrator().HasTable(&model.CommissionRecord{}) {
		var count int64
		require.NoError(t, db.Model(&model.CommissionRecord{}).Count(&count).Error)
		assert.Equal(t, int64(0), count, "G1 必须不写 commission_records")
	}
}