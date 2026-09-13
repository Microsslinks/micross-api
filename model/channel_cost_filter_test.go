package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedCostFilterChannels 造一组带进货折扣的线路，并在测试结束后还原包级缓存。
// 101 进货 0.50、102 进货 0.55、103 进货 0.56、104 未录、105 空串、106 非数字、107 为 0。
func seedCostFilterChannels(t *testing.T) {
	t.Helper()
	previousChannels := channelsIDM
	previousCacheEnabled := common.MemoryCacheEnabled
	previousGroupCache := group2model2channels
	common.MemoryCacheEnabled = true
	channelsIDM = map[int]*Channel{
		101: {Id: 101, CostRatio: common.GetPointer("0.50")},
		102: {Id: 102, CostRatio: common.GetPointer("0.55")},
		103: {Id: 103, CostRatio: common.GetPointer("0.56")},
		104: {Id: 104},
		105: {Id: 105, CostRatio: common.GetPointer("")},
		106: {Id: 106, CostRatio: common.GetPointer("abc")},
		107: {Id: 107, CostRatio: common.GetPointer("0")},
	}
	group2model2channels = map[string]map[string][]int{
		"default": {"gpt-4o": {101, 102, 103, 104, 105, 106, 107}},
	}
	t.Cleanup(func() {
		channelsIDM = previousChannels
		common.MemoryCacheEnabled = previousCacheEnabled
		group2model2channels = previousGroupCache
	})
}

func TestChannelCostFilterEnabled(t *testing.T) {
	var nilFilter *ChannelCostFilter
	assert.False(t, nilFilter.Enabled(), "nil 过滤器不动手")
	assert.False(t, (&ChannelCostFilter{SellRatio: 0}).Enabled(), "折扣为 0 不动手")
	assert.False(t, (&ChannelCostFilter{SellRatio: 1}).Enabled(), "折扣为 1 不动手")
	assert.False(t, (&ChannelCostFilter{SellRatio: 1.5}).Enabled(), "折扣超过 1 不动手")
	assert.True(t, (&ChannelCostFilter{SellRatio: 0.8}).Enabled())
}

func TestFilterChannelsByCostKeepsOnlyProfitable(t *testing.T) {
	seedCostFilterChannels(t)

	channels := []int{101, 102, 103, 104, 105, 106, 107}

	// 售价 5.5 折、底线 0：进货 ≤ 0.55 的留下（含等于），未录／坏值一律出局。
	assert.Equal(t, []int{101, 102}, filterChannelsByCost(channels, &ChannelCostFilter{SellRatio: 0.55}))

	// 底线 0.05 → 预算 0.50：只剩 101。
	assert.Equal(t, []int{101}, filterChannelsByCost(channels, &ChannelCostFilter{SellRatio: 0.55, MinMarginRatio: 0.05}))

	// 折扣为 1（未绑方案）时筛子不动手，连未录进货价的线路也照原样返回。
	assert.Equal(t, channels, filterChannelsByCost(channels, &ChannelCostFilter{SellRatio: 1}))

	// 过滤器为 nil 同样不动手。
	assert.Equal(t, channels, filterChannelsByCost(channels, nil))
}

func TestFilterChannelsByCostDropsChannelsWithoutCost(t *testing.T) {
	seedCostFilterChannels(t)

	// 未录、空串、非数字、0 —— 全部按「没有成本信息」处理并排除（口径 S1）。
	assert.Empty(t, filterChannelsByCost([]int{104, 105, 106, 107}, &ChannelCostFilter{SellRatio: 0.9}))
}

func TestNewChannelCostFilterReadsMinMarginRatio(t *testing.T) {
	setting := operation_setting.GetDiscountSetting()
	previous := setting.MinMarginRatio
	setting.MinMarginRatio = "0.05"
	defer func() { setting.MinMarginRatio = previous }()

	filter := NewChannelCostFilter(0.8)
	require.NotNil(t, filter)
	assert.Equal(t, 0.8, filter.SellRatio)
	assert.Equal(t, 0.05, filter.MinMarginRatio)

	setting.MinMarginRatio = ""
	assert.Equal(t, 0.0, NewChannelCostFilter(0.8).MinMarginRatio, "空串按 0 处理")
}

func TestGetRandomSatisfiedChannelFiltersCost(t *testing.T) {
	seedCostFilterChannels(t)

	// 售价 5.1 折：只有 101（进货 0.50）保本 → 确定性地选到它。
	channel, err := GetRandomSatisfiedChannel("default", "gpt-4o", 0, "", &ChannelCostFilter{SellRatio: 0.51})
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 101, channel.Id)

	// 售价 4.9 折：连 101 都不保本 → 明确报错，而不是静默挑一条亏损线路。
	_, err = GetRandomSatisfiedChannel("default", "gpt-4o", 0, "", &ChannelCostFilter{SellRatio: 0.49})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不亏本")

	// 折扣为 1（未绑方案）：筛子不动手，仍有候选可挑。
	channel, err = GetRandomSatisfiedChannel("default", "gpt-4o", 0, "", &ChannelCostFilter{SellRatio: 1})
	require.NoError(t, err)
	require.NotNil(t, channel)

	// 分组里根本没有这个模型时，仍返回 (nil, nil)，不因为成本过滤改写语义。
	channel, err = GetRandomSatisfiedChannel("default", "no-such-model", 0, "", &ChannelCostFilter{SellRatio: 0.55})
	require.NoError(t, err)
	assert.Nil(t, channel)
}

// ChannelPassesCostBudget 是粘连旁路复用的那一次校验。
func TestChannelPassesCostBudget(t *testing.T) {
	seedCostFilterChannels(t)

	// 筛子不动手时一律放行，连没录进货价的线路也放行——行为与改造前一致。
	assert.True(t, ChannelPassesCostBudget(104, nil))
	assert.True(t, ChannelPassesCostBudget(104, &ChannelCostFilter{SellRatio: 1}))

	filter := &ChannelCostFilter{SellRatio: 0.55}
	assert.True(t, ChannelPassesCostBudget(101, filter))
	assert.True(t, ChannelPassesCostBudget(102, filter), "进货折扣正好等于预算是保本，放行")
	assert.False(t, ChannelPassesCostBudget(103, filter), "进货折扣超出预算，不放行")
	assert.False(t, ChannelPassesCostBudget(104, filter), "没录进货价按无成本信息处理，不放行")

	withFloor := &ChannelCostFilter{SellRatio: 0.55, MinMarginRatio: 0.05}
	assert.True(t, ChannelPassesCostBudget(101, withFloor))
	assert.False(t, ChannelPassesCostBudget(102, withFloor), "毛利没到底线，不放行")
}
