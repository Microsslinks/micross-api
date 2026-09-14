package controller

import (
	"errors"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// discountCustomerAuditRequest 客户档案核算的入参：一个客户 + 他要的整份模型清单。
type discountCustomerAuditRequest struct {
	UserId int      `json:"user_id"`
	Models []string `json:"models"`
}

// AuditCustomerPricing 客户档案核算：签客户之前把「这个客户 + 这份模型清单」一次算完，
// 每个模型按几折、命中哪条价、还有几条线路、最便宜线路成本多少、每单毛利多少，
// 最后一句「这份报价能不能签」。
// POST /api/discount/admin/customer-audit —— 只读接口，不写任何表。
func AuditCustomerPricing(c *gin.Context) {
	var req discountCustomerAuditRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		common.ApiErrorMsg(c, "请求参数格式错误")
		return
	}
	if req.UserId <= 0 {
		common.ApiErrorMsg(c, "user_id 必须是正整数")
		return
	}
	if len(req.Models) > service.CustomerAuditMaxModels {
		common.ApiErrorMsg(c, service.ErrCustomerAuditTooManyModels.Error())
		return
	}

	result, err := service.AuditCustomerPricing(req.UserId, req.Models)
	if err != nil {
		// 客户不存在、清单是空的都属于「输入的问题」，回 400 且把话说清楚；
		// 其余（数据库等）走通用错误处理。
		if errors.Is(err, service.ErrCustomerAuditUserNotFound) ||
			errors.Is(err, service.ErrCustomerAuditNoModels) ||
			errors.Is(err, service.ErrCustomerAuditTooManyModels) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}
