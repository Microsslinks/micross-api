package service

import (
	"errors"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"gorm.io/gorm"
)

// 客户查价（task-06 第四步「一键查价」）：客服、销售问「这个客户调某某模型多少钱」，
// 一次问完就答完——他身上挂了哪几套价、现价按几折、命中的是哪套、其余几套输在哪一层、
// 每个模型还有几条线路、毛利多少。
//
// 与「客户报价核算」的分工：核算面向「这份报价能不能签」（贴一整份清单、看汇总是否可签），
// 查价面向「现在是什么价」（看身上有几套价、每套管什么、落选的差在哪）。
// 两者不重算两套钱的口径：折扣、线路、毛利直接取核算的逐模型结果；候选与落选原因取
// 多方案裁决本身给的。因此「查价显示的价」就是「扣费的那个价」，
// 两处对不上就是 bug（见 discount_price_query_test.go）。

// CustomerPriceQueryModel 一个模型的查价行：核算行 + 这一行的候选明细（谁赢了、谁输在哪一层）。
type CustomerPriceQueryModel struct {
	CustomerPricingAuditModel
	Candidates []*DiscountSimulateCandidate `json:"candidates"`
}

// CustomerPriceQueryResult 查价的完整答案。
// 只填了客户没填模型时 Models 为空、Summary 为零值——价目本本身就是要先看的东西。
type CustomerPriceQueryResult struct {
	User           DiscountSimulateUser            `json:"user"`
	MinMarginRatio string                          `json:"min_margin_ratio"`
	PriceBook      []*model.CustomerPriceBookEntry `json:"price_book"`
	Models         []*CustomerPriceQueryModel      `json:"models"`
	Summary        CustomerPricingAuditSummary     `json:"summary"`
	Warnings       []string                        `json:"warnings"`
}

// QueryCustomerPricing 查「这个客户 + 这些模型」的现价。
// modelNames 为空（或全是空白）时只回价目本：先把客户身上有几套价看清楚，再决定问哪个模型。
func QueryCustomerPricing(userId int, modelNames []string) (*CustomerPriceQueryResult, error) {
	user, err := model.GetUserById(userId, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerAuditUserNotFound
		}
		return nil, err
	}
	priceBook, err := model.GetCustomerPriceBook(userId)
	if err != nil {
		return nil, err
	}

	result := &CustomerPriceQueryResult{
		User:           DiscountSimulateUser{Id: user.Id, Username: user.Username, Group: user.Group},
		MinMarginRatio: normalizeDiscountRatioText(operation_setting.GetDiscountSetting().MinMarginRatio),
		PriceBook:      priceBook,
		Models:         make([]*CustomerPriceQueryModel, 0),
		Warnings:       make([]string, 0),
	}

	// 与核算同一份清单修剪规则（换行、中英文逗号、顿号都容得下，空格不算分隔符）。
	nameList := auditModelNames(modelNames)
	if len(nameList) == 0 {
		result.Warnings = append(result.Warnings, "还没填模型名：先看上方的价目本，填个模型名再查现价")
		return result, nil
	}
	if len(nameList) > CustomerAuditMaxModels {
		return nil, ErrCustomerAuditTooManyModels
	}

	audit, err := AuditCustomerPricing(userId, nameList)
	if err != nil {
		return nil, err
	}
	// 候选明细（含落选原因）只有多方案裁决会给，核算结果里没有这一层：按同一份模型清单
	// 再解析一次取候选。口径没变——用的就是扣费那条路；钱的数字仍取上面的核算结果。
	_, candidatesByModel, err := model.ResolveUserDiscountDetailedForModels(userId, nameList)
	if err != nil {
		return nil, err
	}

	result.MinMarginRatio = audit.MinMarginRatio
	result.Summary = audit.Summary
	result.Warnings = audit.Warnings
	for _, row := range audit.Models {
		result.Models = append(result.Models, &CustomerPriceQueryModel{
			CustomerPricingAuditModel: *row,
			Candidates:                buildSimulateCandidates(candidatesByModel[row.Model]),
		})
	}
	return result, nil
}
