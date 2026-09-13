package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChannelCostTest(t *testing.T) {
	t.Helper()
	truncateTables(t)
	require.NoError(t, DB.Exec("DELETE FROM channels").Error)
}

func createCostTestChannel(t *testing.T, name string, costRatio *string, costUpdatedAt int64) Channel {
	t.Helper()
	channel := Channel{
		Name:          name,
		Key:           "cost-test-key-" + name,
		Status:        common.ChannelStatusEnabled,
		CostRatio:     costRatio,
		CostUpdatedAt: costUpdatedAt,
	}
	require.NoError(t, DB.Create(&channel).Error)
	return channel
}

func TestBatchUpdateChannelCostRefreshesTimestampOnlyWhenValueChanges(t *testing.T) {
	setupChannelCostTest(t)

	// 原值相同的那条不该被算成「重新核对过进价」，也就不该刷时间。
	unchanged := createCostTestChannel(t, "cost-unchanged", common.GetPointer("0.27"), 1000)
	changed := createCostTestChannel(t, "cost-changed", common.GetPointer("0.30"), 1000)
	empty := createCostTestChannel(t, "cost-empty", nil, 0)

	updated, err := BatchUpdateChannelCost([]int{unchanged.Id, changed.Id, empty.Id}, "0.27")
	require.NoError(t, err)
	assert.Equal(t, int64(2), updated)

	var storedUnchanged, storedChanged, storedEmpty Channel
	require.NoError(t, DB.First(&storedUnchanged, unchanged.Id).Error)
	require.NoError(t, DB.First(&storedChanged, changed.Id).Error)
	require.NoError(t, DB.First(&storedEmpty, empty.Id).Error)

	require.NotNil(t, storedUnchanged.CostRatio)
	assert.Equal(t, "0.27", *storedUnchanged.CostRatio)
	assert.Equal(t, int64(1000), storedUnchanged.CostUpdatedAt)

	require.NotNil(t, storedChanged.CostRatio)
	assert.Equal(t, "0.27", *storedChanged.CostRatio)
	assert.NotEqual(t, int64(1000), storedChanged.CostUpdatedAt)

	require.NotNil(t, storedEmpty.CostRatio)
	assert.Equal(t, "0.27", *storedEmpty.CostRatio)
	assert.NotZero(t, storedEmpty.CostUpdatedAt)
}

func TestBatchUpdateChannelCostClearsRatioWithEmptyString(t *testing.T) {
	setupChannelCostTest(t)

	channel := createCostTestChannel(t, "cost-clear", common.GetPointer("0.27"), 1000)

	updated, err := BatchUpdateChannelCost([]int{channel.Id}, "")
	require.NoError(t, err)
	assert.Equal(t, int64(1), updated)

	var stored Channel
	require.NoError(t, DB.First(&stored, channel.Id).Error)
	require.NotNil(t, stored.CostRatio)
	assert.Equal(t, "", *stored.CostRatio)
}

func TestGetChannelCostOverviewClassifiesUnconfiguredAndStale(t *testing.T) {
	setupChannelCostTest(t)

	now := int64(1_700_000_000)
	thirtyDays := int64(30 * 24 * 60 * 60)

	fresh := createCostTestChannel(t, "cost-fresh", common.GetPointer("0.27"), now-100)
	noTimestamp := createCostTestChannel(t, "cost-no-timestamp", common.GetPointer("0.27"), 0)
	stale := createCostTestChannel(t, "cost-stale", common.GetPointer("0.30"), now-thirtyDays-1)
	malformed := createCostTestChannel(t, "cost-malformed", common.GetPointer("abc"), now-100)
	missing := createCostTestChannel(t, "cost-missing", nil, 0)

	items, err := GetChannelCostOverview(now, thirtyDays)
	require.NoError(t, err)
	require.Len(t, items, 4)

	reasons := make(map[int]string, len(items))
	for _, item := range items {
		reasons[item.Id] = item.Reason
	}
	// 刚录过且没过期的线路不该出现在待补清单里。
	_, listed := reasons[fresh.Id]
	assert.False(t, listed)
	// 有值但没有录入时间：无从判断新旧，按过期处理。
	assert.Equal(t, ChannelCostReasonStale, reasons[noTimestamp.Id])
	assert.Equal(t, ChannelCostReasonStale, reasons[stale.Id])
	// 写进来的东西解析不出正数，等于没录。
	assert.Equal(t, ChannelCostReasonUnconfigured, reasons[malformed.Id])
	assert.Equal(t, ChannelCostReasonUnconfigured, reasons[missing.Id])
}

func TestGetChannelCostOverviewSkipsStaleCheckWhenDisabled(t *testing.T) {
	setupChannelCostTest(t)

	createCostTestChannel(t, "cost-old-but-valid", common.GetPointer("0.27"), 1)
	missing := createCostTestChannel(t, "cost-never-set", nil, 0)

	items, err := GetChannelCostOverview(2_000_000_000, 0)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, missing.Id, items[0].Id)
	assert.Equal(t, ChannelCostReasonUnconfigured, items[0].Reason)
}
