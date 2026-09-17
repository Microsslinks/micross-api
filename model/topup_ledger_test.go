package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestTopUpWritesAccountLedger 守住 task-20 §20.1：topup 完成时必须写一行 ledger，
// commission 写账已守护的合约在此复用：append-only + balance_after 一致 + ref 指向
// top_ups.id。RechargeEpay 系列用例覆盖幂等与状态机，本用例覆盖 ledger 合约。
func TestTopUpWritesAccountLedger(t *testing.T) {
	// 用一个空 user + 一笔 topup，模拟"用户首次充值"路径
	user := &User{
		Id:       9001,
		Username: "ledger-topup-user",
		AffCode:  "ledger-topup-aff",
		Status:   common.UserStatusEnabled,
		Quota:    0,
	}
	require.NoError(t, DB.Create(user).Error)

	topUp := &TopUp{
		UserId:          user.Id,
		Amount:          1000, // 1000 cents
		Money:           10.0,
		PaymentMethod:   "stripe",
		TradeNo:         "test-trade-001",
		Status:          common.TopUpStatusPending,
		PaymentProvider: PaymentProviderStripe,
	}
	require.NoError(t, topUp.Insert())

	// 走 stripe 完成路径（model/topup.go:payOrderToStripe 之类）
	// 这里直接调 creditTopUpQuota 是反模式——更接近真实路径的是用 ManualCompleteTopUp
	// 或封装过的充值完成函数。但本测试只验证 ledger 合约，所以直接调底层更稳。
	err := DB.Transaction(func(tx *gorm.DB) error {
		return creditTopUpQuota(tx, user.Id, 5000, nil, int64(topUp.Id))
	})
	require.NoError(t, err)

	// 验证 ledger 行
	var ledgers []AccountLedger
	require.NoError(t, DB.Where("subject_id = ? AND event_type = ?", user.Id, AccountEventTopup).
		Order("id ASC").Find(&ledgers).Error)
	require.Len(t, ledgers, 1, "topup 必须写 1 行 ledger")
	assert.Equal(t, int64(5000), ledgers[0].Amount)
	assert.Equal(t, "top_up", ledgers[0].RefType)
	assert.Equal(t, int64(topUp.Id), ledgers[0].RefId)
	assert.Equal(t, int64(5000), ledgers[0].BalanceAfter, "ledger balance_after 必须等于 user.quota")

	// 用户余额也对账
	var reload User
	require.NoError(t, DB.First(&reload, user.Id).Error)
	assert.Equal(t, 5000, reload.Quota)
	assert.Equal(t, int64(reload.Quota), ledgers[0].BalanceAfter)
}

// TestTopUpNoLedgerWithoutId 守住：手工 quota 调整（非 topup 流程）走 0 id 入参，
// creditTopUpQuota 不写 ledger——避免 audit 误把管理员手动维护算成"充值"。
func TestTopUpNoLedgerWithoutId(t *testing.T) {
	user := &User{
		Id:       9002,
		Username: "manual-quota-user",
		AffCode:  "manual-quota-aff",
		Status:   common.UserStatusEnabled,
		Quota:    0,
	}
	require.NoError(t, DB.Create(user).Error)

	err := DB.Transaction(func(tx *gorm.DB) error {
		return creditTopUpQuota(tx, user.Id, 3000, nil, 0) // topUpId=0
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, DB.Model(&AccountLedger{}).
		Where("subject_id = ?", user.Id).
		Count(&count).Error)
	assert.Equal(t, int64(0), count, "topUpId=0 不写 ledger")
}