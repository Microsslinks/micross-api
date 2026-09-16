/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
package model

import (
	"errors"

	"gorm.io/gorm"
)

// ============================================================================
// 任务文档 §三 12.1 订阅绑折扣（master-plan §4.6 口径）
//
// SubscriptionPlan.DiscountPlanId > 0 时，购买订阅自动给用户绑定该折扣方案。
// 绑定行 source='subscription'，effective_to = user_subscription.EndTime。
// 到期后 pickActiveDiscountBinding 自动跳过，fallback 到下一档方案。
//
// 不复用 BindDiscountPlan：它自己开 DB.Transaction 嵌套 savepoint，但这里
// 我们已经在外层事务里（CreateUserSubscriptionFromPlanTx / AdminInvalidateUserSubscription）。
// 改为直接 tx.Create + syncUserDiscountPlanId。
// ============================================================================

// bindSubscriptionDiscountTx 在外层事务里给用户写一条 source='subscription' 的折扣绑定。
//
// 触发条件：plan.DiscountPlanId > 0；否则 noop。
// 写一行 DiscountBinding（subject_type='user', source='subscription',
// effective_from=sub.StartTime, effective_to=sub.EndTime），
// 然后 syncUserDiscountPlanId 重算 users.discount_plan_id 快路径。
func bindSubscriptionDiscountTx(tx *gorm.DB, userId int, plan *SubscriptionPlan, sub *UserSubscription) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if plan == nil || plan.DiscountPlanId <= 0 {
		return nil // 没绑折扣方案 → 不写 binding
	}
	if userId <= 0 || sub == nil {
		return errors.New("invalid userId or subscription")
	}

	now := GetDBTimestamp()
	binding := &DiscountBinding{
		SubjectType:   DiscountSubjectUser,
		SubjectId:     userId,
		PlanId:        plan.DiscountPlanId,
		EffectiveFrom: sub.StartTime,
		EffectiveTo:   sub.EndTime,
		Source:        DiscountSourceSubscription,
		Status:        DiscountStatusEnabled,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := tx.Create(binding).Error; err != nil {
		return err
	}
	return syncUserDiscountPlanId(tx, userId)
}

// unbindSubscriptionDiscountTx 在外层事务里停用用户所有 source='subscription' 的绑定。
//
// 用于：AdminInvalidateUserSubscription（管理员取消）/ 订阅自然过期（sweeper）。
// 不直接 DELETE（保留事实记录可审计），只把 status 改成 0（Disabled）。
// 然后 syncUserDiscountPlanId 重算快路径。
func unbindSubscriptionDiscountTx(tx *gorm.DB, userId int, sub *UserSubscription) error {
	if tx == nil {
		return errors.New("tx is nil")
	}
	if userId <= 0 {
		return errors.New("invalid userId")
	}
	now := GetDBTimestamp()
	_ = sub // 当前实现对所有 status=enabled 的 subscription 绑定一刀切；
	// 一个用户通常只有一条 active 订阅，少数多档并行（终身+年卡）场景下
	// 取消中间档不会影响其他档（effective_to 已经把它们隔开）。

	if err := tx.Model(&DiscountBinding{}).
		Where("subject_type = ? AND subject_id = ? AND source = ? AND status = ?",
			DiscountSubjectUser, userId, DiscountSourceSubscription, DiscountStatusEnabled).
		Updates(map[string]interface{}{
			"status":     DiscountStatusDisabled,
			"updated_at": now,
		}).Error; err != nil {
		return err
	}
	return syncUserDiscountPlanId(tx, userId)
}