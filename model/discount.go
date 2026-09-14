package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	// 折扣方案归属：平台自营方案，或代理商自有方案
	DiscountOwnerPlatform = "platform"
	DiscountOwnerAgent    = "agent"

	// 折扣规则作用范围：具体模型（支持 claude-* 通配），或整个厂商
	DiscountScopeModel  = "model"
	DiscountScopeVendor = "vendor"

	// 折扣绑定主体
	DiscountSubjectUser  = "user"
	DiscountSubjectAgent = "agent"

	// 计费模式：按用量 / 订阅 / 免费
	DiscountBillingUsage        = "usage"
	DiscountBillingSubscription = "subscription"
	DiscountBillingFree         = "free"

	// 绑定来源；同一主体存在多条绑定时按
	// manual > agent > subscription > customer_code > migration 取用
	DiscountSourceManual       = "manual"
	DiscountSourceAgent        = "agent"
	DiscountSourceSubscription = "subscription"
	DiscountSourceCustomerCode = "customer_code"
	DiscountSourceMigration    = "migration"

	DiscountStatusDisabled = 0
	DiscountStatusEnabled  = 1

	// 无折扣（即官方标价）
	DiscountNone = "1.000000"
)

// DiscountPlan 是给客户计价的折扣方案：客户实付 = 官方标价 × 命中折扣。
// 折扣一律以 decimal(10,6) 字符串存取，避免 float 精度漂移产生的计费尾差。
type DiscountPlan struct {
	Id              int    `json:"id"`
	Name            string `json:"name" gorm:"type:varchar(64);not null;uniqueIndex:uk_plan_owner_name,priority:3"`
	OwnerType       string `json:"owner_type" gorm:"type:varchar(16);not null;default:'platform';uniqueIndex:uk_plan_owner_name,priority:1"`
	OwnerId         int    `json:"owner_id" gorm:"type:int;not null;default:0;uniqueIndex:uk_plan_owner_name,priority:2"`
	BaseDiscount    string `json:"base_discount" gorm:"type:decimal(10,6);not null"`
	MinDiscount     string `json:"min_discount" gorm:"type:decimal(10,6);not null"`
	BillingMode     string `json:"billing_mode" gorm:"type:varchar(16);not null;default:'usage'"`
	CommissionRatio string `json:"commission_ratio" gorm:"type:decimal(10,6);not null"`
	Status          int    `json:"status" gorm:"type:int;not null;default:1;index:idx_plan_status"`
	Remark          string `json:"remark" gorm:"type:varchar(255);default:''"`
	CreatedAt       int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt       int64  `json:"updated_at" gorm:"bigint"`
}

func (DiscountPlan) TableName() string {
	return "discount_plans"
}

func (p *DiscountPlan) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	p.CreatedAt = now
	p.UpdatedAt = now
	return nil
}

func (p *DiscountPlan) BeforeUpdate(tx *gorm.DB) error {
	p.UpdatedAt = common.GetTimestamp()
	return nil
}

// NormalizeDefaults 补全折扣的零值口径。列定义里刻意不写 DEFAULT：
// SQLite 对 decimal 列默认值的判定不稳定，写在列上会导致每次启动重建整张表。
func (p *DiscountPlan) NormalizeDefaults() {
	if p.BaseDiscount == "" {
		p.BaseDiscount = DiscountNone
	}
	if p.MinDiscount == "" {
		p.MinDiscount = "0"
	}
	if p.CommissionRatio == "" {
		p.CommissionRatio = "0"
	}
}

// DiscountRule 是方案内的模型级 / 厂商级覆盖，命中时优先于方案基础折扣。
type DiscountRule struct {
	Id int `json:"id"`
	// plan_id 必须进 uk_rule_plan_scope：这个索引的本意是「同一方案内 (范围类型, 范围取值) 唯一」，
	// IsDiscountRuleScopeDuplicated 一直按这个口径查询。最初建表标签漏了 plan_id，
	// 建出来的是全表唯一——任何两个方案不能有同名规则，多方案定价（task-06）直接被卡死。
	// 存量库由 fixDiscountRuleUniqueIndex 在启动时把旧索引换成这里的定义。
	PlanId     int    `json:"plan_id" gorm:"type:int;not null;index:idx_rule_plan;uniqueIndex:uk_rule_plan_scope,priority:1"`
	ScopeType  string `json:"scope_type" gorm:"type:varchar(16);not null;uniqueIndex:uk_rule_plan_scope,priority:2"`
	ScopeValue string `json:"scope_value" gorm:"type:varchar(128);not null;uniqueIndex:uk_rule_plan_scope,priority:3"`
	Discount   string `json:"discount" gorm:"type:decimal(10,6);not null"`
	Priority   int    `json:"priority" gorm:"type:int;not null;default:0"`
	Status     int    `json:"status" gorm:"type:int;not null;default:1"`
	CreatedAt  int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt  int64  `json:"updated_at" gorm:"bigint"`
}

func (DiscountRule) TableName() string {
	return "discount_rules"
}

func (r *DiscountRule) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	r.CreatedAt = now
	r.UpdatedAt = now
	return nil
}

func (r *DiscountRule) BeforeUpdate(tx *gorm.DB) error {
	r.UpdatedAt = common.GetTimestamp()
	return nil
}

func (r *DiscountRule) NormalizeDefaults() {
	if r.Discount == "" {
		r.Discount = "0"
	}
}

var (
	// ErrDiscountPlanInUse 方案还被绑定引用时不允许删除
	ErrDiscountPlanInUse = errors.New("该方案已被客户绑定，请先解绑后再删除")
	// ErrDiscountBindingExists 同一主体对同一方案不允许重复绑定
	ErrDiscountBindingExists = errors.New("该主体已绑定此方案，请勿重复绑定")
)

// ---- 折扣口径校验 ----

// NormalizeDiscount 校验折扣并规范成固定 6 位小数，让三库存取口径一致。
// 折扣是计费乘数：必须大于 0（0 等于免费，绝不允许），且不超过 1（不允许加价）。
func NormalizeDiscount(raw string) (string, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("折扣必须是数字")
	}
	if value.LessThanOrEqual(decimal.Zero) {
		return "", errors.New("折扣必须大于 0")
	}
	if value.GreaterThan(decimal.NewFromInt(1)) {
		return "", errors.New("折扣不能超过 1")
	}
	return value.StringFixed(6), nil
}

// NormalizeDiscountRatio 用于 min_discount、commission_ratio 这类比例：允许 0，上限 1。
func NormalizeDiscountRatio(raw string) (string, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("比例必须是数字")
	}
	if value.LessThan(decimal.Zero) {
		return "", errors.New("比例不能为负数")
	}
	if value.GreaterThan(decimal.NewFromInt(1)) {
		return "", errors.New("比例不能超过 1")
	}
	return value.StringFixed(6), nil
}

// ValidateDiscountFloor 校验最低保护折扣不高于基础折扣：底价高于成交价没有意义。
func ValidateDiscountFloor(baseDiscount string, minDiscount string) error {
	base, err := decimal.NewFromString(strings.TrimSpace(baseDiscount))
	if err != nil {
		return err
	}
	floor, err := decimal.NewFromString(strings.TrimSpace(minDiscount))
	if err != nil {
		return err
	}
	if floor.GreaterThan(base) {
		return errors.New("最低折扣不能高于基础折扣")
	}
	return nil
}

// ---- 折扣方案 ----

// GetDiscountPlans 分页查询方案，status 传 -1 表示不按状态过滤。
func GetDiscountPlans(keyword string, ownerType string, status int, offset int, limit int) ([]*DiscountPlan, int64, error) {
	db := DB.Model(&DiscountPlan{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR remark LIKE ?", like, like)
	}
	if ownerType = strings.TrimSpace(ownerType); ownerType != "" {
		db = db.Where("owner_type = ?", ownerType)
	}
	if status >= 0 {
		db = db.Where("status = ?", status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	plans := make([]*DiscountPlan, 0)
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&plans).Error; err != nil {
		return nil, 0, err
	}
	for _, plan := range plans {
		plan.NormalizeDefaults()
	}
	return plans, total, nil
}

func GetDiscountPlanById(id int) (*DiscountPlan, error) {
	var plan DiscountPlan
	if err := DB.First(&plan, id).Error; err != nil {
		return nil, err
	}
	plan.NormalizeDefaults()
	return &plan, nil
}

// IsDiscountPlanNameDuplicated 同一归属下的方案名唯一（对应 uk_plan_owner_name）。
func IsDiscountPlanNameDuplicated(excludeId int, ownerType string, ownerId int, name string) (bool, error) {
	var count int64
	err := DB.Model(&DiscountPlan{}).
		Where("owner_type = ? AND owner_id = ? AND name = ? AND id <> ?", ownerType, ownerId, name, excludeId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p *DiscountPlan) Insert() error {
	p.NormalizeDefaults()
	return DB.Create(p).Error
}

// Update 整体更新方案。这里显式列出列名：status 取 0（停用）时不能被 GORM 的零值规则跳过。
func (p *DiscountPlan) Update() error {
	p.NormalizeDefaults()
	return DB.Model(&DiscountPlan{}).Where("id = ?", p.Id).
		Select("name", "owner_type", "owner_id", "base_discount", "min_discount", "billing_mode", "commission_ratio", "status", "remark", "updated_at").
		Updates(p).Error
}

// DeleteDiscountPlan 删除方案：还有生效中的绑定时拒绝（已解绑的历史记录不阻塞），规则随方案一起删。
func DeleteDiscountPlan(id int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var bindings int64
		if err := tx.Model(&DiscountBinding{}).
			Where("plan_id = ? AND status = ?", id, DiscountStatusEnabled).
			Count(&bindings).Error; err != nil {
			return err
		}
		if bindings > 0 {
			return ErrDiscountPlanInUse
		}
		if err := tx.Where("plan_id = ?", id).Delete(&DiscountRule{}).Error; err != nil {
			return err
		}
		return tx.Delete(&DiscountPlan{}, id).Error
	})
}

// ---- 折扣规则 ----

func GetDiscountRulesByPlanId(planId int) ([]*DiscountRule, error) {
	rules := make([]*DiscountRule, 0)
	if err := DB.Where("plan_id = ?", planId).Order("priority DESC, id ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func GetDiscountRuleById(id int) (*DiscountRule, error) {
	var rule DiscountRule
	if err := DB.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// IsDiscountRuleScopeDuplicated 同一方案内的 (范围类型, 范围取值) 唯一（对应 uk_rule_plan_scope）。
func IsDiscountRuleScopeDuplicated(planId int, scopeType string, scopeValue string, excludeId int) (bool, error) {
	var count int64
	err := DB.Model(&DiscountRule{}).
		Where("plan_id = ? AND scope_type = ? AND scope_value = ? AND id <> ?", planId, scopeType, scopeValue, excludeId).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *DiscountRule) Insert() error {
	r.NormalizeDefaults()
	return DB.Create(r).Error
}

// Update 更新规则。规则改挂到方案下面没有意义，故 plan_id 不在可更新列里。
func (r *DiscountRule) Update() error {
	r.NormalizeDefaults()
	return DB.Model(&DiscountRule{}).Where("id = ?", r.Id).
		Select("scope_type", "scope_value", "discount", "priority", "status", "updated_at").
		Updates(r).Error
}

func DeleteDiscountRule(id int) error {
	return DB.Delete(&DiscountRule{}, id).Error
}

// ---- 客户绑定 ----

func GetDiscountBindings(subjectType string, subjectId int, planId int, offset int, limit int) ([]*DiscountBinding, int64, error) {
	db := DB.Model(&DiscountBinding{})
	if subjectType = strings.TrimSpace(subjectType); subjectType != "" {
		db = db.Where("subject_type = ?", subjectType)
	}
	if subjectId > 0 {
		db = db.Where("subject_id = ?", subjectId)
	}
	if planId > 0 {
		db = db.Where("plan_id = ?", planId)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	bindings := make([]*DiscountBinding, 0)
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&bindings).Error; err != nil {
		return nil, 0, err
	}
	return bindings, total, nil
}

func GetDiscountBindingById(id int) (*DiscountBinding, error) {
	var binding DiscountBinding
	if err := DB.First(&binding, id).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

// BindDiscountPlan 写入绑定事实，并在同一事务里重算 users.discount_plan_id 快路径。
// 两者必须同处写入，否则会出现"绑定生效了但计费读不到"的不一致。
func BindDiscountPlan(binding *DiscountBinding) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var existed int64
		if err := tx.Model(&DiscountBinding{}).
			Where("subject_type = ? AND subject_id = ? AND plan_id = ? AND status = ?",
				binding.SubjectType, binding.SubjectId, binding.PlanId, DiscountStatusEnabled).
			Count(&existed).Error; err != nil {
			return err
		}
		if existed > 0 {
			return ErrDiscountBindingExists
		}
		if err := tx.Create(binding).Error; err != nil {
			return err
		}
		return syncUserDiscountPlanId(tx, binding.SubjectId)
	})
}

// UnbindDiscountPlan 解绑：停用绑定（保留事实记录可审计）并重算快路径。
func UnbindDiscountPlan(id int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var binding DiscountBinding
		if err := tx.First(&binding, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&DiscountBinding{}).Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":     DiscountStatusDisabled,
				"updated_at": common.GetTimestamp(),
			}).Error; err != nil {
			return err
		}
		return syncUserDiscountPlanId(tx, binding.SubjectId)
	})
}

// syncUserDiscountPlanId 重算用户的 discount_plan_id 快路径：0 表示当前没有生效方案。
// 只写这一列，不碰 users 的其他字段与 updated_at。
// 注意这里不校验方案本身是否停用——方案状态由解析折扣时判断，快路径只记"命中了哪条绑定"。
func syncUserDiscountPlanId(tx *gorm.DB, userId int) error {
	if userId <= 0 {
		return nil
	}
	bindings := make([]DiscountBinding, 0)
	if err := tx.Where("subject_id = ? AND status = ?", userId, DiscountStatusEnabled).Find(&bindings).Error; err != nil {
		return err
	}
	planId := 0
	if binding := pickActiveDiscountBinding(bindings, common.GetTimestamp()); binding != nil {
		planId = binding.PlanId
	}
	return tx.Model(&User{}).Where("id = ?", userId).UpdateColumn("discount_plan_id", planId).Error
}

// pickActiveDiscountBinding 从候选绑定里挑出真正生效的一条：
// 先看生效时间窗口（0 表示立即 / 不限），多条命中时按来源优先级
// manual > subscription > customer_code > migration，同来源取 id 最大的一条（最新）。
func pickActiveDiscountBinding(bindings []DiscountBinding, now int64) *DiscountBinding {
	var picked *DiscountBinding
	for i := range bindings {
		candidate := &bindings[i]
		if candidate.Status != DiscountStatusEnabled {
			continue
		}
		if candidate.EffectiveFrom > now {
			continue
		}
		if candidate.EffectiveTo > 0 && candidate.EffectiveTo <= now {
			continue
		}
		if picked == nil {
			picked = candidate
			continue
		}
		candidateRank := discountSourceRank(candidate.Source)
		pickedRank := discountSourceRank(picked.Source)
		if candidateRank < pickedRank || (candidateRank == pickedRank && candidate.Id > picked.Id) {
			picked = candidate
		}
	}
	return picked
}

// discountSourceRank 越小优先级越高；未知来源排在最后。
func discountSourceRank(source string) int {
	switch source {
	case DiscountSourceManual:
		return 0
	case DiscountSourceAgent:
		// 经销商给下属客户定的价：比平台手工定价低一档（平台能改回来），
		// 比套餐和客户号高一档（他明确指定过的那份算数）。
		return 1
	case DiscountSourceSubscription:
		return 2
	case DiscountSourceCustomerCode:
		return 3
	case DiscountSourceMigration:
		return 4
	default:
		return 5
	}
}

// DiscountBinding 是「谁在什么时间段用哪个方案」的事实记录。
// users.discount_plan_id 只是它的快路径冗余，两者必须由同一处代码写入。
type DiscountBinding struct {
	Id            int    `json:"id"`
	SubjectType   string `json:"subject_type" gorm:"type:varchar(16);not null;index:idx_binding_subject,priority:1"`
	SubjectId     int    `json:"subject_id" gorm:"type:int;not null;index:idx_binding_subject,priority:2"`
	PlanId        int    `json:"plan_id" gorm:"type:int;not null;index:idx_binding_plan"`
	EffectiveFrom int64  `json:"effective_from" gorm:"type:bigint;not null;default:0"`
	EffectiveTo   int64  `json:"effective_to" gorm:"type:bigint;not null;default:0"`
	Source        string `json:"source" gorm:"type:varchar(16);not null;default:'manual'"`
	Status        int    `json:"status" gorm:"type:int;not null;default:1;index:idx_binding_subject,priority:3"`
	CreatedAt     int64  `json:"created_at" gorm:"bigint"`
	UpdatedAt     int64  `json:"updated_at" gorm:"bigint"`
}

func (DiscountBinding) TableName() string {
	return "discount_bindings"
}

func (b *DiscountBinding) BeforeCreate(tx *gorm.DB) error {
	now := common.GetTimestamp()
	b.CreatedAt = now
	b.UpdatedAt = now
	return nil
}

func (b *DiscountBinding) BeforeUpdate(tx *gorm.DB) error {
	b.UpdatedAt = common.GetTimestamp()
	return nil
}
