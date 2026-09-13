package controller

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 经销商自助台账。
//
// 前面两个接口（controller/agent.go）是平台侧在管经销商；这里是反过来的一面：
// 经销商登录后看自己——钱包还剩多少，发出去的每个 Key 用了多少、花了多少。
// 他拿这些数去跟自己的客户对账，平台不参与他卖给客户多少钱这件事。
//
// 这条路径挂在 /api/user/self 下，任何登录用户都能访问，是不是经销商的判断在
// service/model 里做，并且必须做：普通客户拿到 403 而不是一张空表，前端也就不必
// 靠藏菜单来当权限用。

// GetSelfAgentLedger 经销商看自己的台账。
func GetSelfAgentLedger(c *gin.Context) {
	ledger, err := model.GetAgentLedger(c.GetInt("id"))
	if err != nil {
		if errors.Is(err, model.ErrAgentProfileNotFound) {
			common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, ledger)
}
