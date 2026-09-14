package controller

import (
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 折扣方案管理（G0 影子阶段：只做配置读写，不参与计费）。
// 路由挂在 /api/discount/admin 下，全部要求管理员权限。

// parseDiscountIdParam 读取路径里的 :id，非法值直接回错。
func parseDiscountIdParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "无效的 ID")
		return 0, false
	}
	return id, true
}

// ---- 折扣方案 ----

type discountPlanRequest struct {
	Name            *string `json:"name"`
	OwnerType       *string `json:"owner_type"`
	OwnerId         *int    `json:"owner_id"`
	BaseDiscount    *string `json:"base_discount"`
	MinDiscount     *string `json:"min_discount"`
	BillingMode     *string `json:"billing_mode"`
	CommissionRatio *string `json:"commission_ratio"`
	Status          *int    `json:"status"`
	Remark          *string `json:"remark"`
}

// applyDiscountPlanRequest 把请求里出现过的字段覆盖到 plan 上并校验。
// 新建与更新共用：只覆盖请求里给了的字段，没给的保持 plan 上的原值。
func applyDiscountPlanRequest(plan *model.DiscountPlan, req *discountPlanRequest) error {
	if req.Name != nil {
		plan.Name = strings.TrimSpace(*req.Name)
	}
	if plan.Name == "" {
		return errors.New("方案名不能为空")
	}
	if utf8.RuneCountInString(plan.Name) > 64 {
		return errors.New("方案名不能超过 64 个字符")
	}

	if req.OwnerType != nil {
		plan.OwnerType = strings.TrimSpace(*req.OwnerType)
	}
	if plan.OwnerType == "" {
		plan.OwnerType = model.DiscountOwnerPlatform
	}
	if plan.OwnerType != model.DiscountOwnerPlatform && plan.OwnerType != model.DiscountOwnerAgent {
		return errors.New("归属只能是 platform 或 agent")
	}
	if req.OwnerId != nil {
		plan.OwnerId = *req.OwnerId
	}
	if plan.OwnerType == model.DiscountOwnerPlatform {
		plan.OwnerId = 0
	} else if plan.OwnerId <= 0 {
		return errors.New("代理商方案必须指定 owner_id")
	}

	if req.BillingMode != nil {
		plan.BillingMode = strings.TrimSpace(*req.BillingMode)
	}
	if plan.BillingMode == "" {
		plan.BillingMode = model.DiscountBillingUsage
	}
	switch plan.BillingMode {
	case model.DiscountBillingUsage, model.DiscountBillingSubscription, model.DiscountBillingFree:
	default:
		return errors.New("计费模式只能是 usage、subscription 或 free")
	}

	if req.Remark != nil {
		plan.Remark = strings.TrimSpace(*req.Remark)
	}
	if req.Status != nil {
		plan.Status = *req.Status
	}
	if plan.Status != model.DiscountStatusEnabled && plan.Status != model.DiscountStatusDisabled {
		return errors.New("状态只能是 0（停用）或 1（启用）")
	}

	baseDiscount := plan.BaseDiscount
	if req.BaseDiscount != nil {
		baseDiscount = *req.BaseDiscount
	}
	minDiscount := plan.MinDiscount
	if req.MinDiscount != nil {
		minDiscount = *req.MinDiscount
	}
	commissionRatio := plan.CommissionRatio
	if req.CommissionRatio != nil {
		commissionRatio = *req.CommissionRatio
	}
	// 折扣列刻意没写列级默认值（SQLite 会因此反复重建表），缺省口径在这里补：缺省即无折扣
	if strings.TrimSpace(baseDiscount) == "" {
		baseDiscount = model.DiscountNone
	}
	if strings.TrimSpace(minDiscount) == "" {
		minDiscount = "0"
	}
	if strings.TrimSpace(commissionRatio) == "" {
		commissionRatio = "0"
	}
	normalizedBase, err := model.NormalizeDiscount(baseDiscount)
	if err != nil {
		return err
	}
	normalizedMin, err := model.NormalizeDiscountRatio(minDiscount)
	if err != nil {
		return err
	}
	normalizedCommission, err := model.NormalizeDiscountRatio(commissionRatio)
	if err != nil {
		return err
	}
	if err := model.ValidateDiscountFloor(normalizedBase, normalizedMin); err != nil {
		return err
	}
	plan.BaseDiscount, plan.MinDiscount, plan.CommissionRatio = normalizedBase, normalizedMin, normalizedCommission
	return nil
}

type discountPlanDetailDTO struct {
	Plan  *model.DiscountPlan   `json:"plan"`
	Rules []*model.DiscountRule `json:"rules"`
}

func GetDiscountPlans(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	status := -1
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || (parsed != model.DiscountStatusEnabled && parsed != model.DiscountStatusDisabled) {
			common.ApiErrorMsg(c, "状态只能是 0 或 1")
			return
		}
		status = parsed
	}
	plans, total, err := model.GetDiscountPlans(c.Query("keyword"), c.Query("owner_type"), status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(plans)
	common.ApiSuccess(c, pageInfo)
}

func GetDiscountPlan(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	plan, err := model.GetDiscountPlanById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	rules, err := model.GetDiscountRulesByPlanId(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, discountPlanDetailDTO{Plan: plan, Rules: rules})
}

func CreateDiscountPlan(c *gin.Context) {
	var req discountPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan := &model.DiscountPlan{Status: model.DiscountStatusEnabled}
	if err := applyDiscountPlanRequest(plan, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsDiscountPlanNameDuplicated(0, plan.OwnerType, plan.OwnerId, plan.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "同一归属下已存在同名方案")
		return
	}
	if err := plan.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func UpdateDiscountPlan(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	var req discountPlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	plan, err := model.GetDiscountPlanById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := applyDiscountPlanRequest(plan, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsDiscountPlanNameDuplicated(id, plan.OwnerType, plan.OwnerId, plan.Name)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "同一归属下已存在同名方案")
		return
	}
	if err := plan.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

type discountPlanStatusRequest struct {
	Status int `json:"status"`
}

func UpdateDiscountPlanStatus(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	var req discountPlanStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if req.Status != model.DiscountStatusEnabled && req.Status != model.DiscountStatusDisabled {
		common.ApiErrorMsg(c, "状态只能是 0（停用）或 1（启用）")
		return
	}
	plan, err := model.GetDiscountPlanById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	plan.Status = req.Status
	if err := plan.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, plan)
}

func DeleteDiscountPlan(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	if err := model.DeleteDiscountPlan(id); err != nil {
		if errors.Is(err, model.ErrDiscountPlanInUse) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- 折扣规则 ----

type discountRuleRequest struct {
	ScopeType  *string `json:"scope_type"`
	ScopeValue *string `json:"scope_value"`
	// Discount 字段已废弃——规则改为纯范围标记，命中时按方案基础折扣（plan.BaseDiscount）出价。
	// 保留 JSON 字段是为了不破坏老客户端：仍会被解析进来，但 applyDiscountRuleRequest 不再写入、
	// 不再校验，存库时也不会被任何路径覆盖。前端表单已停止发送该字段。
	Discount *string `json:"discount"`
	Priority *int    `json:"priority"`
	Status   *int    `json:"status"`
}

func applyDiscountRuleRequest(rule *model.DiscountRule, req *discountRuleRequest) error {
	if req.ScopeType != nil {
		rule.ScopeType = strings.TrimSpace(*req.ScopeType)
	}
	switch rule.ScopeType {
	case model.DiscountScopeModel, model.DiscountScopeVendor:
	default:
		return errors.New("范围类型只能是 model 或 vendor")
	}
	if req.ScopeValue != nil {
		rule.ScopeValue = strings.TrimSpace(*req.ScopeValue)
	}
	if rule.ScopeValue == "" {
		return errors.New("范围取值不能为空")
	}
	if utf8.RuneCountInString(rule.ScopeValue) > 128 {
		return errors.New("范围取值不能超过 128 个字符")
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.Status != nil {
		rule.Status = *req.Status
	}
	if rule.Status != model.DiscountStatusEnabled && rule.Status != model.DiscountStatusDisabled {
		return errors.New("状态只能是 0（停用）或 1（启用）")
	}
	// discount 字段已废弃：以前在这里 normalize + 写入；现在规则不再携带折扣，
	// 命中时按 plan.BaseDiscount 出价（见 discount_resolve.go / discount_multi_resolve.go）。
	// rule.Discount 仍保留 DB 列（model.DiscountRule 注释里有写），但本函数不再写它。
	return nil
}

func GetDiscountRules(c *gin.Context) {
	planId, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	if _, err := model.GetDiscountPlanById(planId); err != nil {
		common.ApiError(c, err)
		return
	}
	rules, err := model.GetDiscountRulesByPlanId(planId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rules)
}

func CreateDiscountRule(c *gin.Context) {
	planId, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	var req discountRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if _, err := model.GetDiscountPlanById(planId); err != nil {
		common.ApiError(c, err)
		return
	}
	rule := &model.DiscountRule{PlanId: planId, Status: model.DiscountStatusEnabled}
	if err := applyDiscountRuleRequest(rule, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsDiscountRuleScopeDuplicated(planId, rule.ScopeType, rule.ScopeValue, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "该方案下已存在相同范围的规则")
		return
	}
	if err := rule.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rule)
}

func UpdateDiscountRule(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	var req discountRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	rule, err := model.GetDiscountRuleById(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := applyDiscountRuleRequest(rule, &req); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	dup, err := model.IsDiscountRuleScopeDuplicated(rule.PlanId, rule.ScopeType, rule.ScopeValue, rule.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if dup {
		common.ApiErrorMsg(c, "该方案下已存在相同范围的规则")
		return
	}
	if err := rule.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, rule)
}

func DeleteDiscountRule(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	if err := model.DeleteDiscountRule(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// ---- 客户绑定 ----

type discountBindingRequest struct {
	SubjectType   string `json:"subject_type"`
	SubjectId     int    `json:"subject_id"`
	PlanId        int    `json:"plan_id"`
	EffectiveFrom int64  `json:"effective_from"`
	EffectiveTo   int64  `json:"effective_to"`
	Source        string `json:"source"`
}

func GetDiscountBindings(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	subjectId, _ := strconv.Atoi(c.Query("subject_id"))
	planId, _ := strconv.Atoi(c.Query("plan_id"))
	// 不传 status 就是不过滤（老调用方的口径不变）；传 1 只看此刻生效的绑定。
	status, _ := strconv.Atoi(c.Query("status"))
	bindings, total, err := model.GetDiscountBindings(c.Query("subject_type"), subjectId, planId, status, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(bindings)
	common.ApiSuccess(c, pageInfo)
}

func CreateDiscountBinding(c *gin.Context) {
	var req discountBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	req.SubjectType = strings.TrimSpace(req.SubjectType)
	if req.SubjectType != model.DiscountSubjectUser && req.SubjectType != model.DiscountSubjectAgent {
		common.ApiErrorMsg(c, "绑定主体只能是 user 或 agent")
		return
	}
	if req.SubjectId <= 0 {
		common.ApiErrorMsg(c, "缺少绑定主体 ID")
		return
	}
	if req.PlanId <= 0 {
		common.ApiErrorMsg(c, "缺少方案 ID")
		return
	}
	// P1 阶段代理也是用户（users.id），两种主体都要能在 users 表里找到
	if _, err := model.GetUserById(req.SubjectId, false); err != nil {
		common.ApiErrorMsg(c, "绑定主体不存在")
		return
	}
	if _, err := model.GetDiscountPlanById(req.PlanId); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.EffectiveFrom < 0 || req.EffectiveTo < 0 {
		common.ApiErrorMsg(c, "生效时间不能为负数")
		return
	}
	if req.EffectiveFrom > 0 && req.EffectiveTo > 0 && req.EffectiveTo <= req.EffectiveFrom {
		common.ApiErrorMsg(c, "失效时间必须晚于生效时间")
		return
	}
	source := strings.TrimSpace(req.Source)
	if source == "" {
		source = model.DiscountSourceManual
	}
	switch source {
	case model.DiscountSourceManual, model.DiscountSourceSubscription,
		model.DiscountSourceCustomerCode, model.DiscountSourceMigration:
	default:
		common.ApiErrorMsg(c, "来源只能是 manual、subscription、customer_code 或 migration")
		return
	}
	binding := &model.DiscountBinding{
		SubjectType:   req.SubjectType,
		SubjectId:     req.SubjectId,
		PlanId:        req.PlanId,
		EffectiveFrom: req.EffectiveFrom,
		EffectiveTo:   req.EffectiveTo,
		Source:        source,
		Status:        model.DiscountStatusEnabled,
	}
	if err := model.BindDiscountPlan(binding); err != nil {
		if errors.Is(err, model.ErrDiscountBindingExists) {
			common.ApiErrorMsg(c, err.Error())
			return
		}
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}

func DeleteDiscountBinding(c *gin.Context) {
	id, ok := parseDiscountIdParam(c)
	if !ok {
		return
	}
	if _, err := model.GetDiscountBindingById(id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.UnbindDiscountPlan(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}
