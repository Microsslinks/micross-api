package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 一个客户最多能挂几套折扣方案的用例。
// 库与夹具复用 agent_code_test.go 里的 setupAgentCodeTest / seedCodeTestUser / seedCodeTestPlan。

// seedBoundPlans 给一位客户挂满 count 套方案，返回最后一套的 ID。
func seedBoundPlans(t *testing.T, userId int, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		plan := seedCodeTestPlan(t, DiscountStatusEnabled)
		require.NoError(t, BindDiscountPlan(&DiscountBinding{
			SubjectType: DiscountSubjectUser,
			SubjectId:   userId,
			PlanId:      plan.Id,
			Source:      DiscountSourceManual,
			Status:      DiscountStatusEnabled,
		}))
	}
}

// 挂到第 11 套要被挡住，且一个字都不该落库。
func TestBindDiscountPlanStopsAtLimit(t *testing.T) {
	setupAgentCodeTest(t)
	customer := seedCodeTestUser(t, false)
	seedBoundPlans(t, customer.Id, DiscountMaxPlansPerSubject)

	extra := seedCodeTestPlan(t, DiscountStatusEnabled)
	err := BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   customer.Id,
		PlanId:      extra.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	})
	require.ErrorIs(t, err, ErrDiscountPlanLimitReached)

	var bound int64
	require.NoError(t, DB.Model(&DiscountBinding{}).
		Where("subject_id = ? AND status = ?", customer.Id, DiscountStatusEnabled).
		Count(&bound).Error)
	assert.Equal(t, int64(DiscountMaxPlansPerSubject), bound, "被拒绝的绑定不能留下痕迹")
}

// 重复绑同一条方案，该听到的是"已经绑过了"，而不是"你的方案太多了"——
// 上限检查排在重复检查之后，说的才是他真正该改的那件事。
func TestBindDiscountPlanReportsDuplicateBeforeLimit(t *testing.T) {
	setupAgentCodeTest(t)
	customer := seedCodeTestUser(t, false)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)
	require.NoError(t, BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   customer.Id,
		PlanId:      plan.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	}))
	seedBoundPlans(t, customer.Id, DiscountMaxPlansPerSubject-1)

	err := BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   customer.Id,
		PlanId:      plan.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	})
	require.ErrorIs(t, err, ErrDiscountBindingExists)
}

// 解绑过的不占位：挂满 10 套之后解掉一套，还能再挂一套新的。
// 否则一个人绑够 10 套就永远换不了价。
func TestBindDiscountPlanFreesSlotAfterUnbind(t *testing.T) {
	setupAgentCodeTest(t)
	customer := seedCodeTestUser(t, false)
	seedBoundPlans(t, customer.Id, DiscountMaxPlansPerSubject)

	var first DiscountBinding
	require.NoError(t, DB.Where("subject_id = ?", customer.Id).First(&first).Error)
	require.NoError(t, UnbindDiscountPlan(first.Id))

	extra := seedCodeTestPlan(t, DiscountStatusEnabled)
	require.NoError(t, BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   customer.Id,
		PlanId:      extra.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	}))

	var bound int64
	require.NoError(t, DB.Model(&DiscountBinding{}).
		Where("subject_id = ? AND status = ?", customer.Id, DiscountStatusEnabled).
		Count(&bound).Error)
	assert.Equal(t, int64(DiscountMaxPlansPerSubject), bound)
}

// 客户号这条侧门也要认同一个上限：挂满的客户拿号来绑，整笔拒绝——
// 归属不落、号上的用量不扣，平台解绑一套之后他还能用同一张号。
func TestBindCustomerCodeRespectsPlanLimit(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	seedBoundPlans(t, customer.Id, DiscountMaxPlansPerSubject)

	codePlan := seedCodeTestPlan(t, DiscountStatusEnabled)
	code := seedCodeTestCode(t, agent.Id, codePlan.Id)

	_, err := BindCustomerCode(customer.Id, code.Code)
	require.ErrorIs(t, err, ErrDiscountPlanLimitReached)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, customer.Id).Error)
	assert.Equal(t, 0, reloaded.ParentAgentId, "被拒绝的绑号不该落归属")

	var storedCode CustomerCode
	require.NoError(t, DB.First(&storedCode, code.Id).Error)
	assert.Equal(t, 0, storedCode.UsedCount, "被拒绝的绑号不该扣用量")
	assert.Equal(t, 0, storedCode.BoundUserId)
}
