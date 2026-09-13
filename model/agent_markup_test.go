package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 加价率的取值范围：缺省 +10%，下限 +5%，再往上由运营按生意自己定。
func TestNormalizeAgentMarkup(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{name: "缺省档 1.1", raw: AgentWholesaleMarkupDefault, want: "1.100000"},
		{name: "下限 1.05 允许", raw: "1.05", want: "1.050000"},
		{name: "高于下限允许", raw: "1.2", want: "1.200000"},
		{name: "带空格的数字", raw: " 1.10 ", want: "1.100000"},
		{name: "低于下限拒绝", raw: "1.04", wantErr: true},
		{name: "等于 1（平台不赚钱）拒绝", raw: "1", wantErr: true},
		{name: "小于 1（平台倒贴）拒绝", raw: "0.9", wantErr: true},
		{name: "非数字拒绝", raw: "abc", wantErr: true},
		{name: "空白拒绝", raw: "  ", wantErr: true},
		{name: "过大拒绝", raw: AgentWholesaleMarkupCap, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := NormalizeAgentMarkup(tc.raw)
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// 加价率是每个经销商自己的：同一个模型、同一条线路，两个人拿到的价不同。
func TestResolveAgentWholesaleUsesProfileMarkupRatio(t *testing.T) {
	setupAgentWholesaleTest(t)
	seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")

	// 0.27 × 1.05 = 0.2835
	lowMarkup := seedWholesaleAgentWithMarkup(t, "default", "1.050000")
	// 0.27 × 1.2 = 0.324
	highMarkup := seedWholesaleAgentWithMarkup(t, "default", "1.200000")

	low, err := ResolveAgentWholesale(lowMarkup.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, low)
	assert.Equal(t, "0.283500", low.Discount)

	high, err := ResolveAgentWholesale(highMarkup.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, high)
	assert.Equal(t, "0.324000", high.Discount)
}

// 早年建的档案没有加价率这一列的值：按缺省档 +10% 算，不是按原价、更不是按 0。
func TestResolveAgentWholesaleFallsBackToDefaultMarkup(t *testing.T) {
	setupAgentWholesaleTest(t)
	agent := seedWholesaleUser(t, "default", true)
	seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")
	require.NoError(t, DB.Model(&AgentProfile{}).Where("user_id = ?", agent.Id).
		Update("markup_ratio", "").Error)

	wholesale, err := ResolveAgentWholesale(agent.Id, "gpt-4o")
	require.NoError(t, err)
	require.NotNil(t, wholesale)
	assert.Equal(t, "0.297000", wholesale.Discount)
}

// 档案里那档加价率读出来解析不了（数据坏了）时按原价，绝不用一个没有依据的数计价。
func TestResolveAgentWholesaleReturnsNilWhenMarkupBroken(t *testing.T) {
	setupAgentWholesaleTest(t)
	agent := seedWholesaleUser(t, "default", true)
	seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")
	require.NoError(t, DB.Model(&AgentProfile{}).Where("user_id = ?", agent.Id).
		Update("markup_ratio", "abc").Error)

	wholesale, err := ResolveAgentWholesale(agent.Id, "gpt-4o")
	require.NoError(t, err)
	assert.Nil(t, wholesale)
}

// 档案里没有值或值不可读时按缺省档取用，缺省不是猜数。
func TestAgentProfileEffectiveMarkupRatio(t *testing.T) {
	assert.Equal(t, AgentWholesaleMarkupDefault, (&AgentProfile{}).EffectiveMarkupRatio())
	assert.Equal(t, AgentWholesaleMarkupDefault, (*AgentProfile)(nil).EffectiveMarkupRatio())
	assert.Equal(t, AgentWholesaleMarkupDefault, (&AgentProfile{MarkupRatio: "   "}).EffectiveMarkupRatio())
	assert.Equal(t, "1.200000", (&AgentProfile{MarkupRatio: "1.200000"}).EffectiveMarkupRatio())
}

// NormalizeDefaults 要把空的加价率补成缺省档，不能留空串进库。
func TestAgentProfileNormalizeDefaultsFillsMarkupRatio(t *testing.T) {
	profile := &AgentProfile{}
	profile.NormalizeDefaults()
	assert.Equal(t, AgentWholesaleMarkupDefault, profile.MarkupRatio)

	kept := &AgentProfile{MarkupRatio: "1.300000"}
	kept.NormalizeDefaults()
	assert.Equal(t, "1.300000", kept.MarkupRatio)
}

// 价目表：每个模型按各自最便宜一条线路的成本 × 加价率，并带上那条线路的名字。
// 没录进货折扣的线路算不上价，它名下的模型不进这张表。
func TestListAgentWholesaleQuote(t *testing.T) {
	setupAgentWholesaleTest(t)
	agent := seedWholesaleAgentWithMarkup(t, "default", "1.100000")

	seedWholesaleChannel(t, "便宜线路", wholesaleCostPtr("0.27"), "default", "gpt-4o")
	seedWholesaleChannel(t, "偏贵线路", wholesaleCostPtr("0.55"), "default", "gpt-4o")
	seedWholesaleChannel(t, "没报价的线路", (*string)(nil), "default", "claude-3")

	items, err := ListAgentWholesale(agent.Id, "1.100000")
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "gpt-4o", items[0].ModelName)
	assert.Equal(t, "0.297000", items[0].Discount)
	assert.Equal(t, "0.270000", items[0].CostRatio)
	assert.Equal(t, "便宜线路", items[0].ChannelName)
}

// 加价后已经不低于官方标价时，这一档毛利下他没有便宜可拿，那张价目表里就不该出现这个模型。
func TestListAgentWholesaleQuoteSkipsModelsAboveListPrice(t *testing.T) {
	setupAgentWholesaleTest(t)
	agent := seedWholesaleAgentWithMarkup(t, "default", "1.100000")
	seedWholesaleChannel(t, "成本接近标价", wholesaleCostPtr("0.98"), "default", "gpt-4o")

	items, err := ListAgentWholesale(agent.Id, "1.100000")
	require.NoError(t, err)
	assert.Empty(t, items)
}

// seedWholesaleAgentWithMarkup 建一个经销商，并把他的加价率设成给定值。
func seedWholesaleAgentWithMarkup(t *testing.T, group string, markupRatio string) *User {
	t.Helper()
	user := seedWholesaleUser(t, group, false)
	require.NoError(t, PromoteUserToAgentWithMarkup(
		user.Id, markupRatio, CustomerCodeStatusEnabled, ""))
	return user
}
