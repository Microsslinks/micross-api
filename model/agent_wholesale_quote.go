package model

import (
	"errors"
	"sort"
	"strings"

	"github.com/shopspring/decimal"
)

// AgentWholesaleQuoteItem 是价目表里的一行：这个模型，这位经销商按几折拿货。
type AgentWholesaleQuoteItem struct {
	ModelName   string `json:"model_name"`
	Discount    string `json:"discount"`     // 他的拿货折扣 = 成本 × 加价率
	CostRatio   string `json:"cost_ratio"`   // 这个折扣是从哪条线路的成本算出来的
	ChannelName string `json:"channel_name"` // 那条线路的名字，让他自己看得懂这个价怎么来的
}

// ListAgentWholesale 列出这位经销商当前每个模型的拿货价，也就是他拿去给客户报价的那张底价表。
//
// 只列算得出价的模型：分组下没有可用线路、或可用线路都没录进货折扣的模型不会出现。
// 不把它们列成 0 或空串，是因为「这个模型暂时拿不到价」和「这个模型拿货价很低」
// 在界面上必须看得出区别，混在一起会让人按错的价格报价。
func ListAgentWholesale(userId int, markupRatio string) ([]*AgentWholesaleQuoteItem, error) {
	if userId <= 0 {
		return nil, errors.New("无效的用户 ID")
	}
	markup, err := decimal.NewFromString(strings.TrimSpace(markupRatio))
	if err != nil {
		return nil, errors.New("毛利必须是数字")
	}

	// 整行读而不是挑列：分组列在库里是保留字，挑列要靠 initCol() 按方言拼列名。
	var user User
	if err := DB.First(&user, userId).Error; err != nil {
		return nil, err
	}
	groups := agentWholesaleGroups(user.Group)
	modelNames, err := listEnabledModelNames(groups)
	if err != nil {
		return nil, err
	}
	byModel, err := GetDiscountChannelCandidatesByModels(groups, modelNames)
	if err != nil {
		return nil, err
	}

	items := make([]*AgentWholesaleQuoteItem, 0, len(modelNames))
	for _, modelName := range modelNames {
		lowestCost, channelName, ok := lowestCostCandidate(byModel[modelName])
		if !ok {
			continue
		}
		price := lowestCost.Mul(markup)
		// 加价后已经不低于官方标价：这一档毛利下他没有便宜可拿，列出来只会让他按错价报价。
		if price.LessThanOrEqual(decimal.Zero) || price.GreaterThanOrEqual(decimal.NewFromInt(1)) {
			continue
		}
		items = append(items, &AgentWholesaleQuoteItem{
			ModelName:   modelName,
			Discount:    price.StringFixed(6),
			CostRatio:   lowestCost.StringFixed(6),
			ChannelName: channelName,
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ModelName < items[j].ModelName })
	return items, nil
}

// listEnabledModelNames 列出这些分组下当前启用的全部模型名，已去重、按名排序。
// 同一渠道挂在多个分组下会在能力表里留下多行，所以要去重。
func listEnabledModelNames(groups []string) ([]string, error) {
	groupList := normalizeLookupValues(groups)
	if len(groupList) == 0 {
		return nil, nil
	}
	names := make([]string, 0)
	if err := DB.Model(&Ability{}).
		Distinct("model").
		Where(commonGroupCol+" IN ? AND enabled = ?", groupList, true).
		Order("model ASC").
		Pluck("model", &names).Error; err != nil {
		return nil, err
	}
	return names, nil
}
