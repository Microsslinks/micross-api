package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 客户绑号的用例。库与夹具复用 agent_code_test.go 里的
// setupAgentCodeTest / seedCodeTestUser / seedCodeTestPlan。

// seedCodeTestCode 签一张号给某个经销商用。
//
// 没有"能用几次"这个参数：一张号只拉一位客户，由模型层写死（见 CustomerCodeMaxUsesPerCode），
// 测试里也就没有旋钮可拧。要造过期、作废这类状态，各用例自己直接改库。
func seedCodeTestCode(t *testing.T, agentId int, planId int) *CustomerCode {
	t.Helper()
	codes, err := CreateCustomerCodes(CustomerCodeIssue{
		AgentId: agentId,
		PlanId:  planId,
		Count:   1,
	})
	require.NoError(t, err)
	require.Len(t, codes, 1)
	return codes[0]
}

// 绑一次要同时落三件事：归属、折扣（含快路径）、号上用量。
// 少任何一件都会出现「归属落了但价没变」或「号能反复用」。
func TestBindCustomerCodeBindsOwnershipDiscountAndUsage(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)
	code := seedCodeTestCode(t, agent.Id, plan.Id)

	// 客户手抄进来的号常常是小写，这里故意用小写入参。
	binding, err := BindCustomerCode(customer.Id, strings.ToLower(code.Code))
	require.NoError(t, err)

	assert.Equal(t, agent.Id, binding.ParentAgentId)
	assert.Equal(t, agent.Username, binding.AgentName)
	assert.Equal(t, plan.Id, binding.PlanId)
	assert.Equal(t, plan.Name, binding.PlanName)
	assert.Equal(t, "0.900000", binding.PlanDiscount)
	assert.Equal(t, DiscountSourceCustomerCode, binding.BindingSource)

	var storedUser User
	require.NoError(t, DB.First(&storedUser, customer.Id).Error)
	assert.Equal(t, agent.Id, storedUser.ParentAgentId)
	assert.Equal(t, plan.Id, storedUser.DiscountPlanId, "计费的快路径必须同一处写好")

	var storedCode CustomerCode
	require.NoError(t, DB.First(&storedCode, code.Id).Error)
	assert.Equal(t, 1, storedCode.UsedCount)
	assert.Equal(t, customer.Id, storedCode.BoundUserId, "号上要记下是谁用掉的")

	var storedBinding DiscountBinding
	require.NoError(t, DB.Where("subject_id = ?", customer.Id).First(&storedBinding).Error)
	assert.Equal(t, DiscountSubjectUser, storedBinding.SubjectType)
	assert.Equal(t, DiscountSourceCustomerCode, storedBinding.Source)
	assert.Equal(t, DiscountStatusEnabled, storedBinding.Status)

	// 绑完之后这个客户真的按号上的折扣计价：这是整个功能的验收口径。
	resolution, err := ResolveUserDiscount(customer.Id, "claude-3-5-sonnet")
	require.NoError(t, err)
	assert.True(t, decimal.RequireFromString("0.9").
		Equal(decimal.RequireFromString(resolution.Discount)), resolution.Discount)
	assert.Equal(t, DiscountResolvedFromPlanBase, resolution.Source)

	// 列表里看得出这张号给了谁：经销商打开「我的客户」第一眼要找的就是这个。
	listed, total, err := ListCustomerCodes(agent.Id, 0, 10, false)
	require.NoError(t, err)
	assert.Equal(t, int64(1), total)
	require.Len(t, listed, 1)
	assert.Equal(t, customer.Id, listed[0].BoundUserId)
	assert.Equal(t, customer.Username, listed[0].BoundUsername)
}

// 不能用的号要各自给出具体理由：客户看到的得是能据以行动的那一句。
func TestBindCustomerCodeRejectsUnusableCodes(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)

	revoked := seedCodeTestCode(t, agent.Id, 0)
	require.NoError(t, RevokeCustomerCode(agent.Id, revoked.Id))

	expired := seedCodeTestCode(t, agent.Id, 0)
	require.NoError(t, DB.Model(&CustomerCode{}).Where("id = ?", expired.Id).
		Updates(map[string]interface{}{"expired_at": common.GetTimestamp() - 1}).Error)

	exhausted := seedCodeTestCode(t, agent.Id, 0)
	_, err := BindCustomerCode(customer.Id, exhausted.Code)
	require.NoError(t, err)
	// 换个人来用这张只剩 1 次的号（第一个人已经用掉了）
	other := seedCodeTestUser(t, false)
	err = func() error {
		_, err := BindCustomerCode(other.Id, exhausted.Code)
		return err
	}()
	require.ErrorIs(t, err, ErrCustomerCodeExhausted)

	disabledPlan := seedCodeTestPlan(t, DiscountStatusDisabled)
	codeWithDisabledPlan, errCreate := CreateCustomerCodes(CustomerCodeIssue{
		AgentId: agent.Id, PlanId: disabledPlan.Id, Count: 1,
	})
	require.ErrorIs(t, errCreate, ErrCustomerCodePlanUnavailable)
	require.Empty(t, codeWithDisabledPlan, "方案不可用时不该签出任何号")

	cases := []struct {
		name string
		code string
		want error
	}{
		{"空号", "   ", ErrCustomerCodeNotFound},
		{"不存在的号", "AGZZZZZZZZZZ", ErrCustomerCodeNotFound},
		{"已作废", revoked.Code, ErrCustomerCodeRevoked},
		{"已过期", expired.Code, ErrCustomerCodeExpired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BindCustomerCode(seedCodeTestUser(t, false).Id, tc.code)
			require.ErrorIs(t, err, tc.want)
		})
	}

	// 号上的方案在发出后被停用：也不能绑，否则客户以为自己有折扣。
	code := seedCodeTestCode(t, agent.Id, plan.Id)
	require.NoError(t, DB.Model(&DiscountPlan{}).Where("id = ?", plan.Id).
		Update("status", DiscountStatusDisabled).Error)
	_, err = BindCustomerCode(seedCodeTestUser(t, false).Id, code.Code)
	require.ErrorIs(t, err, ErrCustomerCodePlanUnavailable)
}

// 两种身份上的拒绝：自己发自己用、以及已经归属了别人。
// 后者只能由平台改——否则谁手里有张号就能把别人的客户挖走。
func TestBindCustomerCodeRejectsSelfUseAndSwitchingOwner(t *testing.T) {
	setupAgentCodeTest(t)
	agentA := seedCodeTestUser(t, true)
	agentB := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)

	codeOfA := seedCodeTestCode(t, agentA.Id, 0)
	_, err := BindCustomerCode(agentA.Id, codeOfA.Code)
	require.ErrorIs(t, err, ErrCustomerCodeSelfUse)

	_, err = BindCustomerCode(agentB.Id, codeOfA.Code)
	require.ErrorIs(t, err, ErrCustomerCodeOwnedByAgent)

	require.NoError(t, DB.Model(&User{}).Where("id = ?", customer.Id).
		UpdateColumn("parent_agent_id", agentA.Id).Error)
	codeOfB := seedCodeTestCode(t, agentB.Id, 0)
	_, err = BindCustomerCode(customer.Id, codeOfB.Code)
	require.ErrorIs(t, err, ErrCustomerBelongsToOtherAgent)

	// 被拒绝的三次都不该留下痕迹：归属没动、用量没扣。
	var reloaded User
	require.NoError(t, DB.First(&reloaded, customer.Id).Error)
	assert.Equal(t, agentA.Id, reloaded.ParentAgentId)
	var storedCode CustomerCode
	require.NoError(t, DB.First(&storedCode, codeOfA.Id).Error)
	assert.Equal(t, 0, storedCode.UsedCount)
	assert.Equal(t, 0, storedCode.BoundUserId, "被拒绝的绑定不该留下绑定人")
}

// 同一个人拿同一张号再绑一次：不重复建折扣绑定，也不再扣一次用量。
func TestBindCustomerCodeIsIdempotentForSameCustomer(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)
	code := seedCodeTestCode(t, agent.Id, plan.Id)

	_, err := BindCustomerCode(customer.Id, code.Code)
	require.NoError(t, err)

	// 一张号只给一位客户，所以"再绑一次"时号上已经是用满状态。这一步要断言的是：
	// 他说听到的是"你已经是这位经销商的客户了"，而不是"使用次数已用完"——
	// 后者会让他以为自己拿错号了。
	_, err = BindCustomerCode(customer.Id, code.Code)
	require.ErrorIs(t, err, ErrCustomerCodeAlreadyBound)

	var bindingCount int64
	require.NoError(t, DB.Model(&DiscountBinding{}).Where("subject_id = ?", customer.Id).
		Count(&bindingCount).Error)
	assert.Equal(t, int64(1), bindingCount, "不能出现两条一样的绑定")

	var storedCode CustomerCode
	require.NoError(t, DB.First(&storedCode, code.Id).Error)
	assert.Equal(t, 1, storedCode.UsedCount, "重复绑不算新的用量")
}

// 平台手工给的价优先于客户号带的价：客户号是经销商给的，不该顶掉平台的决定。
func TestBindCustomerCodeDoesNotOverrideManualDiscount(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	customer := seedCodeTestUser(t, false)
	manualPlan := seedCodeTestPlan(t, DiscountStatusEnabled)
	codePlan := seedCodeTestPlan(t, DiscountStatusEnabled)

	require.NoError(t, BindDiscountPlan(&DiscountBinding{
		SubjectType: DiscountSubjectUser,
		SubjectId:   customer.Id,
		PlanId:      manualPlan.Id,
		Source:      DiscountSourceManual,
		Status:      DiscountStatusEnabled,
	}))
	code := seedCodeTestCode(t, agent.Id, codePlan.Id)

	binding, err := BindCustomerCode(customer.Id, code.Code)
	require.NoError(t, err)
	// 归属照样落，号上的方案也照样记下来，只是它此刻不生效
	assert.Equal(t, agent.Id, binding.ParentAgentId)
	assert.Equal(t, manualPlan.Id, binding.PlanId)
	assert.Equal(t, DiscountSourceManual, binding.BindingSource)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, customer.Id).Error)
	assert.Equal(t, manualPlan.Id, reloaded.DiscountPlanId)
}
