package model

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

// 经销商经营档案的修改。
//
// 设成经销商时（agent.go）档案一次写全；这里管的是之后的调整：平台改这位经销商的
// 加价率、给他定一个零售折扣下限、停/开他的发额度能力、改备注。
// 四个字段都是可选的——传哪个改哪个，没传的保持原值，避免"改备注顺手把加价率清零"。

// AgentProfileSettings 是一次档案调整里可以带上来的字段；nil 表示这次不动它。
type AgentProfileSettings struct {
	MarkupRatio       *string
	MinDiscount       *string
	IssueQuotaEnabled *int
	Remark            *string
}

// NormalizeAgentMinDiscount 校验经销商的零售折扣下限并规范成 6 位小数。
//
// 0 表示平台不给这位经销商设下限（他给客户定多低都行）；除此之外必须在 (0, 1] 之间。
// 大于 1 等于要求客户按高于官方标价付费，那是把参数填错了，直接拒绝。
func NormalizeAgentMinDiscount(raw string) (string, error) {
	value, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return "", errors.New("最低折扣必须是数字")
	}
	if value.IsNegative() {
		return "", errors.New("最低折扣不能为负数")
	}
	if value.GreaterThan(decimal.NewFromInt(1)) {
		return "", errors.New("最低折扣不能大于 1")
	}
	return value.StringFixed(6), nil
}

// UpdateAgentProfileSettings 把本次带上来、且调用方已规范过的字段写进档案。
//
// 档案不存在（这个人不是经销商）时返回 ErrAgentProfileNotFound，由调用方决定怎么回话。
// 这里只写档案该管的那几列，不碰身份（users.subject_type）与余额。
func UpdateAgentProfileSettings(userId int, settings AgentProfileSettings) (*AgentProfile, error) {
	profile, err := GetAgentProfileByUserId(userId)
	if err != nil {
		return nil, err
	}
	if settings.MarkupRatio != nil {
		profile.MarkupRatio = *settings.MarkupRatio
	}
	if settings.MinDiscount != nil {
		profile.MinDiscount = *settings.MinDiscount
	}
	if settings.IssueQuotaEnabled != nil {
		profile.IssueQuotaEnabled = *settings.IssueQuotaEnabled
	}
	if settings.Remark != nil {
		remark := strings.TrimSpace(*settings.Remark)
		if utf8.RuneCountInString(remark) > 255 {
			return nil, errors.New("备注不能超过 255 个字符")
		}
		profile.Remark = remark
	}
	// 这一行的加价率可能还是空的（建档时这一列还不存在），落库前落到缺省档上。
	profile.NormalizeDefaults()

	// 显式列出列名：issue_quota_enabled 取 0（停发）时不能被 GORM 的零值规则跳过。
	if err := DB.Model(&AgentProfile{}).Where("id = ?", profile.Id).
		Select("markup_ratio", "min_discount", "issue_quota_enabled", "remark", "updated_at").
		Updates(profile).Error; err != nil {
		return nil, err
	}
	return profile, nil
}
