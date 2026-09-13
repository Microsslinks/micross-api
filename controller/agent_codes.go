package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 客户号：平台替某位经销商签号，以及经销商自己签号。
//
// 两条路走同一份业务（model 里的 CreateCustomerCodes / ListCustomerCodes / RevokeCustomerCode），
// 差别只在「动谁的号」：平台走路径上的 :id，经销商走会话里的自己。
// 所以这里只有三个薄薄的解析壳，真正的判断都在模型层。

type customerCodeIssueRequest struct {
	PlanId    int    `json:"plan_id"`
	Count     int    `json:"count"`
	MaxUses   int    `json:"max_uses"`
	ExpiredAt int64  `json:"expired_at"`
	Remark    string `json:"remark"`
}

// ---- 平台侧：动的是路径上那个经销商 ----

// GetAgentCustomerCodes 列出这位经销商签出去的所有客户号。
func GetAgentCustomerCodes(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}
	listCustomerCodesFor(c, user.Id)
}

// CreateAgentCustomerCodes 替这位经销商签一批客户号。
func CreateAgentCustomerCodes(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}
	issueCustomerCodesFor(c, user.Id)
}

// RevokeAgentCustomerCode 作废这位经销商名下的一个客户号。
func RevokeAgentCustomerCode(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}
	revokeCustomerCodeFor(c, user.Id)
}

// ---- 经销商自己：动的是会话里的自己 ----

// GetSelfAgentCodes 经销商看自己签出去的号。
func GetSelfAgentCodes(c *gin.Context) {
	listCustomerCodesFor(c, c.GetInt("id"))
}

// CreateSelfAgentCodes 经销商自己签一批号发给他的客户。
func CreateSelfAgentCodes(c *gin.Context) {
	issueCustomerCodesFor(c, c.GetInt("id"))
}

// RevokeSelfAgentCode 经销商作废自己的一个号。
func RevokeSelfAgentCode(c *gin.Context) {
	revokeCustomerCodeFor(c, c.GetInt("id"))
}

// listCustomerCodesFor 把某位经销商的号按页列出来。
// 不是经销商时列出空表也说得过去，但「他到底是不是经销商」这件事不该由前端猜，
// 所以仍然先看身份：不是就直接说不是。
func listCustomerCodesFor(c *gin.Context, agentId int) {
	isAgent, err := model.IsUserAgent(agentId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !isAgent {
		common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
		return
	}

	pageInfo := common.GetPageQuery(c)
	codes, total, err := model.ListCustomerCodes(agentId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(codes)
	common.ApiSuccess(c, pageInfo)
}

// issueCustomerCodesFor 签一批号。
// 入参先在这里校验掉，于是下面只剩三种情况：他不是经销商、号上那个方案不能用、库真出错了。
func issueCustomerCodesFor(c *gin.Context, agentId int) {
	var req customerCodeIssueRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	issue := model.CustomerCodeIssue{
		AgentId:   agentId,
		PlanId:    req.PlanId,
		Count:     req.Count,
		MaxUses:   req.MaxUses,
		ExpiredAt: req.ExpiredAt,
		Remark:    req.Remark,
	}
	if _, err := model.NormalizeCustomerCodeIssue(issue); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	codes, err := model.CreateCustomerCodes(issue)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrAgentProfileNotFound):
			common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
		case errors.Is(err, model.ErrCustomerCodePlanUnavailable):
			common.ApiErrorMsg(c, err.Error())
		default:
			common.ApiError(c, err)
		}
		return
	}
	common.ApiSuccess(c, gin.H{"items": codes})
}

// revokeCustomerCodeFor 作废一个号：只能作废自己（或自己名下那位经销商）签的号。
func revokeCustomerCodeFor(c *gin.Context, agentId int) {
	codeId, err := strconv.Atoi(c.Param("codeId"))
	if err != nil || codeId <= 0 {
		common.ApiErrorMsg(c, "无效的客户号 ID")
		return
	}
	if err := model.RevokeCustomerCode(agentId, codeId); err != nil {
		if errors.Is(err, model.ErrCustomerCodeNotFound) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
