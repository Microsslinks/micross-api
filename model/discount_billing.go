package model

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/shopspring/decimal"
)

// BillingDiscount 是「这一单要乘的折扣」——计费链路唯一需要的折扣形态。
//
// 折扣在库里按 decimal(10,6) 字符串存，计费要的是能直接乘的 float64，
// 中间还隔着一道总开关和一层合法性兜底。把它收成一个只读结构放在这里，
// 是为了让「预扣」「结算」「任务重算」几处拿到的一定是同一份答案——
// 分别去解析字符串，早晚会解析出两个不一样的值。
type BillingDiscount struct {
	// Ratio 是计费乘数。1 表示不打折：没绑方案、方案停用、开关关闭、数据异常，全归到这里。
	Ratio float64
	// Source 是折扣来源（DiscountResolvedFrom* 之一），未打折时为空。
	Source string
	// PlanId 是命中的折扣方案 id，未打折时为 0。
	PlanId int
	// Rejected 是这位客户身上、这一单没用上的那几条绑定（含落选原因）。
	// 不参与计算：只为了日志答得出"为什么不是按他另一套方案算"。
	Rejected []BillingDiscountRejection
}

// BillingDiscountRejection 是一条挂在这个客户身上、这一单没用上的绑定。
type BillingDiscountRejection struct {
	PlanId   int
	Source   string
	Discount string
	Reason   string
}

// Applied 表示这一单是否真的要按低于官方标价收费。
// 只有它为 true，调用方才把 Ratio 乘进倍率——这也是「没绑方案的用户扣费逐笔一致」的开关。
func (d BillingDiscount) Applied() bool {
	return d.Ratio != 1
}

// ResolveBillingDiscount 算出「这个客户 + 这个模型」计费时要乘的折扣。
//
// 四种情况一律回落到 1（不打折）：
//  1. 总开关关着——G1 内部阶段默认关，管理员显式打开后才参与计费；
//  2. 没有客户身份（userId <= 0，例如内部调用）；
//  3. 客户没绑方案，或绑的方案已停用（ResolveUserDiscountMulti 会给出 1.0）；
//  4. 折扣读不出来，或落到了 (0,1] 之外。
//
// 第 4 种必须记日志：折扣子系统出故障不能把整条计费链路带停，但也不能静默发生。
// 回落到「按官方标价计费」是保守侧——宁可这一单不打折，也不能因为一个读不到的数字
// 把客户的钱算错。所以这里只记 SysError，不往上抛错。
func ResolveBillingDiscount(userId int, modelName string) BillingDiscount {
	none := BillingDiscount{Ratio: 1}
	if !operation_setting.GetDiscountSetting().EnableBillingDiscount {
		return none
	}
	if userId <= 0 {
		return none
	}

	// 多方案解析：读这个人所有生效绑定按五层裁决，当前每人至多一条生效，
	// 行为与单方案解析一致；界面放开多挂后这里自动成为唯一计价口径。
	// 用 Detailed 版是为了顺带拿到落选的那几条——只多带了数据，不多查一次库。
	resolution, candidates, err := ResolveUserDiscountDetailed(userId, modelName)
	if err != nil {
		common.SysError(fmt.Sprintf("resolve billing discount failed, userId=%d, model=%s: %s", userId, modelName, err.Error()))
		return none
	}
	if resolution == nil {
		return none
	}

	parsed, err := decimal.NewFromString(strings.TrimSpace(resolution.Discount))
	if err != nil {
		common.SysError(fmt.Sprintf("billing discount is not a number, userId=%d, model=%s, discount=%q: %s", userId, modelName, resolution.Discount, err.Error()))
		return none
	}
	value := parsed.InexactFloat64()
	if value <= 0 || value > 1 {
		common.SysError(fmt.Sprintf("billing discount out of range, userId=%d, model=%s, discount=%s", userId, modelName, resolution.Discount))
		return none
	}
	if value == 1 {
		return none
	}

	return BillingDiscount{
		Ratio:    value,
		Source:   resolution.Source,
		PlanId:   resolution.PlanId,
		Rejected: collectRejectedBillingCandidates(candidates),
	}
}

// collectRejectedBillingCandidates 挑出这一单落选的绑定。
// 只挂一套方案的客户这里是空的（没人跟它抢），所以绝大多数请求不会多出这几个字段。
func collectRejectedBillingCandidates(candidates []*DiscountCandidate) []BillingDiscountRejection {
	rejected := make([]BillingDiscountRejection, 0, len(candidates))
	for _, candidate := range candidates {
		if !candidate.Rejected {
			continue
		}
		reason := candidate.RejectReason
		if reason == "" {
			// 裁决没写明原因时也不能留空：读日志的人会以为字段坏了。
			reason = "未被选中"
		}
		rejected = append(rejected, BillingDiscountRejection{
			PlanId:   candidate.Binding.PlanId,
			Source:   candidate.Binding.Source,
			Discount: candidate.Discount,
			Reason:   reason,
		})
	}
	return rejected
}
