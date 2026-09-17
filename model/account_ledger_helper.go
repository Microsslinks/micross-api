package model

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// AccountLedgerEventType 已知事件类型（白名单）。
//
// 不是枚举——go 没枚举；这里只列**已有 caller** 的字符串，其他事件类型
// 由 caller 自由扩展。_reverse 后缀用于冲销，是 grep 上的约定。
const (
	AccountEventCommission        = "commission"         // P4 返佣入账
	AccountEventCommissionReverse = "commission_reverse" // P4 返佣冲销（退款 / 风控）
	AccountEventTopup             = "topup"              // 充值（task-17 §17.3 后续 PR）
	AccountEventRefund            = "refund"             // 退款（task-17 §17.3 后续 PR）
	AccountEventAgentQuotaGrant   = "agent_quota_grant"  // 经销商给客户发额度（后续 PR）
	AccountEventAdjustment        = "adjustment"         // 超管手动调整
)

// RecordAccountLedger 在事务里写一行账。
//
// 必须在调用方的事务里调用（不自动开事务）——写账必须与余额变更同事务，
// 否则一笔钱可能"动了余额但没记账"或"记账了但余额没动"。
//
// 参数：
//   - tx: 调用方的事务
//   - subjectType: "user" 或 "agent"（暂统一用 "user"，与 users 表一致）
//   - subjectId: 用户 id
//   - eventType: 事件类型常量（AccountEvent*）
//   - amount: 正数=入账，负数=出账
//   - balanceAfter: 入账后余额快照（caller 必须在事务内用 row lock 算出）
//   - refType / refId: 引用类型与 id（commission_records.id / top_up.id / consume_log.id）
//   - memo: 备注（人类可读，列出关键字段值）
//   - operatorId: 0=系统；>0=超管 user_id
//
// 返回：error。caller 应当把此错误向上抛——账记错就是错，宁可让外层事务回滚。
//
// 设计：append-only。这函数不 UPDATE / DELETE 已有行；若 caller 想冲销，
// 再调一次 RecordAccountLedger(amount = -原 amount)，用 commission_reverse 标识。
func RecordAccountLedger(
	tx *gorm.DB,
	subjectType string,
	subjectId int,
	eventType string,
	amount int64,
	balanceAfter int64,
	refType string,
	refId int64,
	memo string,
	operatorId int,
) error {
	if tx == nil {
		return errors.New("RecordAccountLedger: tx is required (ledger write must be inside caller transaction)")
	}
	if subjectId <= 0 {
		return fmt.Errorf("RecordAccountLedger: subject_id=%d invalid", subjectId)
	}
	if eventType == "" {
		return errors.New("RecordAccountLedger: event_type is required")
	}
	ledger := &AccountLedger{
		SubjectType:  subjectType,
		SubjectId:    subjectId,
		EventType:    eventType,
		Amount:       amount,
		BalanceAfter: balanceAfter,
		Currency:     "USD",
		RefType:      refType,
		RefId:        refId,
		Memo:         memo,
		OperatorId:   operatorId,
	}
	return tx.Create(ledger).Error
}

// SumLedgerAmount 算一个用户在 ledger 上的累计金额（用于对账脚本）。
// 不含 status 过滤——append-only 表没有 status 字段，所有行都算。
//
// 对账用法（参见 scripts/account_audit.go 的伪代码）：
//
//	expected := ledgerSum - topUpSum + refundSum
//	actual   := user.quota + user.aff_commission_balance
//	if expected != actual { ... 报警 ... }
func SumLedgerAmount(subjectType string, subjectId int) (int64, error) {
	var sum int64
	err := DB.Model(&AccountLedger{}).
		Where("subject_type = ? AND subject_id = ?", subjectType, subjectId).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error
	if err != nil {
		return 0, err
	}
	return sum, nil
}