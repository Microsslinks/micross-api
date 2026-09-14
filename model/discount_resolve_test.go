package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 折扣解析要一次读用户、方案、规则、模型目录、厂商五张表，这里给每个子测试一份干净的 SQLite。
// 用的是全局 DB，所以子测试之间必须隔离，收尾要还原回去（与 discount_test.go 同一套做法）。
//
// 注：本测试套件里规则的 Discount 字段全部写成 0.900000（与方案基础折扣对齐）。
// rule.Discount 已废弃——规则改为纯范围标记，命中时一律按 plan.BaseDiscount 出价。
// 解析时不再读 rule.Discount，这里只是为了与「测试数据应该自洽」的旧约定保持一致，
// 让代码里残留的旧值不会继续误导将来的读者。
func setupResolveTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	// 子测试名里有中文，不能直接拿去拼 DSN（URI 得是 ASCII），用时间戳保证唯一。
	dsn := fmt.Sprintf("file:resolve-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&User{}, &Vendor{}, &Model{}, &DiscountPlan{}, &DiscountRule{}, &DiscountBinding{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// seedResolveCustomer 建一个客户；vendorName / modelName 非空时一并建好厂商与模型目录记录。
func seedResolveCustomer(t *testing.T, vendorName string, modelName string) *User {
	t.Helper()
	user := &User{
		Username: fmt.Sprintf("resolve-%d", time.Now().UnixNano()%1000000000),
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, DB.Create(user).Error)

	vendorID := 0
	if vendorName != "" {
		vendor := &Vendor{Name: vendorName}
		require.NoError(t, DB.Create(vendor).Error)
		vendorID = vendor.Id
	}
	if modelName != "" {
		meta := &Model{ModelName: modelName, VendorID: vendorID, Status: 1}
		require.NoError(t, meta.Insert())
	}
	return user
}

// bindResolvePlan 建方案与规则并绑到客户。绑定走 BindDiscountPlan，快路径由它同步，
// 与真实链路一致——解析函数读的正是这条快路径。
func bindResolvePlan(t *testing.T, userId int, baseDiscount string, rules ...*DiscountRule) *DiscountPlan {
	t.Helper()
	plan := &DiscountPlan{
		Name:         fmt.Sprintf("resolve-plan-%d", time.Now().UnixNano()),
		OwnerType:    DiscountOwnerPlatform,
		BaseDiscount: baseDiscount,
		MinDiscount:  "0",
		BillingMode:  DiscountBillingUsage,
		Status:       DiscountStatusEnabled,
	}
	require.NoError(t, plan.Insert())
	for _, rule := range rules {
		rule.PlanId = plan.Id
		require.NoError(t, rule.Insert())
	}
	require.NoError(t, BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   userId,
		PlanId:      plan.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	}))
	return plan
}

func TestResolveUserDiscount(t *testing.T) {
	// 模型级比厂商级更具体，哪怕厂商级规则的 priority 高得多，也必须模型级赢。
	t.Run("模型级规则压过厂商级规则", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeVendor, ScopeValue: "OpenAI", Discount: "0.900000", Priority: 100, Status: DiscountStatusEnabled},
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.900000", Priority: 1, Status: DiscountStatusEnabled},
		)

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromModel, resolution.Source)
		require.NotNil(t, resolution.Rule)
		assert.Equal(t, "gpt-4o", resolution.Rule.ScopeValue)
		require.NotNil(t, resolution.Plan)
		assert.Equal(t, DiscountStatusEnabled, resolution.Plan.Status)
	})

	t.Run("命中厂商级规则", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeVendor, ScopeValue: "OpenAI", Discount: "0.900000", Status: DiscountStatusEnabled},
		)

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromVendor, resolution.Source)
		require.NotNil(t, resolution.Rule)
		assert.Equal(t, "OpenAI", resolution.Rule.ScopeValue)
	})

	t.Run("没有规则命中时用方案基础折扣", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeVendor, ScopeValue: "Anthropic", Discount: "0.900000", Status: DiscountStatusEnabled},
		)

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromPlanBase, resolution.Source)
		assert.Nil(t, resolution.Rule)
	})

	// 通配只认结尾的 `*`，这是运营配 `claude-*` 这类规则时唯一被支持的写法。
	t.Run("模型级规则支持结尾通配", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "Anthropic", "claude-3-5-sonnet")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "claude-*", Discount: "0.900000", Status: DiscountStatusEnabled},
		)

		resolution, err := ResolveUserDiscount(user.Id, "claude-3-5-sonnet")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromModel, resolution.Source)
	})

	// 同一层级的优先级由 priority 决定：priority 高的先排到，先命中先赢。
	t.Run("同级规则按优先级取高的那条", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-*", Discount: "0.900000", Status: DiscountStatusEnabled},
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.900000", Priority: 9, Status: DiscountStatusEnabled},
		)

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
	})

	// priority 一样时取先创建的那条（id 更小），避免"同分时结果随机"。
	t.Run("同级规则优先级相同时取先创建的那条", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-*", Discount: "0.900000", Status: DiscountStatusEnabled},
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.900000", Status: DiscountStatusEnabled},
		)

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
	})

	t.Run("停用的规则不参与解析", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.900000", Status: DiscountStatusEnabled},
		)
		// 规则得"先建后停"：直接插 Status=0 会被 GORM 的 default 标签掰回 1，
		// 走 Update 才是运营实际经历的路径。
		require.NoError(t, DB.Model(&DiscountRule{}).Where("plan_id = ?", plan.Id).
			UpdateColumn("status", DiscountStatusDisabled).Error)

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromPlanBase, resolution.Source)
	})

	t.Run("方案停用后回落到官方标价", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		plan := bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.900000", Status: DiscountStatusEnabled},
		)

		plan.Status = DiscountStatusDisabled
		require.NoError(t, plan.Update())

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, DiscountNone, resolution.Discount)
		assert.Equal(t, DiscountResolvedFromDefault, resolution.Source)
		assert.Nil(t, resolution.Plan)
		assert.Equal(t, plan.Id, resolution.PlanId, "快路径仍指向该方案，便于排查为什么没生效")
	})

	t.Run("没有绑定方案的客户按官方标价", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, DiscountNone, resolution.Discount)
		assert.Equal(t, DiscountResolvedFromDefault, resolution.Source)
		assert.Equal(t, 0, resolution.PlanId)
		assert.Nil(t, resolution.Plan)
	})

	// 模型目录缺这条记录时取不到厂商名，厂商级规则不该命中——但也不能报错。
	t.Run("模型目录里没有该模型时跳过厂商级匹配", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "OpenAI", "gpt-4o")
		bindResolvePlan(t, user.Id, "0.900000",
			&DiscountRule{ScopeType: DiscountScopeVendor, ScopeValue: "OpenAI", Discount: "0.900000", Status: DiscountStatusEnabled},
		)

		resolution, err := ResolveUserDiscount(user.Id, "not-in-catalog")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromPlanBase, resolution.Source)
	})

	// 折扣存进 decimal(10,6) 列后读回来可能是 "0.9"，对外必须一律是 6 位。
	t.Run("折扣按 6 位小数返回", func(t *testing.T) {
		setupResolveTest(t)
		user := seedResolveCustomer(t, "", "")
		bindResolvePlan(t, user.Id, "0.9")

		resolution, err := ResolveUserDiscount(user.Id, "any-model")
		require.NoError(t, err)
		assert.Equal(t, "0.900000", resolution.Discount)
	})
}

// 通配规则是运营最容易写错的地方，边界逐条钉死。
func TestMatchDiscountScope(t *testing.T) {
	tests := []struct {
		scope    string
		name     string
		expected bool
	}{
		{scope: "gpt-4o", name: "gpt-4o", expected: true},
		{scope: "gpt-4o", name: "gpt-4o-mini", expected: false},
		{scope: "claude-*", name: "claude-3-5-sonnet", expected: true},
		{scope: "claude-*", name: "claude", expected: false},
		{scope: "claude-*", name: "anthropic-claude-3", expected: false},
		{scope: "*", name: "anything", expected: true},
		{scope: "a*b*", name: "axby", expected: false},
		{scope: "*claude", name: "*claude", expected: true},
		{scope: "*claude", name: "claude-3", expected: false},
		{scope: "claude-*", name: "", expected: false},
		{scope: "", name: "gpt-4o", expected: false},
		{scope: " gpt-4o ", name: "gpt-4o", expected: true},
	}
	for _, test := range tests {
		assert.Equal(t, test.expected, matchDiscountScope(test.scope, test.name),
			"scope=%q name=%q", test.scope, test.name)
	}
}
