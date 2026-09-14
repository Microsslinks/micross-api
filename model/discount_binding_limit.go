package model

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// DiscountMaxPlansPerSubject 是"一个人最多同时挂几套折扣方案"。
//
// 10 是业务口径（task-06）：一个客户挂多套是本来的能力，但挂太多就没法回答
// "哪个模型走哪套"，而且每套都要参与解析，成本跟着套数涨。
const DiscountMaxPlansPerSubject = 10

// ErrDiscountPlanLimitReached 已经挂满，再多要绑得先解绑不再用的。
// 调用方用 errors.Is 判断，具体上限数字见错误文本。
var ErrDiscountPlanLimitReached = errors.New("绑定的折扣方案已达上限")

// ensureDiscountBindingQuota 挡住"挂到第 11 套"。
//
// 只数生效中的绑定，历史记录（解绑过的）不占位——不然一个人绑过一次就永远回不来了。
// 判断必须在写绑定之前、且在同一个事务里做：并发绑两条时，两边都先读到 9 就会一起挤进第 10、11 套。
func ensureDiscountBindingQuota(tx *gorm.DB, subjectType string, subjectId int) error {
	var bound int64
	if err := tx.Model(&DiscountBinding{}).
		Where("subject_type = ? AND subject_id = ? AND status = ?",
			subjectType, subjectId, DiscountStatusEnabled).
		Count(&bound).Error; err != nil {
		return err
	}
	if bound >= DiscountMaxPlansPerSubject {
		return fmt.Errorf("%w：一个人最多同时挂 %d 套，请先解绑不再用的",
			ErrDiscountPlanLimitReached, DiscountMaxPlansPerSubject)
	}
	return nil
}
