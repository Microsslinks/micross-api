package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// 没有配置行是绝大多数客户的常态，这时必须走默认口径：毛利优先、不允许走亏损线路。
// 这两条如果反了，等于给所有客户默认开了「平台认赔」。
func TestDiscountRoutingPolicyDefaultsWithoutRow(t *testing.T) {
	var missing *DiscountRoutingPolicy

	assert.True(t, missing.PrioritizesMargin(), "没有配置行时按毛利优先")
	assert.False(t, missing.AllowsCostBreach(), "没有配置行时不允许走亏损线路")
}

// 库里存的是字符串，认不出的取值必须收敛到默认口径，不能让它悄悄变成「允许击穿」。
func TestDiscountRoutingPolicyNormalizeDefaults(t *testing.T) {
	policy := &DiscountRoutingPolicy{RoutingStrategy: "unknown", AllowCostBreach: 7}
	policy.NormalizeDefaults()
	assert.Equal(t, RoutingStrategyMargin, policy.RoutingStrategy)
	assert.Equal(t, 1, policy.AllowCostBreach, "非零一律收成 1，避免出现 2、7 这种取值")

	empty := &DiscountRoutingPolicy{}
	empty.NormalizeDefaults()
	assert.Equal(t, RoutingStrategyMargin, empty.RoutingStrategy)
	assert.Equal(t, 0, empty.AllowCostBreach)
}

// 切成稳定优先、或单独开击穿开关，都要按配置生效，且两件事互不影响。
func TestDiscountRoutingPolicyReadsConfiguredValues(t *testing.T) {
	priority := &DiscountRoutingPolicy{RoutingStrategy: RoutingStrategyPriority}
	assert.False(t, priority.PrioritizesMargin())
	assert.False(t, priority.AllowsCostBreach())

	breachOnly := &DiscountRoutingPolicy{RoutingStrategy: RoutingStrategyMargin, AllowCostBreach: 1}
	assert.True(t, breachOnly.PrioritizesMargin(), "开击穿不改变择优策略")
	assert.True(t, breachOnly.AllowsCostBreach())
}
