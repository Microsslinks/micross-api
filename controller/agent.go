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
	MarkupRatio       *string `json:"markup_ratio"`
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

// SetUserAsAgent 把一个用户设为经销商：写身份 + 建经营档案（同一事务），
// 并定下平台卖给这位经销商的价。
//
// 这个价不逐个模型手填，只定一档毛利：每个模型各自按它最便宜一条线路的成本乘上这档毛利
// （见 model/agent_wholesale.go）。所以请求里只收毛利，界面上给 5% / 10% / 20% 三档、
// 也允许手改，但不得低于 5%——再低平台就白干了。
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

	// 没给毛利时用缺省档 +10%：平台照常赚钱，也不会算错钱。
	markupRatio := model.AgentWholesaleMarkupDefault
	if req.MarkupRatio != nil {
		markupRatio = *req.MarkupRatio
	}
	normalizedMarkup, err := model.NormalizeAgentMarkup(markupRatio)
	if err != nil {
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

	if err := model.PromoteUserToAgentWithMarkup(user.Id, normalizedMarkup, issueQuotaEnabled, remark); err != nil {
		common.ApiError(c, err)
		return
	}
	recordManageAuditFor(c, user.Id, "user.set_agent", map[string]interface{}{
		"username":     user.Username,
		"markup_ratio": normalizedMarkup,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"subject_type": model.SubjectTypeAgent,
		},
	})
}

// GetAgentWholesaleQuote 给出这位经销商当前每个模型的拿货价，也就是他拿去给客户报价的底价表。
//
// 界面在设毛利时会反复拉它试算：数字一改就拉一次，下面那张胶囊清单跟着变。
// 不传 markup_ratio 时按缺省档 +10% 试算，所以还没设过档案的用户也能先看一眼这张表。
func GetAgentWholesaleQuote(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}

	markupRatio := strings.TrimSpace(c.Query("markup_ratio"))
	if markupRatio == "" {
		markupRatio = model.AgentWholesaleMarkupDefault
	}
	normalizedMarkup, err := model.NormalizeAgentMarkup(markupRatio)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}

	items, err := model.ListAgentWholesale(user.Id, normalizedMarkup)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"markup_ratio": normalizedMarkup,
		"items":        items,
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
