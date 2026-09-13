package middleware

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// 令牌上写了 specific_channel_id 时，选线路整个绕开候选集与成本过滤：管理员指定的那条线路
// 哪怕亏本也照走——这是有意保留的特权（.docs/task-04-p1-discount/02-work-queue.md §八 L2）。
// 这里钉住两个口径：**照走不拦**，但**必须留一行痕**；线路保本、或客户根本没折扣时一个字都不写。
const pinnedChannelCostModel = "pinned-channel-cost-model"

type pinnedChannelCostFixture struct {
	ctx    *gin.Context
	logBuf *bytes.Buffer
}

// 夹具造一件事：一个客户发一单 chat 请求，令牌把渠道钉在 channelId 上。
// withDiscount 决定这一单是否真的打折——只有打折客户才有「会亏本」这回事。
func setupPinnedChannelCostTest(t *testing.T, channelId int, costRatio string, withDiscount bool) *pinnedChannelCostFixture {
	t.Helper()

	originalDB := model.DB
	originalMemoryCacheEnabled := common.MemoryCacheEnabled
	discountSetting := operation_setting.GetDiscountSetting()
	originalEnableBillingDiscount := discountSetting.EnableBillingDiscount
	originalMinMarginRatio := discountSetting.MinMarginRatio

	dsn := fmt.Sprintf("file:pinned-channel-cost-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Channel{}, &model.Ability{}, &model.User{},
		&model.DiscountPlan{}, &model.DiscountRule{}, &model.DiscountBinding{},
		&model.DiscountRoutingPolicy{}))
	model.DB = db
	common.MemoryCacheEnabled = true

	discountSetting.EnableBillingDiscount = true
	discountSetting.MinMarginRatio = "0"

	priority := int64(0)
	weight := uint(100)
	require.NoError(t, db.Create(&model.Channel{
		Id:        channelId,
		Type:      constant.ChannelTypeOpenAI,
		Key:       "unused-key",
		Status:    common.ChannelStatusEnabled,
		Name:      fmt.Sprintf("channel-%d", channelId),
		Weight:    &weight,
		Models:    pinnedChannelCostModel,
		Group:     "default",
		Priority:  &priority,
		CostRatio: common.GetPointer(costRatio),
	}).Error)
	require.NoError(t, db.Create(&model.Ability{
		Group:     "default",
		Model:     pinnedChannelCostModel,
		ChannelId: channelId,
		Enabled:   true,
		Priority:  &priority,
		Weight:    weight,
	}).Error)
	// 成本过滤读的是内存里的渠道表，造完渠道必须重建缓存。
	model.InitChannelCache()

	user := &model.User{
		Username: fmt.Sprintf("pinned-channel-%d", time.Now().UnixNano()%1000000000),
		Password: "unused-password-hash",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(user).Error)
	if withDiscount {
		plan := &model.DiscountPlan{
			Name:         fmt.Sprintf("pinned-channel-plan-%d", time.Now().UnixNano()),
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
			ScopeValue: pinnedChannelCostModel,
			Discount:   "0.500000",
			Status:     model.DiscountStatusEnabled,
		}).Insert())
		require.NoError(t, model.BindDiscountPlan(&model.DiscountBinding{
			SubjectType: model.DiscountSubjectUser,
			SubjectId:   user.Id,
			PlanId:      plan.Id,
			Source:      model.DiscountSourceManual,
			Status:      model.DiscountStatusEnabled,
		}))
		require.True(t, model.ResolveBillingDiscount(user.Id, pinnedChannelCostModel).Applied(),
			"夹具没让这一单打折，成本过滤不会启用，后面的断言就没有意义")
	}

	// 告警写进 gin.DefaultErrorWriter，与既有测试同一套抓法。
	logBuf := &bytes.Buffer{}
	common.LogWriterMu.Lock()
	previousErrorWriter := gin.DefaultErrorWriter
	gin.DefaultErrorWriter = logBuf
	common.LogWriterMu.Unlock()

	t.Cleanup(func() {
		common.LogWriterMu.Lock()
		gin.DefaultErrorWriter = previousErrorWriter
		common.LogWriterMu.Unlock()
		model.DB = originalDB
		common.MemoryCacheEnabled = originalMemoryCacheEnabled
		discountSetting.EnableBillingDiscount = originalEnableBillingDiscount
		discountSetting.MinMarginRatio = originalMinMarginRatio
		if originalMemoryCacheEnabled && originalDB != nil && originalDB.Migrator().HasTable(&model.Channel{}) {
			model.InitChannelCache()
		}
		if sqlDB, err := db.DB(); err == nil {
			require.NoError(t, sqlDB.Close())
		}
	})

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions",
		strings.NewReader(fmt.Sprintf(`{"model":%q}`, pinnedChannelCostModel)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	common.SetContextKey(ctx, constant.ContextKeyUserId, user.Id)
	common.SetContextKey(ctx, constant.ContextKeyTokenSpecificChannelId, strconv.Itoa(channelId))

	return &pinnedChannelCostFixture{ctx: ctx, logBuf: logBuf}
}

// 指定的线路会亏本（进货 9 折 > 售价 5 折）：这一单照走，但必须留一行痕。
func TestDistributePinnedChannelWarnsWhenItLosesMoney(t *testing.T) {
	fixture := setupPinnedChannelCostTest(t, 2501, "0.90", true)

	Distribute()(fixture.ctx)

	assert.Equal(t, 2501, common.GetContextKeyInt(fixture.ctx, constant.ContextKeyChannelId),
		"特权保留：管理员指定的线路照走，不能因为亏本就拦下这一单")
	warn := fixture.logBuf.String()
	assert.Contains(t, warn, "指定渠道仍按管理员指定使用")
	assert.Contains(t, warn, "#2501")
	assert.Contains(t, warn, pinnedChannelCostModel)
}

// 指定的线路保本（进货 4.5 折 < 售价 5 折）：一个字都不写。
// 假警报多了运营就不看日志了，那时真出事也没人注意。
func TestDistributePinnedChannelStaysQuietWhenItIsProfitable(t *testing.T) {
	fixture := setupPinnedChannelCostTest(t, 2502, "0.45", true)

	Distribute()(fixture.ctx)

	assert.Equal(t, 2502, common.GetContextKeyInt(fixture.ctx, constant.ContextKeyChannelId))
	assert.Empty(t, fixture.logBuf.String())
}

// 客户没折扣（全价）：本来就不会亏，成本过滤不启用，同样不该写。
func TestDistributePinnedChannelStaysQuietForFullPriceCustomer(t *testing.T) {
	fixture := setupPinnedChannelCostTest(t, 2503, "0.90", false)

	Distribute()(fixture.ctx)

	assert.Equal(t, 2503, common.GetContextKeyInt(fixture.ctx, constant.ContextKeyChannelId))
	assert.Empty(t, fixture.logBuf.String())
}

// 不计费的路径（这里是从上游拉取任务状态）也不该写：这一单不产生费用，
// 在那里报「会亏本」是假警报。
func TestDistributePinnedChannelStaysQuietOnTaskFetch(t *testing.T) {
	fixture := setupPinnedChannelCostTest(t, 2504, "0.90", true)
	fixture.ctx.Request.URL.Path = "/mj/task/1/fetch"

	Distribute()(fixture.ctx)

	assert.Equal(t, 2504, common.GetContextKeyInt(fixture.ctx, constant.ContextKeyChannelId))
	assert.Empty(t, fixture.logBuf.String())
}
