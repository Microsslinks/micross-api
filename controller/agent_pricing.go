package controller

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 经销商自己的货架：他能拿去给客户报价的方案清单。
//
// 发客户号时要挑一套方案（客户绑号后按它计价），给下属客户改价时也要挑，
// 所以这一份清单是两个动作共用的。谁能看到它由身份决定，不是管理员权限。
func GetSelfAgentPlans(c *gin.Context) {
	userId := c.GetInt("id")
	isAgent, err := model.IsUserAgent(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !isAgent {
		common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
		return
	}

	floor, plans, err := model.ListSellablePlansForAgent(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"min_discount": floor,
		"items":        plans,
	})
}
