package model

import (
	"errors"
	"fmt"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/shopspring/decimal"
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
	// ErrAgentTopupRateZero 经销商给客户发额度时方案折算比例为 0，禁止（钱白送）。
	ErrAgentTopupRateZero = errors.New("经销商发额度折算比例不能为 0")
	// ErrAgentTopupRateTooHigh 经销商给客户发额度时方案折算比例 > 1，禁止（扣得比面值还多）。
	ErrAgentTopupRateTooHigh = errors.New("经销商发额度折算比例必须 ≤ 1")
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
	/**
	 * 主方案的折算比例字符串（"0.875000"），task-09 用：经销商给客户发额度时按这个比例
	 * 扣经销商余额。前端预览与后端实扣都参考它；空字符串表示该客户没有生效的折扣方案，
	 * 服务端走 1.0 兜底（保持与改前一致）。
	 */
	TopupConversionRate string `json:"topup_conversion_rate"`
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
		row.TopupConversionRate = plan.TopupConversionRate
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
//
// task-09 加折算：扣经销商的钱按客户主方案的 TopupConversionRate（0~1 之间）折算，
// 客户按面值原额收。比例 = 0 或 > 1 都拒，避免钱白送或被多扣。
// 扣减用 ceil，cents 留给平台——经销商扣得略多一点（≤ 1 单位/笔），客户拿到的是干净的面值。
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

	var agentCost int // 提到事务外：缓存更新也要用实扣数，不能盲扣面值
	err = DB.Transaction(func(tx *gorm.DB) error {
		var agent User
		if err := lockForUpdate(tx).Select("id", "quota").First(&agent, agentId).Error; err != nil {
			return err
		}
		var customer User
		if err := lockForUpdate(tx).
			Select("id", "quota", "parent_agent_id", "subject_type", "discount_plan_id").
			First(&customer, customerId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrAgentCustomerNotFound
			}
			return err
		}
		if customer.ParentAgentId != agentId || customer.SubjectType == SubjectTypeAgent {
			return ErrAgentCustomerNotFound
		}

		// 折算比例：取客户主方案（task-06 维护的快路径）的 TopupConversionRate。
		// 没挂方案 / 方案被删 → 按缺省 1.0（保持与改前一致）。
		// agentCost 已被声明在事务外，这里用 = 赋值（不是 :=）。
		var rate decimal.Decimal
		agentCost, rate, err = resolveAgentTopupCost(tx, customer.DiscountPlanId, quota)
		if err != nil {
			return err
		}
		if rate.IsZero() {
			return ErrAgentTopupRateZero
		}
		if rate.GreaterThan(decimal.NewFromInt(1)) {
			return ErrAgentTopupRateTooHigh
		}
		if agent.Quota < agentCost {
			return ErrAgentQuotaNotEnough
		}

		if err := tx.Model(&User{}).Where("id = ?", agentId).
			Update("quota", gorm.Expr("quota - ?", agentCost)).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Where("id = ?", customerId).
			Update("quota", gorm.Expr("quota + ?", quota)).Error; err != nil {
			return err
		}

		// task-20 §20.3：经销商 grant 写两条 ledger 行（agent 扣 + customer 加），
		// 同事务：回滚时两条 ledger 同步回滚。
		//
		// amount 符号约定：amount 正数=入账（customer 收到），负数=出账（agent 付出）。
		// balance_after 紧跟 UPDATE 之后 SELECT，与 §20.1 topup、§20.2 refund 同口径。
		//
		// refType="agent_quota_grant"，refId=0 表示 grant 还没建专门的 grant_records 表——
		// 等 P3 客户管理加 grant 历史表时，refId 改为 grant_records.id，本接口签名不变。
		var agentBalanceAfter, customerBalanceAfter int64
		if err := tx.Model(&User{}).Select("quota").Where("id = ?", agentId).Scan(&agentBalanceAfter).Error; err != nil {
			return err
		}
		if err := tx.Model(&User{}).Select("quota").Where("id = ?", customerId).Scan(&customerBalanceAfter).Error; err != nil {
			return err
		}
		if err := RecordAccountLedger(tx,
			"user", agentId,
			AccountEventAgentQuotaGrant, -int64(agentCost), agentBalanceAfter,
			"agent_quota_grant", 0,
			fmt.Sprintf("agent grant to customer=%d face=%d cost=%d rate=%s", customerId, quota, agentCost, rate.String()),
			agentId, // operator_id=经销商自己（区别于系统自动 topup）
		); err != nil {
			return err
		}
		if err := RecordAccountLedger(tx,
			"user", customerId,
			AccountEventAgentQuotaGrant, int64(quota), customerBalanceAfter,
			"agent_quota_grant", 0,
			fmt.Sprintf("agent grant from agent=%d face=%d cost=%d rate=%s", agentId, quota, agentCost, rate.String()),
			agentId,
		); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 提交之后再动缓存，写法与 IncreaseUserQuota / DecreaseUserQuota 保持一致：
	// 缓存没跟上只是让界面上慢一拍，不能反过来影响已经落库的账。
	// 折算比例 < 1 时经销商实扣小于面值：缓存按实际扣减/增加走，不按面值盲扣。
	gopool.Go(func() {
		if err := cacheDecrUserQuota(agentId, int64(agentCost)); err != nil {
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

// resolveAgentTopupCost 把面值 quota 按客户主方案的比例折算成经销商实扣 agentCost。
//
// 返回 (agentCost, rate, error)：rate 留给 caller 校验边界（0 / >1）。
// planId = 0 或 plan 被删 → 按 1.0 兜底（兼容没挂方案的客户）。
// agentCost 用 ceil：cents 留在平台，避免浮点给经销商多扣 0.000...1 这种尾差。
func resolveAgentTopupCost(tx *gorm.DB, planId int, quota int) (int, decimal.Decimal, error) {
	rateStr := DiscountTopupConversionDefault
	if planId > 0 {
		var plan DiscountPlan
		if err := tx.Select("id", "topup_conversion_rate").First(&plan, planId).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 方案被删了，按 1.0 走；不让"客户身上方案被人删"成为整笔发额度失败的根因。
			} else {
				return 0, decimal.Zero, err
			}
		} else {
			rateStr = plan.TopupConversionRate
		}
	}
	if rateStr == "" {
		rateStr = DiscountTopupConversionDefault
	}
	rate, err := decimal.NewFromString(rateStr)
	if err != nil {
		return 0, decimal.Zero, fmt.Errorf("解析 topup_conversion_rate %q 失败: %w", rateStr, err)
	}
	quotaDec := decimal.NewFromInt(int64(quota))
	cost := quotaDec.Mul(rate).Ceil()
	agentCost := int(cost.IntPart())
	return agentCost, rate, nil
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
