package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 折扣解析结果的来源：这一单的折扣是从哪一层取到的。
const (
	DiscountResolvedFromModel          = "model"           // 命中模型级规则（最具体，永远优先）
	DiscountResolvedFromVendor         = "vendor"          // 命中厂商级规则
	DiscountResolvedFromPlanBase       = "plan_base"       // 回落到方案的基础折扣
	DiscountResolvedFromAgentWholesale = "agent_wholesale" // 经销商自己消费，按他的拿货价
	DiscountResolvedFromDefault        = "default"         // 没绑定方案或方案已停用，即官方标价
)

// DiscountResolution 是「某个客户对某个模型实际按几折算」的答案。
// Discount 固定为 6 位小数字符串，与折扣在库里的存取口径保持一致。
type DiscountResolution struct {
	Discount string        // 命中的折扣；没有任何方案生效时是 DiscountNone（"1.000000"）
	Source   string        // DiscountResolvedFrom* 常量之一
	Rule     *DiscountRule // Source 为 model / vendor 时非空，指出是哪条规则给的价
	Plan     *DiscountPlan // 方案未绑定或已停用时为空
	PlanId   int           // 快路径上的方案 id；非 0 表示这个客户绑定过方案
}

// ResolveUserDiscount 算出「这个客户 + 这个模型」该按几折，以及这个折扣从哪来。
// 分两层：先看绑定的折扣方案（模型级规则 → 厂商级规则 → 方案基础折扣 → 官方标价），
// 再让经销商自己消费的拿货价参与比较，取更便宜的那个。
// 计费、试算、保存前校验三处共用这一个函数，折扣口径只能有这一份。
func ResolveUserDiscount(userId int, modelName string) (*DiscountResolution, error) {
	resolution, err := resolvePlanDiscount(userId, modelName)
	if err != nil {
		return nil, err
	}
	return applyAgentWholesaleDiscount(resolution, userId, modelName)
}

// resolvePlanDiscount 只看折扣方案这一层，是 ResolveUserDiscount 的前半段。
// 优先级：模型级规则 → 厂商级规则 → 方案基础折扣 → 官方标价（1.0）。
func resolvePlanDiscount(userId int, modelName string) (*DiscountResolution, error) {
	resolution := &DiscountResolution{
		Discount: DiscountNone,
		Source:   DiscountResolvedFromDefault,
	}
	if userId <= 0 {
		return resolution, nil
	}

	// 第 1 步：读快路径 users.discount_plan_id。它只记「命中了哪条绑定」，
	// 方案本身是否停用要留到第 2 步判，这是 syncUserDiscountPlanId 定下的口径。
	var user User
	err := DB.Select("id", "discount_plan_id").First(&user, userId).Error
	if err != nil {
		// 用户不存在视同没有绑定方案；数据库真出错必须报上去，不能悄悄按 1.0 计费。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resolution, nil
		}
		return nil, err
	}
	if user.DiscountPlanId <= 0 {
		return resolution, nil
	}
	resolution.PlanId = user.DiscountPlanId

	// 第 2 步：读方案并检查是否启用。停用即回落到官方标价。
	plan, err := GetDiscountPlanById(user.DiscountPlanId)
	if err != nil {
		// 快路径指向的方案已经不存在，按未绑定处理，不让计费跟着失败。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return resolution, nil
		}
		return nil, err
	}
	if plan.Status != DiscountStatusEnabled {
		return resolution, nil
	}
	resolution.Plan = plan

	// 第 3 步：取该方案下的规则。GetDiscountRulesByPlanId 已按 priority DESC, id ASC 排好，
	// 所以同一层级里第一条命中的就是该用的那条。
	rules, err := GetDiscountRulesByPlanId(plan.Id)
	if err != nil {
		return nil, err
	}

	// 第 4 步：模型级规则。
	if rule := pickDiscountRule(rules, DiscountScopeModel, modelName); rule != nil {
		resolution.Source = DiscountResolvedFromModel
		resolution.Rule = rule
		resolution.Discount = formatResolvedDiscount(rule.Discount)
		return resolution, nil
	}

	// 第 5 步：厂商级规则。厂商名顺着 模型名 → 模型目录 → 厂商 取。
	vendorName, err := resolveVendorName(modelName)
	if err != nil {
		return nil, err
	}
	if rule := pickDiscountRule(rules, DiscountScopeVendor, vendorName); rule != nil {
		resolution.Source = DiscountResolvedFromVendor
		resolution.Rule = rule
		resolution.Discount = formatResolvedDiscount(rule.Discount)
		return resolution, nil
	}

	// 第 6 步：回落到方案基础折扣。
	resolution.Source = DiscountResolvedFromPlanBase
	resolution.Discount = formatResolvedDiscount(plan.BaseDiscount)
	return resolution, nil
}

// applyAgentWholesaleDiscount 让「经销商自己消费的拿货价」参与比较。
//
// 没有生效方案时直接按拿货价；已经有方案价时取更便宜的那个——两条价都在，
// 让经销商吃亏没有道理。不是经销商时这一次判断只花一条索引查询。
// 拿货价算不出来（没线路、没录进货折扣、加价后不低于标价）时原样返回，
// 交给上层按原价处理，绝不用一个没有依据的数字替换现有结论。
func applyAgentWholesaleDiscount(resolution *DiscountResolution, userId int, modelName string) (*DiscountResolution, error) {
	wholesale, err := ResolveAgentWholesale(userId, modelName)
	if err != nil {
		// 拿货价是叠在方案折扣之上的一层，它自己出错不该把方案折扣一起拖下水：
		// 记一笔日志，这一单照方案折扣算，跟没做这个功能时一样。
		common.SysError(fmt.Sprintf("解析经销商拿货价失败，客户 %d 本单按方案折扣计费：%v", userId, err))
		return resolution, nil
	}
	return applyWholesaleToResolution(resolution, wholesale), nil
}

// applyWholesaleToResolution 把算好的拿货价合进解析结果（纯计算，不查库）。
// 单模型与批量两条路共用这一段，合入规则只有一份。
func applyWholesaleToResolution(resolution *DiscountResolution, wholesale *AgentWholesale) *DiscountResolution {
	if resolution == nil {
		return nil
	}
	if wholesale == nil {
		return resolution
	}
	if resolution.Source == DiscountResolvedFromDefault {
		resolution.Discount = wholesale.Discount
		resolution.Source = DiscountResolvedFromAgentWholesale
		resolution.Rule = nil
		return resolution
	}
	planValue, err := decimal.NewFromString(strings.TrimSpace(resolution.Discount))
	if err != nil {
		// 方案折扣解析不出来属于数据异常：不猜，保持原样让上层按原价兜底。
		return resolution
	}
	wholesaleValue, err := decimal.NewFromString(wholesale.Discount)
	if err != nil {
		return resolution
	}
	if wholesaleValue.LessThan(planValue) {
		resolution.Discount = wholesale.Discount
		resolution.Source = DiscountResolvedFromAgentWholesale
		resolution.Rule = nil
	}
	return resolution
}

// pickDiscountRule 在规则里找第一条命中给定取值的规则。
// rules 已按 priority DESC, id ASC 排序，所以「多条命中取 priority 最大、priority 相同取 id 最小」
// 由这个顺序直接保证，这里不再排序。
func pickDiscountRule(rules []*DiscountRule, scopeType string, scopeValue string) *DiscountRule {
	if strings.TrimSpace(scopeValue) == "" {
		return nil
	}
	for _, rule := range rules {
		if rule.Status != DiscountStatusEnabled || rule.ScopeType != scopeType {
			continue
		}
		if matchDiscountScope(rule.ScopeValue, scopeValue) {
			return rule
		}
	}
	return nil
}

// matchDiscountScope 判断规则的 scope_value 是否命中给定的模型名或厂商名。
// 只认写在结尾的 `*`（如 `claude-*`）：不做正则，也不支持中间通配；
// 不带 `*` 时按完整字符串相等比较。`a*b*` 这种含多个 `*` 的写法不按通配处理。
func matchDiscountScope(scopeValue string, name string) bool {
	scopeValue = strings.TrimSpace(scopeValue)
	name = strings.TrimSpace(name)
	if scopeValue == "" || name == "" {
		return false
	}
	prefix, wildcard := strings.CutSuffix(scopeValue, "*")
	if !wildcard {
		return scopeValue == name
	}
	if strings.Contains(prefix, "*") {
		return false
	}
	return strings.HasPrefix(name, prefix)
}

// resolveVendorName 顺着 模型名 → 模型目录 → 厂商 取厂商名。
// 目录里没有这个模型、或它没有挂厂商时返回空串，调用方据此跳过厂商级匹配。
func resolveVendorName(modelName string) (string, error) {
	meta, err := GetModelByName(modelName)
	if err != nil {
		return "", err
	}
	if meta == nil || meta.VendorID <= 0 {
		return "", nil
	}
	vendor, err := GetVendorByID(meta.VendorID)
	if err != nil {
		// 厂商被删了但模型还挂着它的 id：视同没有厂商，跳过厂商级匹配。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(vendor.Name), nil
}

// GetModelByName 按模型名取模型目录记录，软删的不算。
// 查不到返回 (nil, nil)：目录里没有这个模型是正常情况，不是错误。
// 这里用 Find 而不是 First——解析折扣时每次都会问一次，用 First 会给"目录里还没建这个模型"
// 这种情况刷一屏 record not found 的错误日志，而那并不是错误。
func GetModelByName(name string) (*Model, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, nil
	}
	var metas []Model
	if err := DB.Where("model_name = ?", name).Limit(1).Find(&metas).Error; err != nil {
		return nil, err
	}
	if len(metas) == 0 {
		return nil, nil
	}
	return &metas[0], nil
}

// formatResolvedDiscount 把库里读到的折扣统一成固定 6 位小数字符串。
// 折扣列在 SQLite 上按 NUMERIC 亲和性存取，读回来可能是 "0.9"，对外口径必须一致。
// 解析不出来时回落到无折扣——折扣缺省只能落在「不打折」，绝不能变成 0（0 等于免费）。
func formatResolvedDiscount(raw string) string {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return DiscountNone
	}
	return value.StringFixed(6)
}
