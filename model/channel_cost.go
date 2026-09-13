package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
)

// 进货折扣的运营侧能力：批量录入 + 待补清单。
// 单条录入走渠道编辑（controller/channel.go 的 UpdateChannel），这里补的是「一次改一批」
// 和「哪条线路还没录 / 录的价已经太旧」——两者都是运营对着几十条线路干活时的必需动作。

// 待补清单里一条线路被列出来的原因。
const (
	// ChannelCostReasonUnconfigured 没录进货折扣。按 S1 口径这种线路不参与折扣客户的候选，
	// 也就是「客户明明有线路却走不了」的直接原因。
	ChannelCostReasonUnconfigured = "unconfigured"
	// ChannelCostReasonStale 录了，但太久没更新。上游调价后系统还按旧价判保本，会静悄悄亏钱。
	ChannelCostReasonStale = "stale"
)

// ChannelCostOverviewItem 是待补清单里的一行。
type ChannelCostOverviewItem struct {
	Id            int    `json:"id"`
	Name          string `json:"name"`
	Status        int    `json:"status"`
	CostRatio     string `json:"cost_ratio"`      // 未配置时为空串
	CostUpdatedAt int64  `json:"cost_updated_at"` // 未配置时为 0
	Reason        string `json:"reason"`
}

// BatchUpdateChannelCost 给一批线路写同一个进货折扣，返回真正被改动的条数。
// costRatio 传空串表示清空，与单条编辑的「显式清空」是同一口径。
// 只有值真的变了才刷 cost_updated_at —— 把同一个数再存一遍不算「重新核对过进价」。
func BatchUpdateChannelCost(ids []int, costRatio string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var channels []Channel
	if err := DB.Select("id", "cost_ratio").Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return 0, err
	}
	changedIds := make([]int, 0, len(channels))
	for _, channel := range channels {
		current := ""
		if channel.CostRatio != nil {
			current = strings.TrimSpace(*channel.CostRatio)
		}
		if current != costRatio {
			changedIds = append(changedIds, channel.Id)
		}
	}
	if len(changedIds) == 0 {
		return 0, nil
	}

	result := DB.Model(&Channel{}).Where("id IN ?", changedIds).Updates(map[string]interface{}{
		"cost_ratio":      costRatio,
		"cost_updated_at": common.GetTimestamp(),
	})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// GetChannelCostOverview 列出需要补一下进货折扣的线路：没录的，以及录了但超过
// staleSeconds 没更新的。staleSeconds <= 0 表示不做超期判断，只列没录的。
//
// 已停用的线路照样列出，交给调用方决定怎么展示——停用的线路一旦恢复，成本还是旧的。
func GetChannelCostOverview(now int64, staleSeconds int64) ([]ChannelCostOverviewItem, error) {
	var channels []Channel
	if err := DB.Select("id", "name", "status", "cost_ratio", "cost_updated_at").
		Order("id").Find(&channels).Error; err != nil {
		return nil, err
	}

	items := make([]ChannelCostOverviewItem, 0, len(channels))
	for _, channel := range channels {
		raw := ""
		if channel.CostRatio != nil {
			raw = strings.TrimSpace(*channel.CostRatio)
		}
		item := ChannelCostOverviewItem{
			Id:            channel.Id,
			Name:          channel.Name,
			Status:        channel.Status,
			CostRatio:     raw,
			CostUpdatedAt: channel.CostUpdatedAt,
		}

		// 与 channelCostRatio 同一口径：空串、非数字、非正数都算「没有成本信息」。
		// 区别是这里判的是库里的原值，channelCostRatio 判的是内存缓存。
		configured := false
		if raw != "" {
			if parsed, err := decimal.NewFromString(raw); err == nil && parsed.GreaterThan(decimal.Zero) {
				configured = true
			}
		}

		switch {
		case !configured:
			item.Reason = ChannelCostReasonUnconfigured
		case staleSeconds > 0 && channel.CostUpdatedAt <= 0:
			// 有值却没有录入时间：说明这个值绕过了录入界面写进来，无从判断新旧。
			item.Reason = ChannelCostReasonStale
		case staleSeconds > 0 && now-channel.CostUpdatedAt > staleSeconds:
			item.Reason = ChannelCostReasonStale
		default:
			continue
		}

		items = append(items, item)
	}
	return items, nil
}
