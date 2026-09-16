package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupCommissionControllerTest 给 controller/commission.go 用例一份干净的内存 SQLite。
//
// 与 controller/register_customer_code_test.go:setupRegisterCustomerCodeTest 同套路。
// 注意：option 表也建出来——AdminSetCommissionRate 走 model.UpdateOption 路径，
// 写 options 表 + 同步内存 var。
func setupCommissionControllerTest(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousRate := operation_setting.CommissionRate
	// model.UpdateOption 内部会写 common.OptionMap；测试 setup 没跑 InitOptionMap，
	// OptionMap 是 nil map 会 panic。手动初始化 + 还原。
	previousOptionMap := common.OptionMap
	common.OptionMap = make(map[string]string)

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:commission-ctrl-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.CommissionRecord{}, &model.Option{}))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		operation_setting.SetCommissionRate(previousRate)
		common.OptionMap = previousOptionMap
		_ = sqlDB.Close()
	})
	return db
}

// commissionTestSeq 给测试用例造唯一 aff_code / consume_log_id，避免 uniqueIndex 冲突。
//
// 不能用时间戳：Windows 上 time.Now() 的分辨率不足以让同一测试里的两次创建取到不同值，
// 而 users.aff_code / commission_records.(consume_log_id, inviter_id) 都是唯一索引——
// 参考 model/agent_test.go:agentTestSeq 的做法。
var commissionTestSeq int

func uniqueCtrlAffCode(t *testing.T) string {
	t.Helper()
	commissionTestSeq++
	return fmt.Sprintf("aff-ctrl-%d-%d", time.Now().UnixNano()%1000000, commissionTestSeq)
}

// newCommissionCtx 建一个挂上 userId 的 gin test context，模拟 middleware.UserAuth 设的 id。
func newCommissionCtx(userId int) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("id", userId)
	return c, w
}

// seedCommissionUser 建一个用户（含 unique aff_code），给 User.FillUserById 用。
func seedCommissionUser(t *testing.T, db *gorm.DB, username string, affCommissionBalance int) *model.User {
	t.Helper()
	user := &model.User{
		Username:             username,
		Password:             "unused",
		Role:                 common.RoleCommonUser,
		Status:               common.UserStatusEnabled,
		Group:                "default",
		AffCode:              uniqueCtrlAffCode(t),
		AffCommissionBalance: affCommissionBalance,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// seedCommissionRecord 建一条 commission_records（ConsumeLogId 用 commissionTestSeq 保证唯一）。
func seedCommissionRecord(db *gorm.DB, rec model.CommissionRecord) {
	commissionTestSeq++
	rec.ConsumeLogId = int64(commissionTestSeq) * 1000 // 与 user id 错开
	rec.SettledAt = time.Now().Unix()
	rec.CreatedAt = time.Now().Unix()
	if rec.Currency == "" {
		rec.Currency = "USD"
	}
	if err := db.Create(&rec).Error; err != nil {
		panic(fmt.Sprintf("seed commission record failed: %v", err))
	}
}

// TestGetAffCommissionBalanceReturnsUserBalance
//
// 正常路径：用户登录后调 GET /api/user/aff/commission/balance，返回独立钱包余额。
func TestGetAffCommissionBalanceReturnsUserBalance(t *testing.T) {
	db := setupCommissionControllerTest(t)
	user := seedCommissionUser(t, db, "balance-user", 12345)

	c, w := newCommissionCtx(user.Id)
	GetAffCommissionBalance(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.EqualValues(t, 12345, resp.Data["balance"], "应返回用户的独立钱包余额")
	assert.Equal(t, "USD", resp.Data["currency"])
}

// TestListAffCommissionRecordsPagesByInviter
//
// 验证分页 + 仅返回 inviter_id = 当前用户的记录。
func TestListAffCommissionRecordsPagesByInviter(t *testing.T) {
	db := setupCommissionControllerTest(t)
	inviter := seedCommissionUser(t, db, "inviter-records", 0)
	other := seedCommissionUser(t, db, "other-inviter", 0)

	// 给 inviter 插 3 条记录，other 1 条
	for i := 0; i < 3; i++ {
		seedCommissionRecord(db, model.CommissionRecord{
			InviterId: inviter.Id, InviteeId: 1000 + i,
			Gross: 100, Rate: "0.05", Amount: 5, Margin: 20, Breach: false,
		})
	}
	seedCommissionRecord(db, model.CommissionRecord{
		InviterId: other.Id, InviteeId: 9999,
		Gross: 100, Rate: "0.05", Amount: 5, Margin: 20, Breach: false,
	})

	c, w := newCommissionCtx(inviter.Id)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/user/aff/commission/records?page=1&page_size=50", nil)
	ListAffCommissionRecords(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Items    []model.CommissionRecord `json:"items"`
			Page     int                       `json:"page"`
			PageSize int                       `json:"page_size"`
			Total    int64                     `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.Equal(t, int64(3), resp.Data.Total, "只能看到自己作为邀请人的 3 条记录")
	assert.Len(t, resp.Data.Items, 3)
	for _, item := range resp.Data.Items {
		assert.Equal(t, inviter.Id, item.InviterId, "列表里不能混进其他邀请人的记录")
	}
	assert.Equal(t, 1, resp.Data.Page)
	assert.Equal(t, 50, resp.Data.PageSize)
}

// TestGetAffCommissionSummaryAggregatesByInviter
//
// 验证汇总：总额 / 笔数 / breach 笔数。
func TestGetAffCommissionSummaryAggregatesByInviter(t *testing.T) {
	db := setupCommissionControllerTest(t)
	inviter := seedCommissionUser(t, db, "summary-inviter", 0)
	other := seedCommissionUser(t, db, "summary-other", 0)

	// inviter 名下：5 + 10（正常）+ 8（breach）+ 0（breach, margin_unwired 场景）= 23
	seedCommissionRecord(db, model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: 1, Gross: 100, Rate: "0.05", Amount: 5, Margin: 20, Breach: false,
	})
	seedCommissionRecord(db, model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: 2, Gross: 200, Rate: "0.05", Amount: 10, Margin: 40, Breach: false,
	})
	seedCommissionRecord(db, model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: 3, Gross: 100, Rate: "0.5", Amount: 8, Margin: 10, Breach: true,
	})
	seedCommissionRecord(db, model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: 4, Gross: 100, Rate: "0.05", Amount: 0, Margin: 0, Breach: true,
	})
	// 其他邀请人插一条，不应被汇总
	seedCommissionRecord(db, model.CommissionRecord{
		InviterId: other.Id, InviteeId: 999,
		Gross: 999, Rate: "0.99", Amount: 999, Margin: 999, Breach: false,
	})

	c, w := newCommissionCtx(inviter.Id)
	GetAffCommissionSummary(c)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.Success)
	assert.EqualValues(t, 23, resp.Data["total_amount"], "5+10+8+0=23")
	assert.EqualValues(t, 4, resp.Data["record_count"])
	assert.EqualValues(t, 2, resp.Data["breach_count"], "true=2 条")
	assert.Equal(t, "USD", resp.Data["currency"])
}

// TestAdminSetCommissionRatePersistsAndValidates
//
// 验证 admin 端点：
//   - 校验通过 → 落 DB + 同步内存 var
//   - 校验失败（>1 / 负数 / 空串）→ 拒绝、DB 不变、内存 var 不变
func TestAdminSetCommissionRatePersistsAndValidates(t *testing.T) {
	db := setupCommissionControllerTest(t)

	// 1. 正常路径：rate=0.05 → 落 DB + 同步 var
	c, w := newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":"0.05"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	AdminSetCommissionRate(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// 1.1 内存 var 已更新
	assert.Equal(t, "0.05", operation_setting.CommissionRate)
	// 1.2 DB option 表已写入
	var opt model.Option
	require.NoError(t, db.Where("key = ?", "CommissionRate").First(&opt).Error)
	assert.Equal(t, "0.05", opt.Value)

	// 失败用例的统一判定：HTTP 200 但 success=false（项目所有 ApiError* 都是 200 + success=false 格式）。
	assertFalse := func(body []byte, msg string) {
		var resp struct {
			Success bool   `json:"success"`
			Message string `json:"message"`
		}
		require.NoError(t, json.Unmarshal(body, &resp))
		assert.False(t, resp.Success, msg)
	}

	// 2. 拒绝：rate > 1
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":"1.5"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	assertFalse(w.Body.Bytes(), ">1 应被拒绝")
	assert.Equal(t, "0.05", operation_setting.CommissionRate, "拒绝时内存 var 不变")

	// 3. 拒绝：rate 负数
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":"-0.1"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	assertFalse(w.Body.Bytes(), "负数应被拒绝")

	// 4. 拒绝：rate 空串
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":""}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	assertFalse(w.Body.Bytes(), "空串应被拒绝")

	// 5. 拒绝：rate 字段缺失
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	assertFalse(w.Body.Bytes(), "缺 rate 字段应被拒绝")

	// 6. 拒绝：rate 非数字
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":"abc"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	assertFalse(w.Body.Bytes(), "非数字应被拒绝")

	// 校验失败后 DB 仍是 0.05
	require.NoError(t, db.Where("key = ?", "CommissionRate").First(&opt).Error)
	assert.Equal(t, "0.05", opt.Value, "所有失败用例后 DB 不应被覆盖")
}

// TestAdminSetCommissionRateAcceptsBoundaryValues
//
// 边界：rate = 0（关闭返佣）应允许；rate = "1.000000"（1.0）应允许。
func TestAdminSetCommissionRateAcceptsBoundaryValues(t *testing.T) {
	setupCommissionControllerTest(t)

	// 边界 1：rate = "0" → 关闭返佣（合法）
	c, w := newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":"0"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	assert.Equal(t, http.StatusOK, w.Code, "rate=0 应允许（关闭返佣）")
	assert.Equal(t, "0", operation_setting.CommissionRate)

	// 边界 2：rate = "1.000000" → 全额返佣（合法但要小心——运营场景罕见）
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":"1.000000"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	assert.Equal(t, http.StatusOK, w.Code, "rate=1.000000 应允许（边界值）")
	assert.Equal(t, "1.000000", operation_setting.CommissionRate)

	// 边界 3：rate = "1.000001" → 略大于 1，应拒绝（success=false）
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost, "/api/admin/commission/rate", strings.NewReader(`{"rate":"1.000001"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	AdminSetCommissionRate(c)
	var resp struct {
		Success bool `json:"success"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Success, "rate > 1 应被拒绝")
}