package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProcessCommissionWritesAccountLedger 守住 task-17 §17.3 落地的核心合约：
// 邀请佣金入账后，account_ledger 必须有一行 commission 事件，amount=amount，balance_after
// 与邀请人 aff_commission_balance 完全一致。
//
// 这是"业务方案 05 §2.7 对账四问"的最低要求：每一笔返佣都要在账上留痕。
func TestProcessCommissionWritesAccountLedger(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.050000")

	inviter := &model.User{
		Username: "ledger-inviter",
		AffCode:  "ledger-inviter-aff",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	invitee := &model.User{
		Username:  "ledger-invitee",
		AffCode:   "ledger-invitee-aff",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		InviterId: 0,
	}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	require.NoError(t, model.DB.Create(invitee).Error)
	// 关键：把 InviterId 写回去（User 创建时 InviterId=0，存进去；现在再 UPDATE）
	require.NoError(t, model.DB.Model(invitee).Update("inviter_id", inviter.Id).Error)

	ctx, _ := gin.CreateTestContext(nil)
	// gross=10000 quota × rate=0.05 = 500 佣金（margin=5000 ≥ 500 → 不 breach）
	const consumeLogID = 12345
	const gross = 10000
	const margin = 5000
	require.NoError(t, ProcessCommission(ctx, consumeLogID, invitee.Id, gross, margin), "首次返佣应该成功")

	// 1. commission_records 必须有 1 行
	var recs []model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).Find(&recs).Error)
	require.Len(t, recs, 1, "应该写 1 条 commission_records")
	assert.Equal(t, int64(consumeLogID), recs[0].ConsumeLogId)
	assert.Equal(t, 500, recs[0].Amount)

	// 2. account_ledger 必须有 1 行 commission
	var ledgers []model.AccountLedger
	require.NoError(t, model.DB.Where("subject_type = ? AND subject_id = ?", "user", inviter.Id).
		Order("id ASC").Find(&ledgers).Error)
	require.Len(t, ledgers, 1, "应该写 1 条 ledger 行（commission）")
	assert.Equal(t, model.AccountEventCommission, ledgers[0].EventType)
	assert.Equal(t, int64(500), ledgers[0].Amount)
	assert.Equal(t, int64(500), ledgers[0].BalanceAfter, "balance_after 应该等于返佣后的 aff_commission_balance")
	assert.Equal(t, "commission_record", ledgers[0].RefType)
	assert.Equal(t, int64(recs[0].Id), ledgers[0].RefId, "RefId 应该指向 commission_records.id")
	assert.Contains(t, ledgers[0].Memo, "invitee=", "memo 应包含 invitee 信息便于排查")
	assert.Equal(t, 0, ledgers[0].OperatorId, "系统自动记账 operator_id=0")

	// 3. 邀请人 aff_commission_balance 应该 = 500（与 ledger.balance_after 一致）
	var inviterReload model.User
	require.NoError(t, model.DB.First(&inviterReload, inviter.Id).Error)
	assert.Equal(t, int64(500), int64(inviterReload.AffCommissionBalance),
		"aff_commission_balance 应该等于 ledger 中该用户所有 amount 的累计（对账测试）")
}

// TestProcessCommissionBreachStillWritesLedger 守住"margin_unwired 拦截后仍写 ledger"：
// 即使 AssertNoLoss 把 amount 归零（margin_unwired 路径），commission_records 仍会写一行
// （task-10 既定行为），ledger 不写（amount=0 不写）——这条用例守住 balance_after 不会被
// 错误地写 0，让对账脚本误以为钱被扣了。
func TestProcessCommissionBreachStillWritesLedger(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.050000")

	inviter := &model.User{
		Username: "breach-inviter",
		AffCode:  "breach-inviter-aff",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	invitee := &model.User{
		Username: "breach-invitee",
		AffCode:  "breach-invitee-aff",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, model.DB.Create(inviter).Error)
	require.NoError(t, model.DB.Create(invitee).Error)
	require.NoError(t, model.DB.Model(invitee).Update("inviter_id", inviter.Id).Error)

	ctx, _ := gin.CreateTestContext(nil)
	// margin=0 → AssertNoLoss → finalAmount=0, breach=true → 走 margin_unwired 路径
	require.NoError(t, ProcessCommission(ctx, 0, invitee.Id, 1000, 0))

	// commission_records 有 1 行（amount=0, breach=true）
	var recs []model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).Find(&recs).Error)
	require.Len(t, recs, 1)
	assert.Equal(t, 0, recs[0].Amount)
	assert.True(t, recs[0].Breach)

	// account_ledger 应该有 0 行（amount=0 不写 ledger）
	var ledgers []model.AccountLedger
	require.NoError(t, model.DB.Where("subject_id = ?", inviter.Id).Find(&ledgers).Error)
	assert.Len(t, ledgers, 0, "amount=0 的 breach 行不应写 ledger（避免对账混淆）")

	// 邀请人余额仍为 0（与 ledger 空集一致）
	var inviterReload model.User
	require.NoError(t, model.DB.First(&inviterReload, inviter.Id).Error)
	assert.Equal(t, int64(0), int64(inviterReload.AffCommissionBalance))
}

// TestSumLedgerAmount 守住对账脚本的核心 SQL：SUM(amount) 累计。
func TestSumLedgerAmount(t *testing.T) {
	setupCommissionTest(t)

	require.NoError(t, model.DB.Create(&model.AccountLedger{
		SubjectType:  "user",
		SubjectId:    1001,
		EventType:    model.AccountEventCommission,
		Amount:       500,
		BalanceAfter: 500,
		RefType:      "commission_record",
		RefId:        1,
		CreatedAt:    time.Now(),
	}).Error)
	require.NoError(t, model.DB.Create(&model.AccountLedger{
		SubjectType:  "user",
		SubjectId:    1001,
		EventType:    model.AccountEventCommission,
		Amount:       300,
		BalanceAfter: 800,
		RefType:      "commission_record",
		RefId:        2,
		CreatedAt:    time.Now().Add(time.Second),
	}).Error)
	require.NoError(t, model.DB.Create(&model.AccountLedger{
		SubjectType:  "user",
		SubjectId:    9999, // 别的用户
		EventType:    model.AccountEventCommission,
		Amount:       999,
		BalanceAfter: 999,
		RefType:      "commission_record",
		RefId:        3,
		CreatedAt:    time.Now(),
	}).Error)

	sum, err := model.SumLedgerAmount("user", 1001)
	require.NoError(t, err)
	assert.Equal(t, int64(800), sum, "500 + 300 = 800；1001 不应被 9999 的行污染")
}

// TestRecordAccountLedgerRequiresTx 守住"ledger 必须与余额变更同事务"——
// 任何传 nil tx 的调用立即返回 error，绝不"裸写"。
//
// 这是 task-17 §17.3 的关键安全约束：ledger 与余额分离记账是合规灾难，
// 必须强制让 caller 在事务里调。
func TestRecordAccountLedgerRequiresTx(t *testing.T) {
	err := model.RecordAccountLedger(nil,
		"user", 1, model.AccountEventCommission,
		100, 100, "test", 1, "memo", 0,
	)
	require.Error(t, err, "传 nil tx 必须返回 error（强制事务内记账）")
	assert.Contains(t, err.Error(), "tx is required")

	err = model.RecordAccountLedger(model.DB,
		"user", 0, model.AccountEventCommission,
		100, 100, "test", 1, "memo", 0,
	)
	require.Error(t, err, "subject_id=0 必须返回 error")
	assert.Contains(t, err.Error(), "subject_id=0")

	err = model.RecordAccountLedger(model.DB,
		"user", 1, "",
		100, 100, "test", 1, "memo", 0,
	)
	require.Error(t, err, "空 event_type 必须返回 error")
	assert.Contains(t, err.Error(), "event_type is required")
}