package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ----------------------------------------------------------------------
// task-20 §20.7: admin commission_records 管理接口测试
// ----------------------------------------------------------------------

// adminTestSeq 独立于 commission_test.go 的 commissionTestSeq，避免跨文件干扰。
var adminTestSeq int

// seedAdminCommissionRecord 插一条 commission_records 行（绕过 ProcessCommission，
// 让 controller 测试聚焦于 list/reverse 接口本身）。
//
// ConsumeLogId 用 adminTestSeq*1000000 保证唯一（commission_records 有
// uniqueIndex:uk_consume_inviter，覆盖 (consume_log_id, inviter_id)），
// 与 controller/commission_test.go 的 commissionTestSeq 完全独立。
//
// 注意：setupCommissionControllerTest 没 migrate account_ledger；这里
// 手动 migrate，避免 §20.6 service.ReverseCommission 写 ledger 触发表不存在。
func seedAdminCommissionRecord(t *testing.T, db *gorm.DB, rec *model.CommissionRecord) *model.CommissionRecord {
	t.Helper()
	adminTestSeq++
	if rec.ConsumeLogId == 0 {
		rec.ConsumeLogId = int64(adminTestSeq) * 1000000
	}
	if rec.SettledAt == 0 {
		rec.SettledAt = common.GetTimestamp()
	}
	if rec.CreatedAt == 0 {
		rec.CreatedAt = common.GetTimestamp()
	}
	// account_ledger 表在 setupCommissionControllerTest 不存在；
	// 手动 migrate，幂等（AUTO_MIGRATE 不会重建已有表）。
	if !db.Migrator().HasTable(&model.AccountLedger{}) {
		require.NoError(t, db.AutoMigrate(&model.AccountLedger{}))
	}
	require.NoError(t, db.Create(rec).Error)
	return rec
}

// TestAdminListCommissionRecordsPagesAndFilters:
// happy path：建 5 条 records，page_size=2 → 翻 3 页都通；按 inviter_id 过滤
// 只返回该 inviter 的；reversed=true 过滤只返回已撤销的。
func TestAdminListCommissionRecordsPagesAndFilters(t *testing.T) {
	db := setupCommissionControllerTest(t)

	// 建 3 个 inviter 各 1 条 + 1 个 invitee 3 条
	for i := 0; i < 3; i++ {
		inviter := &model.User{
			Username: fmt.Sprintf("list-inv-%d-%d", i, common.GetTimestamp()),
			Password: "unused", Role: common.RoleCommonUser,
			Status: common.UserStatusEnabled, Group: "default",
			AffCode: fmt.Sprintf("list-inv-aff-%d-%d", i, common.GetTimestamp()),
		}
		require.NoError(t, db.Create(inviter).Error)
		seedAdminCommissionRecord(t, db, &model.CommissionRecord{
			InviterId: inviter.Id, InviteeId: 9000 + i,
			Gross: 100, Rate: "0.05", Amount: 5, Margin: 50,
			Currency: "USD", Reversed: false,
		})
	}
	// 第 4 个 inviter 多条（便于按 inviter 过滤）
	multiInviter := &model.User{
		Username: "list-multi-inv-" + fmt.Sprint(common.GetTimestamp()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "list-multi-aff-" + fmt.Sprint(common.GetTimestamp()),
	}
	require.NoError(t, db.Create(multiInviter).Error)
	for i := 0; i < 3; i++ {
		seedAdminCommissionRecord(t, db, &model.CommissionRecord{
			InviterId: multiInviter.Id, InviteeId: 8000 + i,
			Gross: 200, Rate: "0.05", Amount: 10, Margin: 100,
			Currency: "USD", Reversed: i == 0, // 第 1 条标 Reversed
		})
	}

	// 1. 默认分页：page=1, page_size=20 → 返回所有 6 条
	c, w := newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/commission/records", nil)
	AdminListCommissionRecords(c)
	assert.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data struct {
			Items    []model.CommissionRecord `json:"items"`
			Total    int64                    `json:"total"`
			Page     int                      `json:"page"`
			PageSize int                      `json:"page_size"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(6), resp.Data.Total)
	assert.Len(t, resp.Data.Items, 6)
	assert.Equal(t, 1, resp.Data.Page)
	assert.Equal(t, 20, resp.Data.PageSize)

	// 2. page_size=2, page=1 → 2 条
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/commission/records?page=1&page_size=2", nil)
	AdminListCommissionRecords(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(6), resp.Data.Total)
	assert.Len(t, resp.Data.Items, 2)

	// 3. 按 inviter_id 过滤：只返回 multiInviter 的 3 条
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/admin/commission/records?inviter_id=%d", multiInviter.Id), nil)
	AdminListCommissionRecords(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(3), resp.Data.Total)
	for _, item := range resp.Data.Items {
		assert.Equal(t, multiInviter.Id, item.InviterId)
	}

	// 4. reversed=true → 只返回 multiInviter 的第 1 条
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/commission/records?reversed=true", nil)
	AdminListCommissionRecords(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(1), resp.Data.Total)
	assert.True(t, resp.Data.Items[0].Reversed)

	// 5. reversed=false → 5 条
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/commission/records?reversed=false", nil)
	AdminListCommissionRecords(c)
	require.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, int64(5), resp.Data.Total)
}

// TestAdminReverseCommissionHappyPath:
// POST /admin/commission/records/:id/reverse body={"reason":"..."}
// → service.ReverseCommission 走通 + 返回 record 详情。
func TestAdminReverseCommissionHappyPath(t *testing.T) {
	db := setupCommissionControllerTest(t)

	inviter := &model.User{
		Username: "rev-ctrl-inv-" + fmt.Sprint(common.GetTimestamp()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "rev-ctrl-inv-aff-" + fmt.Sprint(common.GetTimestamp()),
		AffCommissionBalance: 100, // 模拟 commission 写过的余额
	}
	require.NoError(t, db.Create(inviter).Error)
	rec := seedAdminCommissionRecord(t, db, &model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: 7001,
		Gross: 200, Rate: "0.05", Amount: 10, Margin: 100,
		Currency: "USD",
	})

	c, w := newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/admin/commission/records/%d/reverse", rec.Id),
		strings.NewReader(`{"reason":"客户投诉金额算错"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(rec.Id)}}
	AdminReverseCommission(c)
	assert.Equal(t, http.StatusOK, w.Code)

	// 验证：record 标记已撤销
	require.NoError(t, db.First(&rec, rec.Id).Error)
	assert.True(t, rec.Reversed)
	assert.Equal(t, 1, rec.ReversedBy)
	assert.Contains(t, rec.ReverseReason, "客户投诉")

	// 验证：钱包扣回
	require.NoError(t, db.First(inviter, inviter.Id).Error)
	assert.Equal(t, 90, inviter.AffCommissionBalance, "100 - 10 = 90")

	// 验证：ledger 行
	var ledgerCount int64
	require.NoError(t, db.Model(&model.AccountLedger{}).
		Where("ref_type = ? AND ref_id = ? AND event_type = ?",
			"commission_record", rec.Id, model.AccountEventCommissionReverse).
		Count(&ledgerCount).Error)
	assert.Equal(t, int64(1), ledgerCount, "应写 commission_reverse ledger 行")
}

// TestAdminReverseCommissionAlreadyReversedReturns409:
// 二次撤销 → errors.Is(err, ErrCommissionAlreadyReversed) → HTTP 409。
func TestAdminReverseCommissionAlreadyReversedReturns409(t *testing.T) {
	db := setupCommissionControllerTest(t)

	inviter := &model.User{
		Username: "rev-409-inv-" + fmt.Sprint(common.GetTimestamp()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "rev-409-inv-aff-" + fmt.Sprint(common.GetTimestamp()),
		AffCommissionBalance: 50,
	}
	require.NoError(t, db.Create(inviter).Error)
	rec := seedAdminCommissionRecord(t, db, &model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: 7002,
		Gross: 100, Rate: "0.05", Amount: 5, Margin: 50,
		Currency: "USD",
	})

	// 第一次撤销成功
	c, w := newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/admin/commission/records/%d/reverse", rec.Id),
		strings.NewReader(`{"reason":"first"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(rec.Id)}}
	AdminReverseCommission(c)
	require.Equal(t, http.StatusOK, w.Code, "第一次撤销应成功")

	// 第二次撤销 → 409
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/admin/commission/records/%d/reverse", rec.Id),
		strings.NewReader(`{"reason":"second"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprint(rec.Id)}}
	AdminReverseCommission(c)
	assert.Equal(t, http.StatusConflict, w.Code, "二次撤销应返回 HTTP 409")
	assert.Contains(t, w.Body.String(), "已被撤销")
}

// TestAdminReverseCommissionInvalidID: 不存在的 ID / 非数字 ID 都返错误响应。
//
// common.ApiErrorMsg 与 common.ApiError 都返 HTTP 200 + success=false（项目约定），
// 不是 4xx/5xx。所以这里断言 success=false + message 含具体内容，而非 status code。
func TestAdminReverseCommissionInvalidID(t *testing.T) {
	setupCommissionControllerTest(t)

	// 非数字 ID：strconv.ParseInt 失败 → handler 自己拦
	c, w := newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/admin/commission/records/abc/reverse",
		strings.NewReader(`{"reason":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	AdminReverseCommission(c)
	assert.Equal(t, http.StatusOK, w.Code, "项目约定: 错误也返 200")
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Success, "非数字 ID 应 success=false")
	assert.Contains(t, resp.Message, "无效的")

	// 数字 ID 但 record 不存在：service 返 wrapped error, handler 走 ApiError
	c, w = newCommissionCtx(1)
	c.Request = httptest.NewRequest(http.MethodPost,
		"/api/admin/commission/records/9999999/reverse",
		strings.NewReader(`{"reason":"x"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: "9999999"}}
	AdminReverseCommission(c)
	assert.Equal(t, http.StatusOK, w.Code)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.False(t, resp.Success, "不存在 ID 应 success=false")
	assert.NotEmpty(t, resp.Message, "error message 不应为空")
}