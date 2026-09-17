package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

const (
	// commissionRiskScanTickInterval 风控扫描周期（默认 6 小时）。
	//
	// 设计取舍：
	//   - 太短（1h 内）→ commission_records 大表扫描 IO 高，意义不大
	//     （薅羊毛往往 24h 内出现，6h 粒度够）。
	//   - 太长（24h+）→ 漏掉快速膨胀的刷单事件。
	//   - 6h 是 ops 心里能接受的延迟 + 单实例一次扫 10000 条记录的合理边界。
	commissionRiskScanTickInterval = 6 * time.Hour

	// commissionRiskScanBatchSize 每批扫的 commission_records 行数。
	//
	// SQLite 内存模式下 1000 条足以在秒级返回；MySQL/Postgres 也无压力。
	// 循环直到扫完所有 Reversed=false + 创建时间在 7 天内的记录，
	// 7 天是 P4 风的业务容忍度（再老的 ring 早被上游风控拦住）。
	commissionRiskScanBatchSize = 1000

	// firstTopupThresholdQuota 首充门槛硬编码值（quota 单位）。
	//
	// 与 model.GetUserTotalConsumeQuota 的 FirstTopUpQuota 默认值对齐：
	// 5000 quota ≈ $0.01（默认 QuotaPerUnit=500000 = $1），
	// 任何"没真实消费过"的 inviter 都低于此值。
	firstTopupThresholdQuota int64 = 5000

	// ringDetectionMaxDepth 成环检测的最大跳数（与 model.DetectInviteRing 默认对齐）。
	ringDepthMax int = 5
)

// 状态变量 —— 单进程单跑语义。
var (
	commissionRiskScanOnce   sync.Once
	commissionRiskScanBusy  atomic.Bool
)

// StartCommissionRiskScanTask 启动风控扫描后台任务（task-20 §20.7）。
//
// 调用入口：main.go / 启动钩子（与 StartSubscriptionQuotaResetTask 同位置）。
//
// 多实例集群：仅 IsMasterNode=true 才跑——避免同一批 record 被多机并发
// 重复扫描（每个 record 实例的 commission_processed_at 标记防并发，但
// 减少不必要的 SQL 流量更友好）。
//
// 单实例：CAS 防重入（subscription_reset_task 同模式）。
func StartCommissionRiskScanTask() {
	commissionRiskScanOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			logger.LogInfo(context.Background(), fmt.Sprintf(
				"commission risk scan task started: tick=%s batch_size=%d",
				commissionRiskScanTickInterval, commissionRiskScanBatchSize))
			ticker := time.NewTicker(commissionRiskScanTickInterval)
			defer ticker.Stop()

			runCommissionRiskScanOnce()
			for range ticker.C {
				runCommissionRiskScanOnce()
			}
		}()
	})
}

// runCommissionRiskScanOnce 单次扫描所有未冲销 commission_records。
//
// 流程：
//   1. 防重入（CAS busy=true）。
//   2. 找出 candidate = commission_records where reversed=false
//      AND settled_at > now-7d 的所有记录（按 id 升序）。
//   3. 对每条：
//      a) DetectInviteRing(inviterId, ringDepthMax) → true → ReverseCommission
//         reason="auto-risk: invite_ring detected (scan cron)"
//      b) GetUserTotalConsumeQuota(inviterId) < firstTopupThresholdQuota → ReverseCommission
//         reason="auto-risk: first_topup_threshold not met (scan cron)"
//   4. 任何一条 ReverseCommission 失败只记录日志，不中断整批。
//
// 7 天窗口理由：更老的记录大概率已经 manual 过一遍；7 天内的新记录
// 才是 cron 真正要保护的实时风险。如果未来发现 P4 风控问题持续期
// 超过 7 天，加个 cron 参数化即可，不必重写。
func runCommissionRiskScanOnce() {
	if !commissionRiskScanBusy.CompareAndSwap(false, true) {
		return
	}
	defer commissionRiskScanBusy.Store(false)

	ctx := context.Background()
	start := time.Now()

	totalScanned := 0
	totalReversed := 0
	var scanErr error

	// 候选记录：未冲销 + 7 天内
	cutoff := common.GetTimestamp() - int64(7*24*time.Hour/time.Second)
	offset := 0
	for {
		var batch []model.CommissionRecord
		if err := model.DB.Model(&model.CommissionRecord{}).
			Where("reversed = ? AND settled_at > ?", false, cutoff).
			Order("id ASC").
			Limit(commissionRiskScanBatchSize).
			Offset(offset).
			Find(&batch).Error; err != nil {
			scanErr = err
			break
		}
		if len(batch) == 0 {
			break
		}

		for _, rec := range batch {
			totalScanned++
			// 重新加载以防 race：scan 期间 record 可能被 admin 撤销
			var fresh model.CommissionRecord
			if err := model.DB.First(&fresh, rec.Id).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue
				}
				logger.LogError(ctx, fmt.Sprintf("commission risk scan: load record %d failed: %v", rec.Id, err))
				continue
			}
			if fresh.Reversed {
				continue
			}

			reason := detectRiskReasonForCommissionRecord(&fresh)
			if reason == "" {
				continue
			}

			// 自动 reverse —— operator_id=0 标识"系统自动"（区别于 admin 人工）。
			if err := ReverseCommission(ctx, fresh.Id, reason, 0); err != nil {
				logger.LogError(ctx, fmt.Sprintf(
					"commission risk scan: reverse record %d failed: %v (reason=%s)", fresh.Id, err, reason))
				continue
			}
			totalReversed++
		}

		if len(batch) < commissionRiskScanBatchSize {
			break
		}
		offset += len(batch)
	}

	duration := time.Since(start)
	logFields := fmt.Sprintf(
		"commission risk scan done: scanned=%d reversed=%d duration_ms=%d",
		totalScanned, totalReversed, duration.Milliseconds())
	if scanErr != nil {
		logFields += fmt.Sprintf(" error=%v", scanErr)
		logger.LogError(ctx, logFields)
	} else {
		logger.LogInfo(ctx, logFields)
	}
}

// detectRiskReasonForCommissionRecord 返回命中风控的原因描述；未命中返回 ""。
//
// 风控规则：
//   - invite_ring:  inviter 链上出现循环（A→B→A 或更长）。
//   - first_topup:  inviter 累计消费 < firstTopupThresholdQuota。
//
// 实现细节：
//   - DetectInviteRing 是深度优先搜索 + 集合，O(MaxRingDepth) 每条。
//   - GetUserTotalConsumeQuota 走 logs 表 SUM，索引命中 type+user_id。
//   - 两次都失败 → 不 reverse，宁可放过也不误伤。
func detectRiskReasonForCommissionRecord(rec *model.CommissionRecord) string {
	if rec.InviterId <= 0 {
		return ""
	}
	// 规则 1：成环
	if model.DetectInviteRing(rec.InviterId) {
		return "auto-risk: invite_ring detected (scan cron)"
	}
	// 规则 2：首充门槛
	total, err := model.GetUserTotalConsumeQuota(rec.InviterId)
	if err != nil {
		// 读取失败宁可放过：日志记一行，不 reverse。
		logger.LogWarn(context.Background(), fmt.Sprintf(
			"commission risk scan: GetUserTotalConsumeQuota(%d) failed: %v",
			rec.InviterId, err))
		return ""
	}
	if total < firstTopupThresholdQuota {
		return fmt.Sprintf("auto-risk: first_topup_threshold not met (consumed=%d < %d, scan cron)", total, firstTopupThresholdQuota)
	}
	return ""
}