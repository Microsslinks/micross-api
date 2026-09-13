package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/setting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// 经销商拿货价的解析。
//
// 平台卖给经销商的价不手工填，算出来：拿这个模型最便宜一条可用线路的进货折扣，
// 乘上平台固定加价档，就是经销商的拿货价。
// 例：上游按 0.27 给我们（cost_ratio = 0.27），加价档 1.1 → 经销商拿货 0.297。
//
// 为什么按「最便宜那条」算：拿货价是平台给经销商的报价，报价要盖得住路由实际
// 可能走到的线路成本；取最低进货折扣是这套体系判断成本的一贯口径——保存前校验
// 与试算都拿「规则范围内最低的进货折扣」当成本（见 service/discount_validate.go）。
//
// 价格铁律：线路成本 ≤ 经销商拿货价 ≤ 他卖给客户的价。

const (
	// AgentWholesaleMarkupDefault 是平台加价率的缺省档：1.1，即在自采成本之上加 10%。
	// 新设经销商不填毛利时用它；早年建的档案（这一列还没值）也按它处理。
	AgentWholesaleMarkupDefault = "1.1"
	// AgentWholesaleMarkupMin 是加价率的下限：1.05，即至少加 5%。再低平台就白干了，
	// 所以这里是硬校验，不是建议值。
	AgentWholesaleMarkupMin = "1.05"
	// AgentWholesaleMarkupCap 是加价率的硬上界，只用来挡住误输入（比如把 1.1 敲成 110）：
	// 列的容量是 decimal(10,6)，超过这个量级的数根本存不进去。
	AgentWholesaleMarkupCap = "100"
)

// NormalizeAgentMarkup 校验平台加价率并规范成固定 6 位小数字符串。
// 加价率是「成本 × 加价率 = 经销商拿货价」里的那个乘数，所以下限是 1.05 而不是 0.05：
// 小于 1 意味着平台倒贴钱卖，绝不允许。
func NormalizeAgentMarkup(raw string) (string, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("毛利必须是数字")
	}
	minimum, err := decimal.NewFromString(AgentWholesaleMarkupMin)
	if err != nil {
		return "", fmt.Errorf("经销商加价率下限无法解析：%s", AgentWholesaleMarkupMin)
	}
	if value.LessThan(minimum) {
		return "", errors.New("毛利不能低于 5%")
	}
	capValue, err := decimal.NewFromString(AgentWholesaleMarkupCap)
	if err != nil {
		return "", fmt.Errorf("经销商加价率上界无法解析：%s", AgentWholesaleMarkupCap)
	}
	if value.GreaterThanOrEqual(capValue) {
		return "", errors.New("毛利设置过大，请检查是否填错了")
	}
	return value.StringFixed(6), nil
}

// AgentWholesale 是「这个经销商对这个模型按几折拿货」的答案。
// Discount 是拿货折扣（固定 6 位小数字符串，与折扣在库里的口径一致）；
// CostRatio 与 ChannelName 指出这个价是从哪条线路的成本算出来的，
// 只用于试算展示与后台排查，不参与计费。
type AgentWholesale struct {
	Discount    string
	CostRatio   string
	ChannelName string
}

// ResolveAgentWholesale 算出经销商自己消费时的拿货价。
//
// 返回 (nil, nil) 的几种情形一律是「算不出来，按原价」：
//  1. 这个用户不是经销商（没有经营档案）；
//  2. 他分组下这个模型没有可用线路；
//  3. 可用线路都没录进货折扣，或录的数字解析不出来（含 0）；
//  4. 档案里那档加价率解析不出来（数据坏了）；
//  5. 按成本加价后已经不低于官方标价——这套体系不允许加价（见 NormalizeDiscount）。
//
// 后三种都是在没有依据的情况下算出来的数，猜一个价会把平台的钱算亏，
// 所以宁可这一单按原价收。数据库真出错则返回错误，由调用方决定怎么处置。
func ResolveAgentWholesale(userId int, modelName string) (*AgentWholesale, error) {
	modelName = strings.TrimSpace(modelName)
	if userId <= 0 || modelName == "" {
		return nil, nil
	}

	// 这里读整行而不是挑列：分组列在库里是保留字，列名要靠 initCol() 按方言拼出来，
	// 而 initCol() 只在 InitDB() 里跑过。整行读不依赖它，代价只是多读几列。
	var user User
	if err := DB.First(&user, userId).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	// 身份读 users 上的快路径字段，跟经营档案同源（设/取消经销商都在同一事务里改它）。
	if user.SubjectType != SubjectTypeAgent {
		return nil, nil
	}

	// 加价率只在经营档案上，所以这里必须碰一次 agent_profiles（按 user_id 唯一索引）。
	// 档案和身份理论上不会不一致；真遇上不一致（身份是经销商但档案没了）也按原价走。
	profile, err := GetAgentProfileByUserId(userId)
	if err != nil {
		if errors.Is(err, ErrAgentProfileNotFound) {
			return nil, nil
		}
		return nil, err
	}
	markup, err := decimal.NewFromString(strings.TrimSpace(profile.EffectiveMarkupRatio()))
	if err != nil {
		return nil, nil
	}

	candidates, err := GetDiscountChannelCandidates(agentWholesaleGroups(user.Group), modelName)
	if err != nil {
		return nil, err
	}
	lowestCost, channelName, ok := lowestCostCandidate(candidates)
	if !ok {
		return nil, nil
	}

	price := lowestCost.Mul(markup)
	if price.LessThanOrEqual(decimal.Zero) || price.GreaterThanOrEqual(decimal.NewFromInt(1)) {
		return nil, nil
	}
	return &AgentWholesale{
		Discount:    price.StringFixed(6),
		CostRatio:   lowestCost.StringFixed(6),
		ChannelName: channelName,
	}, nil
}

// agentWholesaleGroups 算经销商自己消费时该看哪些分组的线路。
// 与试算、保存前校验同源：分组为空按 default；配成 auto 的展开成全局自动分组。
func agentWholesaleGroups(userGroup string) []string {
	group := strings.TrimSpace(userGroup)
	switch {
	case group == "":
		return []string{"default"}
	case group == "auto":
		groups := setting.GetAutoGroups()
		if len(groups) == 0 {
			return []string{"default"}
		}
		return groups
	default:
		return []string{group}
	}
}

// lowestCostCandidate 在候选线路里挑进货折扣最低的一条，返回它的折扣与线路名。
// 没录进货折扣、或录的数字解析不出来（含 0）的线路一律跳过——它们给不出成本，
// 与试算、保存前校验的处理保持一致。
func lowestCostCandidate(candidates []*DiscountChannelCandidate) (decimal.Decimal, string, bool) {
	var lowest decimal.Decimal
	channelName := ""
	found := false
	for _, candidate := range candidates {
		if candidate == nil || candidate.CostRatio == nil {
			continue
		}
		cost, err := decimal.NewFromString(strings.TrimSpace(*candidate.CostRatio))
		if err != nil || cost.LessThanOrEqual(decimal.Zero) {
			continue
		}
		if !found || cost.LessThan(lowest) {
			lowest = cost
			channelName = candidate.ChannelName
			found = true
		}
	}
	return lowest, channelName, found
}
