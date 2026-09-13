package controller

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// 注册时带客户号：经销商把号交给客户，客户在建账号这一趟就把归属与折扣一起落下来
// （master-plan §4.4「注册接口接受可选 customer_code」）。
//
// 分成前后两步是因为时机不同：号不能用时必须在**建账号之前**就被拦下——否则客户拿到
// 一个没有折扣的账号，还以为自己是 8 折，只能等他自己发现按原价在计费；
// 而真正绑定必须等到 user.id 存在之后。

// registerRequest 是注册请求：model.User 原有的字段照旧解析（用户名、密码、邮箱、
// 验证码、邀请码），另外多认一个可选的 customer_code。
//
// 用内嵌而不是给 model.User 加字段：客户号只在注册这一趟是输入，
// 加到用户模型上会让每个返回用户的接口都多出一个空字段。
type registerRequest struct {
	model.User
	CustomerCode string `json:"customer_code"`
}

// rejectUnusableCustomerCode 建账号之前先验号：不能用就写回具体理由并返回 true。
// 没带号（只有空白也算没带）时什么都不做，保持既有注册流程原样。
func rejectUnusableCustomerCode(c *gin.Context, rawCode string) bool {
	if strings.TrimSpace(rawCode) == "" {
		return false
	}
	_, err := model.CheckCustomerCodeUsable(rawCode)
	if err == nil {
		return false
	}
	if key := customerCodeErrorMessageKey(err); key != "" {
		common.ApiErrorI18n(c, key)
	} else {
		common.ApiError(c, err)
	}
	return true
}

// applyCustomerCodeOnRegister 账号建好之后真正绑号：落归属 + 绑折扣 + 号上用量 +1。
// 返回要回给界面的那一小块数据，没带号时返回 nil。
//
// 这一步失败不推翻注册：账号已经建好了，这时说「注册失败」只会让客户以为自己没注册上，
// 再点一次反而撞「用户名已存在」。所以照常返回注册成功，但把「号没生效」和原因一起带回去，
// 让界面能立刻说清楚——他还可以登录后在个人资料里重新绑。
func applyCustomerCodeOnRegister(c *gin.Context, userId int, rawCode string) map[string]any {
	if strings.TrimSpace(rawCode) == "" {
		return nil
	}
	if _, err := model.BindCustomerCode(userId, rawCode); err == nil {
		return map[string]any{
			"customer_code_applied": true,
			"customer_code_error":   "",
		}
	} else {
		// 预检放行之后仍失败，多半是并发抢用（号在两步之间刚好被用满）。
		// 这是异常路径，留一条日志便于日后核对。
		common.SysLog(fmt.Sprintf("register: customer code did not apply for user %d: %v", userId, err))
		reason := ""
		if key := customerCodeErrorMessageKey(err); key != "" {
			reason = common.TranslateMessage(c, key)
		}
		return map[string]any{
			"customer_code_applied": false,
			"customer_code_error":   reason,
		}
	}
}
