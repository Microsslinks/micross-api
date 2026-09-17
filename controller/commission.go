package controller

import (
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
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