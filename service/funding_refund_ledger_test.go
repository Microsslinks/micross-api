package service

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWalletFundingRefundWritesAccountLedger 守住 task-20 §20.2：
// 调用 WalletFunding.Refund() 时必须写一行 ledger（event_type=refund），
// amount=consumed, balance_after=user.quota。ledger 与 quota 变更同事务。
//
// 这是 task-17 §17.3 commission 写账合约的复刻：append-only + balance_after 一致。
func TestWalletFundingRefundWritesAccountLedger(t *testing.T) {
	truncate(t) // 共用 task_billing_test.go 的 helper 清 users / account_ledger
	const userID = 9101
	const initialQuota = 100_000
	const refundAmount = 30_000

	user := &model.User{
		Id:       userID,
		Username: "wallet-refund-user",
		AffCode:  "wallet-refund-aff",
		Status:   common.UserStatusEnabled,
		Quota:    initialQuota,
	}
	require.NoError(t, model.DB.Create(user).Error)

	// 模拟失败 / 取消：WalletFunding 已经预扣 30k，现在 Refund 还回去。
	w := &WalletFunding{userId: userID, consumed: refundAmount}
	require.NoError(t, w.Refund())

	// 1. ledger 必须有 1 行 refund 事件
	var ledgers []model.AccountLedger
	require.NoError(t, model.DB.Where("subject_id = ? AND event_type = ?", userID, model.AccountEventRefund).
		Order("id ASC").Find(&ledgers).Error)
	require.Len(t, ledgers, 1, "WalletFunding.Refund 必须写 1 行 ledger")
	assert.Equal(t, int64(refundAmount), ledgers[0].Amount)
	assert.Equal(t, "billing_session_refund", ledgers[0].RefType,
		"refund 来源应标识 ref_type")
	assert.Equal(t, int64(0), ledgers[0].RefId,
		"refund 没关联到具体 topup/consume_log, RefId=0")

	// 2. 用户余额也对账
	var reload model.User
	require.NoError(t, model.DB.First(&reload, userID).Error)
	assert.Equal(t, initialQuota+refundAmount, reload.Quota,
		"余额应等于 initialQuota + refunded")
	assert.Equal(t, int64(reload.Quota), ledgers[0].BalanceAfter,
		"ledger.balance_after 必须等于 user.quota 同步快照")
}

// TestWalletFundingSettleNegativeWritesAccountLedger 守住：Settle(delta<0)
// 表示预扣超额（实际用了比预扣少的钱）→ 给用户加回 → 写 ledger。
func TestWalletFundingSettleNegativeWritesAccountLedger(t *testing.T) {
	truncate(t)
	const userID = 9102
	const initialQuota = 50_000
	const overReserve = 20_000

	user := &model.User{
		Id:       userID,
		Username: "wallet-settle-neg-user",
		AffCode:  "wallet-settle-neg-aff",
		Status:   common.UserStatusEnabled,
		Quota:    initialQuota,
	}
	require.NoError(t, model.DB.Create(user).Error)

	w := &WalletFunding{userId: userID}
	// Settle(20_000) 返回额度 = -(20_000) 即 "超额预扣了 20k" → 加回 20k
	require.NoError(t, w.Settle(-overReserve))

	var ledgers []model.AccountLedger
	require.NoError(t, model.DB.Where("subject_id = ? AND event_type = ?", userID, model.AccountEventRefund).
		Order("id ASC").Find(&ledgers).Error)
	require.Len(t, ledgers, 1)
	assert.Equal(t, int64(overReserve), ledgers[0].Amount)
	assert.Equal(t, "billing_session_settle", ledgers[0].RefType,
		"Settle 负数的 refund 来源与 Refund() 不同，便于排查")

	var reload model.User
	require.NoError(t, model.DB.First(&reload, userID).Error)
	assert.Equal(t, initialQuota+overReserve, reload.Quota)
}

// TestRefundWalletQuotaNoOpOnZero 守住：refundWalletQuota(quota=0) 直接 return，
// 不写 ledger。这是常见的"实际未消费 → 不需要 refund"边界。
func TestRefundWalletQuotaNoOpOnZero(t *testing.T) {
	truncate(t)
	const userID = 9103

	user := &model.User{
		Id:       userID,
		Username: "refund-noop-user",
		AffCode:  "refund-noop-aff",
		Status:   common.UserStatusEnabled,
		Quota:    1000,
	}
	require.NoError(t, model.DB.Create(user).Error)

	// 内部 helper：传 quota=0 应该早返回
	require.NoError(t, refundWalletQuota(userID, 0, "test_noop", 0))

	var count int64
	require.NoError(t, model.DB.Model(&model.AccountLedger{}).
		Where("subject_id = ?", userID).
		Count(&count).Error)
	assert.Equal(t, int64(0), count, "quota=0 does not write ledger")
}