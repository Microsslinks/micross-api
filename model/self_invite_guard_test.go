package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateInviterForRegistration_* 守住 task-20 §20.4 自邀拦截四条规则：
//   - 24h 新账号不能作为邀请人（批号薅羊毛防护）
//   - 同 email 域名（最常见的自邀伪装）
//   - 24h 内邀请人数 ≥ 5（刷单主控账号防护）
//   - 不存在的 inviter / 软删 inviter（基础完整性，沿用 ErrInviterNotFound / Deleted）
//
// 这些是 §20.4 第一阶段（自邀拦截）的"基础事实"——后续 §20.5（首充门槛 / 成环检测）
// 在同一 controller 入口叠加，但不互斥。

func seedInviterWithCreatedAt(t *testing.T, id int, email string, createdAt int64) *User {
	t.Helper()
	user := &User{
		Id:        id,
		Username:  "inviter-" + email,
		AffCode:   "inviter-aff-" + email,
		Email:     email,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		CreatedAt: createdAt,
	}
	require.NoError(t, DB.Create(user).Error)
	return user
}

// 1. inviter 注册 < 24h → ErrInviterTooNew
func TestValidateInviterTooNew(t *testing.T) {
	// TestMain 启动时 setMaxOpenConns(1)，DB 是 sqlite in-memory。直接调。
	now := common.GetTimestamp()
	recentCreated := now - 3600 // 1h ago
	inviter := seedInviterWithCreatedAt(t, 6001, "newinviter@example.com", recentCreated)

	err := ValidateInviterForRegistration(inviter.Id, "newbie@example.org")
	assert.ErrorIs(t, err, ErrInviterTooNew,
		"inviter 注册 1h < 24h 阈值，应返回 ErrInviterTooNew")
}

// 2. inviter 与被摄者 email 域名相同 → ErrInviterSameEmailDomain
func TestValidateInviterSameEmailDomain(t *testing.T) {
	now := common.GetTimestamp()
	oldCreated := now - int64(48*3600) // 48h ago（ageHours 满足）
	inviter := seedInviterWithCreatedAt(t, 6002, "master@badco.com", oldCreated)

	err := ValidateInviterForRegistration(inviter.Id, "newbie@badco.com")
	assert.ErrorIs(t, err, ErrInviterSameEmailDomain,
		"同 email 域名应返回 ErrInviterSameEmailDomain")
}

// 3. inviter 过去 24h 邀请 ≥ 5 → ErrInviterTooActive
func TestValidateInviterTooActive(t *testing.T) {
	now := common.GetTimestamp()
	oldCreated := now - int64(48*3600)
	inviter := seedInviterWithCreatedAt(t, 6003, "busyinviter@example.com", oldCreated)

	// 模拟过去 24h 内已邀请了 5 个用户（不同 email 域名以免触碰第 3 条）
	for i := 0; i < 5; i++ {
		require.NoError(t, DB.Create(&User{
			Id:        7000 + i,
			Username:  "downstream-" + emailAlias(i),
			AffCode:   "downstream-aff-" + emailAlias(i),
			Email:     "downstream" + emailAlias(i) + "@example.org",
			Status:    common.UserStatusEnabled,
			InviterId: inviter.Id,
			CreatedAt: now - int64(i*3600), // 都 < 24h
		}).Error)
	}

	err := ValidateInviterForRegistration(inviter.Id, "newbie@example.org")
	assert.ErrorIs(t, err, ErrInviterTooActive,
		"24h 内邀请数 ≥ 5 应返回 ErrInviterTooActive")
}

// 4. 24h 边界：24h 内恰好 4 个邀请 → 通过
func TestValidateInviterAtBoundaryPasses(t *testing.T) {
	now := common.GetTimestamp()
	oldCreated := now - int64(48*3600)
	inviter := seedInviterWithCreatedAt(t, 6004, "healthyinviter@example.com", oldCreated)

	// 24h 内邀请 4 个（< 5 阈值）
	for i := 0; i < 4; i++ {
		require.NoError(t, DB.Create(&User{
			Id:        8000 + i,
			Username:  "downstream-healthy-" + emailAlias(i),
			AffCode:   "downstream-healthy-aff-" + emailAlias(i),
			Email:     "downstream-healthy-" + emailAlias(i) + "@example.org",
			Status:    common.UserStatusEnabled,
			InviterId: inviter.Id,
			CreatedAt: now - int64(i*3600),
		}).Error)
	}

	err := ValidateInviterForRegistration(inviter.Id, "newbie@example.org")
	assert.NoError(t, err,
		"24h 内 4 个邀请（< 5 阈值）应通过")
}

// 5. inviter 24h+ 前 + 域名不同 + 邀请 < 5 → 无错误（happy path）
func TestValidateInviterHappyPath(t *testing.T) {
	now := common.GetTimestamp()
	oldCreated := now - int64(48*3600)
	inviter := seedInviterWithCreatedAt(t, 6005, "cleanmaster@example.com", oldCreated)

	err := ValidateInviterForRegistration(inviter.Id, "newbie@example.org")
	assert.NoError(t, err)
}

// 6. inviterId=0 跳过所有风控
func TestValidateInviterSkipsOnZeroID(t *testing.T) {
	assert.NoError(t, ValidateInviterForRegistration(0, "any@email.com"))
}

// 7. inviter 不存在 → ErrInviterNotFound
func TestValidateInviterNotFound(t *testing.T) {
	err := ValidateInviterForRegistration(9999999, "newbie@example.org")
	assert.ErrorIs(t, err, ErrInviterNotFound)
}

// emailAlias helper：给 0..N 索引返回 "a","b","c",..（避免 i 拼字符串歧义）。
func emailAlias(i int) string {
	if i < 0 {
		i = 0
	}
	alphabet := "abcdefghijklmnopqrstuvwxyz"
	if i >= len(alphabet) {
		i = len(alphabet) - 1
	}
	return string(alphabet[i])
}