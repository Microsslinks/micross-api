package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 注册带号进来的第一道关：这张号能不能用、不能用是哪一条。
//
// 口径必须与真正绑号（BindCustomerCode）一致——注册放行、绑号又被拒，
// 客户就会拿到一个没有折扣的账号，而他还以为自己已经在按经销商的价计费。
// 库与夹具复用 agent_code_test.go 里的那几个 seed 函数。

func TestCheckCustomerCodeUsableAcceptsGoodCode(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)
	code := seedCodeTestCode(t, agent.Id, plan.Id)

	// 号在库里是大写，客户手抄进来常常是小写、前后还带空格，这里就该被抹平
	got, err := CheckCustomerCodeUsable("  " + strings.ToLower(code.Code) + "  ")
	require.NoError(t, err)
	assert.Equal(t, code.Id, got.Id)
	assert.Equal(t, agent.Id, got.AgentId)
	assert.Equal(t, plan.Id, got.PlanId)
}

// 预检只是看一眼，不能动任何东西：注册被拒时不该顺手把号上的用量吃掉一次，
// 也不该让这张号从此绑不上人。
func TestCheckCustomerCodeUsableDoesNotConsumeUsage(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)
	code := seedCodeTestCode(t, agent.Id, plan.Id)

	_, err := CheckCustomerCodeUsable(code.Code)
	require.NoError(t, err)

	var reloaded CustomerCode
	require.NoError(t, DB.First(&reloaded, code.Id).Error)
	assert.Equal(t, 0, reloaded.UsedCount, "预检不该改用量")

	// 预检放行之后，这张号仍然真的能绑上
	customer := seedCodeTestUser(t, false)
	_, err = BindCustomerCode(customer.Id, code.Code)
	require.NoError(t, err)
}

func TestCheckCustomerCodeUsableRejectsReasons(t *testing.T) {
	setupAgentCodeTest(t)
	agent := seedCodeTestUser(t, true)
	plan := seedCodeTestPlan(t, DiscountStatusEnabled)

	t.Run("号不存在", func(t *testing.T) {
		_, err := CheckCustomerCodeUsable("AGNOSUCHCODE")
		require.ErrorIs(t, err, ErrCustomerCodeNotFound)
	})

	t.Run("没填号", func(t *testing.T) {
		_, err := CheckCustomerCodeUsable("   ")
		require.ErrorIs(t, err, ErrCustomerCodeNotFound)
	})

	t.Run("已作废", func(t *testing.T) {
		code := seedCodeTestCode(t, agent.Id, plan.Id)
		require.NoError(t, RevokeCustomerCode(agent.Id, code.Id))

		_, err := CheckCustomerCodeUsable(code.Code)
		require.ErrorIs(t, err, ErrCustomerCodeRevoked)
	})

	t.Run("已过期", func(t *testing.T) {
		// 签发时不接受过去时，所以只能签发后再把它改成过期——和真实情形一样。
		code := seedCodeTestCode(t, agent.Id, plan.Id)
		require.NoError(t, DB.Model(&CustomerCode{}).Where("id = ?", code.Id).
			UpdateColumn("expired_at", common.GetTimestamp()-1).Error)

		_, err := CheckCustomerCodeUsable(code.Code)
		require.ErrorIs(t, err, ErrCustomerCodeExpired)
	})

	t.Run("次数用满", func(t *testing.T) {
		code := seedCodeTestCode(t, agent.Id, plan.Id)
		first := seedCodeTestUser(t, false)
		_, err := BindCustomerCode(first.Id, code.Code)
		require.NoError(t, err)

		_, err = CheckCustomerCodeUsable(code.Code)
		require.ErrorIs(t, err, ErrCustomerCodeExhausted)
	})

	t.Run("号上的方案签发后被停用", func(t *testing.T) {
		codePlan := seedCodeTestPlan(t, DiscountStatusEnabled)
		code := seedCodeTestCode(t, agent.Id, codePlan.Id)
		require.NoError(t, DB.Model(&DiscountPlan{}).Where("id = ?", codePlan.Id).
			Update("status", DiscountStatusDisabled).Error)

		_, err := CheckCustomerCodeUsable(code.Code)
		require.ErrorIs(t, err, ErrCustomerCodePlanUnavailable)
	})
}
