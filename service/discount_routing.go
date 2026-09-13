package service

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/model"

	"gorm.io/gorm"
)

var (
	// ErrDiscountRoutingUserNotFound 路由策略指定的客户不存在。
	ErrDiscountRoutingUserNotFound = errors.New("客户不存在")
	// ErrDiscountRoutingInvalidStrategy 择优策略取值不在允许的两档里。
	ErrDiscountRoutingInvalidStrategy = errors.New("routing_strategy 只能是 margin 或 priority")
	// ErrDiscountRoutingInvalidUser user_id 不是正整数。
	ErrDiscountRoutingInvalidUser = errors.New("user_id 必须是正整数")
)

// 客户路由策略（择优策略 + 允许走亏损线路的开关）的运营侧读写。
//
// 表里没有配置行 = 默认口径（毛利优先、不允许走亏损线路），所以读接口在没有配置行时
// 也返回一份完整结果，只是 configured 为 false——界面上就是「这个客户没单独配过」，
// 而不是「读失败」。

// DiscountRoutingView 是读接口的出参。
type DiscountRoutingView struct {
	UserId   int    `json:"user_id"`
	Username string `json:"username"`
	// RoutingStrategy 是当前生效的策略：没配置过时给默认的 margin。
	RoutingStrategy string `json:"routing_strategy"`
	AllowCostBreach bool   `json:"allow_cost_breach"`
	Remark          string `json:"remark"`
	// UpdatedBy 是最后改这一行的人（管理员 id）；没配置过时为 0。
	UpdatedBy int   `json:"updated_by"`
	UpdatedAt int64 `json:"updated_at"`
	// Configured 表示这个客户有没有单独的配置行。
	Configured bool `json:"configured"`
}

// GetCustomerRouting 读一个客户当前生效的路由策略。客户不存在时报错，没配过策略不算错。
func GetCustomerRouting(userId int) (*DiscountRoutingView, error) {
	user, err := model.GetUserById(userId, false)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDiscountRoutingUserNotFound
		}
		return nil, err
	}

	policy, err := model.GetDiscountRoutingPolicy(userId)
	if err != nil {
		return nil, err
	}
	view := &DiscountRoutingView{
		UserId:          user.Id,
		Username:        user.Username,
		RoutingStrategy: model.RoutingStrategyMargin,
	}
	if policy != nil {
		view.Configured = true
		view.RoutingStrategy = policy.RoutingStrategy
		view.AllowCostBreach = policy.AllowsCostBreach()
		view.Remark = policy.Remark
		view.UpdatedBy = policy.UpdatedBy
		view.UpdatedAt = policy.UpdatedAt
	}
	return view, nil
}

// CustomerRoutingUpdate 是写接口的入参。OperatorId 是操作的管理员，写进配置行供事后追责。
type CustomerRoutingUpdate struct {
	UserId          int
	RoutingStrategy string
	AllowCostBreach bool
	Remark          string
	OperatorId      int
}

// SaveCustomerRouting 写入（或更新）一个客户的路由策略。
func SaveCustomerRouting(update *CustomerRoutingUpdate) error {
	if update == nil || update.UserId <= 0 {
		return ErrDiscountRoutingInvalidUser
	}
	strategy := strings.TrimSpace(update.RoutingStrategy)
	if strategy != model.RoutingStrategyMargin && strategy != model.RoutingStrategyPriority {
		return ErrDiscountRoutingInvalidStrategy
	}
	// 先确认客户真的存在，否则会写出一行指向空气的配置。
	if _, err := model.GetUserById(update.UserId, false); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrDiscountRoutingUserNotFound
		}
		return err
	}

	breach := 0
	if update.AllowCostBreach {
		breach = 1
	}
	return model.SaveDiscountRoutingPolicy(&model.DiscountRoutingPolicy{
		UserId:          update.UserId,
		RoutingStrategy: strategy,
		AllowCostBreach: breach,
		Remark:          strings.TrimSpace(update.Remark),
		UpdatedBy:       update.OperatorId,
	})
}

// DeleteCustomerRouting 删掉一个客户的例外配置，回到默认口径（毛利优先、不允许击穿）。
func DeleteCustomerRouting(userId int) error {
	if userId <= 0 {
		return ErrDiscountRoutingInvalidUser
	}
	return model.DeleteDiscountRoutingPolicy(userId)
}
