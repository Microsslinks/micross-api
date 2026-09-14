package helper

import (
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	hosttypes "github.com/QuantumNous/new-api/types"
)

// ApplyUserDiscountRatio 把「客户折扣」合进分组倍率。
//
// 为什么合进去、而不是另开一个乘数位：分组倍率本来就是
// 「客户实付 = 官方标价 × 分组倍率」里的那个倍率，客户折扣是同一件事的第二层。
// 合成一个数之后，预扣、结算、重试、任务计费读到的必然是同一个值——
// 分成两处最容易出的错就是「预扣按折后算、结算按原价扣」，差额退还又是另一笔账。
//
// 没打折时原样返回（连折扣字段都不填），保证未绑方案的请求与改造前逐笔一致。
func ApplyUserDiscountRatio(relayInfo *relaycommon.RelayInfo, groupRatioInfo hosttypes.GroupRatioInfo) hosttypes.GroupRatioInfo {
	if relayInfo == nil {
		return groupRatioInfo
	}
	discount := model.ResolveBillingDiscount(relayInfo.UserId, relayInfo.OriginModelName)
	if !discount.Applied() {
		return groupRatioInfo
	}
	groupRatioInfo.GroupRatio = groupRatioInfo.GroupRatio * discount.Ratio
	groupRatioInfo.DiscountRatio = discount.Ratio
	groupRatioInfo.DiscountSource = discount.Source
	groupRatioInfo.DiscountPlanId = discount.PlanId
	groupRatioInfo.DiscountRejected = convertDiscountRejections(discount.Rejected)
	return groupRatioInfo
}

// convertDiscountRejections 把落选的绑定搬进 PriceData。
//
// 两个包各有一份同名结构（types 不允许反向依赖 model），所以转换只能在这里做。
// 没落选时给 nil，日志里就不会多出这个字段——老请求的日志形状一个字都不变。
func convertDiscountRejections(rejected []model.BillingDiscountRejection) []hosttypes.DiscountRejection {
	if len(rejected) == 0 {
		return nil
	}
	notes := make([]hosttypes.DiscountRejection, 0, len(rejected))
	for _, item := range rejected {
		notes = append(notes, hosttypes.DiscountRejection{
			PlanId:   item.PlanId,
			Source:   item.Source,
			Discount: item.Discount,
			Reason:   item.Reason,
		})
	}
	return notes
}
