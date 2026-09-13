package controller

import (
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

// 注册请求里带的客户号要能解出来，而 model.User 原有的字段（含校验标签）照旧。
// 这一层用内嵌结构体接 json，一旦有人换成别的接法，这两件事会一起悄悄失效：
// 客户号解不出来（号白填），或者校验被绕过（弱密码能注册）。
func TestRegisterRequestDecodesUserFieldsAndCustomerCode(t *testing.T) {
	var req registerRequest
	require.NoError(t, common.DecodeJson(strings.NewReader(`{
		"username": "code-customer",
		"password": "password123",
		"email": "customer@example.com",
		"aff_code": "abcd",
		"customer_code": "ag7k2m9p4qxz"
	}`), &req))

	assert.Equal(t, "code-customer", req.Username)
	assert.Equal(t, "password123", req.Password)
	assert.Equal(t, "customer@example.com", req.Email)
	assert.Equal(t, "abcd", req.AffCode)
	assert.Equal(t, "ag7k2m9p4qxz", req.CustomerCode)

	// 校验仍按 model.User 的标签走：这里故意给一个过短的密码，必须被挡下
	weak := req.User
	weak.Password = "123"
	require.Error(t, common.Validate.Struct(&weak))
	require.NoError(t, common.Validate.Struct(&req.User))
}

// 注册带客户号这一趟的两个关键判定：建账号之前号能不能用（不能用就不该建出账号来），
// 以及账号建好之后号有没有真的落到归属与折扣上。

func setupRegisterCustomerCodeTest(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousMainType, previousLogType := common.MainDatabaseType(), common.LogDatabaseType()
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:register-customer-code-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.CustomerCode{}, &model.DiscountPlan{}, &model.DiscountBinding{},
	))

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.SetDatabaseTypes(previousMainType, previousLogType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func newRegisterCustomerCodeContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	require.NoError(t, i18n.Init())
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	// 响应要走 i18n（按 Accept-Language 出文案），所以请求对象不能省
	c.Request = httptest.NewRequest(http.MethodPost, "/api/user/register", nil)
	c.Request.Header.Set("Content-Type", "application/json")
	return c, recorder
}

// 带一张不存在的号进来：当场拒绝，并且要说清是「号不存在」——
// 只说「注册失败」的话，客户只会反复重试同一个错号。
func TestRejectUnusableCustomerCodeRejectsUnknownCode(t *testing.T) {
	setupRegisterCustomerCodeTest(t)
	c, recorder := newRegisterCustomerCodeContext(t)

	rejected := rejectUnusableCustomerCode(c, "AGNOSUCHCODE")

	require.True(t, rejected)
	assert.Contains(t, recorder.Body.String(), `"success":false`)
	// 返回的是翻好的那句话（按 Accept-Language），不是 i18n 键
	assert.Contains(t, recorder.Body.String(), i18n.Translate(i18n.LangEn, i18n.MsgCustomerCodeNotFound))
}

// 没带号就是普通注册：客户号这道关不能拦住走到这里的任何人。
func TestRejectUnusableCustomerCodePassesWhenAbsent(t *testing.T) {
	setupRegisterCustomerCodeTest(t)
	c, recorder := newRegisterCustomerCodeContext(t)

	assert.False(t, rejectUnusableCustomerCode(c, ""))
	assert.False(t, rejectUnusableCustomerCode(c, "   "))
	assert.Empty(t, recorder.Body.String(), "没带号时不该写任何响应")
}

// 没带号时不该往注册响应里塞 customer_code 那一块。
func TestApplyCustomerCodeOnRegisterSkippedWhenAbsent(t *testing.T) {
	setupRegisterCustomerCodeTest(t)
	c, _ := newRegisterCustomerCodeContext(t)

	assert.Nil(t, applyCustomerCodeOnRegister(c, 1, ""))
	assert.Nil(t, applyCustomerCodeOnRegister(c, 1, "   "))
}

// 预检放行之后号仍可能绑不上（并发抢用同一张号）：这时账号已经建好了，
// 不能反过来报「注册失败」——那会让客户以为自己没注册上、再点一次撞用户名已存在。
// 于是照常返回成功，但把「号没生效」和原因带上，让界面当场说清楚。
func TestApplyCustomerCodeOnRegisterReportsFailureWithReason(t *testing.T) {
	setupRegisterCustomerCodeTest(t)
	c, _ := newRegisterCustomerCodeContext(t)

	data := applyCustomerCodeOnRegister(c, 1, "AGNOSUCHCODE")

	require.NotNil(t, data)
	assert.Equal(t, false, data["customer_code_applied"])
	assert.Equal(t, i18n.Translate(i18n.LangEn, i18n.MsgCustomerCodeNotFound), data["customer_code_error"])
}

// 预检能被放行的号，绑定时也必须真的落下：这两步的口径一旦走散，
// 客户就会拿到一个「注册时说有折扣、其实没有」的账号。
func TestApplyCustomerCodeOnRegisterBindsUsableCode(t *testing.T) {
	db := setupRegisterCustomerCodeTest(t)
	c, _ := newRegisterCustomerCodeContext(t)

	// aff_code 上有唯一索引，两个用户得给不同的值（空串也算同值）
	agent := &model.User{Username: "agent-for-register", Status: common.UserStatusEnabled, AffCode: "REGAG"}
	require.NoError(t, db.Create(agent).Error)
	customer := &model.User{Username: "customer-for-register", Status: common.UserStatusEnabled, AffCode: "REGCU"}
	require.NoError(t, db.Create(customer).Error)
	plan := &model.DiscountPlan{Name: "register-code-plan", Status: model.DiscountStatusEnabled}
	require.NoError(t, db.Create(plan).Error)

	code := &model.CustomerCode{Code: "AGREGISTER1", AgentId: agent.Id, PlanId: plan.Id, MaxUses: 1, Status: model.CustomerCodeStatusEnabled}
	require.NoError(t, db.Create(code).Error)

	// 注册前先验号放行，再真正绑定，走的就是注册那两步的顺序
	require.False(t, rejectUnusableCustomerCode(c, code.Code))
	data := applyCustomerCodeOnRegister(c, customer.Id, code.Code)

	require.NotNil(t, data)
	assert.Equal(t, true, data["customer_code_applied"])

	var reloaded model.User
	require.NoError(t, db.First(&reloaded, customer.Id).Error)
	assert.Equal(t, agent.Id, reloaded.ParentAgentId)
}
