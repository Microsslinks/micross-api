package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 进货折扣的取值校验必须「单条录入」和「批量录入」同一处口径。
// 只在批量那侧拦，就会出现同一张表里两种值：批量填不进去的 27，点开单条编辑反而存得下。
func TestChannelCostRatioRejectsInvalidValuesOnBothEntryPoints(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))

	tests := []struct {
		name      string
		costRatio string
		wantErr   bool
	}{
		{name: "empty means not recorded", costRatio: ""},
		{name: "blank means not recorded", costRatio: "   "},
		{name: "normal discount", costRatio: "0.27"},
		{name: "list price itself", costRatio: "1"},
		{name: "upper bound itself", costRatio: "100"},
		{name: "above list price is allowed for loss-leaders", costRatio: "2.5"},
		{name: "zero", costRatio: "0", wantErr: true},
		{name: "negative", costRatio: "-0.1", wantErr: true},
		{name: "not a number", costRatio: "abc", wantErr: true},
		{name: "above upper bound", costRatio: "100.000001", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			costRatio := test.costRatio
			single := &model.Channel{Type: constant.ChannelTypeOpenAI, CostRatio: &costRatio}

			singleErr := validateChannel(single, false)

			body, err := common.Marshal(channelCostBatchRequest{Ids: []int{1}, CostRatio: test.costRatio})
			require.NoError(t, err)
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			ctx.Request = httptest.NewRequest(http.MethodPost, "/api/channel/cost/batch", bytes.NewReader(body))
			ctx.Request.Header.Set("Content-Type", "application/json")

			BatchSetChannelCost(ctx)

			var response struct {
				Success bool `json:"success"`
			}
			require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))

			if test.wantErr {
				require.Error(t, singleErr)
				assert.False(t, response.Success, "批量录入必须和单条编辑一样拒绝这个值")
				return
			}
			require.NoError(t, singleErr)
			assert.True(t, response.Success, "合法值两端都该放行")
		})
	}
}

// 请求没带 cost_ratio（nil）时不校验：只改备注的请求，不该被这条渠道里存着的旧值挡住。
func TestValidateChannelIgnoresOmittedCostRatio(t *testing.T) {
	require.NoError(t, validateChannel(&model.Channel{Type: constant.ChannelTypeOpenAI}, false))
}
