package model

import (
	"fmt"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedRouteTierChannels 造一组带毛利和优先级的线路，供择优／放行用例使用：
// 201 进货 2.5 折、优先级 10（毛利最高，优先级也最高）
// 202 进货 2.5 折、优先级 5 （同毛利，优先级次之）
// 203 进货 2.9 折、优先级 99（优先级最高，毛利最低）
// 204 未录进货折扣、优先级 99
func seedRouteTierChannels(t *testing.T) {
	t.Helper()
	previousChannels := channelsIDM
	previousCacheEnabled := common.MemoryCacheEnabled
	previousGroupCache := group2model2channels
	common.MemoryCacheEnabled = true

	newChannel := func(id int, priority int64, costRatio string) *Channel {
		channel := &Channel{
			Id:       id,
			Name:     fmt.Sprintf("channel-%d", id),
			Priority: &priority,
			Weight:   common.GetPointer(uint(10)),
		}
		if costRatio != "" {
			channel.CostRatio = common.GetPointer(costRatio)
		}
		return channel
	}
	channelsIDM = map[int]*Channel{
		201: newChannel(201, 10, "0.25"),
		202: newChannel(202, 5, "0.25"),
		203: newChannel(203, 99, "0.29"),
		204: newChannel(204, 99, ""),
	}
	group2model2channels = map[string]map[string][]int{
		"default": {"gpt-4o": {201, 202, 203, 204}},
	}
	t.Cleanup(func() {
		channelsIDM = previousChannels
		common.MemoryCacheEnabled = previousCacheEnabled
		group2model2channels = previousGroupCache
	})
}

// 毛利优先（默认）时先走毛利最高的层；优先级只在同一毛利内起作用。
// 没录进货折扣的线路排最后——它成本未知，不能因为「读不到」就被当成最便宜的。
func TestBuildChannelRouteTiersMarginFirst(t *testing.T) {
	seedRouteTierChannels(t)

	tiers, err := buildChannelRouteTiers([]int{201, 202, 203, 204}, &ChannelCostFilter{SellRatio: 0.30, PrioritizeMargin: true})
	require.NoError(t, err)
	require.Len(t, tiers, 4)

	assert.Equal(t, []int{201}, tiers[0].channels, "毛利最高的层先走")
	assert.Equal(t, []int{202}, tiers[1].channels, "同毛利、优先级次之的排在第二")
	assert.Equal(t, []int{203}, tiers[2].channels, "优先级最高但毛利最低，只能排在后面")
	assert.Equal(t, []int{204}, tiers[3].channels, "没录进货折扣的排最后")
	assert.True(t, tiers[0].costRatio.Equal(decimal.RequireFromString("0.25")))
	assert.False(t, tiers[3].costKnown)
}

// 毛利和优先级都相同的线路必须留在同一层：否则「层内按权重随机」会退化成
// 每次都走同一条，那条一挂整批请求跟着断（文档 §6 的那条权衡）。
func TestBuildChannelRouteTiersKeepsSameKeyChannelsInOneTier(t *testing.T) {
	previousChannels := channelsIDM
	t.Cleanup(func() { channelsIDM = previousChannels })

	priority := int64(10)
	weight := uint(10)
	channelsIDM = map[int]*Channel{
		301: {Id: 301, Priority: &priority, Weight: &weight, CostRatio: common.GetPointer("0.25")},
		302: {Id: 302, Priority: &priority, Weight: &weight, CostRatio: common.GetPointer("0.25")},
	}

	tiers, err := buildChannelRouteTiers([]int{301, 302}, &ChannelCostFilter{SellRatio: 0.30, PrioritizeMargin: true})
	require.NoError(t, err)
	require.Len(t, tiers, 1)
	assert.ElementsMatch(t, []int{301, 302}, tiers[0].channels)
}

// 按客户切成「稳定优先」后只看上游供应商优先级，不再看毛利。
func TestBuildChannelRouteTiersPriorityStrategy(t *testing.T) {
	seedRouteTierChannels(t)

	tiers, err := buildChannelRouteTiers([]int{201, 202, 203, 204}, &ChannelCostFilter{SellRatio: 0.30, PrioritizeMargin: false})
	require.NoError(t, err)
	require.Len(t, tiers, 4)

	assert.Equal(t, []int{203}, tiers[0].channels, "稳定优先只看优先级：203/204 同级，先到先排")
	assert.Equal(t, []int{204}, tiers[1].channels, "204 是同一优先级但成本未知，另立一层")
	assert.Equal(t, []int{201}, tiers[2].channels, "毛利最高的 201 在稳定优先下反而靠后")
	assert.Equal(t, []int{202}, tiers[3].channels)
}

// 「没有不亏本线路」的报错必须一次说清三件事：为什么失败、还有哪条线路能走、
// 走它每单亏多少（文档 §7 的 E1）。三件事缺一件，客户和运营就得再来问一轮。
func TestDescribeCostBreachStatesWhyWhatAndLoss(t *testing.T) {
	previousChannels := channelsIDM
	t.Cleanup(func() { channelsIDM = previousChannels })
	channelsIDM = map[int]*Channel{
		401: {Id: 401, Name: "provider-a", CostRatio: common.GetPointer("0.58")},
		402: {Id: 402, Name: "provider-b", CostRatio: common.GetPointer("0.60")},
		403: {Id: 403, Name: "provider-c"},
	}

	err := describeCostBreach("default", "gpt-4o", []int{401, 402, 403}, &ChannelCostFilter{SellRatio: 0.55})
	require.Error(t, err)
	message := err.Error()

	assert.Contains(t, message, "5.5 折", "说清当前的售价折扣（为什么失败）")
	assert.Contains(t, message, "provider-a", "点名还有哪条线路能走")
	assert.Contains(t, message, "进货 5.8 折", "给出这条线路的进货折扣")
	assert.Contains(t, message, "每单亏 3.0 个百分点", "给出走它会亏多少")
	assert.Contains(t, message, "另有 1 条线路未录进货折扣", "没录进货价的线路要单独说明，否则看起来像系统坏了")
	assert.Contains(t, message, "允许走亏损线路", "告诉客户下一步该找谁做什么")
}

// 线路多的时候只列亏得最少的三条，其余归并成一句，免得一条报错刷满整屏。
func TestDescribeCostBreachListsCheapestThreeFirst(t *testing.T) {
	previousChannels := channelsIDM
	t.Cleanup(func() { channelsIDM = previousChannels })

	channelsIDM = map[int]*Channel{}
	channelIds := make([]int, 0, 5)
	for index, costRatio := range []string{"0.56", "0.57", "0.58", "0.59", "0.60"} {
		id := 500 + index
		name := fmt.Sprintf("provider-%d", id)
		channelsIDM[id] = &Channel{Id: id, Name: name, CostRatio: common.GetPointer(costRatio)}
		channelIds = append(channelIds, id)
	}

	err := describeCostBreach("default", "gpt-4o", channelIds, &ChannelCostFilter{SellRatio: 0.55})
	require.Error(t, err)
	message := err.Error()

	assert.Contains(t, message, "provider-500", "亏得最少的先列")
	assert.Contains(t, message, "provider-502")
	assert.NotContains(t, message, "provider-503", "第四条起不再点名")
	assert.Contains(t, message, "另有 2 条线路同样亏本")
}

// 客户被明确允许走亏损线路时放行，并按「亏得最少」优先挑，同时留下击穿记录
// （消费日志据此在 admin_info.cost_breach 留痕）。
func TestGetRandomSatisfiedChannelAllowsCostBreach(t *testing.T) {
	seedRouteTierChannels(t)

	filter := &ChannelCostFilter{SellRatio: 0.20, AllowCostBreach: true, PrioritizeMargin: true}
	channel, err := GetRandomSatisfiedChannel("default", "gpt-4o", 0, "", filter)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 201, channel.Id, "放行时也优先挑亏得最少的那一层")

	require.NotNil(t, filter.Breach)
	assert.Equal(t, 201, filter.Breach.ChannelId)
	assert.Equal(t, "channel-201", filter.Breach.ChannelName)
	assert.Equal(t, "0.200000", filter.Breach.SellRatio)
	assert.Equal(t, "0.250000", filter.Breach.CostRatio)
	assert.Equal(t, "0.050000", filter.Breach.LossRatio)

	// 同一个售价、开关关着：明确报错，不放行，也不留击穿记录。
	blocked := &ChannelCostFilter{SellRatio: 0.20}
	_, err = GetRandomSatisfiedChannel("default", "gpt-4o", 0, "", blocked)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "不亏本")
	assert.Nil(t, blocked.Breach)

	// 有保本线路时不走放行分支，也不该留下击穿记录。
	profitable := &ChannelCostFilter{SellRatio: 0.30, AllowCostBreach: true, PrioritizeMargin: true}
	channel, err = GetRandomSatisfiedChannel("default", "gpt-4o", 0, "", profitable)
	require.NoError(t, err)
	require.NotNil(t, channel)
	assert.Equal(t, 201, channel.Id, "保本线路里毛利最高的那条")
	assert.Nil(t, profitable.Breach)
}

// 折扣要按中文习惯写成「折」：0.55 是 5.5 折，不是 55 折；也不能因为去尾零把 10 折写成 1 折。
func TestFormatRatioAsDiscount(t *testing.T) {
	assert.Equal(t, "5.5", formatRatioAsDiscount(decimal.RequireFromString("0.55")))
	assert.Equal(t, "6", formatRatioAsDiscount(decimal.RequireFromString("0.6")))
	assert.Equal(t, "2", formatRatioAsDiscount(decimal.RequireFromString("0.2")))
	assert.Equal(t, "10", formatRatioAsDiscount(decimal.RequireFromString("1")))
	assert.Equal(t, "3.0 个百分点", formatLossRatio(decimal.RequireFromString("0.03")))
	assert.Equal(t, "0.0 个百分点", formatLossRatio(decimal.RequireFromString("-0.02")), "保本时不说成负数亏损")
}
