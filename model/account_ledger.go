package model

import (
	"time"

	"gorm.io/gorm"
)

// AccountLedger 统一账本——记录一切动了 user.Quota / user.AffCommissionBalance 的事件。
//
// task-17 §17.3 落地：业务方案 05 §2.7「对账四问」硬要求"每一笔进出都有据可查"。
// 之前只有 commission_records（仅 P4 返佣）+ top_ups（充值）分散在多张表，
// 退款 / 补偿 / 经销商争议 / 风控冲销发生时无法做"用户余额 = Σ ledger"的原子对账。
//
// 设计要点：
//
//  1. **append-only**：从不 UPDATE / DELETE——错了写一条反向冲销，不动旧账。
//     冲销由 EventType 带 "_reverse" 后缀标识（如 commission → commission_reverse）。
//  2. **balance_after snapshot**：每行存余额快照，便于"任意一行回放余额曲线"。
//     计算方式 = 入账前余额 ± amount（由 caller 在事务里用 row lock 取最新值后填）。
//  3. **RefType / RefId**：指回事件来源（commission_records.id / top_up.id / consume_log.id）。
//     联合索引便于"按 ref 查 ledger 行"。
//  4. **subject_type / subject_id**：user 与 agent 都落 users.id（与 DiscountBinding 同口径），
//     索引 + Where("subject_type = ? AND subject_id = ?") 加速对账查询。
//  5. **Currency**：暂全部 USD；多币种时此列换成 decimal(10,4) + lookup 表。
//  6. **OperatorId**：0=系统自动；>0=超管手动（如调整、补偿）。
//
// 已知未写账的钩子（task-17 §17.3 后续 PR 补，不在本次 commit 范围）：
//   - 充值：service/topup.go → EventType="topup"
//   - 退款：service/refund.go → EventType="refund"
//   - 风控冲销：task-17.4 加风控时，commission_reverse 也写 ledger
//   - 经销商给客户发额度：model/agent_customers.go IssueQuotaToCustomer → EventType="agent_quota_grant"
//
// 对账脚本：scripts/account_audit.go
//   每用户扫：user.quota + user.aff_commission_balance - SUM(ledger.amount WHERE subject_id=u.id)
//   差额 ≠ 0 → 该用户有未记账事件，需人工补一笔反向 ledger。
type AccountLedger struct {
	Id           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	SubjectType  string    `json:"subject_type" gorm:"type:varchar(16);not null;index:idx_ledger_subject,priority:1"`
	SubjectId    int       `json:"subject_id" gorm:"not null;index:idx_ledger_subject,priority:2"`
	EventType    string    `json:"event_type" gorm:"type:varchar(32);not null;index:idx_ledger_event"`
	Amount       int64     `json:"amount"` // 正数=入账，负数=出账
	BalanceAfter int64     `json:"balance_after" gorm:"not null;default:0"`
	Currency     string    `json:"currency" gorm:"type:varchar(8);not null;default:'USD'"`
	RefType      string    `json:"ref_type" gorm:"type:varchar(32)"`
	RefId        int64     `json:"ref_id" gorm:"index:idx_ledger_ref"`
	Memo         string    `json:"memo" gorm:"type:text"`
	OperatorId   int       `json:"operator_id" gorm:"not null;default:0"` // 0=系统；>0=超管 user_id
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}

// TableName 显式声明，避免 GORM 命名推断把 AccountLedger 改成 account_ledgers 时
// 与旧版 DDL（如果历史库曾建过单数表）冲突。这里用单数 account_ledger 与「账本」语义对齐。
func (AccountLedger) TableName() string {
	return "account_ledger"
}

// BeforeCreate 自动填 CreatedAt（与项目其他表保持一致：Unix 秒 + caller 可覆盖）。
func (l *AccountLedger) BeforeCreate(tx *gorm.DB) error {
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	if l.Currency == "" {
		l.Currency = "USD"
	}
	return nil
}