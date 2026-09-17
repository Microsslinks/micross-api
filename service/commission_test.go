package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupCommissionTest 给 commission service 用例一份干净的内存 SQLite + 还原 CommissionRate。
//
// 与 model/commission_migrate_test.go:setupCommissionMigrateTest 同模式，但额外
// 还原 operation_setting.CommissionRate（test 间不能污染全局运营设置）。
func setupCommissionTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousRate := operation_setting.CommissionRate

	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:commission-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	// Log 表让 TestProcessCommissionBreachWritesAudit 能 SELECT 验证 audit 日志；
	// User + CommissionRecord 是 ProcessCommission 直接读写两张表。
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.CommissionRecord{}, &model.Log{},
		// task-17 §17.3：ProcessCommission 写 ledger 必须在 setup 里建 account_ledger 表，
		// 否则 SQLite 报 "no such table: account_ledger"。
		&model.AccountLedger{},
	))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		operation_setting.SetCommissionRate(previousRate)
		_ = sqlDB.Close()
	})
}

// TestCalculateCommissionGrossTimesRate
//
// 基本算术：gross=100, rate=0.05 → 5（整数 quota，Round(0) 不留小数）。
func TestCalculateCommissionGrossTimesRate(t *testing.T) {
	assert.Equal(t, int64(5), CalculateCommission(100, "0.050000"))
	// 边界：gross 较大
	assert.Equal(t, int64(500), CalculateCommission(10000, "0.050000"))
	// 边界：rate=1.0 全额
	assert.Equal(t, int64(100), CalculateCommission(100, "1.000000"))
}

// TestCalculateCommissionZeroGrossReturnsZero
//
// gross=0 → 直接返回 0（不开返佣）。
func TestCalculateCommissionZeroGrossReturnsZero(t *testing.T) {
	assert.Equal(t, int64(0), CalculateCommission(0, "0.050000"))
}

// TestCalculateCommissionEmptyRateReturnsZero
//
// rate 空字符串 / "0" / 非正数 → 返回 0（运营未启用返佣）。
func TestCalculateCommissionEmptyRateReturnsZero(t *testing.T) {
	assert.Equal(t, int64(0), CalculateCommission(100, ""))
	assert.Equal(t, int64(0), CalculateCommission(100, "0"))
	assert.Equal(t, int64(0), CalculateCommission(100, "0.000000"))
	// 负数 rate 也应归零——不应该用负返佣率
	assert.Equal(t, int64(0), CalculateCommission(100, "-0.05"))
}

// TestAssertNoLossAmountUnderCap
//
// amount 低于 cap → 原值返回 + breach=false。
func TestAssertNoLossAmountUnderCap(t *testing.T) {
	final, breach := AssertNoLoss(5, 100)
	// margin=100 → cap = 80；amount=5 < 80 → 原值
	assert.Equal(t, int64(5), final)
	assert.False(t, breach)
}

// TestAssertNoLossAmountOverCapBreaches
//
// amount > cap → 降级到 cap + breach=true。
// 任务文档示例：margin=10, cap=8, raw=10 → 最终 8, breach=true。
func TestAssertNoLossAmountOverCapBreaches(t *testing.T) {
	final, breach := AssertNoLoss(10, 10)
	// margin=10 → cap = 8（10 × 0.8 = 8）；amount=10 > 8 → 降级到 8
	assert.Equal(t, int64(8), final)
	assert.True(t, breach)
}

// TestAssertNoLossZeroMarginReturnsZeroBreach
//
// margin<=0 → 归零 + breach=true（毛利为 0 不返佣）。
func TestAssertNoLossZeroMarginReturnsZeroBreach(t *testing.T) {
	final, breach := AssertNoLoss(5, 0)
	assert.Equal(t, int64(0), final)
	assert.True(t, breach)

	// 边界：margin 负数（不可能但要兜住）
	final, breach = AssertNoLoss(5, -10)
	assert.Equal(t, int64(0), final)
	assert.True(t, breach)
}

// seedFirstTopupConsume 给 inviter 喂一条满足 §20.5 首充门槛的 consume log。
// task-20 §20.5：ProcessCommission 加了首充门槛（inviter 累计消费 < 5000 quota
// 时 commission amount 归零）。现有测试的 inviter 大多没消费历史，跑旧测试会全 FAIL。
// 这个 helper 给 inviter 喂一条 10000 quota 的 consume log，门槛满足，测试期望
// 的 amount 行为不被 §20.5 影响。
//
// 调用时机：创建 inviter + invitee 之后、调 ProcessCommission 之前。
func seedFirstTopupConsume(t *testing.T, inviter *model.User) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.Log{
		UserId:    inviter.Id,
		Username:  inviter.Username,
		Type:      model.LogTypeConsume,
		Content:   "test seed consume to satisfy §20.5 first-topup threshold",
		Quota:     10000,
		CreatedAt: common.GetTimestamp(),
	}).Error)
}

// TestProcessCommissionNoInviterSkipped
//
// invitee.InviterId == 0（无邀请关系）→ 直接返回 nil，commission_records 不写。
func TestProcessCommissionNoInviterSkipped(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.05")

	// 建一个无邀请人的 invitee
	invitee := &model.User{
		Username:  "no-inviter-test",
		Password:  "unused",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		InviterId: 0,
		AffCode:   uniqueAffCode(t),
	}
	require.NoError(t, model.DB.Create(invitee).Error)

	// 跑 ProcessCommission——应直接返回 nil（无 inviteeId=invitee.Id 查不到 InviterId）
	require.NoError(t, ProcessCommission(nil, 0, invitee.Id, 100, 50))

	// commission_records 表应为空
	var count int64
	require.NoError(t, model.DB.Model(&model.CommissionRecord{}).Count(&count).Error)
	assert.Equal(t, int64(0), count, "无邀请关系应不写 commission_records")
}

// TestProcessCommissionInviterBalanceIncreased
//
// 正常路径：invitee.InviterId 非 0、rate 非 0、amount 非 0 → 写 commission_records +
// 邀请人钱包 += amount。
func TestProcessCommissionInviterBalanceIncreased(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.05")

	// 建邀请人 + 被邀请人
	inviter := &model.User{
		Username: "inviter-test",
		Password: "unused",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
		AffCode:  uniqueAffCode(t),
	}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	seedFirstTopupConsume(t, inviter)

	invitee := &model.User{
		Username:  "invitee-test",
		Password:  "unused",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		InviterId: inviter.Id,
		AffCode:   uniqueAffCode(t),
	}
	require.NoError(t, model.DB.Create(invitee).Error)

	// margin=50（足够覆盖 cap=40），gross=100 → rawAmount=5，finalAmount=5
	require.NoError(t, ProcessCommission(nil, 0, invitee.Id, 100, 50))

	// 1. commission_records 写了一条
	var rec model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).First(&rec).Error)
	assert.Equal(t, inviter.Id, rec.InviterId)
	assert.Equal(t, invitee.Id, rec.InviteeId)
	assert.Equal(t, int(100), rec.Gross)
	assert.Equal(t, "0.05", rec.Rate, "operation_setting.CommissionRate 原值写入，不补 0")
	assert.Equal(t, int(5), rec.Amount)
	assert.Equal(t, int(50), rec.Margin)
	assert.False(t, rec.Breach, "amount <= cap 时不应 breach")

	// 2. 邀请人钱包 +=5
	var inviter2 model.User
	require.NoError(t, model.DB.First(&inviter2, inviter.Id).Error)
	assert.Equal(t, 5, inviter2.AffCommissionBalance, "邀请人 aff_commission_balance 必须 += amount")
}

// TestProcessCommissionBreachWritesAudit
//
// breach=true 且 amount > 0 时写 audit 日志。
// 用 margin=10 + gross=100 + rate=0.05：rawAmount=5, cap=8（10×0.8），5<8 → 不 breach。
// 所以这里让 amount 远超 cap：rate=0.5, gross=100, margin=10 → rawAmount=50, cap=8, breach=true。
func TestProcessCommissionBreachWritesAudit(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.500000")

	inviter := &model.User{Username: "inviter-breach", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	invitee := &model.User{Username: "invitee-breach", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", InviterId: inviter.Id, AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(invitee).Error)

	// margin=10, cap=8; rate=0.5; gross=100 → rawAmount=50, breach=true, finalAmount=8
	require.NoError(t, ProcessCommission(nil, 0, invitee.Id, 100, 10))

	// 1. commission_records 写了一条 breach=true
	var rec model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).First(&rec).Error)
	assert.True(t, rec.Breach)
	assert.Equal(t, int(8), rec.Amount, "降级到 cap=8")
	assert.Equal(t, int(10), rec.Margin)

	// 2. 邀请人钱包 += 8
	var inviter2 model.User
	require.NoError(t, model.DB.First(&inviter2, inviter.Id).Error)
	assert.Equal(t, 8, inviter2.AffCommissionBalance)

	// 3. audit 日志写了一条 commission_breach，挂在邀请人（inviter.Id）名下
	var auditLog model.Log
	require.NoError(t, model.DB.Where("user_id = ? AND type = ?", inviter.Id, model.LogTypeManage).
		Order("created_at DESC").First(&auditLog).Error)
	assert.Contains(t, auditLog.Content, "commission_breach")
	assert.Contains(t, auditLog.Content, fmt.Sprintf("invitee=%d", invitee.Id))
	assert.Contains(t, auditLog.Content, fmt.Sprintf("inviter=%d", inviter.Id))

	// 4. admin_info JSON 嵌入了完整上下文
	assert.Contains(t, auditLog.Other, "admin_info", "audit 日志必须嵌 admin_info")
	assert.Contains(t, auditLog.Other, `"final_amount":8`)
}

// TestProcessCommissionMarginUnwiredAudit
//
// margin=0（cost_ratio 未接入的占位状态）→ 走 commission_margin_unwired 路径，
// 写 commission_records amount=0 + 一条 margin_unwired audit（区别于真实 breach）。
//
// 验证：log 内容关键词不同，运维 grep "commission_margin_unwired" 与 "commission_breach" 能区分。
func TestProcessCommissionMarginUnwiredAudit(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.05")

	inviter := &model.User{Username: "inviter-unwired", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	invitee := &model.User{Username: "invitee-unwired", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", InviterId: inviter.Id, AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(invitee).Error)

	// margin=0 占位，gross=100, rate=0.05 → rawAmount=5, finalAmount=0, breach=true
	// 期望：commission_records 写 amount=0；audit 写 "margin_unwired" 不是 "breach"
	require.NoError(t, ProcessCommission(nil, 0, invitee.Id, 100, 0))

	var rec model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).First(&rec).Error)
	assert.Equal(t, int(0), rec.Amount, "margin=0 时 amount 必须为 0")
	assert.True(t, rec.Breach)

	var auditLog model.Log
	require.NoError(t, model.DB.Where("user_id = ? AND type = ?", inviter.Id, model.LogTypeManage).
		Order("created_at DESC").First(&auditLog).Error)
	assert.Contains(t, auditLog.Content, "commission_margin_unwired",
		"margin=0 时应写 margin_unwired 而不是 breach，让运维清晰识别 cost_ratio 未接入")
	assert.NotContains(t, auditLog.Content, "commission_breach:",
		"不应写 breach 关键词（避免运维误判）")
}

// uniqueAffCode 给测试用例造唯一 aff_code，避免 users.aff_code uniqueIndex 冲突。
//
// 不能用时间戳：Windows 上 time.Now() 的分辨率不足以让同一测试里的两次创建取到不同值，
// 而 users.aff_code 是唯一索引——参考 model/agent_test.go:agentTestSeq 的做法。
var commissionTestSeq int

func uniqueAffCode(t *testing.T) string {
	t.Helper()
	commissionTestSeq++
	name := t.Name()
	if len(name) > 8 {
		name = name[:8]
	}
	return fmt.Sprintf("aff-%d-%d-%s", time.Now().UnixNano()%1000000, commissionTestSeq, name)
}

// TestProcessCommissionIdempotentOnRetry
//
// 同一笔消费 + 同一邀请人 重试场景：commission_records 的 uk_consume_inviter 唯一约束兜底，
// 第二次调 ProcessCommission 应该：
//   - 不 panic / 不向上抛错（tx.Create 报错时 return nil 吞掉）
//   - 不新增 commission_records（唯一约束冲突）
//   - 不重复加钱包（只第一次加了）
//
// 这是 README §七第 4 条防重试返佣的兜底。
func TestProcessCommissionIdempotentOnRetry(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.05")

	inviter := &model.User{Username: "inviter-retry", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	invitee := &model.User{Username: "invitee-retry", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", InviterId: inviter.Id, AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(invitee).Error)

	consumeLogId := int64(12345)

	// 第一次：正常返佣
	require.NoError(t, ProcessCommission(nil, consumeLogId, invitee.Id, 100, 50))

	var rec1 model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).First(&rec1).Error)
	assert.Equal(t, int(5), rec1.Amount)

	var inviterAfter1 model.User
	require.NoError(t, model.DB.First(&inviterAfter1, inviter.Id).Error)
	assert.Equal(t, 5, inviterAfter1.AffCommissionBalance, "第一次应 +5")

	// 第二次：同一 consume_log_id + 同一 invitee，应被唯一约束吞掉
	require.NoError(t, ProcessCommission(nil, consumeLogId, invitee.Id, 100, 50),
		"重试场景 ProcessCommission 不应向上抛错（DB 唯一约束兜底）")

	// 验证：commission_records 仍只有 1 条
	var count int64
	require.NoError(t, model.DB.Model(&model.CommissionRecord{}).
		Where("invitee_id = ?", invitee.Id).Count(&count).Error)
	assert.Equal(t, int64(1), count, "唯一约束阻止重复返佣，commission_records 应仍只有 1 条")

	// 验证：钱包仍只 +5，没被重复加
	var inviterAfter2 model.User
	require.NoError(t, model.DB.First(&inviterAfter2, inviter.Id).Error)
	assert.Equal(t, 5, inviterAfter2.AffCommissionBalance, "钱包不应被重复加")
}

// TestProcessCommissionNegativeGrossWritesNegativeAmount
//
// 退款场景：gross < 0 → rawAmount < 0 → finalAmount < 0 → 钱包 -= |amount|。
//
// 与正向消费共享同一 AssertNoLoss 路径：amount < cap（永远成立），不 breach，不写 audit。
// 这是 task-09 / task-10 一直强调的"反向扣减必须可"——退款时邀请人佣金要相应冲销。
func TestProcessCommissionNegativeGrossWritesNegativeAmount(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.050000")

	inviter := &model.User{Username: "inviter-refund", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	invitee := &model.User{Username: "invitee-refund", Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", InviterId: inviter.Id, AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(invitee).Error)

	// 先正向消费一笔，让钱包 = 5
	require.NoError(t, ProcessCommission(nil, 1, invitee.Id, 100, 50))

	// 再退款：gross=-100, rate=0.05 → rawAmount=-5, finalAmount=-5（小于 cap=40），不 breach
	require.NoError(t, ProcessCommission(nil, 2, invitee.Id, -100, 50))

	// 验证：钱包 = 5 + (-5) = 0
	var inviterAfter model.User
	require.NoError(t, model.DB.First(&inviterAfter, inviter.Id).Error)
	assert.Equal(t, 0, inviterAfter.AffCommissionBalance, "退款后钱包应归零（5 - 5）")

	// 验证：commission_records 有 2 条（正向 1 条 amount=5 + 退款 1 条 amount=-5）
	var records []model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).
		Order("id ASC").Find(&records).Error)
	assert.Len(t, records, 2)
	assert.Equal(t, int(5), records[0].Amount, "正向消费 amount=5")
	assert.Equal(t, int(-5), records[1].Amount, "退款 amount=-5")
	assert.False(t, records[1].Breach, "退款场景 amount<cap 不应 breach")

	// 验证：没有 audit 日志（负 amount 不写 audit）
	var auditCount int64
	require.NoError(t, model.DB.Model(&model.Log{}).
		Where("user_id = ? AND type = ?", inviter.Id, model.LogTypeManage).
		Count(&auditCount).Error)
	assert.Equal(t, int64(0), auditCount, "负 amount 不应写 audit")
}

// 这里的 _ = time.Now() 不会因 unused import 编译失败。
// （commission_records 需要时间戳字段，但 Go 静态检查不强制每个 time 引用都被用上，
// 上面的 fmt.Sprintf 没用 time，这里补一个无副作用的引用避免 unused import。）
var _ = time.Now

// ----------------------------------------------------------------------
// task-20 §20.6: ReverseCommission 测试（风控冲销 / admin 撤销 commission_records）
// ----------------------------------------------------------------------

// TestReverseCommissionRollsBackWalletAndWritesReverseLedger
// happy path: 写 commission → ReverseCommission → 钱包 -= amount + ledger commission_reverse
func TestReverseCommissionRollsBackWalletAndWritesReverseLedger(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.050000")

	operator := &model.User{Username: "rev-op-" + uniqueAffCode(t), Password: "unused", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "rev-op-" + uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(operator).Error)
	inviter := &model.User{Username: "rev-inv-" + uniqueAffCode(t), Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	invitee := &model.User{Username: "rev-invee-" + uniqueAffCode(t), Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", InviterId: inviter.Id, AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(invitee).Error)

	consumeLogId := int64(999001)
	require.NoError(t, ProcessCommission(nil, consumeLogId, invitee.Id, 200, 100),
		"写入一笔 commission amount=10（200 * 0.05）")

	// 验证：钱包已 +10
	require.NoError(t, model.DB.First(&inviter, inviter.Id).Error)
	assert.Equal(t, 10, inviter.AffCommissionBalance, "commission 入账后钱包应 +10")

	// 拿 commission_record id
	var rec model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).First(&rec).Error)
	assert.False(t, rec.Reversed, "初始 Reversed=false")

	// 触发 ReverseCommission
	require.NoError(t, ReverseCommission(nil, rec.Id, "风控扫描发现返佣为薅羊毛产出", operator.Id))

	// 验证：钱包被扣回
	require.NoError(t, model.DB.First(&inviter, inviter.Id).Error)
	assert.Equal(t, 0, inviter.AffCommissionBalance, "撤销后钱包应扣回 amount=10 → 0")

	// 验证：commission_record 标记已撤销
	require.NoError(t, model.DB.First(&rec, rec.Id).Error)
	assert.True(t, rec.Reversed, "Reversed=true")
	assert.Equal(t, operator.Id, rec.ReversedBy)
	assert.Contains(t, rec.ReverseReason, "风控扫描")

	// 验证：ledger 行 event_type=commission_reverse, amount=-10
	var ledger model.AccountLedger
	require.NoError(t, model.DB.Where("ref_type = ? AND ref_id = ? AND event_type = ?",
		"commission_record", rec.Id, model.AccountEventCommissionReverse).First(&ledger).Error)
	assert.Equal(t, model.AccountEventCommissionReverse, ledger.EventType)
	assert.Equal(t, int64(-10), ledger.Amount)
	assert.Equal(t, int64(0), ledger.BalanceAfter, "balance_after = 扣回后余额")
	assert.Equal(t, operator.Id, ledger.OperatorId)
	assert.Contains(t, ledger.Memo, "风控扫描")
}

// TestReverseCommissionIdempotent: 重复调用返回 ErrCommissionAlreadyReversed
func TestReverseCommissionIdempotent(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.050000")

	operator := &model.User{Username: "rev-idem-op-" + uniqueAffCode(t), Password: "unused", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(operator).Error)
	inviter := &model.User{Username: "rev-idem-inv-" + uniqueAffCode(t), Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(inviter).Error)
	seedFirstTopupConsume(t, inviter)
	invitee := &model.User{Username: "rev-idem-invee-" + uniqueAffCode(t), Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", InviterId: inviter.Id, AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(invitee).Error)

	consumeLogId := int64(999002)
	require.NoError(t, ProcessCommission(nil, consumeLogId, invitee.Id, 200, 100))

	var rec model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).First(&rec).Error)

	// 第一次撤销成功
	require.NoError(t, ReverseCommission(nil, rec.Id, "first reverse", operator.Id))

	// 第二次撤销失败
	err := ReverseCommission(nil, rec.Id, "second reverse", operator.Id)
	require.Error(t, err)
	assert.True(t, errors.Is(err, model.ErrCommissionAlreadyReversed),
		"二次撤销应返回 ErrCommissionAlreadyReversed, got: %v", err)

	// 验证：钱包只扣一次
	require.NoError(t, model.DB.First(&inviter, inviter.Id).Error)
	assert.Equal(t, 0, inviter.AffCommissionBalance, "二次撤销不应再扣款")
}

// TestReverseCommissionZeroAmountOnlyWritesAuditLedger:
// amount=0 的 commission 撤销不扣钱包，但仍写 ledger 留下审计痕迹。
// 典型场景：ProcessCommission 内 AssertNoLoss 触发 breach 的 commission_records，
// 这条记录 Amount=0 但 Breach=true，风控扫描发现后撤销只为审计。
func TestReverseCommissionZeroAmountOnlyWritesAuditLedger(t *testing.T) {
	setupCommissionTest(t)
	operation_setting.SetCommissionRate("0.050000")

	operator := &model.User{Username: "rev-zero-op-" + uniqueAffCode(t), Password: "unused", Role: common.RoleAdminUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(operator).Error)
	inviter := &model.User{Username: "rev-zero-inv-" + uniqueAffCode(t), Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(inviter).Error)
	// 不调 seedFirstTopupConsume —— 让 §20.5 首充门槛触发，amount=0
	invitee := &model.User{Username: "rev-zero-invee-" + uniqueAffCode(t), Password: "unused", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", InviterId: inviter.Id, AffCode: uniqueAffCode(t)}
	require.NoError(t, model.DB.Create(invitee).Error)

	consumeLogId := int64(999003)
	require.NoError(t, ProcessCommission(nil, consumeLogId, invitee.Id, 100, 50),
		"首充门槛未满足，amount 应为 0")

	var rec model.CommissionRecord
	require.NoError(t, model.DB.Where("invitee_id = ?", invitee.Id).First(&rec).Error)
	assert.Equal(t, 0, rec.Amount, "首充门槛触发后 Amount=0")

	require.NoError(t, ReverseCommission(nil, rec.Id, "audit only", operator.Id))

	// 验证：钱包不变（amount=0 → 无扣款）
	require.NoError(t, model.DB.First(&inviter, inviter.Id).Error)
	assert.Equal(t, 0, inviter.AffCommissionBalance, "amount=0 撤销不应动钱包")

	// 验证：ledger 行依然写入（amount=0 但 event_type=commission_reverse 留痕）
	var ledgerCount int64
	require.NoError(t, model.DB.Model(&model.AccountLedger{}).
		Where("ref_type = ? AND ref_id = ? AND event_type = ?",
			"commission_record", rec.Id, model.AccountEventCommissionReverse).
		Count(&ledgerCount).Error)
	assert.Equal(t, int64(1), ledgerCount, "amount=0 也写 ledger 留痕")
}

// TestReverseCommissionInvalidRecordID: 不存在的 ID 返回 error，不 panic
func TestReverseCommissionInvalidRecordID(t *testing.T) {
	err := ReverseCommission(nil, 999999999, "test invalid", 1)
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "panic")
	assert.False(t, errors.Is(err, model.ErrCommissionAlreadyReversed),
		"无效 ID 不应返回 ErrCommissionAlreadyReversed（区分 not found 与 already reversed）")
}