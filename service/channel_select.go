package service

import (
	"errors"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx          *gin.Context
	TokenGroup   string
	ModelName    string
	RequestPath  string
	Retry        *int
	resetNextTry bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
//
//   - priorityRetry = Retry - startRetryIndex, represents the priority level within current group.
//     priorityRetry = Retry - startRetryIndex，表示当前分组内的优先级级别。
//
//   - When GetRandomSatisfiedChannel returns nil (priorities exhausted), moves to next group.
//     当 GetRandomSatisfiedChannel 返回 nil（优先级用完）时，切换到下一个分组。
//
// Example flow (2 groups, each with 2 priorities, RetryTimes=3):
// 示例流程（2个分组，每个有2个优先级，RetryTimes=3）：
//
//	Retry=0: GroupA, priority0 (startRetryIndex=0, priorityRetry=0)
//	         分组A, 优先级0
//
//	Retry=1: GroupA, priority1 (startRetryIndex=0, priorityRetry=1)
//	         分组A, 优先级1
//
//	Retry=2: GroupA exhausted → GroupB, priority0 (startRetryIndex=2, priorityRetry=0)
//	         分组A用完 → 分组B, 优先级0
//
//	Retry=3: GroupB, priority1 (startRetryIndex=2, priorityRetry=1)
//	         分组B, 优先级1
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	// costBreachErr 留住「这个分组有线路、但一条都不保本」的报错（见下面 auto 分支）。
	var costBreachErr error
	selectGroup := param.TokenGroup
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)
	costFilter := BuildChannelCostFilter(param)

	if param.TokenGroup == "auto" {
		autoGroups := GetRequestAutoGroups(param.Ctx, userGroup)
		if len(autoGroups) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			// Calculate priorityRetry for current group
			// 计算当前分组的 priorityRetry
			priorityRetry := param.GetRetry()
			// If moved to a new group, reset priorityRetry and update startRetryIndex
			// 如果切换到新分组，重置 priorityRetry 并更新 startRetryIndex
			if i > startGroupIndex {
				priorityRetry = 0
			}
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, priorityRetry: %d", autoGroup, priorityRetry)

			channel, err = model.GetRandomSatisfiedChannel(autoGroup, param.ModelName, priorityRetry, param.RequestPath, costFilter)
			if err != nil && costBreachErr == nil {
				// 「这个分组有线路、但一条都不保本」不是"没渠道"，原因必须留住：auto 分组本来就是
				// 按客户给的顺序往下试，后面那个分组可能是赚的，所以不能一遇到就放弃整单。
				// 但整条链都试完还是没挑到线路时，要把这条报错透出去（见本函数末尾）。
				// 只留第一条——报客户最先想要的那个分组，对他最有用。
				costBreachErr = err
			}
			if channel == nil {
				// Current group has no available channel for this model, try next group
				// 当前分组没有该模型的可用渠道，尝试下一个分组
				reason := "no channel for this model"
				if err != nil {
					reason = err.Error()
				}
				logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityRetry %d, trying next group (reason: %s)", autoGroup, param.ModelName, priorityRetry, reason)
				// 重置状态以尝试下一个分组
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				continue
			}
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)

			// Prepare state for next retry
			// 为下一次重试准备状态
			if crossGroupRetry && priorityRetry >= common.RetryTimes {
				// Current group has exhausted all retries, prepare to switch to next group
				// This request still uses current group, but next retry will use next group
				// 当前分组已用完所有重试次数，准备切换到下一个分组
				// 本次请求仍使用当前分组，但下次重试将使用下一个分组
				logger.LogDebug(param.Ctx, "Current group %s retries exhausted (priorityRetry=%d >= RetryTimes=%d), preparing switch to next group for next retry", autoGroup, priorityRetry, common.RetryTimes)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				param.ResetRetryNextTry()
			} else {
				// Stay in current group, save current state
				// 保持在当前分组，保存当前状态
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			}
			break
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannel(param.TokenGroup, param.ModelName, param.GetRetry(), param.RequestPath, costFilter)
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}

	if channel == nil && costBreachErr != nil {
		// auto 链全试完了仍然没挑到线路，而其中至少有一个分组是"有线路、但都不保本"：
		// 把"为什么"透出去，否则客户只看到"没找到可用渠道"，运营也分不清是"这个模型真没线路"
		// 还是"线路全亏本"（口径见 .docs/task-02-business-goals/04-cost-aware-routing.md §7 的 E1）。
		return nil, selectGroup, costBreachErr
	}

	// 客户被允许走亏损线路时，这一单必须在消费日志里留痕。记在请求上下文上是因为写日志的
	// 地方在计费链路深处，够不着这里的局部变量；由 attachCostBreach 落到 admin_info 下。
	if costFilter != nil && costFilter.Breach != nil {
		common.SetContextKey(param.Ctx, constant.ContextKeyCostBreach, costFilter.Breach)
	}
	return channel, selectGroup, nil
}

// BuildChannelCostFilter 算出这一单的成本过滤条件，交给选线路逻辑剔掉会亏本的线路。
//
// 折扣从 model.ResolveBillingDiscount 取——它自带总开关：开关关着、没绑方案、
// 方案停用都给出 1，于是过滤器不启用、候选集一条不动。这里刻意放在
// channelSyncLock 之外调用：解析折扣要查库，不能占着缓存锁去查。
//
// 客户的路由策略（毛利优先／稳定优先、是否允许走亏损线路）跟折扣一起取：两者都只在
// 「这一单真的要打折」时才有意义，所以都放在 discount.Applied() 这道门后面。
func BuildChannelCostFilter(param *RetryParam) *model.ChannelCostFilter {
	if param == nil || param.Ctx == nil {
		return nil
	}
	userId := common.GetContextKeyInt(param.Ctx, constant.ContextKeyUserId)
	discount := model.ResolveBillingDiscount(userId, param.ModelName)
	if !discount.Applied() {
		return nil
	}
	return model.NewChannelCostFilter(discount.Ratio, model.ResolveCustomerRouting(userId))
}
