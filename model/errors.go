package model

import "errors"

// Common errors
var (
	ErrDatabase = errors.New("database error")
)

// User auth errors
var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserEmptyCredentials = errors.New("empty credentials")
	ErrEmailAlreadyTaken    = errors.New("email already taken")
	ErrEmailNotFound        = errors.New("email not found")
	ErrEmailAmbiguous       = errors.New("email matches multiple users")
)

// Token auth errors
var (
	ErrTokenNotProvided = errors.New("token not provided")
	ErrTokenInvalid     = errors.New("token invalid")
)

// Redemption errors
var ErrRedeemFailed = errors.New("redeem.failed")

// 2FA errors
var ErrTwoFANotEnabled = errors.New("2fa not enabled")
var ErrTwoFAAlreadyEnabled = errors.New("2fa already enabled")

// Reset-inviter errors (task-16)
// 一律用显式 sentinel errors，方便 controller 直接 errors.Is 判别 → 翻译为
// 对应的 HTTP 状态码 + i18n 错误信息。target 用户不存在这种"恰好等于 gorm 错"
// 的场景不在这里——直接复用 gorm.ErrRecordNotFound。
var (
	ErrInviterSelf      = errors.New("inviter cannot be the same as target user")
	ErrInviterNotFound  = errors.New("inviter user not found")
	ErrInviterDeleted   = errors.New("inviter user has been deleted")
)
