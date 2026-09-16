package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// performResetInviterRequest 仿 user_manage_test.go 的 performManageUserRequest：
// 直接调用 handler、塞 id/role 到 gin context，跳过中间件（中间件在
// router 层管，这里只测 controller 逻辑）。
//
// actorRole 决定能否过守卫——用 common.RoleAdminUser (10) 和 common.RoleRootUser
// (100) 两种值覆盖 403 和 200 两条路径。
func performResetInviterRequest(t *testing.T, actorRole int, userId int, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/user/%d/reset_inviter", userId),
		strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = gin.Params{{Key: "id", Value: fmt.Sprintf("%d", userId)}}
	c.Set("id", 9999) // actor user id（不在 guard 里用，留痕用）
	c.Set("role", actorRole)
	c.Set("username", "test-actor")
	AdminResetUserInviter(c)
	return recorder
}

// seedManageTestUser 建一个普通用户作为 reset target。与 setupManageUserTestDB
// 配合使用：测试先把 DB 切到内存 SQLite，建 target user，再调 handler。
func seedResetInviterTestUser(t *testing.T, db *gorm.DB, usernameSuffix string) *model.User {
	t.Helper()
	user := &model.User{
		Username:  "ri-target-" + usernameSuffix,
		Password:  "unused-hash",
		Role:      common.RoleCommonUser,
		Status:    common.UserStatusEnabled,
		Group:     "default",
		AffCode:   "ri-aff-" + usernameSuffix,
		InviterId: 0,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

// 1. 普通管理员（role=10）调用 → 守卫拒绝 → "success":false。
//    项目约定所有 error 都返 HTTP 200 + success=false，所以这里断言 success。
func TestAdminResetUserInviterForbiddenForNonRoot(t *testing.T) {
	db := setupManageUserTestDB(t)
	target := seedResetInviterTestUser(t, db, "nonroot")

	recorder := performResetInviterRequest(t,
		common.RoleAdminUser,
		target.Id,
		`{"inviter_id":123,"reason":"forbidden test reason"}`)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), "no permission")

	// target.inviter_id 不应被改
	var reloaded model.User
	require.NoError(t, db.First(&reloaded, target.Id).Error)
	assert.Equal(t, 0, reloaded.InviterId, "非超管调用被拒，inviter_id 应保持原值")
}

// 2. 超管 happy path：target 已绑定 inviter=B → reset 到 inviter=C → 200 success。
func TestAdminResetUserInviterRootHappyPath(t *testing.T) {
	db := setupManageUserTestDB(t)
	oldInviter := seedResetInviterTestUser(t, db, "oldinviter")
	newInviter := seedResetInviterTestUser(t, db, "newinviter")
	target := &model.User{
		Username: "ri-target-happy", Password: "unused-hash", Role: common.RoleCommonUser,
		Status: common.UserStatusEnabled, Group: "default", AffCode: "ri-aff-happy",
		InviterId: oldInviter.Id,
	}
	require.NoError(t, db.Create(target).Error)

	recorder := performResetInviterRequest(t,
		common.RoleRootUser,
		target.Id,
		fmt.Sprintf(`{"inviter_id":%d,"reason":"happy path unit test"}`, newInviter.Id))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":true`)

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, target.Id).Error)
	assert.Equal(t, newInviter.Id, reloaded.InviterId, "inviter_id 应被改成 newInviter")
}

// 3. 超管 + target 不存在 → "success":false，message 含"目标用户不存在"。
func TestAdminResetUserInviterTargetNotFound(t *testing.T) {
	_ = setupManageUserTestDB(t)

	recorder := performResetInviterRequest(t,
		common.RoleRootUser,
		999999, // 不存在的 user id
		`{"inviter_id":123,"reason":"missing target test"}`)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), "目标用户不存在")
}

// 4. 超管 + reason 长度 < 10 字符 → "success":false，message 含长度说明。
//    这是 API 边界防御性校验（前端表单已硬约束，这里再挡一道）。
func TestAdminResetUserInviterReasonTooShort(t *testing.T) {
	db := setupManageUserTestDB(t)
	target := seedResetInviterTestUser(t, db, "shortreason")

	recorder := performResetInviterRequest(t,
		common.RoleRootUser,
		target.Id,
		`{"inviter_id":123,"reason":"abc"}`) // 只有 3 字符
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	assert.Contains(t, recorder.Body.String(), "原因长度")

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, target.Id).Error)
	assert.Equal(t, 0, reloaded.InviterId, "reason 太短被拒，inviter_id 应保持原值")
}