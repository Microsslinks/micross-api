package controller

// P3 task-08 集成测试：覆盖验收 checklist 的 5 个点。
//
// 测试环境：SQLite 内存库 + 关闭 Redis（cache helper 在 RedisEnabled=false 时 graceful 退化）。
// 不依赖任何外部服务，可以直接 `go test ./controller/...` 跑。

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// ===================== 环境 =====================

func setupCustomerRegistrationTest(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	previousRedisEnabled := common.RedisEnabled
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	// 单元测试不连 Redis：关掉以后所有 cache helper 走 graceful 退化分支
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:customer-registration-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.UserSession{}))

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		common.RedisEnabled = previousRedisEnabled
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newAgentCustomerContext(t *testing.T, method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	var req *http.Request
	if body != "" {
		req = httptest.NewRequest(method, path, strings.NewReader(body))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	// 路径参数：拿 customerId（如果有）
	for i, seg := range strings.Split(path, "/") {
		if seg == "customers" && i+1 < len(strings.Split(path, "/")) {
			parts := strings.Split(path, "/")
			c.Params = gin.Params{{Key: "customerId", Value: parts[i+1]}}
			break
		}
	}
	return c, recorder
}

// 模拟「AuthMiddleware 把当前用户 id 放进 context」这一步。
// 实际 HTTP 时是 c.Set("id", ...)；这里直接 Set 同样的值即可。
func setCallerId(c *gin.Context, id int) {
	c.Set("id", id)
}

// makeAgent 造一个 enabled 经销商。AffCode 必须各不相同（库里是 uniqueIndex）。
func makeAgent(t *testing.T, db *gorm.DB, name, affCode string) *model.User {
	u := &model.User{
		Username:    name,
		Password:    "placeholder",
		DisplayName: name,
		Status:      common.UserStatusEnabled,
		SubjectType: model.SubjectTypeAgent,
		AffCode:     affCode,
		Role:        common.RoleCommonUser,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

// makeCustomer 造一个 enabled 普通客户，parentAgentId 决定挂在哪个经销商名下。
func makeCustomer(t *testing.T, db *gorm.DB, name, affCode string, parentAgentId int) *model.User {
	u := &model.User{
		Username:       name,
		Password:       "placeholder",
		DisplayName:    name,
		Status:         common.UserStatusEnabled,
		SubjectType:    model.SubjectTypeIndividual,
		AffCode:        affCode,
		ParentAgentId:  parentAgentId,
		Role:           common.RoleCommonUser,
	}
	require.NoError(t, db.Create(u).Error)
	return u
}

// successResponse 是 controller 成功响应的统一格式。
type successResponse struct {
	Success bool            `json:"success"`
	Message string           `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// failureResponse 是 controller 失败响应的统一格式。
type failureResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ===================== 测试用例 =====================

// 1) 经销商代注册成功：parent_agent_id 写入当前经销商 id，新用户 subject_type=individual。
func TestRegisterSelfAgentCustomer_Success(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	agent := makeAgent(t, db, "agent-1", "AGENT01")

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		"/api/user/self/agent/customers",
		`{"username":"customer-x","password":"abcdefgh","display_name":"测试客户"}`)
	setCallerId(c, agent.Id)

	RegisterSelfAgentCustomer(c)

	var resp successResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success, "响应失败：%s", recorder.Body.String())

	var created model.User
	require.NoError(t, json.Unmarshal(resp.Data, &created))
	assert.Equal(t, "customer-x", created.Username)
	assert.Equal(t, "测试客户", created.DisplayName)
	assert.Equal(t, agent.Id, created.ParentAgentId, "parent_agent_id 必须指向调用方经销商")
	assert.Equal(t, model.SubjectTypeIndividual, created.SubjectType)
	assert.NotZero(t, created.Id, "新用户必须有 id")

	// DB 复核
	var reloaded model.User
	require.NoError(t, db.First(&reloaded, created.Id).Error)
	assert.Equal(t, agent.Id, reloaded.ParentAgentId)
	assert.NotEqual(t, "abcdefgh", reloaded.Password, "密码必须 hash，不能明文落库")
}

// 2) 非经销商调代注册被拒（验收第 4 项）。
func TestRegisterSelfAgentCustomer_NonAgentRejected(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	notAgent := makeCustomer(t, db, "regular-1", "REGU01", 0)

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		"/api/user/self/agent/customers",
		`{"username":"customer-y","password":"abcdefgh"}`)
	setCallerId(c, notAgent.Id)

	RegisterSelfAgentCustomer(c)

	var resp failureResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	// controller 走 ApiErrorI18n，按当前语言翻译。默认 i18n 是英文。
	expected := i18n.Translate(i18n.LangEn, i18n.MsgUserNotAgent)
	assert.Equal(t, expected, resp.Message, "非经销商必须被拒，且翻译与 i18n key 一致")
}

// 3) 短密码被拒。
func TestRegisterSelfAgentCustomer_ShortPasswordRejected(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	agent := makeAgent(t, db, "agent-2", "AGENT02")

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		"/api/user/self/agent/customers",
		`{"username":"customer-z","password":"abc"}`)
	setCallerId(c, agent.Id)

	RegisterSelfAgentCustomer(c)

	var resp failureResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Message, "8")
}

// 4) 重置密码：返回新明文，DB 里密码被改写且 auth_version 增加。
func TestResetSelfAgentCustomerPassword_Success(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	agent := makeAgent(t, db, "agent-3", "AGENT03")
	customer := makeCustomer(t, db, "customer-3", "CUST003", agent.Id)
	originalHash := customer.Password
	require.NoError(t, db.Model(customer).Update("auth_version", 1).Error)

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		fmt.Sprintf("/api/user/self/agent/customers/%d/reset-password", customer.Id),
		"")
	setCallerId(c, agent.Id)

	ResetSelfAgentCustomerPassword(c)

	var resp successResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success, "响应失败：%s", recorder.Body.String())

	var payload struct {
		CustomerId int    `json:"customer_id"`
		Password   string `json:"password"`
	}
	require.NoError(t, json.Unmarshal(resp.Data, &payload))
	assert.Equal(t, customer.Id, payload.CustomerId)
	assert.NotEmpty(t, payload.Password, "必须返回新明文密码")
	assert.NotEqual(t, "placeholder", payload.Password)

	// DB 复核：密码被改、auth_version 增加
	var reloaded model.User
	require.NoError(t, db.First(&reloaded, customer.Id).Error)
	assert.NotEqual(t, originalHash, reloaded.Password, "密码必须已改写")
	assert.Greater(t, reloaded.AuthVersion, int64(1), "auth_version 必须递增（让旧 token 失效）")
}

// 5) 经销商 A 重置经销商 B 的客户密码 → 资源归属错被拒（验收第 5 项）。
func TestResetSelfAgentCustomerPassword_OtherAgentCustomerRejected(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	agentA := makeAgent(t, db, "agent-a", "AGENTA1")
	agentB := makeAgent(t, db, "agent-b", "AGENTB1")
	customerOfB := makeCustomer(t, db, "customer-b", "CUSTB01", agentB.Id)

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		fmt.Sprintf("/api/user/self/agent/customers/%d/reset-password", customerOfB.Id),
		"")
	setCallerId(c, agentA.Id) // A 想去动 B 的客户

	ResetSelfAgentCustomerPassword(c)

	var resp failureResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	expected := i18n.Translate(i18n.LangEn, i18n.MsgAgentCustomerNotFound)
	assert.Equal(t, expected, resp.Message, "归属错必须走 ErrAgentCustomerNotFound i18n key 翻译")

	// DB 复核：customerOfB 密码 / auth_version 都没动
	var reloaded model.User
	require.NoError(t, db.First(&reloaded, customerOfB.Id).Error)
	assert.Equal(t, customerOfB.Password, reloaded.Password)
	assert.Equal(t, customerOfB.AuthVersion, reloaded.AuthVersion)
}

// 6) 停用：user.status=disabled，级联 token.status=disabled（验收第 3 项的部分）。
func TestDisableSelfAgentCustomer_Success(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	agent := makeAgent(t, db, "agent-4", "AGENT04")
	customer := makeCustomer(t, db, "customer-4", "CUST004", agent.Id)

	// 给客户造一个启用的 token
	tk := &model.Token{UserId: customer.Id, Name: "test-token", Key: "tk-test-1", Status: common.UserStatusEnabled}
	require.NoError(t, db.Create(tk).Error)

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		fmt.Sprintf("/api/user/self/agent/customers/%d/disable", customer.Id),
		"")
	setCallerId(c, agent.Id)

	DisableSelfAgentCustomer(c)

	var resp successResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	require.True(t, resp.Success, "响应失败：%s", recorder.Body.String())

	// 客户自己被停用
	var reloaded model.User
	require.NoError(t, db.First(&reloaded, customer.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, reloaded.Status)

	// token 一起被停用（级联）
	var reloadedToken model.Token
	require.NoError(t, db.First(&reloadedToken, tk.Id).Error)
	assert.Equal(t, common.UserStatusDisabled, reloadedToken.Status, "停用客户必须级联停 token")
}

// 7) 重复停用幂等（不报错也不重复写）。
func TestDisableSelfAgentCustomer_Idempotent(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	agent := makeAgent(t, db, "agent-5", "AGENT05")
	customer := makeCustomer(t, db, "customer-5", "CUST005", agent.Id)
	require.NoError(t, db.Model(customer).Update("status", common.UserStatusDisabled).Error)

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		fmt.Sprintf("/api/user/self/agent/customers/%d/disable", customer.Id),
		"")
	setCallerId(c, agent.Id)

	DisableSelfAgentCustomer(c)

	var resp successResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.True(t, resp.Success, "重复停用应当幂等成功，不报错")
}

// 8) 经销商 A 停用经销商 B 的客户 → 归属错被拒。
func TestDisableSelfAgentCustomer_OtherAgentCustomerRejected(t *testing.T) {
	db := setupCustomerRegistrationTest(t)
	agentA := makeAgent(t, db, "agent-a2", "AGENTA2")
	agentB := makeAgent(t, db, "agent-b2", "AGENTB2")
	customerOfB := makeCustomer(t, db, "customer-b2", "CUSTB02", agentB.Id)

	c, recorder := newAgentCustomerContext(t, http.MethodPost,
		fmt.Sprintf("/api/user/self/agent/customers/%d/disable", customerOfB.Id),
		"")
	setCallerId(c, agentA.Id)

	DisableSelfAgentCustomer(c)

	var resp failureResponse
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &resp))
	assert.False(t, resp.Success)
	expected := i18n.Translate(i18n.LangEn, i18n.MsgAgentCustomerNotFound)
	assert.Equal(t, expected, resp.Message)

	// 复核：customerOfB 仍然是 enabled
	var reloaded model.User
	require.NoError(t, db.First(&reloaded, customerOfB.Id).Error)
	assert.Equal(t, common.UserStatusEnabled, reloaded.Status)
}