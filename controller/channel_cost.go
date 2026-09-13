package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/shopspring/decimal"

	"github.com/gin-gonic/gin"
)

// 进货折扣多久没更新就算「过期」。上游调价通常按月发生，30 天既不吵人，
// 又能在上游涨价后一个月内提醒到；调用方可以用 query 参数临时改。
const defaultChannelCostStaleDays = 30

// 进货折扣的上限。进价高于标价是可能的（亏本引流），所以不卡在 1 以内；
// 但超过这个数几乎一定是把「2.7 折」误填成「27」，拦下来比算错钱好。
const maxChannelCostRatio = 100

const secondsPerDay = 24 * 60 * 60

type channelCostBatchRequest struct {
	Ids       []int  `json:"ids"`
	CostRatio string `json:"cost_ratio"`
}

// BatchSetChannelCost 批量给选中的线路录同一个进货折扣。cost_ratio 传空串表示清空。
func BatchSetChannelCost(c *gin.Context) {
	request := channelCostBatchRequest{}
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiError(c, err)
		return
	}
	if len(request.Ids) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请先选择要设置的渠道",
		})
		return
	}

	costRatio := strings.TrimSpace(request.CostRatio)
	if costRatio != "" {
		value, err := decimal.NewFromString(costRatio)
		if err != nil || value.LessThanOrEqual(decimal.Zero) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "进货折扣要填一个大于 0 的数，例如 0.27 表示按平台标价的 27% 进货",
			})
			return
		}
		if value.GreaterThan(decimal.NewFromInt(maxChannelCostRatio)) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": fmt.Sprintf("进货折扣不能超过 %d，请确认没有把 0.27 误填成 27", maxChannelCostRatio),
			})
			return
		}
	}

	updated, err := model.BatchUpdateChannelCost(request.Ids, costRatio)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	model.InitChannelCache()
	recordManageAudit(c, "channel.cost_batch_update", map[string]interface{}{
		"count":      updated,
		"cost_ratio": costRatio,
	})
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    updated,
	})
}

// GetStaleChannelCosts 列出还没录进货折扣、以及录了太久没更新的线路，供运营补录。
func GetStaleChannelCosts(c *gin.Context) {
	days := defaultChannelCostStaleDays
	if raw := c.Query("days"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 3650 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "days 必须是 1 到 3650 之间的整数",
			})
			return
		}
		days = parsed
	}

	items, err := model.GetChannelCostOverview(common.GetTimestamp(), int64(days)*secondsPerDay)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"days":  days,
			"items": items,
		},
	})
}
