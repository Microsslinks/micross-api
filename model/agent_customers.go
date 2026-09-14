package model

import (
	"errors"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

// 我的客户：经销商看自己名下的客户，给他们改价、给他们发额度。
//
// 「谁是我的客户」不看客户号，只看 users.parent_agent_id 指向谁——绑号时落的那个归属，
// 是唯一的判据。所以客户号被作废、被用满都不影响他已经归到谁名下。
//
// 三件事在计价上各归各位：
//   - 改价 = 在客户身上落一条 source=agent 的折扣绑定（比套餐、客户号高，比平台手工定价低）
//   - 发额度 = 把经销商自己账上的额度转给客户（钱从他的余额出，不是平台补贴）
//   - 名单 = 读 parent_agent_id，顺带把客户此刻生效的方案与余额摊开给经销商看

var (
	// ErrAgentCustomerNotFound 这个人不在你的名下（或他自己已经是经销商）。
	// 与"查无此人"合并处理，不确认别人名下的客户是否存在。
	ErrAgentCustomerNotFound = errors.New("这位客户不在你的名下")
	// ErrCustomerPricedByPlatform 平台已经单独给这位客户定过价，经销商不能顶掉。
	ErrCustomerPricedByPlatform = errors.New("平台已为这位客户单独定价，请联系平台")
	// ErrAgentPlanNotSellable 这个方案不在经销商的货架上（不存在、已停用或低于他的最低售价折扣）。
	ErrAgentPlanNotSellable = errors.New("这个折扣方案不在你的货架上")
	// ErrAgentQuotaIssuingDisabled 经销商给下属客户发额度的开关是关的。
	ErrAgentQuotaIssuingDisabled = errors.New("给下属客户发额度已关闭")
	// ErrAgentQuotaNotEnough 经销商自己账上的额度不够。
	ErrAgentQuotaNotEnough = errors.New("你自己的额度不足")
	// ErrAgentQuotaInvalid 发的额度数不是一个正数。
	ErrAgentQuotaInvalid = errors.New("额度必须是正数")
)

// AgentCustomer 是经销商名下一个客户的台账行。
//
// Quota / UsedQuota 是只读的展示字段：客户还剩多少、累计用了多少。经销商不能用这里
// 的数字做别的事——发额度走 IssueQuotaToCustomer，改价走 SetAgentCustomerDiscount。
type AgentCustomer struct {
	UserId      int    `json:"user_id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Status      int    `json:"status"`
	CreatedAt   int64  `json:"created_at"`
	Quota       int    `json:"quota"`
	UsedQuota   int    `json:"used_quota"`
	/** 这位客户此刻挂着的全部生效绑定，按来源优先级排序（见 buildAgentCustomerBindings） */
	Bindings []*AgentCustomerBinding `json:"bindings"`
	/** 下面四个是单方案时代的老字段，保留给还没切过来的调用方；新界面读 Bindings */
	PlanId        int    `json:"plan_id"`
	PlanName      string `json:"plan_name"`
	PlanDiscount  string `json:"plan_discount"`
	BindingSource string `json:"binding_source"`
	/** 是否由经销商自己定过价（= 有 source=agent 的生效绑定），界面据此决定"恢复"按钮显不显示 */
	PricedByMe bool `json:"priced_by_me"`
}

// AgentCustomerBinding 是这位客户身上一条正在生效的折扣绑定。
//
// 一个客户可以同时挂多套方案（task-06），所以台账行不能再把它压成一套价：
// 挂了几套就摊几套，每套带上是"谁定的"，经销商才看得见全貌。
type AgentCustomerBinding struct {
	BindingId    int    `json:"binding_id"`
	PlanId       int    `json:"plan_id"`
	PlanName     string `json:"plan_name"`
	PlanDiscount string `json:"plan_discount"`
	Source       string `json:"source"`
}

// ListAgentCustomers 按 offset/limit 列出经销商名下的客户（与 ListCustomerCodes 同一口径）。
//
// 已经在名单里的客户如果后来被平台设成了经销商，就不再算客户：他是同行的对手，
// 归属也已在设为经销商那一刻清掉（见 model/agent.go 的 promoteUserToAgent）。
func ListAgentCustomers(agentId int, offset int, limit int) ([]*AgentCustomer, int, error) {
	return listAgentCustomers(agentId, 0, offset, limit)
}

// GetAgentCustomer 读一位客户此刻的台账行，供改价/发额度之后回给界面刷新那一行。
func GetAgentCustomer(agentId int, customerId int) (*AgentCustomer, error) {
	rows, _, err := listAgentCustomers(agentId, customerId, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrAgentCustomerNotFound
	}
	return rows[0], nil
}

// listAgentCustomers 是列表与单行的共同实现；filterUserId > 0 时只读那一个人。
func listAgentCustomers(agentId int, filterUserId int, offset int, limit int) ([]*AgentCustomer, int, error) {
	if agentId <= 0 {
		return nil, 0, errors.New("无效的经销商 ID")
	}
	if offset < 0 {
		offset = 0
	}
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	query := DB.Model(&User{}).
		Where("parent_agent_id = ? AND subject_type <> ?", agentId, SubjectTypeAgent)
	if filterUserId > 0 {
		query = query.Where("id = ?", filterUserId)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*AgentCustomer{}, 0, nil
	}

	customers := make([]*User, 0)
	if err := query.
		Select("id", "username", "display_name", "status", "created_at", "quota", "used_quota").
		Order("id DESC").Limit(limit).Offset(offset).
		Find(&customers).Error; err != nil {
		return nil, 0, err
	}

	// 这一页客户的生效折扣一次读完：逐行查会让列表变成 N+1 次数据库往返。
	customerIds := make([]int, 0, len(customers))
	for _, customer := range customers {
		customerIds = append(customerIds, customer.Id)
	}
	bindings := make([]DiscountBinding, 0)
	if err := DB.Where("subject_id IN ? AND status = ?", customerIds, DiscountStatusEnabled).
		Find(&bindings).Error; err != nil {
		return nil, 0, err
	}
	planIds := make([]int, 0, len(bindings))
	for i := range bindings {
		planIds = append(planIds, bindings[i].PlanId)
	}
	plans := map[int]*DiscountPlan{}
	if len(planIds) > 0 {
		planRows := make([]*DiscountPlan, 0)
		if err := DB.Where("id IN ?", planIds).Find(&planRows).Error; err != nil {
			return nil, 0, err
		}
		for _, plan := range planRows {
			plans[plan.Id] = plan
		}
	}

	now := common.GetTimestamp()
	rows := make([]*AgentCustomer, 0, len(customers))
	for _, customer := range customers {
		mine := make([]DiscountBinding, 0, len(bindings))
		for i := range bindings {
			if bindings[i].SubjectId == customer.Id {
				mine = append(mine, bindings[i])
			}
		}
		rows = append(rows, buildAgentCustomer(customer, mine, plans, now))
	}
	return rows, int(total), nil
}

// buildAgentCustomer 把一行客户拼成台账行：挂着的绑定全摊开，另外记一条"单方案口径"的生效绑定。
//
// 老字段（PlanId/PlanName/PlanDiscount/BindingSource）留着是为了不打断还没有切到 Bindings 的调用方，
// 它们仍然等于"只挂一套时的那套价"。
func buildAgentCustomer(customer *User, bindings []DiscountBinding, plans map[int]*DiscountPlan, now int64) *AgentCustomer {
	row := &AgentCustomer{
		UserId:      customer.Id,
		Username:    customer.Username,
		DisplayName: customer.DisplayName,
		Status:      customer.Status,
		CreatedAt:   customer.CreatedAt,
		Quota:       customer.Quota,
		UsedQuota:   customer.UsedQuota,
		Bindings:    buildAgentCustomerBindings(bindings, plans, now),
	}
	active := pickActiveDiscountBinding(bindings, now)
	if active == nil {
		return row
	}
	row.PlanId = active.PlanId
	row.BindingSource = active.Source
	row.PricedByMe = active.Source == DiscountSourceAgent
	if plan, ok := plans[active.PlanId]; ok {
		row.PlanName = plan.Name
		row.PlanDiscount = formatResolvedDiscount(plan.BaseDiscount)
	}
	return row
}

// buildAgentCustomerBindings 摊开这位客户此刻生效的全部绑定。
//
// 只有"此刻在生效窗口内"的才列出来：窗口还没开始或已经结束的绑定不是他现在的价，
// 列出来会让经销商以为自己给的价没生效。
//
// 排序沿用计价挑绑定时的那套来源优先级（平台手工 > 经销商 > 套餐 > 客户号 > 迁移），
// 同来源取新绑的那条。要注意的是计价还会先看"规则具体不具体"（某个模型单独设过规则就先用它），
// 所以排序只表示"谁更有话语权"，不等于"这套一定在生效"——界面文案不能写成只有一套价。
func buildAgentCustomerBindings(bindings []DiscountBinding, plans map[int]*DiscountPlan, now int64) []*AgentCustomerBinding {
	inWindow := make([]DiscountBinding, 0, len(bindings))
	for i := range bindings {
		binding := bindings[i]
		if binding.Status != DiscountStatusEnabled || binding.EffectiveFrom > now {
			continue
		}
		if binding.EffectiveTo > 0 && binding.EffectiveTo <= now {
			continue
		}
		inWindow = append(inWindow, binding)
	}
	sort.SliceStable(inWindow, func(a int, b int) bool {
		rankA := discountSourceRank(inWindow[a].Source)
		rankB := discountSourceRank(inWindow[b].Source)
		if rankA != rankB {
			return rankA < rankB
		}
		return inWindow[a].Id > inWindow[b].Id
	})

	rows := make([]*AgentCustomerBinding, 0, len(inWindow))
	for i := range inWindow {
		binding := inWindow[i]
		row := &AgentCustomerBinding{
			BindingId: binding.Id,
			PlanId:    binding.PlanId,
			Source:    binding.Source,
		}
		if plan, ok := plans[binding.PlanId]; ok {
			row.PlanName = plan.Name
			row.PlanDiscount = formatResolvedDiscount(plan.BaseDiscount)
		}
		rows = append(rows, row)
	}
	return rows
}

// SetAgentCustomerDiscount 给一位下属客户定价：planId 传 0 表示撤掉自己定的价，让他回到
// 客户号或平台给的价。
//
// 两条线不能越：方案必须在自己货架上（停用的、低于自己最低售价折扣的都卖不了），
// 平台已经单独定过价的客户不能碰——那是平台的决定，经销商改不动。
func SetAgentCustomerDiscount(agentId int, customerId int, planId int) error {
	if planId < 0 {
		return errors.New("无效的折扣方案 ID")
	}
	if _, err := GetAgentProfileByUserId(agentId); err != nil {
		return err
	}
	if err := ensureCustomerBelongsToAgent(agentId, customerId); err != nil {
		return err
	}

	// 平台单独定过价的客户，经销商不能覆盖；他自己的价（source=agent）不在此列，随便改。
	var platformPriced int64
	if err := DB.Model(&DiscountBinding{}).
		Where("subject_id = ? AND source = ? AND status = ?",
			customerId, DiscountSourceManual, DiscountStatusEnabled).
		Count(&platformPriced).Error; err != nil {
		return err
	}
	if platformPriced > 0 {
		return ErrCustomerPricedByPlatform
	}

	if planId > 0 {
		// 货架是唯一的准入口径：能卖的方案一定来自它（停用的、低于下限的都不在里面）。
		_, plans, err := ListSellablePlansForAgent(agentId)
		if err != nil {
			return err
		}
		onShelf := false
		for _, sellable := range plans {
			if sellable.Id == planId {
				onShelf = true
				break
			}
		}
		if !onShelf {
			return ErrAgentPlanNotSellable
		}
	}

	return DB.Transaction(func(tx *gorm.DB) error {
		// 先撤掉自己上次定的价：一个人身上只留一条经销商定价，不留一堆历史绑定。
		if err := tx.Model(&DiscountBinding{}).
			Where("subject_id = ? AND source = ? AND status = ?",
				customerId, DiscountSourceAgent, DiscountStatusEnabled).
			Update("status", DiscountStatusDisabled).Error; err != nil {
			return err
		}
		if planId <= 0 {
			return syncUserDiscountPlanId(tx, customerId)
		}
		// 直接新插一条，不去改写客户号留下的那条同方案绑定：客户号是经销商签出去的凭证，
		// 上面写着当初答应客户的价，不该被后来的改价抹掉。两条主张并存、由来源优先级
		// 决定谁生效（经销商定价排在客户号之上），撤价时他才真的回到客户号那份价上。
		binding := &DiscountBinding{
			SubjectType: DiscountSubjectUser,
			SubjectId:   customerId,
			PlanId:      planId,
			Source:      DiscountSourceAgent,
			Status:      DiscountStatusEnabled,
		}
		if err := tx.Create(binding).Error; err != nil {
			return err
		}
		return syncUserDiscountPlanId(tx, customerId)
	})
}

// IssueQuotaToCustomer 给一位下属客户发额度：钱从经销商自己的余额出。
//
// 两行必须在同一事务里一起动，并且都锁住：少一半就是"扣了没给"或"给了没扣"。
// 锁的顺序固定为先经销商、后客户，两个方向同时操作也不会互相等死。
func IssueQuotaToCustomer(agentId int, customerId int, quota int) error {
	if quota <= 0 {
		return ErrAgentQuotaInvalid
	}
	profile, err := GetAgentProfileByUserId(agentId)
	if err != nil {
		return err
	}
	if profile.IssueQuotaEnabled != CustomerCodeStatusEnabled {
		return ErrAgentQuotaIssuingDisabled
	}

	err = DB.Transaction(func(tx *gorm.DB) error {
		var agent User
		if err := lockForUpdate(tx).Select("id", "quota").First(&agent, agentId).Error; err != nil {
			return err
		}
		if agent.Quota < quota {
			return ErrAgentQuotaNotEnough
		}
		var customer User
		if err := lockForUpdate(tx).
			Select("id", "quota", "parent_agent_id", "subject_type").
			First(&customer, customerId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAgentCustomerNotFound
			}
			return err
		}
		if customer.ParentAgentId != agentId || customer.SubjectType == SubjectTypeAgent {
			return ErrAgentCustomerNotFound
		}

		if err := tx.Model(&User{}).Where("id = ?", agentId).
			Update("quota", gorm.Expr("quota - ?", quota)).Error; err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", customerId).
			Update("quota", gorm.Expr("quota + ?", quota)).Error
	})
	if err != nil {
		return err
	}

	// 提交之后再动缓存，写法与 IncreaseUserQuota / DecreaseUserQuota 保持一致：
	// 缓存没跟上只是让界面上慢一拍，不能反过来影响已经落库的账。
	gopool.Go(func() {
		if err := cacheDecrUserQuota(agentId, int64(quota)); err != nil {
			common.SysLog("failed to update quota cache for agent: " + err.Error())
		}
	})
	gopool.Go(func() {
		if err := cacheIncrUserQuota(customerId, int64(quota)); err != nil {
			common.SysLog("failed to update quota cache for customer: " + err.Error())
		}
	})
	return nil
}

// ensureCustomerBelongsToAgent 确认这个人确实是这位经销商名下的客户。
// 不是的话一律按"不在你名下"回报，不确认别人名下的客户是否存在。
func ensureCustomerBelongsToAgent(agentId int, customerId int) error {
	if customerId <= 0 {
		return ErrAgentCustomerNotFound
	}
	var customer User
	err := DB.Select("id", "subject_type", "parent_agent_id").
		First(&customer, customerId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrAgentCustomerNotFound
	}
	if err != nil {
		return err
	}
	if customer.SubjectType == SubjectTypeAgent || customer.ParentAgentId != agentId {
		return ErrAgentCustomerNotFound
	}
	return nil
}
