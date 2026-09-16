package operation_setting

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

var DemoSiteEnabled = false
var SelfUseModeEnabled = false

// CommissionRate 邀请佣金全局返佣率（口径 A①），DECIMAL(6,6) 字符串，默认 "0" 表示关闭。
//
// 业务方在运营后台改这个数（task-10 / P4 佣金核心）；修改落库 system_setting 后，
// 启动时由 setting/operation_setting/general_setting.go 的 SystemSetting 解析回填。
//
// 这里只声明 var + Getter/Setter；system_setting 持久化 + 后台 endpoint 在 task-10 Phase 3 完成。
// service/commission.go:CalculateCommission 读这个 var 计算原始返佣额。
var CommissionRate = "0"

func GetCommissionRate() string {
	return CommissionRate
}

func SetCommissionRate(v string) {
	CommissionRate = v
}

// ValidateCommissionRate 校验全局返佣率必须能解析为 decimal 且范围 [0, 1]。
//
// 任务文档 §三口径 A①：返佣率 0 = 关闭；>0 表示比例。>1（多倍返佣）禁止——避免邀请人
// 从平台拿走的比卖出去的多。负数 / NaN / 空串 同样拒绝。
// 返回 error 含具体原因，方便 admin 端点直接展示。
func ValidateCommissionRate(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("CommissionRate 不能为空")
	}
	parsed, err := decimal.NewFromString(trimmed)
	if err != nil {
		return fmt.Errorf("CommissionRate 无法解析为 decimal: %w", err)
	}
	if parsed.IsNegative() {
		return errors.New("CommissionRate 不能为负数")
	}
	if parsed.GreaterThan(decimal.NewFromInt(1)) {
		return errors.New("CommissionRate 不能大于 1（避免多倍返佣）")
	}
	return nil
}

var AutomaticDisableKeywords = []string{
	"Your credit balance is too low",
	"This organization has been disabled.",
	"You exceeded your current quota",
	"Permission denied",
	"The security token included in the request is invalid",
	"Operation not allowed",
	"Your account is not authorized",
}

func AutomaticDisableKeywordsToString() string {
	return strings.Join(AutomaticDisableKeywords, "\n")
}

func AutomaticDisableKeywordsFromString(s string) {
	AutomaticDisableKeywords = []string{}
	ak := strings.Split(s, "\n")
	for _, k := range ak {
		k = strings.TrimSpace(k)
		k = strings.ToLower(k)
		if k != "" {
			AutomaticDisableKeywords = append(AutomaticDisableKeywords, k)
		}
	}
}
