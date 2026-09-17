package service

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 跨包测试没法直接调 model 包内私有的 initCol，而候选线路查询要拼 `group` 这种保留字列名，
// 没初始化就是空串、SQL 直接语法错。做法与 controller/model_list_test.go 一致：
// 先在临时内存库上跑一次 model.InitDB() 把列名口径初始化好，再把库换成自己的。
func initSimulateColumnNames(t *testing.T) {
	t.Helper()
	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalMainType, originalLogType := common.MainDatabaseType(), common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainType, originalLogType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:simulate-init-%d?mode=memory&cache=shared", time.Now().UnixNano())
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))
	require.NoError(t, model.InitDB())
	if model.DB != nil {
		if sqlDB, err := model.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}

// 试算要同时读用户、方案、规则、模型目录、厂商、能力表、渠道七张表，
// 给每个子测试一份干净的内存 SQLite（用的是全局 DB，收尾要还原回去）。
func setupDiscountSimulateTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	initSimulateColumnNames(t)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:simulate-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Vendor{}, &model.Model{}, &model.Channel{}, &model.Ability{},
		&model.DiscountPlan{}, &model.DiscountRule{}, &model.DiscountBinding{},
		// task-17 §17.1 (b) 修复：discount_routing_policies 表在 setup 里漏建，
		// resolve customer routing policy 时报 "no such table"。
		&model.DiscountRoutingPolicy{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// seedSimulateCustomer 建一个客户，并按需建好厂商与模型目录记录。
func seedSimulateCustomer(t *testing.T, vendorName string, modelName string) *model.User {
	t.Helper()
	user := &model.User{
		Username: fmt.Sprintf("simulate-%d", time.Now().UnixNano()%1000000000),
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, model.DB.Create(user).Error)

	vendorID := 0
	if vendorName != "" {
		vendor := &model.Vendor{Name: vendorName}
		require.NoError(t, model.DB.Create(vendor).Error)
		vendorID = vendor.Id
	}
	if modelName != "" {
		require.NoError(t, model.DB.Create(&model.Model{ModelName: modelName, VendorID: vendorID, Status: 1}).Error)
	}
	return user
}

// seedSimulateChannel 建一条上游线路，并在能力表里把它挂到给定的分组上。
// costRatio 传 nil 表示这条线路没录进货折扣。
func seedSimulateChannel(t *testing.T, name string, costRatio *string, groups []string, modelName string) *model.Channel {
	t.Helper()
	channel := &model.Channel{
		Name:      name,
		Key:       "sk-simulate-test",
		Status:    common.ChannelStatusEnabled,
		Group:     groups[0],
		Models:    modelName,
		CostRatio: costRatio,
	}
	require.NoError(t, model.DB.Create(channel).Error)
	for _, group := range groups {
		require.NoError(t, model.DB.Create(&model.Ability{
			Group:     group,
			Model:     modelName,
			ChannelId: channel.Id,
			Enabled:   true,
		}).Error)
	}
	return channel
}

// bindSimulatePlan 建方案与规则并绑到客户（快路径由 BindDiscountPlan 同步，与真实链路一致）。
func bindSimulatePlan(t *testing.T, userId int, baseDiscount string, rules ...*model.DiscountRule) *model.DiscountPlan {
	t.Helper()
	plan := &model.DiscountPlan{
		Name:         fmt.Sprintf("simulate-plan-%d", time.Now().UnixNano()),
		OwnerType:    model.DiscountOwnerPlatform,
		BaseDiscount: baseDiscount,
		MinDiscount:  "0",
		BillingMode:  model.DiscountBillingUsage,
		Status:       model.DiscountStatusEnabled,
	}
	require.NoError(t, plan.Insert())
	for _, rule := range rules {
		rule.PlanId = plan.Id
		require.NoError(t, rule.Insert())
	}
	require.NoError(t, model.BindDiscountPlan(&model.DiscountBinding{
		SubjectType: model.DiscountSubjectUser,
		SubjectId:   userId,
		PlanId:      plan.Id,
		Source:      model.DiscountSourceManual,
		Status:      model.DiscountStatusEnabled,
	}))
	return plan
}

func costRatioPtr(value string) *string {
	return &value
}

// 验收口径 1、5：命中模型级规则的客户 + 线路录了进货价 → 折扣、毛利、是否达标三者一起给对。
// rule.Discount 已废弃：命中规则时按方案基础折扣（plan.BaseDiscount）出价。
// 这条用例特意把 plan.BaseDiscount 设成 0.9，让命中后客户付费按 0.9 算——
// 验证「规则命中不影响价格，只影响范围」这件事能在试算里走通。
func TestSimulateDiscountWithKnownCost(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	plan := bindSimulatePlan(t, user.Id, "0.900000",
		&model.DiscountRule{ScopeType: model.DiscountScopeModel, ScopeValue: "gpt-4o", Discount: "0.300000", Status: model.DiscountStatusEnabled},
	)
	channel := seedSimulateChannel(t, "微观互联-GPT", costRatioPtr("0.27"), []string{"default"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	assert.Equal(t, "0.900000", result.Resolution.Discount, "rule.Discount 已废弃：命中规则时按方案基础折扣出价")
	assert.Equal(t, model.DiscountResolvedFromModel, result.Resolution.Source)
	require.NotNil(t, result.Resolution.MatchedRule)
	assert.Equal(t, "gpt-4o", result.Resolution.MatchedRule.ScopeValue)
	require.NotNil(t, result.Plan)
	assert.Equal(t, plan.Id, result.Plan.Id)
	assert.Equal(t, "OpenAI", result.Vendor)
	assert.Equal(t, "0.000000", result.MinMarginRatio)
	assert.True(t, result.CostKnown)
	assert.Empty(t, result.Warnings)

	require.Len(t, result.Channels, 1)
	calculated := result.Channels[0]
	assert.Equal(t, channel.Id, calculated.ChannelId)
	assert.Equal(t, "微观互联-GPT", calculated.ChannelName)
	require.NotNil(t, calculated.CostRatio)
	assert.Equal(t, "0.270000", *calculated.CostRatio)
	require.NotNil(t, calculated.GrossMargin)
	assert.Equal(t, "2.333333", *calculated.GrossMargin, "客户 0.90 / 进货 0.27 − 1")
	require.NotNil(t, calculated.PassesFloor)
	assert.True(t, *calculated.PassesFloor, "0.27 <= 0.90 − 0")
}

// 验收口径 6：线路没录进货折扣 → 三个值字段都是 null、cost_known 为 false、接口不报错且指明是哪条线路。
func TestSimulateDiscountWithoutKnownCost(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	bindSimulatePlan(t, user.Id, "0.300000")
	seedSimulateChannel(t, "某某-GPT", nil, []string{"default"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	assert.False(t, result.CostKnown)
	require.Len(t, result.Channels, 1)
	assert.Nil(t, result.Channels[0].CostRatio)
	assert.Nil(t, result.Channels[0].GrossMargin)
	assert.Nil(t, result.Channels[0].PassesFloor)
	require.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "某某-GPT")
}

// 验收口径 2：没绑定方案的客户按官方标价试算，不报错，但要明确提示。
func TestSimulateDiscountForUnboundUser(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	seedSimulateChannel(t, "微观互联-GPT", costRatioPtr("0.27"), []string{"default"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	assert.Equal(t, model.DiscountNone, result.Resolution.Discount)
	assert.Equal(t, model.DiscountResolvedFromDefault, result.Resolution.Source)
	assert.Nil(t, result.Resolution.MatchedRule)
	assert.Nil(t, result.Plan)
	require.NotEmpty(t, result.Warnings)
	assert.Contains(t, result.Warnings[0], "未绑定折扣方案")

	require.Len(t, result.Channels, 1)
	require.NotNil(t, result.Channels[0].GrossMargin)
	assert.Equal(t, "2.703704", *result.Channels[0].GrossMargin, "1.0 / 0.27 − 1")
}

// 验收口径 8：客户折扣低于进货价时毛利为负、passes_floor 为 false——只提示，不报错。
func TestSimulateDiscountForLosingRoute(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	bindSimulatePlan(t, user.Id, "0.100000")
	seedSimulateChannel(t, "微观互联-GPT", costRatioPtr("0.27"), []string{"default"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	require.Len(t, result.Channels, 1)
	calculated := result.Channels[0]
	require.NotNil(t, calculated.GrossMargin)
	assert.Equal(t, "-0.629630", *calculated.GrossMargin, "0.10 / 0.27 − 1")
	require.NotNil(t, calculated.PassesFloor)
	assert.False(t, *calculated.PassesFloor)
	assert.True(t, result.CostKnown, "成本是知道的，只是这单会亏")
}

// 毛利底线抬高后，原本达标的线路要变成不达标；底线本身要随出参一起返回，让运营知道按什么标准判的。
func TestSimulateDiscountRespectsMarginFloor(t *testing.T) {
	setupDiscountSimulateTest(t)
	setting := operation_setting.GetDiscountSetting()
	previous := setting.MinMarginRatio
	setting.MinMarginRatio = "0.05"
	t.Cleanup(func() { setting.MinMarginRatio = previous })

	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	bindSimulatePlan(t, user.Id, "0.300000")
	seedSimulateChannel(t, "微观互联-GPT", costRatioPtr("0.27"), []string{"default"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	assert.Equal(t, "0.050000", result.MinMarginRatio)
	require.Len(t, result.Channels, 1)
	require.NotNil(t, result.Channels[0].PassesFloor)
	assert.False(t, *result.Channels[0].PassesFloor, "0.27 > 0.30 − 0.05")
}

// 该客户分组下没有这条模型的线路时不报错，channels 为空数组并给出提示。
func TestSimulateDiscountWithoutAvailableChannel(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	seedSimulateChannel(t, "只给 VIP 的线路", costRatioPtr("0.27"), []string{"vip"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	assert.Empty(t, result.Channels)
	assert.False(t, result.CostKnown, "没有候选线路就谈不上成本已知")
	assert.Contains(t, result.Warnings, "该客户分组下此模型无可用线路")
}

// 指定 channel_id 时只算那一条；同一条线路挂在多个分组下只出现一次。
func TestSimulateDiscountForSpecifiedChannel(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	bindSimulatePlan(t, user.Id, "0.300000")
	shared := seedSimulateChannel(t, "共享线路", costRatioPtr("0.27"), []string{"default", "vip"}, "gpt-4o")
	other := seedSimulateChannel(t, "备用线路", costRatioPtr("0.20"), []string{"default"}, "gpt-4o")

	all, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)
	require.Len(t, all.Channels, 2, "同一条线路挂在两个分组下只应出现一次")
	ids := []int{all.Channels[0].ChannelId, all.Channels[1].ChannelId}
	assert.ElementsMatch(t, []int{shared.Id, other.Id}, ids)

	only, err := SimulateDiscount(user.Id, "gpt-4o", shared.Id)
	require.NoError(t, err)
	require.Len(t, only.Channels, 1)
	assert.Equal(t, shared.Id, only.Channels[0].ChannelId)

	_, err = SimulateDiscount(user.Id, "gpt-4o", 999999)
	assert.ErrorIs(t, err, ErrDiscountSimulateChannelNotAvailable)
}

// 进货折扣被写坏（解析不出数字或为 0）时按未录入处理：不能拿它算出一个荒唐的毛利。
func TestSimulateDiscountWithBrokenCostRatio(t *testing.T) {
	setupDiscountSimulateTest(t)
	user := seedSimulateCustomer(t, "OpenAI", "gpt-4o")
	bindSimulatePlan(t, user.Id, "0.300000")
	seedSimulateChannel(t, "坏数据线路", costRatioPtr("abc"), []string{"default"}, "gpt-4o")
	seedSimulateChannel(t, "零值线路", costRatioPtr("0"), []string{"default"}, "gpt-4o")

	result, err := SimulateDiscount(user.Id, "gpt-4o", 0)
	require.NoError(t, err)

	assert.False(t, result.CostKnown)
	require.Len(t, result.Channels, 2)
	for _, calculated := range result.Channels {
		assert.Nil(t, calculated.CostRatio)
		assert.Nil(t, calculated.GrossMargin)
		assert.Nil(t, calculated.PassesFloor)
	}
	require.Len(t, result.Warnings, 2)
	assert.Contains(t, result.Warnings[0], "坏数据线路")
	assert.Contains(t, result.Warnings[1], "零值线路")
}

// 客户不存在要能区分出来，controller 据此回 400 而不是 500。
func TestSimulateDiscountForMissingUser(t *testing.T) {
	setupDiscountSimulateTest(t)

	_, err := SimulateDiscount(999999, "gpt-4o", 0)
	assert.ErrorIs(t, err, ErrDiscountSimulateUserNotFound)
}
