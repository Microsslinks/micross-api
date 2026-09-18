package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

var _ = common.GetTimestamp // 占位防止 unused import 误删（common 后续 RecordLogWithAdminInfo 可能会用）

// ---------------------------------------------------------------------------
// FundingSource — 资金来源接口（钱包 or 订阅）
// ---------------------------------------------------------------------------

// FundingSource 抽象了预扣费的资金来源。
type FundingSource interface {
	// Source 返回资金来源标识："wallet" 或 "subscription"
	Source() string
	// PreConsume 从该资金来源预扣 amount 额度
	PreConsume(amount int) error
	// Settle 根据差额调整资金来源（正数补扣，负数退还）
	Settle(delta int) error
	// Refund 退还所有预扣费
	Refund() error
}

// ---------------------------------------------------------------------------
// WalletFunding — 钱包资金来源实现
// ---------------------------------------------------------------------------

// ErrInsufficientWalletQuota 钱包原子预扣失败（余额不足），未发生任何扣减。
// BillingSession 据此映射为 ErrorCodeInsufficientUserQuota，
// 使 wallet_first 等计费偏好可以回退到订阅。
var ErrInsufficientWalletQuota = errors.New("wallet quota insufficient")

type WalletFunding struct {
	userId   int
	consumed int // 实际预扣的用户额度
}

func (w *WalletFunding) Source() string { return BillingSourceWallet }

func (w *WalletFunding) PreConsume(amount int) error {
	if amount <= 0 {
		return nil
	}
	reserved, err := model.TryReserveUserQuota(w.userId, amount)
	if err != nil {
		return err
	}
	if !reserved {
		return ErrInsufficientWalletQuota
	}
	w.consumed = amount
	return nil
}

func (w *WalletFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return model.DecreaseUserQuota(w.userId, delta, false)
	}
	// task-20 §20.2：delta<0 表示预扣超额 → 给用户加回额度（refund 路径）。
	// 走 refundWalletQuota：事务内 quota += N + 写 ledger。
	return refundWalletQuota(w.userId, -delta, "billing_session_settle", 0)
}

func (w *WalletFunding) Refund() error {
	if w.consumed <= 0 {
		return nil
	}
	// task-20 §20.2：失败 / 取消时退还预扣 → 走 refundWalletQuota 写 ledger。
	// IncreaseUserQuota 内部是 quota += N 的非幂等操作，不能重试（否则多退）。
	// refundWalletQuota 同样单写一行：重试会双写，所以依赖 IncreaseUserQuota
	// 同款的非重试语义。
	return refundWalletQuota(w.userId, w.consumed, "billing_session_refund", 0)
}

// refundWalletQuota 在事务内给用户加回额度并写一行 ledger。task-20 §20.2：
// 所有给用户"加回 quota"的路径（WalletFunding.Settle 负数 + WalletFunding.Refund
// 失败退还）都走这里。ledger 与 quota 变更同事务——事务回滚时 ledger 同步回滚，
// 不可能"额度退还了但没记账"。
//
// 为什么用独立 helper 而不是在 model.IncreaseUserQuota 里加 eventType 参数：
//
//   - IncreaseUserQuota 现有 7 个调用点（邀请赠送 / 注册赠送 / 退款 / 等），
//     eventType 各不相同，加参数要动 7 处。
//   - task-20 §20.2 只关心"refund"一类事件类型，独立 helper 把范围收紧：
//     之后（§20.3 agent_quota_grant / §20.6 风控冲销）再加事件类型时改
//     helper 而不是改 IncreaseUserQuota。
//   - IncreaseUserQuota 是公共 API（外部包可能引用），改签名影响面大。
//
// 直接绕开 IncreaseUserQuota：因为它有 BatchUpdateEnabled 异步批更新分支，
// ledger 在事务里但 quota 写入异步 batch 会乱序。refundWalletQuota 走底层 DB
// + ledger 同步写，确保 ledger 与 quota 严格同事务。
func refundWalletQuota(userId int, quota int, refType string, refId int64) error {
	if quota <= 0 {
		return nil
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.User{}).
			Where("id = ?", userId).
			Update("quota", gorm.Expr("quota + ?", quota))
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		var balanceAfter int64
		if err := tx.Model(&model.User{}).
			Select("quota").
			Where("id = ?", userId).
			Scan(&balanceAfter).Error; err != nil {
			return err
		}
		return model.RecordAccountLedger(tx,
			"user", userId,
			model.AccountEventRefund, int64(quota), balanceAfter,
			refType, refId,
			fmt.Sprintf("refund quota=%d", quota),
			0,
		)
	})
}

// ---------------------------------------------------------------------------
// SubscriptionFunding — 订阅资金来源实现
// ---------------------------------------------------------------------------

type SubscriptionFunding struct {
	requestId      string
	userId         int
	modelName      string
	amount         int64 // 预扣的订阅额度（subConsume）
	subscriptionId int
	preConsumed    int64
	// 以下字段在 PreConsume 成功后填充，供 RelayInfo 同步使用
	AmountTotal     int64
	AmountUsedAfter int64
	PlanId          int
	PlanTitle       string
}

func (s *SubscriptionFunding) Source() string { return BillingSourceSubscription }

func (s *SubscriptionFunding) PreConsume(_ int) error {
	// amount 参数被忽略，使用内部 s.amount（已在构造时根据 preConsumedQuota 计算）
	res, err := model.PreConsumeUserSubscription(s.requestId, s.userId, s.modelName, 0, s.amount)
	if err != nil {
		return err
	}
	s.subscriptionId = res.UserSubscriptionId
	s.preConsumed = res.PreConsumed
	s.AmountTotal = res.AmountTotal
	s.AmountUsedAfter = res.AmountUsedAfter
	// 获取订阅计划信息
	if planInfo, err := model.GetSubscriptionPlanInfoByUserSubscriptionId(res.UserSubscriptionId); err == nil && planInfo != nil {
		s.PlanId = planInfo.PlanId
		s.PlanTitle = planInfo.PlanTitle
	}
	return nil
}

func (s *SubscriptionFunding) Settle(delta int) error {
	if delta == 0 {
		return nil
	}
	return model.PostConsumeUserSubscriptionDelta(s.subscriptionId, int64(delta))
}

func (s *SubscriptionFunding) Refund() error {
	if s.preConsumed <= 0 {
		return nil
	}
	return refundWithRetry(func() error {
		return model.RefundSubscriptionPreConsume(s.requestId)
	})
}

// refundWithRetry 尝试多次执行退款操作以提高成功率，只能用于基于事务的退款函数！！！！！！
// try to refund with retries, only for refund functions based on transactions!!!
func refundWithRetry(fn func() error) error {
	if fn == nil {
		return nil
	}
	const maxAttempts = 3
	var lastErr error
	for i := 0; i < maxAttempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < maxAttempts-1 {
			time.Sleep(time.Duration(200*(i+1)) * time.Millisecond)
		}
	}
	return lastErr
}
