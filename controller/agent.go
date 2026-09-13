package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 经销商身份管理。
//
// 经销商不是一个权限等级，而是叠在用户身上的业务身份（users.subject_type = 'agent'）。
// 他仍然是普通用户：能不能进后台、能点哪些按钮，只由 users.role 决定。
// 所以这里的两个接口只动身份与经营档案，**不碰 role**。

type setAgentRequest struct {
	WholesaleDiscount *string `json:"wholesale_discount"`
	MinDiscount       *string `json:"min_discount"`
	IssueQuotaEnabled *int    `json:"issue_quota_enabled"`
	Remark            *string `json:"remark"`
}

// loadManageableUser 取路径 :id 指向的用户，并确认当前操作者管得了他。
// 出错时响应已经写好，调用方直接 return 即可。
func loadManageableUser(c *gin.Context, idParam string) (*model.User, bool) {
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的用户 ID")
		return nil, false
	}
	user := &model.User{Id: id}
	if err := user.FillUserById(); err != nil {
		common.ApiError(c, err)
		return nil, false
	}
	if !canManageTargetRole(c.GetInt("role"), user.Role) {
		common.ApiErrorI18n(c, i18n.MsgUserNoPermissionSameLevel)
		return nil, false
	}
	return user, true
}

// SetUserAsAgent 把一个用户设为经销商：写身份 + 建经营档案（同一事务）。
//
// 两个折扣由平台手动设定（见 .docs/task-02-business-goals/03-agent-and-commission.md §5 与开放问题 A3）：
// 批发折扣是平台卖给经销商的价格，最低折扣是平台给他的地板价。
func SetUserAsAgent(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}
	if user.SubjectType == model.SubjectTypeAgent {
		common.ApiErrorI18n(c, i18n.MsgUserAlreadyAgent)
		return
	}

	var req setAgentRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	// 没给批发折扣时用「无折扣」，即按官方标价进货，不会算错钱。
	wholesaleDiscount := model.DiscountNone
	if req.WholesaleDiscount != nil {
		wholesaleDiscount = *req.WholesaleDiscount
	}
	normalizedWholesale, err := model.NormalizeDiscount(wholesaleDiscount)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	minDiscount := "0"
	if req.MinDiscount != nil {
		minDiscount = *req.MinDiscount
	}
	normalizedMin, err := model.NormalizeDiscountRatio(minDiscount)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	// 地板价高于进货价没有意义，与折扣方案用同一条铁律。
	if err := model.ValidateDiscountFloor(normalizedWholesale, normalizedMin); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	issueQuotaEnabled := model.CustomerCodeStatusEnabled
	if req.IssueQuotaEnabled != nil {
		issueQuotaEnabled = *req.IssueQuotaEnabled
	}
	remark := ""
	if req.Remark != nil {
		if utf8.RuneCountInString(*req.Remark) > 255 {
			common.ApiErrorMsg(c, "备注不能超过 255 个字符")
			return
		}
		remark = strings.TrimSpace(*req.Remark)
	}

	if err := model.PromoteUserToAgent(user.Id, normalizedWholesale, normalizedMin, issueQuotaEnabled, remark); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.set_agent", map[string]interface{}{
		"username":           user.Username,
		"wholesale_discount": normalizedWholesale,
		"min_discount":       normalizedMin,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"subject_type": model.SubjectTypeAgent,
		},
	})
}

// UnsetUserAsAgent 取消经销商身份：删经营档案 + 改回普通客户。
func UnsetUserAsAgent(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}
	if user.SubjectType != model.SubjectTypeAgent {
		common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
		return
	}
	if err := model.DemoteAgentToUser(user.Id); err != nil {
		if errors.Is(err, model.ErrAgentHasCustomerCodes) {
			common.ApiErrorI18n(c, i18n.MsgUserAgentHasCustomerCodes)
			return
		}
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.unset_agent", map[string]interface{}{
		"username": user.Username,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"subject_type": model.SubjectTypeIndividual,
		},
	})
}
