package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 我的客户：经销商看自己名下的客户，给他们改价、给他们发额度。
//
// 三个动作都动的是经销商自己的生意，所以都挂在自助路径下，不需要管理员权限：
// 谁是我的客户，由 users.parent_agent_id 说了算，别人看不见也改不了。
//
// 判断全在 model 层（agent_customers.go），这里只做参数解析、把错误翻成人话，
// 以及在改动之后把那位客户的最新台账行回给界面，省掉一次为了刷新而发起的查询。

// agentCustomerDiscountRequest 给客户定价时带的方案；0 表示撤掉自己定的价。
type agentCustomerDiscountRequest struct {
	PlanId int `json:"plan_id"`
}

// agentCustomerQuotaRequest 发给客户的额度，单位与钱包一致。
type agentCustomerQuotaRequest struct {
	Quota int `json:"quota"`
}

// GetSelfAgentCustomers 经销商看自己名下的客户名单。
func GetSelfAgentCustomers(c *gin.Context) {
	listAgentCustomersFor(c, c.GetInt("id"))
}

// SetSelfAgentCustomerDiscount 给一位下属客户定价（plan_id 传 0 撤掉自己定的价）。
func SetSelfAgentCustomerDiscount(c *gin.Context) {
	setAgentCustomerDiscountFor(c, c.GetInt("id"))
}

// IssueSelfAgentCustomerQuota 给一位下属客户发额度，钱从经销商自己的余额出。
func IssueSelfAgentCustomerQuota(c *gin.Context) {
	issueAgentCustomerQuotaFor(c, c.GetInt("id"))
}

// listAgentCustomersFor 把某位经销商名下的客户按页列出来。
// 不是经销商时没必要去查一遍 empty 表：直接说不是，界面才知道该不该显示这一块。
func listAgentCustomersFor(c *gin.Context, agentId int) {
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
	customers, total, err := model.ListAgentCustomers(agentId, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(customers)
	common.ApiSuccess(c, pageInfo)
}

// setAgentCustomerDiscountFor 给一位下属客户定价，并把改完之后的那一行回给界面。
func setAgentCustomerDiscountFor(c *gin.Context, agentId int) {
	customerId, ok := parseAgentCustomerId(c)
	if !ok {
		return
	}
	var req agentCustomerDiscountRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	if err := model.SetAgentCustomerDiscount(agentId, customerId, req.PlanId); err != nil {
		apiErrorForAgentCustomer(c, err)
		return
	}
	replyAgentCustomer(c, agentId, customerId)
}

// issueAgentCustomerQuotaFor 给一位下属客户发额度，并把加完之后的那一行回给界面。
func issueAgentCustomerQuotaFor(c *gin.Context, agentId int) {
	customerId, ok := parseAgentCustomerId(c)
	if !ok {
		return
	}
	var req agentCustomerQuotaRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	if err := model.IssueQuotaToCustomer(agentId, customerId, req.Quota); err != nil {
		apiErrorForAgentCustomer(c, err)
		return
	}
	replyAgentCustomer(c, agentId, customerId)
}

// replyAgentCustomer 把某一位客户此刻的台账行回给界面。
// 改动之后读一遍真实结果，界面拿到的就是库里此刻的状态，而不是我推测的状态。
func replyAgentCustomer(c *gin.Context, agentId int, customerId int) {
	row, err := model.GetAgentCustomer(agentId, customerId)
	if err != nil {
		apiErrorForAgentCustomer(c, err)
		return
	}
	common.ApiSuccess(c, row)
}

// parseAgentCustomerId 取路径上的客户 ID。
func parseAgentCustomerId(c *gin.Context) (int, bool) {
	customerId, err := strconv.Atoi(c.Param("customerId"))
	if err != nil || customerId <= 0 {
		common.ApiErrorMsg(c, "无效的客户 ID")
		return 0, false
	}
	return customerId, true
}

// apiErrorForAgentCustomer 把「我的客户」这张表上可能出的错翻成人话。
//
// 每个拒绝都有具体说法：不在你名下、平台已单独定价、这个方案不在你的货架上、
// 发额度已关闭、你自己的额度不够、额度得是正数。客户看到的都是能据以行动的那一句。
func apiErrorForAgentCustomer(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrAgentCustomerNotFound):
		common.ApiErrorI18n(c, i18n.MsgAgentCustomerNotFound)
	case errors.Is(err, model.ErrCustomerPricedByPlatform):
		common.ApiErrorI18n(c, i18n.MsgAgentCustomerPricedByPlat)
	case errors.Is(err, model.ErrAgentPlanNotSellable):
		common.ApiErrorI18n(c, i18n.MsgAgentCustomerPlanNotSellable)
	case errors.Is(err, model.ErrAgentQuotaIssuingDisabled):
		common.ApiErrorI18n(c, i18n.MsgAgentCustomerQuotaDisabled)
	case errors.Is(err, model.ErrAgentQuotaNotEnough):
		common.ApiErrorI18n(c, i18n.MsgAgentCustomerQuotaNotEnough)
	case errors.Is(err, model.ErrAgentQuotaInvalid):
		common.ApiErrorI18n(c, i18n.MsgAgentCustomerQuotaInvalid)
	case errors.Is(err, model.ErrAgentProfileNotFound):
		common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
	default:
		common.ApiError(c, err)
	}
}
