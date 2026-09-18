package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDetectInviteRing_* 守住 task-20 §20.5 成环检测。
// DetectInviteRing 沿 inviter 链上溯最多 MaxRingDepth（默认 5）跳，
// 看是否能再次访问到起始 userID——是则视为闭环（A→B→A）。
//
// 测试聚焦三种形态：
//   - 直接 2 跳环：A→B→A
//   - 3 跳环：A→B→C→A
//   - 5 跳刚好环：A→B→C→D→E→A（恰好命中边界深度）
//   - 6 跳环（超出 MaxRingDepth=5）：不应被检测到（深度上限保护）
//   - 链头（inviter=0）：无环
//   - 单链（无环）：不返回 true

func seedInviterChain(t *testing.T, chain []int /* user_id 序列 */) {
	t.Helper()
	for i, id := range chain {
		var inviterID int
		if i > 0 {
			inviterID = chain[i-1]
		}
		require.NoError(t, DB.Create(&User{
			Id:        id,
			Username:  "chain-user-" + intToStr(id),
			AffCode:   "chain-aff-" + intToStr(id),
			Status:    common.UserStatusEnabled,
			Group:     "default",
			InviterId: inviterID,
		}).Error)
	}
}

func intToStr(i int) string {
	if i == 0 {
		return "0"
	}
	return string(rune('0' + i%10)) + intToStr(i/10)
}

func TestDetectInviteRing2Hop(t *testing.T) {
	// A(100) → B(101) → A(100)
	// seedInviterChain 已经是 i-1 → i 的链（A.inviter=0, B.inviter=A）；
	// 还要把 A.inviter 改成 B 才能形成 A→B→A 闭环。
	seedInviterChain(t, []int{100, 101})
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 100).Update("inviter_id", 101).Error)

	assert.True(t, DetectInviteRing(100), "2 跳环 A→B→A 必须命中")
}

func TestDetectInviteRing3Hop(t *testing.T) {
	// A → B → C → A
	seedInviterChain(t, []int{200, 201, 202})
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 200).Update("inviter_id", 202).Error)

	assert.True(t, DetectInviteRing(200), "3 跳环 A→B→C→A 必须命中")
}

func TestDetectInviteRing5HopBoundary(t *testing.T) {
	// A → B → C → D → E → A（5 跳刚好命中 MaxRingDepth=5）
	seedInviterChain(t, []int{300, 301, 302, 303, 304})
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 300).Update("inviter_id", 304).Error)

	assert.True(t, DetectInviteRing(300), "5 跳边界环应被命中")
}

func TestDetectInviteRing6HopTooDeep(t *testing.T) {
	// A → B → C → D → E → F → A（6 跳超出 MaxRingDepth=5）
	// 不应被检测到——成环是薅羊毛特征，5 跳以内基本不存在；
	// 5 跳以上视为"自然传播链"，放过避免误拦。
	seedInviterChain(t, []int{400, 401, 402, 403, 404, 405})
	require.NoError(t, DB.Model(&User{}).Where("id = ?", 400).Update("inviter_id", 405).Error)

	assert.False(t, DetectInviteRing(400), "6 跳环超出 MaxRingDepth=5，应放过")
}

func TestDetectInviteRingNoRing(t *testing.T) {
	// A → B → C（链头 + 单链），无环
	seedInviterChain(t, []int{500, 501, 502})
	assert.False(t, DetectInviteRing(500), "单链不应视为环")
}

func TestDetectInviteRingZeroID(t *testing.T) {
	assert.False(t, DetectInviteRing(0), "user_id=0 跳过检测")
}

// TestGetUserTotalConsumeQuota 守住 §20.5 首充门槛的 consume log 聚合路径。
func TestGetUserTotalConsumeQuota(t *testing.T) {
	const userID = 6001

	user := &User{
		Id:       userID,
		Username: "consume-test",
		AffCode:  "consume-test-aff",
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, DB.Create(user).Error)

	// 喂 3 条 consume log：5000 + 3000 + 2000 = 10000 quota
	require.NoError(t, DB.Create(&Log{
		UserId: userID, Username: user.Username,
		Type: LogTypeConsume, Quota: 5000,
		Content: "first consume",
	}).Error)
	require.NoError(t, DB.Create(&Log{
		UserId: userID, Username: user.Username,
		Type: LogTypeConsume, Quota: 3000,
		Content: "second consume",
	}).Error)
	require.NoError(t, DB.Create(&Log{
		UserId: userID, Username: user.Username,
		Type: LogTypeConsume, Quota: 2000,
		Content: "third consume",
	}).Error)

	sum, err := GetUserTotalConsumeQuota(userID)
	require.NoError(t, err)
	assert.Equal(t, int64(10000), sum, "3 条 consume log 应聚合为 10000 quota")
}

func TestGetUserTotalConsumeQuotaIgnoresNonConsume(t *testing.T) {
	// 非 consume 类型的 log 不计入（如 topup / manage）
	const userID = 6002
	user := &User{
		Id:       userID,
		Username: "consume-ignore-test",
		AffCode:  "consume-ignore-aff",
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, DB.Create(user).Error)

	require.NoError(t, DB.Create(&Log{
		UserId: userID, Username: user.Username,
		Type: LogTypeManage, Quota: 99999,
		Content: "admin op should not count as consume",
	}).Error)
	require.NoError(t, DB.Create(&Log{
		UserId: userID, Username: user.Username,
		Type: LogTypeConsume, Quota: 5000,
		Content: "real consume",
	}).Error)

	sum, err := GetUserTotalConsumeQuota(userID)
	require.NoError(t, err)
	assert.Equal(t, int64(5000), sum, "只聚合 LogTypeConsume，不混入 LogTypeManage")
}

func TestGetUserTotalConsumeQuotaEmpty(t *testing.T) {
	const userID = 6003
	_, err := GetUserTotalConsumeQuota(userID)
	require.NoError(t, err) // 不存在的 user_id 不应 panic
}