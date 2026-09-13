package controller

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 经销商经营档案：平台看他现在是什么设置，以及调整这些设置。
//
// 这两个接口只动 agent_profiles 那几列经营参数，**不碰身份、不碰 role、不碰余额**——
// 经销商是叠在用户身上的业务身份而不是权限等级（见 controller/agent.go 开头那段）。

type agentProfileRequest struct {
	MarkupRatio       *string `json:"markup_ratio"`
	MinDiscount       *string `json:"min_discount"`
	IssueQuotaEnabled *int    `json:"issue_quota_enabled"`
	Remark            *string `json:"remark"`
}

// agentProfileData 是把档案摊给前端看的那几个数。
// 加价率取 EffectiveMarkupRatio：早年建的档案这一列为空，界面上要看到真正生效的那一档，
// 而不是一个空框。
func agentProfileData(profile *model.AgentProfile) gin.H {
	return gin.H{
		"user_id":             profile.UserId,
		"markup_ratio":        profile.EffectiveMarkupRatio(),
		"min_discount":        profile.MinDiscount,
		"issue_quota_enabled": profile.IssueQuotaEnabled,
		"remark":              profile.Remark,
	}
}

// GetAgentProfile 读出这位经销商的经营档案，给「经销商设置」回填。
func GetAgentProfile(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}
	profile, err := model.GetAgentProfileByUserId(user.Id)
	if err != nil {
		if errors.Is(err, model.ErrAgentProfileNotFound) {
			common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
			return
		}
		common.ApiError(c, err)
		return
	}
	profile.NormalizeDefaults()
	common.ApiSuccess(c, agentProfileData(profile))
}

// UpdateAgentProfile 改一位经销商的经营档案：加价率 / 最低折扣 / 是否允许发额度 / 备注。
//
// 四个字段都可选，传哪个改哪个。校验放在这里做（口径与设经销商时同一套），
// 模型层只管把规范过的值写进去，出错就是数据库的问题。
func UpdateAgentProfile(c *gin.Context) {
	user, ok := loadManageableUser(c, c.Param("id"))
	if !ok {
		return
	}
	if user.SubjectType != model.SubjectTypeAgent {
		common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
		return
	}

	var req agentProfileRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}

	settings := model.AgentProfileSettings{}
	if req.MarkupRatio != nil {
		normalized, err := model.NormalizeAgentMarkup(*req.MarkupRatio)
		if err != nil {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		settings.MarkupRatio = &normalized
	}
	if req.MinDiscount != nil {
		normalized, err := model.NormalizeAgentMinDiscount(*req.MinDiscount)
		if err != nil {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		settings.MinDiscount = &normalized
	}
	if req.IssueQuotaEnabled != nil {
		mode := *req.IssueQuotaEnabled
		if mode != 0 && mode != 1 {
			common.ApiErrorMsg(c, "发额度开关只能是 0 或 1")
			return
		}
		settings.IssueQuotaEnabled = &mode
	}
	if req.Remark != nil {
		remark := strings.TrimSpace(*req.Remark)
		if utf8.RuneCountInString(remark) > 255 {
			common.ApiErrorMsg(c, "备注不能超过 255 个字符")
			return
		}
		settings.Remark = &remark
	}

	profile, err := model.UpdateAgentProfileSettings(user.Id, settings)
	if err != nil {
		if errors.Is(err, model.ErrAgentProfileNotFound) {
			common.ApiErrorI18n(c, i18n.MsgUserNotAgent)
			return
		}
		common.ApiError(c, err)
		return
	}

	recordManageAuditFor(c, user.Id, "user.update_agent", map[string]interface{}{
		"username":            user.Username,
		"markup_ratio":        profile.EffectiveMarkupRatio(),
		"min_discount":        profile.MinDiscount,
		"issue_quota_enabled": profile.IssueQuotaEnabled,
	})
	common.ApiSuccess(c, agentProfileData(profile))
}
