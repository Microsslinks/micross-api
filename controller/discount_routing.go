package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// 客户路由策略：8b 的择优策略（默认按毛利优先，可按客户切成稳定优先）与 8c 的放行开关
// （允许这个客户走会亏本的线路）。挂在 /api/discount/admin 下，要管理员权限。
//
// 界面文案里这个主体叫「客户／经销商」，但底层就是 users 表里的一行，所以只按 user_id 定位。

type discountRoutingRequest struct {
	UserId          *int    `json:"user_id"`
	RoutingStrategy *string `json:"routing_strategy"`
	AllowCostBreach *bool   `json:"allow_cost_breach"`
	Remark          *string `json:"remark"`
}

// GetDiscountRouting 读一个客户当前生效的路由策略。
// GET /api/discount/admin/routing?user_id=12
//
// 没配置过的客户也返回成功，configured 为 false、策略字段是默认口径——「没配过」是
// 绝大多数客户的常态，界面据此显示「未单独配置」而不是报错。
func GetDiscountRouting(c *gin.Context) {
	userId, ok := parseDiscountRoutingUserId(c, c.Query("user_id"))
	if !ok {
		return
	}
	view, err := service.GetCustomerRouting(userId)
	if err != nil {
		if errors.Is(err, service.ErrDiscountRoutingUserNotFound) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, view)
}

// UpdateDiscountRouting 写入（或更新）一个客户的路由策略。没传的字段按默认口径处理。
// PUT /api/discount/admin/routing
// body: {"user_id":12,"routing_strategy":"priority","allow_cost_breach":true,"remark":"客户要求走更稳的线路"}
func UpdateDiscountRouting(c *gin.Context) {
	var req discountRoutingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "请求参数格式错误")
		return
	}
	if req.UserId == nil || *req.UserId <= 0 {
		common.ApiErrorMsg(c, service.ErrDiscountRoutingInvalidUser.Error())
		return
	}

	update := &service.CustomerRoutingUpdate{
		UserId:          *req.UserId,
		RoutingStrategy: model.RoutingStrategyMargin,
		OperatorId:      c.GetInt("id"),
	}
	if req.RoutingStrategy != nil {
		update.RoutingStrategy = *req.RoutingStrategy
	}
	if req.AllowCostBreach != nil {
		update.AllowCostBreach = *req.AllowCostBreach
	}
	if req.Remark != nil {
		update.Remark = *req.Remark
	}

	if err := service.SaveCustomerRouting(update); err != nil {
		if errors.Is(err, service.ErrDiscountRoutingUserNotFound) || errors.Is(err, service.ErrDiscountRoutingInvalidStrategy) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// DeleteDiscountRouting 删掉一个客户的例外配置，回到默认口径（毛利优先、不允许走亏损线路）。
// DELETE /api/discount/admin/routing?user_id=12
func DeleteDiscountRouting(c *gin.Context) {
	userId, ok := parseDiscountRoutingUserId(c, c.Query("user_id"))
	if !ok {
		return
	}
	if err := service.DeleteCustomerRouting(userId); err != nil {
		if errors.Is(err, service.ErrDiscountRoutingInvalidUser) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// parseDiscountRoutingUserId 解析 ?user_id= 查询参数。参数不合法时已经写好响应，返回 false。
func parseDiscountRoutingUserId(c *gin.Context, raw string) (int, bool) {
	userId, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || userId <= 0 {
		common.ApiErrorMsg(c, service.ErrDiscountRoutingInvalidUser.Error())
		return 0, false
	}
	return userId, true
}
