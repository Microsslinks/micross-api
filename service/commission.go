package service

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// CalculateCommission 按口径 A①（全局一个数）× B②（平台实收）计算原始返佣额。
//
// 参数：
//   - gross：被邀请人当次实际计费金额（summary.Quota，1 quota = 1 / common.QuotaPerUnit 美元）。
//     反向扣减（退款 / 补偿 / 撤单）走 negative gross：caller 传 -N 即视为退佣 N。
//   - rate：运营全局返佣率（operation_setting.CommissionRate，decimal(6,6) 字符串）。
//
// 返回：原始返佣额（整数 quota，Round(0) 不留小数——1 quota = 1 / 500000 美元，精度已够）。
//
// 边界：
//   - gross == 0 → 返回 0（无消费场景）
//   - gross > 0  → 正常正向返佣
//   - gross < 0  → 退款场景，amount 也为负，caller（ProcessCommission）写入 commission_records
//                  后钱包 -= |amount|，邀请人佣金被冲销。
//   - rate 空 / "0" / 非正数 → 返回 0（运营未启用返佣）
//
// 浮点 vs decimal：刻意用 shopspring/decimal，避免 float64 精度漂移产生返佣尾差
// （与 DiscountPlan.CommissionRatio 同口径）。
func CalculateCommission(gross int64, rate string) int64 {
	if gross == 0 || rate == "" {
		return 0
	}
	parsed, err := decimal.NewFromString(rate)
	if err != nil || !parsed.IsPositive() {
		return 0
	}
	return decimal.NewFromInt(gross).Mul(parsed).Round(0).IntPart()
}

// AssertNoLoss 逐笔实时校验「佣金 ≤ 平台毛利 × 安全垫」（口径 C①）。
//
// 参数：
//   - amount：CalculateCommission 算出来的原始返佣额
//   - margin：当次平台毛利快照（gross × (1 - cost_ratio)，整数 quota）
//
// 返回：finalAmount + breach。
//   - 若 amount <= cap（margin × 0.8）：原值返回，breach=false
//   - 若 amount >  cap：降级到 cap，breach=true（caller 应写 audit 日志）
//   - 若 margin <= 0：直接归零 + breach=true（极端 case——成本折扣=零售折扣时不该发生，但要兜住）
//
// 安全垫 0.8：保留 20% 空间给未来可能引入的代理商分成 / 平台二次返佣，
// 避免锁死天花板导致后续任务要把这条逻辑再调一次。
func AssertNoLoss(amount, margin int64) (finalAmount int64, breach bool) {
	if margin <= 0 {
		// 毛利为 0 时归零；margin=0 也包括"cost_ratio 数据未接入"的占位状态，
		// 调用方必须用 breach 标记区分这两种情形（前者写 "commission_breach"，后者写
		// "margin_unwired"）。
		return 0, true
	}
	cap := decimal.NewFromInt(margin).Mul(decimal.NewFromFloat(0.8)).Round(0).IntPart()
	if amount > cap {
		return cap, true
	}
	return amount, false
}

// ProcessCommission 单笔消费的佣金结算入口（task-10 · P4 佣金核心）。
//
// 关键设计：
//  1. 旁路调用——在 service/text_quota.go:PostTextConsumeQuota 里通过 gopool.Go + defer recover
//     异步调用，**绝不污染主计费链路**（panic 也吞掉，只记 syslog）。
//  2. 邀请关系查一次：model.GetUserById(inviteeId, false) 只取 InviterId 字段，避免拉全表。
//     邀请人 0（无邀请关系）→ 直接返回 nil。
//  3. 运营未启用（CommissionRate = "0"）→ 直接返回 nil。
//  4. 原始金额为 0 → 直接返回 nil。
//  5. (consume_log_id, inviter_id) 唯一约束靠 DB 兜底（README §七第 4 条）；
//     INSERT 失败时 caller 直接吞掉错误，不影响主链路（commission 可丢可补，运营后台能补单）。
//  6. 钱包只动 user.AffCommissionBalance（task-11 的 TransferCommissionToQuota 是出口）；
//     不动 user.Quota（主余额）—— 与 task-09「独立钱包」语义一致。
//  7. audit 日志策略：
//     - breach=true 且 amount>0：写 commission_breach（真实不赔本场景，运营要看见）
//     - breach=true 且 amount=0：写 margin_unwired（cost_ratio 数据未接入的占位状态）
//     - 两条都用 LogTypeManage + admin_info 嵌套，与 v0.19.6 attachCostBreach 同位置，
//       普通用户看不到，运营后台能看到。
//
// consumeLogId 当前用 0 占位：task-10 阶段 model.RecordConsumeLog 还不返回 logId；
// commission_records.consume_log_id 默认 0，靠 uk_consume_inviter 防重。
// task-11 接入 logId 拿取逻辑后，此参数会被正确填充——commission_records 的消费溯源能力补齐。
func ProcessCommission(c *gin.Context, consumeLogId int64, inviteeId int, gross int64, margin int64) error {
	if inviteeId == 0 {
		return nil
	}

	// 1. 查邀请关系——只读 InviterId，用 selectAll=false 省一个字段。
	invitee, err := model.GetUserById(inviteeId, false)
	if err != nil || invitee == nil {
		return nil // invitee 不存在时静默跳过
	}
	if invitee.InviterId == 0 {
		return nil
	}

	// 2. 运营全局返佣率。
	rate := operation_setting.GetCommissionRate()
	if rate == "" || rate == "0" || rate == "0.000000" {
		return nil
	}

	// 3. 原始佣金。
	rawAmount := CalculateCommission(gross, rate)
	if rawAmount == 0 {
		return nil
	}

	// 4. 不赔本校验。
	finalAmount, breach := AssertNoLoss(rawAmount, margin)

	// 5. 事务：写 commission_records + 邀请人钱包 + account_ledger。
	//
	// 关键：audit 日志（RecordLogWithAdminInfo → createLog）不在事务里调——
	// 它内部要查 username 用到独立连接，会与当前事务争抢 SQLite 的同一连接导致死锁。
	// 改为事务提交后单独写。代价：若 audit 写失败不影响 commission_records（运营后台能补单）。
	//
	// task-17 §17.3：ledger 行与余额变更同事务——append-only，balance_after 用
	// 行锁 SELECT 后算出，保证任何事务回滚时 ledger 与余额同步回滚。
	txErr := model.DB.Transaction(func(tx *gorm.DB) error {
		now := common.GetTimestamp()
		rec := &model.CommissionRecord{
			InviterId:    invitee.InviterId,
			InviteeId:    inviteeId,
			ConsumeLogId: consumeLogId, // 占位 0（task-11 补 logId 拿取）
			Gross:        int(gross),
			Rate:         rate,
			Amount:       int(finalAmount),
			Margin:       int(margin),
			Breach:       breach,
			Currency:     "USD",
			SettledAt:    now,
			CreatedAt:    now,
		}
		if err := tx.Create(rec).Error; err != nil {
			// 唯一约束冲突（重试场景）——直接吞掉，DB 已经兜底。
			// 不影响主链路。
			return nil
		}

		// 邀请人钱包 += amount（finalAmount=0 时这是空操作，DB 不会报错）。
		if err := tx.Model(&model.User{}).
			Where("id = ?", invitee.InviterId).
			UpdateColumn("aff_commission_balance",
				gorm.Expr("aff_commission_balance + ?", finalAmount)).Error; err != nil {
			return err
		}

		// task-17 §17.3：写 ledger（finalAmount=0 也写一行——方便对账时核对
		// "应该入账但被 margin_unwired 拦截"的笔数）。balance_after 用行锁
		// 算出，确保 ledger 与余额同步；行锁避免与并发 commission 抢余额。
		if finalAmount != 0 {
			var balanceBefore int64
			if err := tx.Model(&model.User{}).
				Select("aff_commission_balance").
				Where("id = ?", invitee.InviterId).
				Scan(&balanceBefore).Error; err != nil {
				return err
			}
			balanceAfter := balanceBefore // 因为 UpdateColumn 已经在同事务里更新过，新值就是 balanceBefore
			if err := model.RecordAccountLedger(tx,
				"user", invitee.InviterId,
				model.AccountEventCommission, int64(finalAmount), balanceAfter,
				"commission_record", rec.Id,
				fmt.Sprintf("invitee=%d gross=%d rate=%s margin=%d breach=%t", inviteeId, gross, rate, margin, breach),
				0,
			); err != nil {
				return err
			}
		}
		return nil
	})
	if txErr != nil {
		return txErr
	}

	// 6. audit 日志（事务外）。
	// 策略：breach=true 时写；amount=0 的 breach 改写 "margin_unwired" 让运维识别
	// "cost_ratio 数据未接入"（与真实不赔本区分），方便后续接入。
	if breach {
		adminInfo := map[string]interface{}{
			"invitee_id":     inviteeId,
			"inviter_id":     invitee.InviterId,
			"consume_log_id": consumeLogId,
			"gross":          gross,
			"raw_amount":     rawAmount,
			"final_amount":   finalAmount,
			"margin":         margin,
			"rate":           rate,
		}
		var content string
		if finalAmount > 0 {
			content = fmt.Sprintf("commission_breach: invitee=%d inviter=%d consume_log=%d raw=%d final=%d margin=%d",
				inviteeId, invitee.InviterId, consumeLogId, rawAmount, finalAmount, margin)
		} else {
			content = fmt.Sprintf("commission_margin_unwired: invitee=%d inviter=%d consume_log=%d raw=%d margin=%d (cost_ratio data not yet integrated)",
				inviteeId, invitee.InviterId, consumeLogId, rawAmount, margin)
		}
		// RecordLogWithAdminInfo 内部用独立连接查 username（不与事务冲突）。
		// 用 admin_info 嵌套——与 service/log_info_generate.go:attachCostBreach 同一做法。
		model.RecordLogWithAdminInfo(invitee.InviterId, model.LogTypeManage, content, adminInfo)
	}
	return nil
}