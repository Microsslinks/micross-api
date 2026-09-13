package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// 客户号在「绑定之前」的可用性预检。
//
// 真正的判定在 agent_customer.go 的 BindCustomerCode 里——它在事务里 lockForUpdate 之后
// 才下结论，那是防并发抢用一张号的地方。这里这一趟只读、不加锁，目的不同：
// 让调用方在动手写数据之前就知道这张号能不能用。注册时带着一张废号进来，
// 应该当场被拦下（一个账号都不建），而不是先把账号建好、再回头告诉客户「号没生效」。

// customerCodeRejectReason 这张号此刻不能用的话，到底是哪一条。
//
// 只看号自身的状态（作废 / 过期 / 次数用满），不看是谁在绑：
// 「谁在绑」那一类限制（自己发给自己、已经归属别人）要等到客户身份确定之后才判得了，
// 留在 BindCustomerCode 里。绑号与预检共用这一份判断，避免两处口径走散。
func customerCodeRejectReason(record *CustomerCode, now int64) error {
	switch {
	case record.Status != CustomerCodeStatusEnabled:
		return ErrCustomerCodeRevoked
	case record.ExpiredAt > 0 && now > record.ExpiredAt:
		return ErrCustomerCodeExpired
	case record.MaxUses > 0 && record.UsedCount >= record.MaxUses:
		return ErrCustomerCodeExhausted
	}
	return nil
}

// CheckCustomerCodeUsable 预检一张客户号：能用就把它取回来，不能用就给出具体理由。
//
// 大小写与前后空白在这里就抹平（号在库里是大写，客户手抄常常是小写），
// 与 BindCustomerCode 同一口径。号上带的方案也必须还能用——号本身有效但方案停用的话，
// 绑上去客户只会以为自己有折扣。
//
// 只读带来的固有缺口：这一趟读到「还能用」，下一瞬间号可能刚好被别人用满。
// 这是可以接受的——真正的判定仍在 BindCustomerCode 的事务里，这里只是让常见错误
// 在建数据之前就暴露出来。
func CheckCustomerCodeUsable(rawCode string) (*CustomerCode, error) {
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		return nil, ErrCustomerCodeNotFound
	}

	record, err := GetCustomerCodeByCode(code)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCustomerCodeNotFound
		}
		return nil, err
	}
	if err := customerCodeRejectReason(record, common.GetTimestamp()); err != nil {
		return nil, err
	}
	if err := ensureCustomerCodePlanUsable(record.PlanId); err != nil {
		return nil, err
	}
	return record, nil
}
