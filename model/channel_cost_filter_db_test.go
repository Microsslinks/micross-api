package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 关缓存 + 真实 DB 走一遍成本过滤：这条路径在 task-14.2 之前完全漏了，断言把行为钉住。
//
// 夹具用 sqlite memory + AutoMigrate，与 service/channel_select_cost_breach_test.go
// 同套路；为避免包级 channelsIDM / group2model2channels 被其他用例污染，每个用例都重置。
const cacheDisabledCostFilterModel = "cache-disabled-cost-filter-model"

type cacheDisabledFixture struct {
	db *gorm.DB
}

func setupCacheDisabledCostFilterTest(t *testing.T) *cacheDisabledFixture {
	t.Helper()

	originalDB := DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	originalChannels := channelsIDM
	originalGroupCache := group2model2channels

	dsn := fmt.Sprintf("file:cache-disabled-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Channel{}, &Ability{}))
	DB = db
	common.MemoryCacheEnabled = false
	// 关缓存场景下 channelsIDM / group2model2channels 不参与选线路，置空以免污染。
	channelsIDM = nil
	group2model2channels = nil

	t.Cleanup(func() {
		DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		channelsIDM = originalChannels
		group2model2channels = originalGroupCache
		sqlDB, err := db.DB()
		if err == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	return &cacheDisabledFixture{db: db}
}

func createCacheDisabledChannel(t *testing.T, db *gorm.DB, id int, costRatio string) {
	t.Helper()
	weight := uint(100)
	priority := int64(0)
	require.NoError(t, db.Create(&Channel{
		Id:        id,
		Type:      constant.ChannelTypeOpenAI,
		Key:       fmt.Sprintf("key-%d", id),
		Status:    common.ChannelStatusEnabled,
		Name:      fmt.Sprintf("channel-%d", id),
		Weight:    &weight,
		Models:    cacheDisabledCostFilterModel,
		Group:     "default",
		Priority:  &priority,
		CostRatio: common.GetPointer(costRatio),
	}).Error)
	require.NoError(t, db.Create(&Ability{
		Group:     "default",
		Model:     cacheDisabledCostFilterModel,
		ChannelId: id,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
}

// 1) 折扣启用 + 关缓存：成本过滤生效，只挑保本线路。
func TestGetChannelFiltersCostWhenMemoryCacheDisabled(t *testing.T) {
	setupCacheDisabledCostFilterTest(t)
	db := DB
	createCacheDisabledChannel(t, db, 3101, "0.50") // 进货 5 折
	createCacheDisabledChannel(t, db, 3102, "0.80") // 进货 8 折 → 售价 5.5 折时这条亏本

	// 售价 5.5 折、底线 0 → 只有 3101（进货 ≤ 0.55）保本。
	channel, err := GetChannel("default", cacheDisabledCostFilterModel, 0, "",
		&ChannelCostFilter{SellRatio: 0.55})
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 3101, channel.Id, "关缓存也得成本过滤：亏本的那条不能被选中")
}

// 2) 折扣启用 + 关缓存：所有候选都亏本且不允许击穿时，按 describeCostBreachDB 报错，
//    说出哪条线亏多少（与缓存分支同口径）。
func TestGetChannelRaisesCostBreachWhenAllChannelsUnprofitableCacheDisabled(t *testing.T) {
	setupCacheDisabledCostFilterTest(t)
	db := DB
	createCacheDisabledChannel(t, db, 3201, "0.90") // 进货 9 折
	createCacheDisabledChannel(t, db, 3202, "0.95") // 进货 9.5 折

	channel, err := GetChannel("default", cacheDisabledCostFilterModel, 0, "",
		&ChannelCostFilter{SellRatio: 0.50})
	require.Error(t, err)
	assert.Nil(t, channel)

	// 必须把亏本事实说清楚（task-02 §7 E1：还有哪条能走、走它亏多少）。
	assert.Contains(t, err.Error(), "不亏本", "没线路时也得说清楚是「不亏本」而不是「找不到」")
	assert.Contains(t, err.Error(), "channel-3201", "亏得最少的那条要列出来供补进货价参考")
	assert.Contains(t, err.Error(), "channel-3202")
	assert.Contains(t, err.Error(), "5 折")
}

// 3) 折扣未启用（没绑方案）：筛子不动手，候选集原样选——行为与改造前一致。
func TestGetChannelKeepsLegacyBehaviorWhenFilterDisabled(t *testing.T) {
	setupCacheDisabledCostFilterTest(t)
	db := DB
	createCacheDisabledChannel(t, db, 3301, "0.90") // 即便亏本，没启用筛子也得能选
	createCacheDisabledChannel(t, db, 3302, "0.95")

	// SellRatio == 1：filter 未启用 → 走原逻辑
	channel, err := GetChannel("default", cacheDisabledCostFilterModel, 0, "",
		&ChannelCostFilter{SellRatio: 1})
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Contains(t, []int{3301, 3302}, channel.Id)

	// nil filter 也得能跑（夹具边界）。
	channel, err = GetChannel("default", cacheDisabledCostFilterModel, 0, "", nil)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Contains(t, []int{3301, 3302}, channel.Id)
}

// 4) 折扣启用 + 关缓存 + AllowCostBreach=true：候选都不保本时按权重随机选一条，
//    并把击穿事实写到 costFilter.Breach 上，供消费日志留痕。
func TestGetChannelAllowsCostBreachWhenEnabledAndCacheDisabled(t *testing.T) {
	setupCacheDisabledCostFilterTest(t)
	db := DB
	createCacheDisabledChannel(t, db, 3401, "0.85")
	createCacheDisabledChannel(t, db, 3402, "0.95")

	filter := &ChannelCostFilter{SellRatio: 0.50, AllowCostBreach: true}
	channel, err := GetChannel("default", cacheDisabledCostFilterModel, 0, "", filter)
	require.NoError(t, err)
	require.NotNil(t, channel, "放行模式下不应报错")
	assert.Contains(t, []int{3401, 3402}, channel.Id)

	require.NotNil(t, filter.Breach, "放行模式下必须记下击穿事实（task-02 §7 E2）")
	assert.Equal(t, channel.Id, filter.Breach.ChannelId)
	assert.Equal(t, channel.Name, filter.Breach.ChannelName)
	assert.NotEmpty(t, filter.Breach.SellRatio)
	assert.NotEmpty(t, filter.Breach.CostRatio)
	assert.NotEmpty(t, filter.Breach.LossRatio)
}

// 5) 端到端：经 GetRandomSatisfiedChannel（缓存关闭分支），验证 task-14.2 修复点。
func TestGetRandomSatisfiedChannelAppliesCostFilterWhenCacheDisabled(t *testing.T) {
	setupCacheDisabledCostFilterTest(t)
	db := DB
	createCacheDisabledChannel(t, db, 3501, "0.50")
	createCacheDisabledChannel(t, db, 3502, "0.80")

	// 售价 5.5 折 → 3501 保本、3502 亏本；缓存关闭时也得只挑 3501。
	for i := 0; i < 5; i++ {
		channel, err := GetRandomSatisfiedChannel("default", cacheDisabledCostFilterModel, 0, "",
			&ChannelCostFilter{SellRatio: 0.55})
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Equal(t, 3501, channel.Id, "第 %d 次迭代都该挑到保本那条", i+1)
	}
}