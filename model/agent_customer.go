package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// 客户绑号：客户拿着经销商发给他的号，把自己归到那位经销商名下，并按号上的方案计价。
//
// 签发只是造了张凭证，绑定才是号真正生效的地方：它在 users.parent_agent_id 上落下归属，
// 在 discount_bindings 上落下折扣（来源 customer_code），在号上记下用量与"是谁用掉的"。
// 这几件事必须在同一个事务里，少一件就会出现「归属落了但价没变」，或者一张号被反复用。
//
// 归属落在客户自己身上，所以经销商名下的客户就是 parent_agent_id 指向他的人；号这边只留
// 一条「谁用掉的」备查（见 CustomerCode.BoundUserId）。一张号对应一位客户，用完即废。

var (
	// ErrCustomerCodeRevoked 号已被经销商作废。
	ErrCustomerCodeRevoked = errors.New("客户号已作废")
	// ErrCustomerCodeExpired 号过了有效期。
	ErrCustomerCodeExpired = errors.New("客户号已过期")
	// ErrCustomerCodeExhausted 号的使用次数用完了。
	ErrCustomerCodeExhausted = errors.New("客户号使用次数已用完")
	// ErrCustomerCodeAlreadyBound 同一个人拿同一张号（或同一个方案）再绑一次。
	// 这不算失败，但要说清楚：没有发生任何变化，也没有再用掉一次。
	ErrCustomerCodeAlreadyBound = errors.New("你已经是这位经销商的客户了")
	// ErrCustomerCodeSelfUse 经销商用自己发的号。
	ErrCustomerCodeSelfUse = errors.New("不能使用自己发出的客户号")
	// ErrCustomerCodeOwnedByAgent 经销商拿别人的号：经销商本身直属平台。
	ErrCustomerCodeOwnedByAgent = errors.New("经销商直属平台，不能使用客户号")
	// ErrCustomerBelongsToOtherAgent 已经归属另一位经销商。
	// 归属只能由平台改：否则任何人随便拿到一张号就能把别人的客户挖走。
	ErrCustomerBelongsToOtherAgent = errors.New("你已经归属另一位经销商，请联系平台")
)

// CustomerBinding 是「这个客户此刻挂在谁名下、按什么价算」的一份快照。
// ParentAgentId 为 0 表示直属平台（没有归属经销商）。
type CustomerBinding struct {
	ParentAgentId int    `json:"parent_agent_id"`
	AgentName     string `json:"agent_name"`
	PlanId        int    `json:"plan_id"`
	PlanName      string `json:"plan_name"`
	PlanDiscount  string `json:"plan_discount"`  // 方案基准折扣（个别模型可能另有专门折扣）
	BindingSource string `json:"binding_source"` // DiscountSource* 之一，空表示没有生效方案
}

// GetCustomerBinding 读出这个客户当前的归属与折扣，供界面显示。
// 没有任何归属也没有生效方案时返回一份零值快照，不报错——那是最常见的正常状态。
func GetCustomerBinding(customerId int) (*CustomerBinding, error) {
	binding := &CustomerBinding{}
	if customerId <= 0 {
		return binding, nil
	}

	var user User
	err := DB.Select("id", "parent_agent_id", "discount_plan_id").First(&user, customerId).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return binding, nil
		}
		return nil, err
	}
	binding.ParentAgentId = user.ParentAgentId
	if user.ParentAgentId > 0 {
		// 只取名字：这一趟是为了把界面上的那一行字填上，没必要把整个用户读出来。
		names := make([]string, 0, 1)
		if err := DB.Model(&User{}).Where("id = ?", user.ParentAgentId).
			Pluck("username", &names).Error; err != nil {
			return nil, err
		}
		if len(names) > 0 {
			binding.AgentName = names[0]
		}
	}

	// 折扣取"此刻真正生效的那一条"，与计费读到的口径一致（同一份 pickActiveDiscountBinding）。
	bindings := make([]DiscountBinding, 0)
	if err := DB.Where("subject_id = ? AND status = ?", customerId, DiscountStatusEnabled).
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	active := pickActiveDiscountBinding(bindings, common.GetTimestamp())
	if active == nil {
		return binding, nil
	}
	binding.PlanId = active.PlanId
	binding.BindingSource = active.Source
	plan, err := GetDiscountPlanById(active.PlanId)
	if err != nil {
		// 快路径指向的方案已经不在了：按"没有方案"显示，不让阅读界面跟着失败。
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return binding, nil
		}
		return nil, err
	}
	binding.PlanName = plan.Name
	binding.PlanDiscount = formatResolvedDiscount(plan.BaseDiscount)
	return binding, nil
}

// BindCustomerCode 让客户用一张客户号绑定：落归属 + 绑折扣 + 号上用量 +1。
//
// 这张号必须是"此刻还能用"的：未作废、未过期、没被用满，且号上指定的方案仍在启用中。
// 两种情况会被明确拒绝：自己发自己的号用、已经归属了别人——后者只能由平台改。
func BindCustomerCode(customerId int, rawCode string) (*CustomerBinding, error) {
	if customerId <= 0 {
		return nil, errors.New("无效的用户 ID")
	}
	// 号在库里是大写，客户手抄进来常常是小写，这里统一成大写去找，不让他为大小写重试。
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return nil, ErrCustomerCodeNotFound
	}

	err := DB.Transaction(func(tx *gorm.DB) error {
		// 锁住这一行再判断：两张请求同时用同一张号的最后一次用量时，
		// 没锁的话两边都会读到"还能用"，然后各扣一次（见 model/locking.go）。
		var record CustomerCode
		err := lockForUpdate(tx).Where("code = ?", code).First(&record).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrCustomerCodeNotFound
			}
			return err
		}
		now := common.GetTimestamp()
		if record.AgentId == customerId {
			return ErrCustomerCodeSelfUse
		}

		var customer User
		if err := lockForUpdate(tx).Select("id", "subject_type", "parent_agent_id").
			First(&customer, customerId).Error; err != nil {
			return err
		}
		if customer.SubjectType == SubjectTypeAgent {
			return ErrCustomerCodeOwnedByAgent
		}
		if customer.ParentAgentId != 0 && customer.ParentAgentId != record.AgentId {
			return ErrCustomerBelongsToOtherAgent
		}

		// 这个人是不是已经有这个方案了（不管当初是平台给的还是上一张号给的）。
		// 已有即视为重复绑定：不重复建绑定、也不再扣一次用量，只把话说清楚。
		//
		// 这一步必须排在「号自身状态」之前。一张号只给一位客户，客户自己用掉之后再绑一次
		// （手滑，或者忘了自己已经绑过），号上就已经是"用满"了——先看号的状态会告诉他
		// "使用次数已用完"，可他正是用掉这张号的人，该听到的是"你已经是这位经销商的客户了"。
		var planBound int64
		if record.PlanId > 0 {
			if err := tx.Model(&DiscountBinding{}).
				Where("subject_id = ? AND plan_id = ? AND status = ?",
					customerId, record.PlanId, DiscountStatusEnabled).
				Count(&planBound).Error; err != nil {
				return err
			}
		}
		if customer.ParentAgentId == record.AgentId && (record.PlanId == 0 || planBound > 0) {
			return ErrCustomerCodeAlreadyBound
		}

		// 到这里才谈得上"号本身还能不能用"：号的状态（作废/过期/次数用满）与注册前的
		// 预检共用同一份判断，避免两处口径走散。
		if err := customerCodeRejectReason(&record, now); err != nil {
			return err
		}
		// 号上带的方案必须还能用：方案停用后客户绑上去只会以为自己有折扣。
		if err := ensureCustomerCodePlanUsable(record.PlanId); err != nil {
			return err
		}

		if err := tx.Model(&User{}).Where("id = ?", customerId).
			UpdateColumn("parent_agent_id", record.AgentId).Error; err != nil {
			return err
		}
		if record.PlanId > 0 && planBound == 0 {
			// 这位客户已经被挂满了：先拒绝整笔，而不是"只落归属、不带折扣"——
			// 后者会让他以为自己拿到了号上的价，而这张号已经被扣掉一次用量。
			// 整笔拒绝意味着什么都没发生，平台解绑一套之后他还能用同一张号。
			if err := ensureDiscountBindingQuota(tx, DiscountSubjectUser, customerId); err != nil {
				return err
			}
			// 与 BindDiscountPlan 同一口径：先写绑定事实，再重算 users.discount_plan_id 快路径。
			// 这里不复用 BindDiscountPlan 是因为它自带事务，开不了嵌套。
			discountBinding := &DiscountBinding{
				SubjectType: DiscountSubjectUser,
				SubjectId:   customerId,
				PlanId:      record.PlanId,
				Source:      DiscountSourceCustomerCode,
				Status:      DiscountStatusEnabled,
			}
			if err := tx.Create(discountBinding).Error; err != nil {
				return err
			}
			if err := syncUserDiscountPlanId(tx, customerId); err != nil {
				return err
			}
		}
		// 用量 +1，同时记下"这张号是谁用掉的"：号用完即废，但这条记录要一直留着——
		// 归属落在客户身上，将来只有这里能回答"这位客户当初是谁带来的"。
		return tx.Model(&CustomerCode{}).Where("id = ?", record.Id).
			Updates(map[string]interface{}{
				"used_count":    record.UsedCount + 1,
				"bound_user_id": customerId,
				"updated_at":    now,
			}).Error
	})
	if err != nil {
		return nil, err
	}

	// 绑定完读一遍真实结果返回：界面拿到的就是库里此刻的状态，而不是我推测的状态。
	return GetCustomerBinding(customerId)
}
