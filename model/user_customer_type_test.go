package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 客户类型筛选是运营在用户列表里按人群捞人的唯一依据，口径必须与前端
// web/src/features/users/constants.ts 的 resolveCustomerType 完全一致：
// 先看是不是经销商，再看有没有绑定折扣方案。顺序一旦反了，经销商就会被
// 算进企业折扣。
func TestSearchUsersByCustomerType(t *testing.T) {
	setupAgentTest(t)

	// 普通客户：没绑任何专属方案
	individual := seedAgentTestUser(t, SubjectTypeIndividual, 0)

	// 企业折扣：绑了专属方案，但不是经销商
	enterprise := seedAgentTestUser(t, SubjectTypeIndividual, 0)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", enterprise.Id).
		Update("discount_plan_id", 9).Error)

	// 经销商：自己身上也挂着专属方案（他的批发价），不能因此被算成企业折扣
	agent := seedAgentTestUser(t, SubjectTypeAgent, 0)
	require.NoError(t, DB.Model(&User{}).Where("id = ?", agent.Id).
		Update("discount_plan_id", 11).Error)

	idsOf := func(users []*User) []int {
		ids := make([]int, 0, len(users))
		for _, user := range users {
			ids = append(ids, user.Id)
		}
		return ids
	}
	search := func(customerType string) []*User {
		users, _, err := SearchUsers("", "", nil, nil, customerType, 0, 50)
		require.NoError(t, err)
		return users
	}

	assert.ElementsMatch(t, []int{individual.Id, enterprise.Id, agent.Id}, idsOf(search("")),
		"customer_type 传空字符串时不得改变原有搜索结果")
	assert.Equal(t, []int{agent.Id}, idsOf(search(CustomerTypeAgent)),
		"筛经销商时只能出经销商")
	assert.Equal(t, []int{enterprise.Id}, idsOf(search(CustomerTypeEnterprise)),
		"筛企业折扣时不能把经销商一起带出来")
	assert.Equal(t, []int{individual.Id}, idsOf(search(CustomerTypeIndividual)),
		"筛普通客户时不能把经销商一起带出来")
}

// 存量行的 subject_type 可能为 NULL（老库新增列没有默认值时就是这个形态）。
// SQL 里 NULL 参与 <> 比较恒不成立，如果不单独兜底，老用户会被整批筛掉——
// 运营看到的是"人凭空少了"，而不是报错。
func TestSearchUsersByCustomerTypeHandlesNullSubjectType(t *testing.T) {
	setupAgentTest(t)

	legacy := seedAgentTestUser(t, SubjectTypeIndividual, 0)
	require.NoError(t, DB.Exec("UPDATE users SET subject_type = NULL WHERE id = ?", legacy.Id).Error)

	users, _, err := SearchUsers("", "", nil, nil, CustomerTypeIndividual, 0, 50)
	require.NoError(t, err)
	require.Len(t, users, 1, "subject_type 为 NULL 的存量用户应当照样算作普通客户")
	assert.Equal(t, legacy.Id, users[0].Id)

	agents, _, err := SearchUsers("", "", nil, nil, CustomerTypeAgent, 0, 50)
	require.NoError(t, err)
	assert.Empty(t, agents, "subject_type 为 NULL 的存量用户不是经销商")
}
