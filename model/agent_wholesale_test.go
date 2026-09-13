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

// 拿货价要读用户、经营档案、能力表、线路四张表，这里给每个子测试一份干净的 SQLite。
// 用的是全局 DB，所以子测试之间必须隔离，收尾要还原回去（与 discount_resolve_test.go 同一套做法）。
func setupAgentWholesaleTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:agent-wholesale-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&User{}, &AgentProfile{}, &Channel{}, &Ability{},
		&DiscountPlan{}, &DiscountRule{}, &DiscountBinding{},
		&Vendor{}, &Model{},
	))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// seedWholesaleUser 建一个客户；isAgent 为真时顺手把他设成经销商。
// username 与 aff_code 都是唯一索引，而一个用例里会建好几个用户（还要比价的两位经销商、
// 跟着一起建的普通客户），所以后缀必须用 uniqueAgentTestSuffix() 而不是裸时间戳——
// 原因见那个函数上的注释：Windows 的时间分辨率不足以让同一测试里的两次创建取到不同的值。
func seedWholesaleUser(t *testing.T, group string, isAgent bool) *User {
	t.Helper()
	suffix := uniqueAgentTestSuffix()
	user := &User{
		Username: "wholesale-" + suffix,
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    group,
		AffCode:  "wholesale-aff-" + suffix,
	}
	require.NoError(t, DB.Create(user).Error)
	if isAgent {
		require.NoError(t, PromoteUserToAgent(user.Id, DiscountNone, "0", CustomerCodeStatusEnabled, ""))
	}
	return user
}

// seedWholesaleChannel 建一条上游线路并挂到给定分组的这个模型上。
// costRatio 传 nil 表示这条线路没录进货折扣。
func seedWholesaleChannel(t *testing.T, name string, costRatio *string, group string, modelName string) *Channel {
	t.Helper()
	channel := &Channel{
		Name:      name,
		Key:       "sk-wholesale-test",
		Status:    common.ChannelStatusEnabled,
		Group:     group,
		Models:    modelName,
		CostRatio: costRatio,
	}
	require.NoError(t, DB.Create(channel).Error)
	require.NoError(t, DB.Create(&Ability{
		Group:     group,
		Model:     modelName,
		ChannelId: channel.Id,
		Enabled:   true,
	}).Error)
	return channel
}

func wholesaleCostPtr(value string) *string {
	return &value
}

// 拿货价 = 最便宜线路的进货折扣 × 1.1：0.27 → 0.297。
func TestResolveAgentWholesaleUsesLowestCostLine(t *testing.T) {
	setupAgentWholesaleTest(t)
	agent := seedWholesaleUser(t, "default", true)
	seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")
	seedWholesaleChannel(t, "偏贵线路", wholesaleCostPtr("0.45"), "default", "gpt-4o")

	wholesale, err := ResolveAgentWholesale(agent.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, wholesale)
	assert.Equal(t, "0.297000", wholesale.Discount)
	assert.Equal(t, "0.270000", wholesale.CostRatio)
	assert.Equal(t, "便宜线路", wholesale.ChannelName)
}

// 不是经销商就根本没有拿货价这回事。
func TestResolveAgentWholesaleSkipsNonAgent(t *testing.T) {
	setupAgentWholesaleTest(t)
	user := seedWholesaleUser(t, "default", false)
	seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")

	wholesale, err := ResolveAgentWholesale(user.Id, "gpt-4o")
	require.NoError(t, err)
	assert.Nil(t, wholesale)
}

// 算不出来的一律返回 nil，交给上层按原价收钱：
// 没录进货折扣、线路挂在别的分组、加价后已经不低于官方标价。
func TestResolveAgentWholesaleReturnsNilWhenCostUnknown(t *testing.T) {
	t.Run("线路没录进货折扣", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		agent := seedWholesaleUser(t, "default", true)
		seedWholesaleChannel(t, "没录成本的线路", nil, "default", "gpt-4o")

		wholesale, err := ResolveAgentWholesale(agent.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Nil(t, wholesale)
	})

	t.Run("这个分组下没有可用线路", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		agent := seedWholesaleUser(t, "default", true)
		seedWholesaleChannel(t, "只给 VIP 的线路", wholesaleCostPtr("0.27"), "vip", "gpt-4o")

		wholesale, err := ResolveAgentWholesale(agent.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Nil(t, wholesale)
	})

	t.Run("加价后不低于官方标价", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		agent := seedWholesaleUser(t, "default", true)
		seedWholesaleChannel(t, "成本很高的线路", wholesaleCostPtr("0.95"), "default", "gpt-4o")

		wholesale, err := ResolveAgentWholesale(agent.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Nil(t, wholesale)
	})

	t.Run("进货折扣是脏数据", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		agent := seedWholesaleUser(t, "default", true)
		seedWholesaleChannel(t, "脏数据线路", wholesaleCostPtr("abc"), "default", "gpt-4o")

		wholesale, err := ResolveAgentWholesale(agent.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Nil(t, wholesale)
	})
}

// 经销商自己消费：没绑方案时按拿货价，绑了方案时取更便宜的那个。
func TestResolveUserDiscountAppliesAgentWholesale(t *testing.T) {
	t.Run("没绑方案按拿货价", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		agent := seedWholesaleUser(t, "default", true)
		seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")

		resolution, err := ResolveUserDiscount(agent.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.297000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromAgentWholesale, resolution.Source)
	})

	t.Run("方案更贵时取拿货价", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		agent := seedWholesaleUser(t, "default", true)
		seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")
		bindWholesalePlan(t, agent.Id, "0.800000")

		resolution, err := ResolveUserDiscount(agent.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.297000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromAgentWholesale, resolution.Source)
	})

	t.Run("方案更便宜时留方案价", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		agent := seedWholesaleUser(t, "default", true)
		seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")
		bindWholesalePlan(t, agent.Id, "0.200000")

		resolution, err := ResolveUserDiscount(agent.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, "0.200000", resolution.Discount)
		assert.Equal(t, DiscountResolvedFromPlanBase, resolution.Source)
	})

	t.Run("普通客户不受影响", func(t *testing.T) {
		setupAgentWholesaleTest(t)
		user := seedWholesaleUser(t, "default", false)
		seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")

		resolution, err := ResolveUserDiscount(user.Id, "gpt-4o")
		require.NoError(t, err)
		assert.Equal(t, DiscountNone, resolution.Discount)
		assert.Equal(t, DiscountResolvedFromDefault, resolution.Source)
	})
}

func bindWholesalePlan(t *testing.T, userId int, baseDiscount string) *DiscountPlan {
	t.Helper()
	plan := &DiscountPlan{
		Name:         fmt.Sprintf("wholesale-plan-%d", time.Now().UnixNano()),
		OwnerType:    DiscountOwnerPlatform,
		BaseDiscount: baseDiscount,
		MinDiscount:  "0",
		BillingMode:  DiscountBillingUsage,
		Status:       DiscountStatusEnabled,
	}
	require.NoError(t, plan.Insert())
	require.NoError(t, BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   userId,
		PlanId:      plan.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	}))
	return plan
}
