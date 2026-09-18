package controller

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GetAffCommissionBalance 当前用户独立钱包余额（task-10 / P4 佣金核心）。
//
// 路径：GET /api/user/aff/commission/balance
// 鉴权：middleware.UserAuth（selfRoute 分组自带）。
// 返回：{ balance: int }——邀请佣金钱包余额，与主余额 quota 区分。
//
// 与 aff_quota 区分：aff_quota 是「邀请人给被邀请人的额度」（已存在的旧机制）；
// aff_commission_balance 是「被邀请人消费后，邀请人拿到的佣金」（task-10 新增）。
func GetAffCommissionBalance(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未登录")
		return
	}
	user := &model.User{Id: userId}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"balance": user.AffCommissionBalance,
		"currency": "USD",
	})
}

// ListAffCommissionRecords 邀请佣金流水分页。
//
// 路径：GET /api/user/aff/commission/records
// 查询参数：page（默认 1）、page_size（默认 50，上限 200）
// 返回：{ items: [...], page, page_size, total }。
//
// 数据范围：当前用户作为邀请人产生的所有 commission_records，按 id DESC（最新优先）。
func ListAffCommissionRecords(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未登录")
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 50
	}
	if page <= 0 {
		page = 1
	}

	db := model.DB
	var total int64
	if err := db.Model(&model.CommissionRecord{}).
		Where("inviter_id = ?", userId).Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	var records []model.CommissionRecord
	if err := db.Where("inviter_id = ?", userId).
		Order("id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&records).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"items":     records,
		"page":      page,
		"page_size": pageSize,
		"total":     total,
	})
}

// GetAffCommissionSummary 邀请佣金累计汇总。
//
// 路径：GET /api/user/aff/commission/summary
// 返回：{ total_amount, record_count, breach_count }。
//
//   - total_amount: 所有 amount 之和（含负——退款冲销）。
//   - record_count: 流水条数。
//   - breach_count: breach=true 的条数（运营排查"是否频繁不赔本"的关键指标）。
//
// 不返回 rate：rate 在流水里已经快照写死了，按笔算比按当前运营设置算更准确。
func GetAffCommissionSummary(c *gin.Context) {
	userId := c.GetInt("id")
	if userId == 0 {
		common.ApiErrorMsg(c, "未登录")
		return
	}
	db := model.DB
	type summaryRow struct {
		TotalAmount  int64
		RecordCount  int64
		BreachCount  int64
	}
	var row summaryRow
	if err := db.Model(&model.CommissionRecord{}).
		Select("COALESCE(SUM(amount), 0) AS total_amount, COUNT(*) AS record_count, COALESCE(SUM(CASE WHEN breach = 1 THEN 1 ELSE 0 END), 0) AS breach_count").
		Where("inviter_id = ?", userId).
		Scan(&row).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"total_amount": row.TotalAmount,
		"record_count": row.RecordCount,
		"breach_count": row.BreachCount,
		"currency":     "USD",
	})
}

// AdminSetCommissionRate 运营设置全局返佣率。
//
// 路径：POST /api/admin/commission/rate
// 鉴权：middleware.AdminAuth（adminRoute 分组自带）。
// 请求体：{ "rate": "0.050000" }——DECIMAL(6,6) 字符串，范围 [0, 1]。
//
// 写入流程：model.UpdateOption("CommissionRate", rate) 同步落 options 表 + 内存 var，
// 启动时 InitOptionMap 会从 DB 反向加载（参考 DemoSiteEnabled 持久化模式）。
//
// 校验：operation_setting.ValidateCommissionRate 拒绝空串、负数、>1、非数字。
// 校验失败时 model.UpdateOption 已经返回 error，handler 直接透传给前端。
type adminSetCommissionRateRequest struct {
	Rate *string `json:"rate"`
}

func AdminSetCommissionRate(c *gin.Context) {
	var req adminSetCommissionRateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求体格式错误："+err.Error())
		return
	}
	if req.Rate == nil {
		common.ApiErrorMsg(c, "缺少 rate 字段")
		return
	}
	// 校验放在 UpdateOption 之前——失败时直接拒绝，不写 DB。
	if err := operation_setting.ValidateCommissionRate(*req.Rate); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	// task-17 §17.4 修复关键点：必须在 UpdateOption **之前**读 oldRate。
	// UpdateOption 内部 model/option.go:620 会把 var 同步到新值，之后再读只会拿到新值，
	// audit 永远显示 from==to，运营出现"我的佣金变少了"工单时无法定位调参时间。
	oldRate := operation_setting.GetCommissionRate()

	if err := model.UpdateOption("CommissionRate", *req.Rate); err != nil {
		common.ApiError(c, err)
		return
	}

	// task-17 §17.4：运营调返佣率是高敏感写操作——直接决定所有邀请人钱包入账速率。
	// 必须在 audit log 留痕：哪个 admin 何时改成多少 rate → 出现"我的佣金变少了"
	// 工单时能立刻定位到调参时间。
	//
	// 双轨日志：
	//   1. logger.LogInfo → stdout / 文件日志，便于运维 grep（与 rate change 同日的事件相关）
	//   2. model.RecordLogWithAdminInfo → logs 表 + admin_info JSON，前端 admin_log 面板能查
	adminId := c.GetInt("id")
	logger.LogInfo(c.Request.Context(), fmt.Sprintf(
		"audit commission_rate_changed admin_id=%d old_rate=%s new_rate=%s",
		adminId, oldRate, *req.Rate))
	if adminId > 0 {
		model.RecordLogWithAdminInfo(adminId, model.LogTypeManage,
			fmt.Sprintf("admin_set_commission_rate: from=%s to=%s", oldRate, *req.Rate),
			map[string]interface{}{
				"old_rate":  oldRate,
				"new_rate":  *req.Rate,
				"admin_id":  adminId,
				"action":    "set_commission_rate",
				"audit_tag": "commission_rate_changed",
			})
	}
	logger.LogInfo(c.Request.Context(), fmt.Sprintf(
		"audit commission_rate_changed admin_id=%d old_rate=%s new_rate=%s",
		adminId, oldRate, *req.Rate))
	if adminId > 0 {
		model.RecordLogWithAdminInfo(adminId, model.LogTypeManage,
			fmt.Sprintf("admin_set_commission_rate: from=%s to=%s", oldRate, *req.Rate),
			map[string]interface{}{
				"old_rate":  oldRate,
				"new_rate":  *req.Rate,
				"admin_id":  adminId,
				"action":    "set_commission_rate",
				"audit_tag": "commission_rate_changed",
			})
	}

	common.ApiSuccess(c, gin.H{
		"rate":     *req.Rate,
		"currency": "USD",
	})
}

// 编译期兜底：model.DB 引用确认（防止 unused import 编译失败）。
var _ = model.DB

// ----------------------------------------------------------------------
// task-20 §20.7: admin commission_records 管理接口
//
//  - GET  /admin/commission/records          分页列出 + 过滤
//  - POST /admin/commission/records/:id/reverse   撤销单条
//
// 为什么需要这两个：
//   - 列出接口：财务/客服在 admin 后台查"某人最近 30 天返佣明细"用——
//     不能直接给 commission_records 表的 SQL 权限给 admin UI。
//   - 撤销接口：把 §20.6 的 service.ReverseCommission 挂到 HTTP 路由；
//     ErrCommissionAlreadyReversed 翻译为 HTTP 409 + 文案复用。
//
// 路由已在 router/api-router.go:adminCommissionRoute 注册（§20.7 一并加）。
// ----------------------------------------------------------------------

// adminListCommissionRecordsRequest 列表查询参数。
//
// 字段可选：
//   - page / page_size: 分页（默认 page=1, page_size=20, page_size 最大 200）
//   - inviter_id / invitee_id: 精确过滤
//   - reversed: true=仅已撤销, false=仅未撤销, 缺省=全部
//   - breach:   true=仅 breach, false=仅非 breach, 缺省=全部
type adminListCommissionRecordsRequest struct {
	Page      int  `json:"page"`
	PageSize  int  `json:"page_size"`
	InviterID int  `json:"inviter_id"`
	InviteeID int  `json:"invitee_id"`
	Reversed  *bool `json:"reversed"`
	Breach    *bool `json:"breach"`
}

// AdminListCommissionRecords 分页列出 commission_records。
//
// 响应：{"items": [...], "total": N, "page": ..., "page_size": ...}。
//
// 与全表扫描相关：commission_records 索引已存在（inviter_id 单列 +
// idx_invitee_consume 复合），按 inviter_id 过滤命中索引；不带过滤的
// 全表扫描返回 page_size 默认 20 条 + total 数字——可接受。
func AdminListCommissionRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 20
	}
	if page <= 0 {
		page = 1
	}

	inviterID, _ := strconv.Atoi(c.Query("inviter_id"))
	inviteeID, _ := strconv.Atoi(c.Query("invitee_id"))
	var reversedFilter *bool
	if s := c.Query("reversed"); s != "" {
		b, err := strconv.ParseBool(s)
		if err == nil {
			reversedFilter = &b
		}
	}
	var breachFilter *bool
	if s := c.Query("breach"); s != "" {
		b, err := strconv.ParseBool(s)
		if err == nil {
			breachFilter = &b
		}
	}

	q := model.DB.Model(&model.CommissionRecord{})
	if inviterID > 0 {
		q = q.Where("inviter_id = ?", inviterID)
	}
	if inviteeID > 0 {
		q = q.Where("invitee_id = ?", inviteeID)
	}
	if reversedFilter != nil {
		q = q.Where("reversed = ?", *reversedFilter)
	}
	if breachFilter != nil {
		q = q.Where("breach = ?", *breachFilter)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	var items []model.CommissionRecord
	if err := q.Order("id DESC").
		Limit(pageSize).Offset((page - 1) * pageSize).
		Find(&items).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	common.ApiSuccess(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

type adminReverseCommissionRequest struct {
	Reason string `json:"reason"`
}

// AdminReverseCommission 撤销一笔 commission_records（task-20 §20.7 admin HTTP 入口）。
//
// 调用契约：
//   - body: {"reason": "..."}，reason 必填；空 reason 会被 service 兜底为
//     "admin reverse (no reason given)"。
//   - 成功 → 200 + commission_record 详情。
//   - record 不存在 → 404（service 内 error 不带 ErrCommissionAlreadyReversed）。
//   - 已撤销 → 409（errors.Is(err, model.ErrCommissionAlreadyReversed)）。
//   - 其它事务错误 → 500。
//
// 审计：admin_id / reason / record 详情都写 logs 表 LogTypeManage 行（§20.6
// 的 ledger 行是账本追踪，logs 表是 admin 行为追踪，两者不冲突）。
func AdminReverseCommission(c *gin.Context) {
	idStr := c.Param("id")
	recordID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || recordID <= 0 {
		common.ApiErrorMsg(c, "无效的 commission record id")
		return
	}

	var req adminReverseCommissionRequest
	_ = c.ShouldBindJSON(&req) // reason 可选：service 会兜底

	adminID := c.GetInt("id")
	if err := service.ReverseCommission(c, recordID, req.Reason, adminID); err != nil {
		if errors.Is(err, model.ErrCommissionAlreadyReversed) {
			// 用 ApiError + 自定义 status code 难，改为直接 409。
			c.JSON(409, gin.H{
				"success": false,
				"message": "该返佣记录已被撤销",
				"data":    nil,
			})
			return
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.ApiErrorMsg(c, fmt.Sprintf("commission record %d 不存在", recordID))
			return
		}
		common.ApiError(c, err)
		return
	}

	// 审计：写一条 logs 表行，便于 admin_log 面板追溯。
	if adminID > 0 {
		model.RecordLogWithAdminInfo(adminID, model.LogTypeManage,
			fmt.Sprintf("admin_reverse_commission: record_id=%d reason=%s", recordID, req.Reason),
			map[string]interface{}{
				"action":        "reverse_commission",
				"record_id":     recordID,
				"reason":        req.Reason,
				"admin_id":      adminID,
				"audit_tag":     "commission_reversed",
			})
	}

	// 读回最新 record 给前端
	var rec model.CommissionRecord
	if err := model.DB.First(&rec, recordID).Error; err != nil {
		common.ApiSuccess(c, gin.H{
			"record_id": recordID,
			"reversed":  true,
		})
		return
	}
	common.ApiSuccess(c, rec)
}