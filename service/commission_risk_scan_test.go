package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----------------------------------------------------------------------
// task-20 §20.7: commission risk scan cron 测试
//
// runCommissionRiskScanOnce 是 package-private 函数，被 main.go 周期调用。
// 测试直接调用它模拟一次 tick —— 不等 ticker，节省 CI 时间。
// ----------------------------------------------------------------------

// buildRing 三节点环：A→B→A。
//
//  userA
//   └── inviter=B → invitee=A(自己), 形成环
//  userB
//   └── inviter=A → invitee=B
//
// 顺序建库：A 的 inviter_id=B, B 的 inviter_id=A —— DetectInviteRing(A)
// 从 A 沿 inviter 上溯：seen={A}, walk A.inviter=B, seen={A,B}, walk B.inviter=A
// already in seen → return true。
func buildRing(t *testing.T) (userA, userB *model.User) {
	t.Helper()
	userA = &model.User{
		Username: "ring-A-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "ring-A-aff-" + fmt.Sprint(time.Now().UnixNano()),
		CreatedAt: time.Now().Unix() - 7*24*3600, // 7 天前注册，让 DetectInviteRing 不被"账户太新"短路
	}
	require.NoError(t, model.DB.Create(userA).Error)
	userB = &model.User{
		Username: "ring-B-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "ring-B-aff-" + fmt.Sprint(time.Now().UnixNano()),
		CreatedAt: time.Now().Unix() - 7*24*3600,
	}
	require.NoError(t, model.DB.Create(userB).Error)
	// 互设 inviter 形成环
	require.NoError(t, model.DB.Model(userA).Update("inviter_id", userB.Id).Error)
	require.NoError(t, model.DB.Model(userB).Update("inviter_id", userA.Id).Error)
	return userA, userB
}

// TestRunCommissionRiskScanOnce_AutoReversesInviteRing:
// 建 ring → commission_records 一条 → run scan → 自动 reverse。
func TestRunCommissionRiskScanOnce_AutoReversesInviteRing(t *testing.T) {
	setupCommissionTest(t)

	userA, userB := buildRing(t)
	// 给 ring 内的 inviter(A) 建一条 commission_records，让 DetectInviteRing 触发。
	now := common.GetTimestamp()
	rec := &model.CommissionRecord{
		InviterId: userA.Id, InviteeId: userB.Id,
		ConsumeLogId: now * 10,
		Gross: 100, Rate: "0.05", Amount: 5, Margin: 50,
		Currency: "USD",
		SettledAt: now,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(rec).Error)

	runCommissionRiskScanOnce()

	// 验证：record 已被 reverse
	var afterRec model.CommissionRecord
	require.NoError(t, model.DB.First(&afterRec, rec.Id).Error)
	assert.True(t, afterRec.Reversed, "成环 commission 应被 cron 自动 reverse")
	assert.Equal(t, 0, afterRec.ReversedBy, "cron 自动 reverse operator_id=0 区别于 admin")
	assert.Contains(t, afterRec.ReverseReason, "invite_ring")
}

// TestRunCommissionRiskScanOnce_AutoReversesFirstTopup:
// inviter 没真实消费记录 → first_topup_threshold 触发。
func TestRunCommissionRiskScanOnce_AutoReversesFirstTopup(t *testing.T) {
	setupCommissionTest(t)

	inviter := &model.User{
		Username: "topup-inv-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "topup-inv-aff-" + fmt.Sprint(time.Now().UnixNano()),
		AffCommissionBalance: 100, // 给钱包 +100 让 reverse 后能扣回
	}
	require.NoError(t, model.DB.Create(inviter).Error)
	invitee := &model.User{
		Username: "topup-invee-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		InviterId: inviter.Id,
		AffCode: "topup-invee-aff-" + fmt.Sprint(time.Now().UnixNano()),
	}
	require.NoError(t, model.DB.Create(invitee).Error)

	now := common.GetTimestamp()
	rec := &model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: invitee.Id,
		ConsumeLogId: now * 20,
		Gross: 100, Rate: "0.05", Amount: 5, Margin: 50,
		Currency: "USD",
		SettledAt: now,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(rec).Error)

	runCommissionRiskScanOnce()

	// 验证：record 被 reverse
	var afterRec model.CommissionRecord
	require.NoError(t, model.DB.First(&afterRec, rec.Id).Error)
	assert.True(t, afterRec.Reversed)
	assert.Contains(t, afterRec.ReverseReason, "first_topup_threshold")

	// 验证：钱包被扣回
	var afterInviter model.User
	require.NoError(t, model.DB.First(&afterInviter, inviter.Id).Error)
	assert.Equal(t, 95, afterInviter.AffCommissionBalance, "100 - 5 = 95")

	// 验证：ledger 行 commission_reverse
	var ledgerCount int64
	require.NoError(t, model.DB.Model(&model.AccountLedger{}).
		Where("ref_type = ? AND ref_id = ? AND event_type = ?",
			"commission_record", rec.Id, model.AccountEventCommissionReverse).
		Count(&ledgerCount).Error)
	assert.Equal(t, int64(1), ledgerCount, "应写 commission_reverse ledger 行")
}

// TestRunCommissionRiskScanOnce_SkipsHealthyRecords:
// 健康 inviter（有真实消费）→ scan 不应 reverse。
func TestRunCommissionRiskScanOnce_SkipsHealthyRecords(t *testing.T) {
	setupCommissionTest(t)

	inviter := &model.User{
		Username: "healthy-inv-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "healthy-inv-aff-" + fmt.Sprint(time.Now().UnixNano()),
		AffCommissionBalance: 100,
	}
	require.NoError(t, model.DB.Create(inviter).Error)
	invitee := &model.User{
		Username: "healthy-invee-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		InviterId: inviter.Id,
		AffCode: "healthy-invee-aff-" + fmt.Sprint(time.Now().UnixNano()),
	}
	require.NoError(t, model.DB.Create(invitee).Error)

	// 给 inviter 喂首充门槛之上的消费记录
	require.NoError(t, model.DB.Create(&model.Log{
		UserId: inviter.Id, Type: model.LogTypeConsume,
		Quota: 100000, // 远 > firstTopupThresholdQuota (5000)
	}).Error)

	now := common.GetTimestamp()
	rec := &model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: invitee.Id,
		ConsumeLogId: now * 30,
		Gross: 100, Rate: "0.05", Amount: 5, Margin: 50,
		Currency: "USD",
		SettledAt: now,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(rec).Error)

	runCommissionRiskScanOnce()

	// 验证：record 没被 reverse
	var afterRec model.CommissionRecord
	require.NoError(t, model.DB.First(&afterRec, rec.Id).Error)
	assert.False(t, afterRec.Reversed, "健康 inviter 的 commission 不应被 cron reverse")

	// 验证：钱包不动
	var afterInviter model.User
	require.NoError(t, model.DB.First(&afterInviter, inviter.Id).Error)
	assert.Equal(t, 100, afterInviter.AffCommissionBalance, "钱包不应动")
}

// TestRunCommissionRiskScanOnce_Respects7DayWindow:
// 旧 record（> 7 天）→ 不被扫描。
// 用 SettledAt 设到 8 天前即可（scan 内 cutoff = now - 7d）。
func TestRunCommissionRiskScanOnce_Respects7DayWindow(t *testing.T) {
	setupCommissionTest(t)

	inviter := &model.User{
		Username: "old-inv-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		AffCode: "old-inv-aff-" + fmt.Sprint(time.Now().UnixNano()),
		AffCommissionBalance: 50,
	}
	require.NoError(t, model.DB.Create(inviter).Error)
	invitee := &model.User{
		Username: "old-invee-" + fmt.Sprint(time.Now().UnixNano()),
		Password: "unused", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default",
		InviterId: inviter.Id,
		AffCode: "old-invee-aff-" + fmt.Sprint(time.Now().UnixNano()),
	}
	require.NoError(t, model.DB.Create(invitee).Error)

	eightDaysAgo := common.GetTimestamp() - int64(8*24*3600)
	rec := &model.CommissionRecord{
		InviterId: inviter.Id, InviteeId: invitee.Id,
		ConsumeLogId: eightDaysAgo * 40,
		Gross: 100, Rate: "0.05", Amount: 5, Margin: 50,
		Currency: "USD",
		SettledAt: eightDaysAgo,
		CreatedAt: eightDaysAgo,
	}
	require.NoError(t, model.DB.Create(rec).Error)

	runCommissionRiskScanOnce()

	// 验证：8 天前的 record 没被扫（仍 Reversed=false）
	var afterRec model.CommissionRecord
	require.NoError(t, model.DB.First(&afterRec, rec.Id).Error)
	assert.False(t, afterRec.Reversed, "7 天窗口外的 record 不应被扫")
}

// TestRunCommissionRiskScanOnce_Idempotent: 重复跑不会二次扣款。
func TestRunCommissionRiskScanOnce_Idempotent(t *testing.T) {
	setupCommissionTest(t)

	_, userB := buildRing(t)
	now := common.GetTimestamp()
	rec := &model.CommissionRecord{
		InviterId: userB.Id, InviteeId: 99999, // 也是 ring 里的 user (DetectInviteRing(B) 也会 true)
		ConsumeLogId: now * 50,
		Gross: 100, Rate: "0.05", Amount: 5, Margin: 50,
		Currency: "USD",
		SettledAt: now,
		CreatedAt: now,
	}
	require.NoError(t, model.DB.Create(rec).Error)

	// 给 userB 设 AffCommissionBalance 让首次 reverse 扣回可见
	require.NoError(t, model.DB.Model(userB).Update("aff_commission_balance", 100).Error)

	runCommissionRiskScanOnce()
	runCommissionRiskScanOnce() // 第二次跑不应再扣

	var afterRec model.CommissionRecord
	require.NoError(t, model.DB.First(&afterRec, rec.Id).Error)
	assert.True(t, afterRec.Reversed)

	var afterUserB model.User
	require.NoError(t, model.DB.First(&afterUserB, userB.Id).Error)
	// 100 - 5 = 95 (只扣一次)
	assert.Equal(t, 95, afterUserB.AffCommissionBalance, "二次 scan 不应再扣")
}