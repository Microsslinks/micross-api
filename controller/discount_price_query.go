package controller

import (
	"errors"
	"io"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// discountPriceQueryRequest 客户查价的入参：一个客户 + 要问的模型。
// models 可以留空（或全是空白行）——那就只回价目本，先看清他身上有几套价。
type discountPriceQueryRequest struct {
	UserId int      `json:"user_id"`
	Models []string `json:"models"`
}

// QueryCustomerPricing 客户查价：这个客户身上挂了哪几套价、现价按几折、命中的是哪套、
// 其余几套输在哪一层、每个模型还有几条线路、毛利多少。
// POST /api/discount/admin/price-query —— 只读接口，不写任何表。
func QueryCustomerPricing(c *gin.Context) {
	var req discountPriceQueryRequest
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

	result, err := service.QueryCustomerPricing(req.UserId, req.Models)
	if err != nil {
		// 客户不存在、清单太大属于「输入的问题」，回 400 并把话说清楚；其余走通用错误处理。
		if errors.Is(err, service.ErrCustomerAuditUserNotFound) ||
			errors.Is(err, service.ErrCustomerAuditTooManyModels) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}
