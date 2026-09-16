// ============================================================================
// 任务文档 §一"task-12 · P5 订阅绑折扣 + 存量迁移 + 倍率退役"
// 12.2 存量迁移幂等脚本（master-plan §4.6 口径）
//
// 把老 users.discount_plan_id=0 且 group != 'default' 的用户，按
// setting/ratio_setting.GroupRatio[group] 折算成 DiscountPlan，
// 写一条 source='migration' 的 discount_bindings 行，
// 由 BindDiscountPlan 事务内同步更新 users.discount_plan_id 快路径。
//
// 幂等：重复跑不写重复行（plan 复用、binding 跳过）。
// 不放进 AutoMigrate：独立命令，运维按需执行。
//
// 用法：
//
//	go run ./cmd/migrate-discount-plans \
//	    --dry-run                         # 打印变更不打库
//	go run ./cmd/migrate-discount-plans  # 实跑（依赖 SQL_DSN / SQLITE_PATH）
//	go run ./cmd/migrate-discount-plans --limit 10          # 每组前 10 条试水
//	go run ./cmd/migrate-discount-plans --groups=vip,svip    # 白名单
//
// 环境：读取 .env / 环境变量 SQL_DSN（MySQL/PG）或 SQLITE_PATH（SQLite）。
// ============================================================================
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	dryRun  = flag.Bool("dry-run", false, "only print planned changes, do not write to DB")
	limit   = flag.Int("limit", 0, "max users per group (0 = unlimited)")
	groupsF = flag.String("groups", "", "comma-separated group whitelist (empty = all non-default groups from GroupRatio)")
)

type migrationStats struct {
	usersProcessed  int
	bindingsCreated int
	plansCreated    int
	skippedExisting int
}

func main() {
	flag.Parse()
	common.InitEnv()

	if err := model.InitDB(); err != nil {
		log.Fatalf("init main DB: %v", err)
	}
	if err := model.InitLogDB(); err != nil {
		log.Fatalf("init log DB: %v", err)
	}
	model.InitOptionMap()

	// GroupRatio 已通过 InitOptionMap 写入 ratio_setting。
	allRatios := ratio_setting.GetGroupRatioCopy()

	// 白名单解析：--groups 空 = 所有非 default 组；非空 = 只跑指定组。
	allowed := parseWhitelist(*groupsF)

	var stats migrationStats
	for group, ratio := range allRatios {
		if group == "default" {
			continue
		}
		if allowed != nil && !allowed[group] {
			continue
		}
		if ratio <= 0 || ratio > 1 {
			log.Printf("[group=%s] skip: group_ratio=%.4f out of (0,1]", group, ratio)
			continue
		}
		log.Printf("[group=%s ratio=%.4f] starting migration", group, ratio)
		if err := migrateGroup(group, ratio, *limit, *dryRun, &stats); err != nil {
			log.Fatalf("[group=%s] migration: %v", group, err)
		}
	}

	mode := "applied"
	if *dryRun {
		mode = "DRY-RUN"
	}
	log.Printf("[%s] done: users_processed=%d bindings_created=%d plans_created=%d skipped_existing=%d",
		mode, stats.usersProcessed, stats.bindingsCreated, stats.plansCreated, stats.skippedExisting)
}

// parseWhitelist 把 --groups=vip,svip 解析成 map；空输入返回 nil 表示「不过滤」。
func parseWhitelist(raw string) map[string]bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]bool{}
	for _, g := range strings.Split(raw, ",") {
		g = strings.TrimSpace(g)
		if g != "" && g != "default" {
			out[g] = true
		}
	}
	return out
}

// migrateGroup 把这一组分组的存量用户从「裸分组」迁移到「migration 折扣绑定」。
func migrateGroup(group string, ratio float64, limit int, dryRun bool, stats *migrationStats) error {
	// 1. 找或创建 DiscountPlan：name="migration_<group>"（owner=platform, owner_id=0）。
	plan, created, err := findOrCreateMigrationPlan(group, ratio)
	if err != nil {
		return fmt.Errorf("plan setup: %w", err)
	}
	if created {
		stats.plansCreated++
		log.Printf("[group=%s] created DiscountPlan id=%d base_discount=%s",
			group, plan.Id, plan.BaseDiscount)
	} else {
		log.Printf("[group=%s] reusing DiscountPlan id=%d base_discount=%s",
			group, plan.Id, plan.BaseDiscount)
	}

	// 2. 查未迁移用户：discount_plan_id = 0。
	// 注意 `group` 是保留字，三库统一用反引号（SQLite / MySQL 用反引号，PG 也接受）。
	q := model.DB.Where("`group` = ?", group).Where("discount_plan_id = ?", 0)
	if limit > 0 {
		q = q.Limit(limit)
	}
	var users []model.User
	if err := q.Find(&users).Error; err != nil {
		return fmt.Errorf("query users: %w", err)
	}
	log.Printf("[group=%s] %d unmigrated users (discount_plan_id=0, limit=%d)",
		group, len(users), limit)

	// 3. 逐用户迁移：先查是否已有 source='migration' 绑定，幂等。
	for _, u := range users {
		stats.usersProcessed++

		var existing int64
		if err := model.DB.Model(&model.DiscountBinding{}).
			Where("subject_type = ? AND subject_id = ? AND source = ?",
				model.DiscountSubjectUser, u.Id, model.DiscountSourceMigration).
			Count(&existing).Error; err != nil {
			log.Printf("[user=%d] count existing binding: %v", u.Id, err)
			continue
		}
		if existing > 0 {
			stats.skippedExisting++
			continue
		}

		if dryRun {
			stats.bindingsCreated++
			log.Printf("[DRY-RUN] would bind user=%d plan_id=%d", u.Id, plan.Id)
			continue
		}

		binding := &model.DiscountBinding{
			SubjectType:   model.DiscountSubjectUser,
			SubjectId:     u.Id,
			PlanId:        plan.Id,
			EffectiveFrom: 0, // 立即生效（pickActiveDiscountBinding 视 0 为「不限」）
			EffectiveTo:   0, // 永久（pickActiveDiscountBinding 视 <=now 为过期，0 不触发）
			Source:        model.DiscountSourceMigration,
			Status:        model.DiscountStatusEnabled,
		}
		// BindDiscountPlan 事务内：去重 + 上限 + Create + syncUserDiscountPlanId。
		if err := model.BindDiscountPlan(binding); err != nil {
			log.Printf("[user=%d] BindDiscountPlan: %v", u.Id, err)
			continue
		}
		stats.bindingsCreated++
	}
	return nil
}

// findOrCreateMigrationPlan 按 name 唯一索引 (owner_type, owner_id, name) 找 plan；
// 不存在则按 group_ratio 创建。created=true 表示本次新插入。
func findOrCreateMigrationPlan(group string, ratio float64) (*model.DiscountPlan, bool, error) {
	name := "migration_" + group
	ratioStr := decimal.NewFromFloat(ratio).StringFixed(6)

	var existing model.DiscountPlan
	err := model.DB.Where("owner_type = ? AND owner_id = ? AND name = ?",
		model.DiscountOwnerPlatform, 0, name).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}

	plan := &model.DiscountPlan{
		Name:            name,
		OwnerType:       model.DiscountOwnerPlatform,
		OwnerId:         0,
		BaseDiscount:    ratioStr,
		MinDiscount:     ratioStr,
		BillingMode:     "usage",
		CommissionRatio: "0.000000",
		Status:          model.DiscountStatusEnabled,
		Remark: fmt.Sprintf(
			"Auto-migrated from group_ratio for group=%s (ratio=%.4f). Created by cmd/migrate-discount-plans.",
			group, ratio),
	}
	if err := plan.Insert(); err != nil {
		return nil, false, fmt.Errorf("insert plan: %w", err)
	}
	return plan, true, nil
}