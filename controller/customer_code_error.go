package controller

import (
	"errors"

	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
)

// customerCodeErrorMessageKey 把客户号这条链上的拒绝理由翻成客户看得懂的那一句。
//
// 绑号和注册前验号是两条入口，但拒绝理由是同一批（model 里的 ErrCustomerCode*），
// 所以翻译只写在这里一份；返回空串表示认不出来，由调用方按普通错误处理。
func customerCodeErrorMessageKey(err error) string {
	switch {
	case errors.Is(err, model.ErrCustomerCodeNotFound):
		return i18n.MsgCustomerCodeNotFound
	case errors.Is(err, model.ErrCustomerCodeRevoked):
		return i18n.MsgCustomerCodeRevoked
	case errors.Is(err, model.ErrCustomerCodeExpired):
		return i18n.MsgCustomerCodeExpired
	case errors.Is(err, model.ErrCustomerCodeExhausted):
		return i18n.MsgCustomerCodeExhausted
	case errors.Is(err, model.ErrCustomerCodeAlreadyBound):
		return i18n.MsgCustomerCodeAlreadyBound
	case errors.Is(err, model.ErrCustomerCodeSelfUse):
		return i18n.MsgCustomerCodeSelfUse
	case errors.Is(err, model.ErrCustomerCodeOwnedByAgent):
		return i18n.MsgCustomerCodeOwnedByAgent
	case errors.Is(err, model.ErrCustomerBelongsToOtherAgent):
		return i18n.MsgCustomerBelongsToOtherAgent
	case errors.Is(err, model.ErrCustomerCodePlanUnavailable):
		return i18n.MsgCustomerCodePlanUnavailable
	}
	return ""
}
