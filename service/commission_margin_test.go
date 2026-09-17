package service

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupCommissionMarginTest 给 calculateCommissionMargin 用例一份干净的内存 SQLite
// （与 commission_test.go 同模式，cache=shared 避免 setup 关闭影响其他测试）。
func setupCommissionMarginTest(t *testing.T) {
	t.Helper()

	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()

	dsn := fmt.Sprintf("file:margin-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// channels 是 calculateCommissionMargin 读 channel.CostRatio 唯一路径。
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.User{}, &model.Log{}))

	model.DB, model.LOG_DB = db, db
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
}

func seedChannelWithCostRatio(t *testing.T, id int, costRatio string) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Channel{
		Id:        id,
		Type:      1,
		Key:       fmt.Sprintf("margin-test-key-%d", id),
		Name:      fmt.Sprintf("margin-channel-%d", id),
		Status:    common.ChannelStatusEnabled,
		Models:    "test-model",
		Group:     "default",
		CostRatio: ptrStringOrNil(costRatio),
	}).Error)
}

func ptrStringOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func TestCalculateCommissionMargin(t *testing.T) {
	cases := []struct {
		name       string
		costRatio  *string
		quota      int
		channelID  int
		wantMargin int64
	}{
		{
			// happy path：进货 0.7 → 毛利 0.3 × 1000 = 300 quota
			name:       "happy path 0.7 cost_ratio with quota 1000 yields margin 300",
			costRatio:  strPtr("0.700000"),
			quota:      1000,
			channelID:  7001,
			wantMargin: 300,
		},
		{
			// 进货 0.9 → 毛利 0.1 × 1000 = 100 quota（边际保本：commission 上限 = margin × 0.8 = 80）
			name:       "high cost_ratio yields small margin",
			costRatio:  strPtr("0.900000"),
			quota:      1000,
			channelID:  7002,
			wantMargin: 100,
		},
		{
			// cost_ratio=1（进货与零售同价）→ 毛利 = 0（不赔本就难）
			name:       "cost_ratio 1 yields zero margin",
			costRatio:  strPtr("1.000000"),
			quota:      1000,
			channelID:  7003,
			wantMargin: 0,
		},
		{
			// nil cost_ratio（运营未录进货折扣）→ 0 + 走 margin_unwired audit
			name:       "nil cost_ratio yields zero margin",
			costRatio:  nil,
			quota:      1000,
			channelID:  7004,
			wantMargin: 0,
		},
		{
			// 空字符串 cost_ratio → 0 + 走 margin_unwired audit
			name:       "empty cost_ratio yields zero margin",
			costRatio:  strPtr(""),
			quota:      1000,
			channelID:  7005,
			wantMargin: 0,
		},
		{
			// 解析失败（非数字）→ 0
			name:       "non-numeric cost_ratio yields zero margin",
			costRatio:  strPtr("not-a-number"),
			quota:      1000,
			channelID:  7006,
			wantMargin: 0,
		},
		{
			// 超出 (0, 1] 范围 → 0
			name:       "cost_ratio > 1 yields zero margin",
			costRatio:  strPtr("1.500000"),
			quota:      1000,
			channelID:  7007,
			wantMargin: 0,
		},
		{
			// 负数 cost_ratio → 0
			name:       "negative cost_ratio yields zero margin",
			costRatio:  strPtr("-0.100000"),
			quota:      1000,
			channelID:  7008,
			wantMargin: 0,
		},
		{
			// quota=0（不计费场景）→ 0（不写 commission_records）
			name:       "zero quota yields zero margin",
			costRatio:  strPtr("0.700000"),
			quota:      0,
			channelID:  7009,
			wantMargin: 0,
		},
		{
			// channelID 不存在 → 0（找不到渠道走 unwired 兜底，不 panic）
			name:       "unknown channel id yields zero margin",
			costRatio:  strPtr("0.700000"),
			quota:      1000,
			channelID:  999999,
			wantMargin: 0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupCommissionMarginTest(t)
			if tc.channelID != 999999 {
				seedChannelWithCostRatio(t, tc.channelID, derefStr(tc.costRatio))
			}

			ctx, _ := gin.CreateTestContext(nil)
			common.SetContextKey(ctx, constant.ContextKeyChannelId, tc.channelID)
			relayInfo := &relaycommon.RelayInfo{
				ChannelMeta: &relaycommon.ChannelMeta{
					ChannelId: tc.channelID,
				},
			}
			summary := &textQuotaSummary{Quota: tc.quota, ModelName: "test-model"}

			got := calculateCommissionMargin(ctx, relayInfo, summary)
			assert.Equal(t, tc.wantMargin, got)
		})
	}
}

// TestCalculateCommissionMarginNilSafety 守住 nil summary / nil relayInfo 不 panic
// —— 计费主链路 panic 会拖垮整条 relay；commission 是旁路，但 panic 仍走 defer recover
// 把 try 写进日志。先单元测守一遍，再走集成链路。
func TestCalculateCommissionMarginNilSafety(t *testing.T) {
	setupCommissionMarginTest(t)

	t.Run("nil summary", func(t *testing.T) {
		ctx, _ := gin.CreateTestContext(nil)
		relayInfo := &relaycommon.RelayInfo{ChannelMeta: &relaycommon.ChannelMeta{ChannelId: 7001}}
		seedChannelWithCostRatio(t, 7001, "0.700000")
		common.SetContextKey(ctx, constant.ContextKeyChannelId, 7001)
		got := calculateCommissionMargin(ctx, relayInfo, nil)
		assert.Equal(t, int64(0), got)
	})

	t.Run("nil relayInfo", func(t *testing.T) {
		ctx, _ := gin.CreateTestContext(nil)
		summary := &textQuotaSummary{Quota: 1000}
		got := calculateCommissionMargin(ctx, nil, summary)
		assert.Equal(t, int64(0), got)
	})

	t.Run("nil channel meta", func(t *testing.T) {
		ctx, _ := gin.CreateTestContext(nil)
		summary := &textQuotaSummary{Quota: 1000}
		relayInfo := &relaycommon.RelayInfo{} // ChannelMeta nil
		got := calculateCommissionMargin(ctx, relayInfo, summary)
		assert.Equal(t, int64(0), got)
	})
}

// helpers（test-only）
func strPtr(s string) *string { return &s }

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// 防止 _ = sqlDB.Close() 这行用到 sql 包被 goimports 误删；真实用 *sql.DB.Close。
var _ = sql.ErrNoRows