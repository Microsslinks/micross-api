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

// P4 风控自邀拦截 (task-20 §20.4) — 注册阶段拒绝以下邀请关系：
//   - 邀请人账户创建时间 < minInviterAgeHours（默认 24h）
//     阻止"批量脚本注册 → 立刻用主控账号 aff_code 互邀"的薅羊毛手法
//   - 邀请人与被摄者 email 域名相同
//     阻止"主控账号与新账号共用 @example.com"的常见自邀伪装
//   - 邀请人累计邀请人数超过 maxInviterItersPerDay（默认 5/24h）
//     阻止"主控账号每天拉 100 个新号"的刷单手法
//
// 失败用显式 sentinel errors，方便 controller 翻译为 i18n + HTTP 状态码；
// 与 ErrInviterSelf/NotFound/Deleted 一并成为 controller 端 errors.Is 判断的家族。
//
// 这些拦截只对**新用户注册时**生效；admin ResetInviter 路径（task-16）不动——
// admin 信任不受风控约束。
var (
	ErrInviterTooNew         = errors.New("inviter account is too new (must be at least 24h old)")
	ErrInviterSameEmailDomain = errors.New("inviter and invitee share the same email domain")
	ErrInviterTooActive       = errors.New("inviter has reached the daily invitation cap")

	// task-20 §20.6: 同一笔 commission 已被冲销，重复调用 ReverseCommission 时返回。
	// controller 看到它会翻译为 HTTP 409 Conflict，UI 上展示"该返佣已撤销"提示。
	ErrCommissionAlreadyReversed = errors.New("commission record already reversed")
)
