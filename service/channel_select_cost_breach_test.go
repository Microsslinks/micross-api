package service

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// auto 分组下「这个分组有线路、但一条都不保本」这一支的行为。
//
// 它曾经是**静默失败**：报错被丢掉，客户只看到"没找到可用渠道"，运营从日志上也分不清
// 是"这个模型真没线路"还是"线路全亏本"（见 .docs/task-04-p1-discount/02-work-queue.md §八 L1）。
// 这里的两个用例把最终口径钉住：**分组继续往下试，但原因必须留住并透给客户**。
const autoGroupCostBreachModel = "auto-group-cost-breach-model"

type autoGroupCostBreachFixture struct {
	ctx   *gin.Context
	param *RetryParam
}

// 造一单「真的打了折」的请求：总开关打开 + 客户绑定启用中的方案（模型级规则 5 折）。
// 只在客户真的享受折扣时成本过滤才动手，所以夹具必须先把这件事做实，否则下面的断言
// 会因为"过滤器根本没启用"而变得毫无意义——所以这里用 ResolveBillingDiscount 自检一次。
func setupAutoGroupCostBreachTest(t *testing.T) (*autoGroupCostBreachFixture, *gorm.DB) {
	t.Helper()

	originalDB := model.DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalRetryTimes := common.RetryTimes
	originalAutoGroups := setting.AutoGroups2JsonString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalGroupRatios := ratio_setting.GroupRatio2JSONString()
	originalMaxTokenAutoGroups := setting.GetMaxTokenAutoGroups()
	discountSetting := operation_setting.GetDiscountSetting()
	originalEnableBillingDiscount := discountSetting.EnableBillingDiscount
	originalMinMarginRatio := discountSetting.MinMarginRatio

	dsn := fmt.Sprintf("file:cost-breach-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.User{},
		&model.DiscountPlan{}, &model.DiscountRule{}, &model.DiscountBinding{}))
	model.DB = db
	common.MemoryCacheEnabled = true
	common.RetryTimes = 0

	require.NoError(t, setting.UpdateAutoGroupsByJsonString(`[]`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":2}`))
	require.NoError(t, setting.UpdateMaxTokenAutoGroups("2"))

	discountSetting.EnableBillingDiscount = true
	discountSetting.MinMarginRatio = "0"

	user := &model.User{
		Username: fmt.Sprintf("cost-breach-%d", time.Now().UnixNano()%1000000000),
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(user).Error)

	plan := &model.DiscountPlan{
		Name:         fmt.Sprintf("cost-breach-plan-%d", time.Now().UnixNano()),
		OwnerType:    model.DiscountOwnerPlatform,
		BaseDiscount: "1.000000",
		MinDiscount:  "0",
		BillingMode:  model.DiscountBillingUsage,
		Status:       model.DiscountStatusEnabled,
	}
	require.NoError(t, plan.Insert())
	require.NoError(t, (&model.DiscountRule{
		PlanId:     plan.Id,
		ScopeType:  model.DiscountScopeModel,
		ScopeValue: autoGroupCostBreachModel,
		Discount:   "0.500000",
		Status:     model.DiscountStatusEnabled,
	}).Insert())
	// 走 BindDiscountPlan 而不是直接改 users.discount_plan_id：快路径由绑定写入，
	// 解析函数读的正是这条快路径，与真实链路保持一致。
	require.NoError(t, model.BindDiscountPlan(&model.DiscountBinding{
		SubjectType: model.DiscountSubjectUser,
		SubjectId:   user.Id,
		PlanId:      plan.Id,
		Source:      model.DiscountSourceManual,
		Status:      model.DiscountStatusEnabled,
	}))
	require.True(t, model.ResolveBillingDiscount(user.Id, autoGroupCostBreachModel).Applied(),
		"夹具没有让这一单打折，成本过滤不会启用，后面的断言就没有意义")

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyUserId, user.Id)
	common.SetContextKey(ctx, constant.ContextKeyTokenAutoGroups, []string{"vip", "default"})
	common.SetContextKey(ctx, constant.ContextKeyTokenCrossGroupRetry, true)

	t.Cleanup(func() {
		model.DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		common.RetryTimes = originalRetryTimes
		discountSetting.EnableBillingDiscount = originalEnableBillingDiscount
		discountSetting.MinMarginRatio = originalMinMarginRatio
		require.NoError(t, setting.UpdateAutoGroupsByJsonString(originalAutoGroups))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatios))
		require.NoError(t, setting.UpdateMaxTokenAutoGroups(fmt.Sprintf("%d", originalMaxTokenAutoGroups)))
		if originalMemoryCacheEnabled && originalDB != nil &&
			originalDB.Migrator().HasTable(&model.Channel{}) && originalDB.Migrator().HasTable(&model.Ability{}) {
			model.InitChannelCache()
		}
		sqlDB, err := db.DB()
		if err == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	retry := 0
	return &autoGroupCostBreachFixture{
		ctx: ctx,
		param: &RetryParam{
			Ctx:         ctx,
			TokenGroup:  "auto",
			ModelName:   autoGroupCostBreachModel,
			RequestPath: "/v1/chat/completions",
			Retry:       &retry,
		},
	}, db
}

func createAutoGroupCostBreachChannel(t *testing.T, db *gorm.DB, id int, group string, costRatio string) {
	t.Helper()
	priority := int64(0)
	weight := uint(100)
	require.NoError(t, db.Create(&model.Channel{
		Id:        id,
		Type:      constant.ChannelTypeOpenAI,
		Key:       fmt.Sprintf("key-%d", id),
		Status:    common.ChannelStatusEnabled,
		Name:      fmt.Sprintf("channel-%d", id),
		Weight:    &weight,
		Models:    autoGroupCostBreachModel,
		Group:     group,
		Priority:  &priority,
		CostRatio: common.GetPointer(costRatio),
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     group,
		Model:     autoGroupCostBreachModel,
		ChannelId: id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
}

// 两个分组都只有亏本线路（进货 9 折、售价 5 折）：客户必须看到"为什么"，而不是笼统的"没有渠道"。
func TestCacheGetRandomSatisfiedChannelSurfacesCostBreachInAutoGroups(t *testing.T) {
	fixture, db := setupAutoGroupCostBreachTest(t)
	createAutoGroupCostBreachChannel(t, db, 2301, "vip", "0.90")
	createAutoGroupCostBreachChannel(t, db, 2302, "default", "0.90")
	model.InitChannelCache()

	channel, _, err := CacheGetRandomSatisfiedChannel(fixture.param)
	require.Error(t, err, "有线路但都不保本，不是「找不到渠道」：报错必须透出去")
	assert.Nil(t, channel)

	// 三件事都要说清：为什么失败、还有哪条能走、走它亏多少（04-cost-aware-routing.md §7 的 E1）。
	assert.Contains(t, err.Error(), "不亏本")
	assert.Contains(t, err.Error(), "5 折")
	assert.Contains(t, err.Error(), "channel-2301")
	assert.Contains(t, err.Error(), "每单亏 40.0 个百分点")
	assert.Contains(t, err.Error(), "允许走亏损线路")
	assert.Contains(t, err.Error(), "分组 vip", "只报客户最先想要的那个分组，报它最有用")
}

// 前一个分组全亏本、后一个分组保本时，必须继续往下试：
// 「某一组全亏」不等于整单失败，否则等于把客户能赚钱的线路也一起堵死。
func TestCacheGetRandomSatisfiedChannelKeepsTryingNextAutoGroupAfterCostBreach(t *testing.T) {
	fixture, db := setupAutoGroupCostBreachTest(t)
	createAutoGroupCostBreachChannel(t, db, 2401, "vip", "0.90")
	createAutoGroupCostBreachChannel(t, db, 2402, "default", "0.40")
	model.InitChannelCache()

	channel, selectedGroup, err := CacheGetRandomSatisfiedChannel(fixture.param)
	require.NoError(t, err, "第一组全亏本只是「这一组」没得走，后面那组能赚就必须走它")
	require.NotNil(t, channel)
	assert.Equal(t, 2402, channel.Id)
	assert.Equal(t, "default", selectedGroup)
	assert.Equal(t, "default", common.GetContextKeyString(fixture.ctx, constant.ContextKeyAutoGroup))
}
