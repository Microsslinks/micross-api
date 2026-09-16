package model

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// setupResetInviterTest 给 task-16 重置邀请人用例一份干净的 SQLite。
// 沿用 setupAgentTest 同款做法（全局 DB swap + t.Cleanup 还原），保证子测试
// 之间不互相污染。
func setupResetInviterTest(t *testing.T) {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	dsn := fmt.Sprintf("file:reset-inviter-test-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	DB, LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&User{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		_ = sqlDB.Close()
	})
}

// userResetInviterTestSeq 唯一 username 后缀——同进程多个 case 串行跑时
// 用纳秒+计数去重，避开 Windows 时钟粒度问题。
var userResetInviterTestSeq int

func seedResetInviterUser(t *testing.T, inviterId int, deleted bool) *User {
	t.Helper()
	userResetInviterTestSeq++
	suffix := fmt.Sprintf("ri-%d-%d", time.Now().UnixNano()%1000000, userResetInviterTestSeq)
	user := &User{
		Username:  "ri-" + suffix,
		Password:  "unused-password-hash",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		AffCode:   "aff-" + suffix,
		InviterId: inviterId,
	}
	require.NoError(t, DB.Create(user).Error)
	if deleted {
		require.NoError(t, DB.Delete(user).Error) // 软删：写 deleted_at
	}
	return user
}

// happy path：target A（已有 inviter=B）→ reset 到 inviter=C → 写库 + 返回 oldInviterId=B。
// 这是 README §五 验收第 1 条的核心 case。
func TestResetInviterHappyPath(t *testing.T) {
	setupResetInviterTest(t)
	inviterOld := seedResetInviterUser(t, 0, false)
	inviterNew := seedResetInviterUser(t, 0, false)
	target := seedResetInviterUser(t, inviterOld.Id, false)

	oldId, err := ResetInviter(ResetInviterParams{
		TargetUserId: target.Id,
		NewInviterId: inviterNew.Id,
		ActorUserId:  1,
		Reason:       "user requested reset after typo",
	})
	require.NoError(t, err)
	assert.Equal(t, inviterOld.Id, oldId, "oldInviterId 应返回原 inviter，便于留痕")

	var reloaded User
	require.NoError(t, DB.First(&reloaded, target.Id).Error)
	assert.Equal(t, inviterNew.Id, reloaded.InviterId, "target.inviter_id 应被改成新邀请人")
}

// 自邀请守卫：inviter_id == target_user_id → ErrInviterSelf，不写库。
func TestResetInviterRejectsSelfInvite(t *testing.T) {
	setupResetInviterTest(t)
	target := seedResetInviterUser(t, 0, false)

	oldId, err := ResetInviter(ResetInviterParams{
		TargetUserId: target.Id,
		NewInviterId: target.Id, // 自邀请
		ActorUserId:  1,
		Reason:       "should be rejected",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInviterSelf), "应返回 ErrInviterSelf，实际: %v", err)
	assert.Equal(t, 0, oldId, "失败时 oldInviterId 不应被赋值（保留 0 兜底）")

	var reloaded User
	require.NoError(t, DB.First(&reloaded, target.Id).Error)
	assert.Equal(t, 0, reloaded.InviterId, "自邀请被拒绝，inviter_id 应保持原值不变")
}

// 清空场景：inviter_id=0 → 写 0（DB 列 int，0 等价于 NULL），旧值正确返回。
func TestResetInviterClearsToZero(t *testing.T) {
	setupResetInviterTest(t)
	inviter := seedResetInviterUser(t, 0, false)
	target := seedResetInviterUser(t, inviter.Id, false)

	oldId, err := ResetInviter(ResetInviterParams{
		TargetUserId: target.Id,
		NewInviterId: 0, // 清空
		ActorUserId:  1,
		Reason:       "ops cleanup before migration",
	})
	require.NoError(t, err)
	assert.Equal(t, inviter.Id, oldId)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, target.Id).Error)
	assert.Equal(t, 0, reloaded.InviterId)
}

// 新邀请人不存在：返回 ErrInviterNotFound，不写库。
func TestResetInviterRejectsNonExistentInviter(t *testing.T) {
	setupResetInviterTest(t)
	target := seedResetInviterUser(t, 0, false)

	_, err := ResetInviter(ResetInviterParams{
		TargetUserId: target.Id,
		NewInviterId: 999999, // 不存在
		ActorUserId:  1,
		Reason:       "inviter id is wrong",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInviterNotFound), "应返回 ErrInviterNotFound，实际: %v", err)

	var reloaded User
	require.NoError(t, DB.First(&reloaded, target.Id).Error)
	assert.Equal(t, 0, reloaded.InviterId, "邀请人不存在时不应改 target.inviter_id")
}

// 新邀请人已软删：返回 ErrInviterDeleted，不写库。
// 这是任务清单里没列但顺手加的边界——同样属于"无效邀请人"，避免运营把客户
// 链到一个"曾经是邀请人但现在已经退号/被封"的用户上。
func TestResetInviterRejectsSoftDeletedInviter(t *testing.T) {
	setupResetInviterTest(t)
	inviter := seedResetInviterUser(t, 0, true) // 已软删
	target := seedResetInviterUser(t, 0, false)

	_, err := ResetInviter(ResetInviterParams{
		TargetUserId: target.Id,
		NewInviterId: inviter.Id,
		ActorUserId:  1,
		Reason:       "inviter got banned",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInviterDeleted), "应返回 ErrInviterDeleted，实际: %v", err)
}

// 边界补充：target 本身已软删（被锁定的 row 不应被改）。
func TestResetInviterTargetSoftDeletedReturnsNotFound(t *testing.T) {
	setupResetInviterTest(t)
	inviter := seedResetInviterUser(t, 0, false)
	target := seedResetInviterUser(t, 0, true) // target 软删

	_, err := ResetInviter(ResetInviterParams{
		TargetUserId: target.Id,
		NewInviterId: inviter.Id,
		ActorUserId:  1,
		Reason:       "should not even try",
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, gorm.ErrRecordNotFound),
		"target 软删应被 gorm.ErrRecordNotFound 兜住，实际: %v", err)
}