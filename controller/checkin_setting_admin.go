package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
)

// 签到设置的管理员专用接口（AdminAuth）。
//
// 业务管理定位是「管理员即可操作」，但签到配置存在系统选项里，
// 原生 /api/option/ 整组是 RootAuth，普通管理员读写都会 403。
// 这里只放行 checkin_setting 三个白名单键，绕开通用选项接口，
// 让普通管理员也能配置签到，同时不暴露选项库里的敏感配置。

const (
	checkinOptionKeyEnabled  = "checkin_setting.enabled"
	checkinOptionKeyMinQuota = "checkin_setting.min_quota"
	checkinOptionKeyMaxQuota = "checkin_setting.max_quota"
)

type checkinSettingRequest struct {
	Enabled  bool `json:"enabled"`
	MinQuota int  `json:"min_quota"`
	MaxQuota int  `json:"max_quota"`
}

// GetAdminCheckinSetting 返回签到配置（管理员权限）
func GetAdminCheckinSetting(c *gin.Context) {
	setting := operation_setting.GetCheckinSetting()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":   setting.Enabled,
			"min_quota": setting.MinQuota,
			"max_quota": setting.MaxQuota,
		},
	})
}

// UpdateAdminCheckinSetting 更新签到配置（管理员权限，仅白名单键）
func UpdateAdminCheckinSetting(c *gin.Context) {
	var request checkinSettingRequest
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		common.ApiErrorMsg(c, "无效的参数")
		return
	}
	if request.MinQuota < 0 || request.MaxQuota < 0 {
		common.ApiErrorMsg(c, "签到额度不能为负数")
		return
	}
	if request.MinQuota > request.MaxQuota {
		common.ApiErrorMsg(c, "签到最小额度不能大于最大额度")
		return
	}

	values := map[string]string{
		checkinOptionKeyEnabled:  strconv.FormatBool(request.Enabled),
		checkinOptionKeyMinQuota: strconv.Itoa(request.MinQuota),
		checkinOptionKeyMaxQuota: strconv.Itoa(request.MaxQuota),
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		common.ApiError(c, err)
		return
	}

	// 与通用选项接口同口径：审计只记键名，不记配置值。
	recordManageAudit(c, "checkin_setting.update", gin.H{
		"keys": []string{
			checkinOptionKeyEnabled,
			checkinOptionKeyMinQuota,
			checkinOptionKeyMaxQuota,
		},
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":   request.Enabled,
			"min_quota": request.MinQuota,
			"max_quota": request.MaxQuota,
		},
	})
}