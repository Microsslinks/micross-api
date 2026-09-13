package controller

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 客户这一侧的客户号：看自己挂在谁名下，以及用一张号把归属与折扣绑上。
//
// 两个接口都挂在 /api/user/self 下——动的是会话里的自己，不需要任何管理权限，
// 客户自己就能完成。判断全在 model 层（agent_customer.go），这里只把错误翻成人话。

type bindCustomerCodeRequest struct {
	Code string `json:"code"`
}

// GetMyAgentBinding 读出我此刻的归属经销商与生效折扣，供个人资料页显示。
// 直属平台、没有方案时会返回一份零值快照，这是最常见的正常状态，不是错误。
func GetMyAgentBinding(c *gin.Context) {
	binding, err := model.GetCustomerBinding(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}

// BindCustomerCode 用一张客户号绑定自己：归属到发号的经销商，并按号上的方案计价。
//
// 每种拒绝都有具体说法——号不存在、已作废、已过期、次数用完、方案停用、
// 自己发的号、已经归属别人，客户看到的都是能据以行动的那一句，而不是笼统的失败。
func BindCustomerCode(c *gin.Context) {
	var req bindCustomerCodeRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	binding, err := model.BindCustomerCode(c.GetInt("id"), req.Code)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrCustomerCodeNotFound):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodeNotFound)
		case errors.Is(err, model.ErrCustomerCodeRevoked):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodeRevoked)
		case errors.Is(err, model.ErrCustomerCodeExpired):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodeExpired)
		case errors.Is(err, model.ErrCustomerCodeExhausted):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodeExhausted)
		case errors.Is(err, model.ErrCustomerCodeAlreadyBound):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodeAlreadyBound)
		case errors.Is(err, model.ErrCustomerCodeSelfUse):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodeSelfUse)
		case errors.Is(err, model.ErrCustomerCodeOwnedByAgent):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodeOwnedByAgent)
		case errors.Is(err, model.ErrCustomerBelongsToOtherAgent):
			common.ApiErrorI18n(c, i18n.MsgCustomerBelongsToOtherAgent)
		case errors.Is(err, model.ErrCustomerCodePlanUnavailable):
			common.ApiErrorI18n(c, i18n.MsgCustomerCodePlanUnavailable)
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, binding)
}
