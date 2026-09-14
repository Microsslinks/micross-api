package controller

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 模型清单（可复用的模型名单）管理。
// 路由挂在 /api/discount/admin/model-lists 下，全部要求管理员权限。
// 这一族接口管的是「填模型清单格子时能一键带入的名单」，不参与任何计价。

type discountModelListRequest struct {
	Name   string   `json:"name"`
	Remark string   `json:"remark"`
	Models []string `json:"models"`
}

// applyDiscountModelListRequest 把请求整份落到清单上并校验。
// 新建与更新共用：这一版接口不做局部更新，名称与名单必须一起给全——
// 否则「没给名单」和「要把名单清空」这两种意思在 JSON 里分不出来。
func applyDiscountModelListRequest(list *model.DiscountModelList, req *discountModelListRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return errors.New("清单名不能为空")
	}
	if utf8.RuneCountInString(name) > model.DiscountModelListNameMaxLength {
		return fmt.Errorf("清单名不能超过 %d 个字符", model.DiscountModelListNameMaxLength)
	}
	remark := strings.TrimSpace(req.Remark)
	if utf8.RuneCountInString(remark) > 255 {
		return errors.New("备注不能超过 255 个字符")
	}
	names, err := model.NormalizeDiscountModelListNames(req.Models)
	if err != nil {
		return err
	}
	list.Name = name
	list.Remark = remark
	return list.SetModelNames(names)
}

func GetDiscountModelLists(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	lists, total, err := model.GetDiscountModelLists(c.Query("keyword"), pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(lists)
	common.ApiSuccess(c, pageInfo)
}

func CreateDiscountModelList(c *gin.Context) {
	var req discountModelListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	list := &model.DiscountModelList{}
	if err := applyDiscountModelListRequest(list, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsDiscountModelListNameDuplicated(0, list.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "已存在同名清单")
		return
	}
	if err := list.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

func UpdateDiscountModelList(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	var req discountModelListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	list, err := model.GetDiscountModelListById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := applyDiscountModelListRequest(list, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsDiscountModelListNameDuplicated(id, list.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "已存在同名清单")
		return
	}
	if err := list.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}

func DeleteDiscountModelList(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	if err := model.DeleteDiscountModelList(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
