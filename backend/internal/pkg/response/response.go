package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一响应信封：{"code":0,"msg":"ok","data":{...}}
// code=0 成功，非 0 为业务错误码。

// 业务错误码表
const (
	CodeOK           = 0
	CodeBadRequest   = 4000
	CodeUnauthorized = 4010
	CodeKicked       = 4011 // Token 被新设备登录踢下线
	CodeNoCredits    = 4020
	CodeOrderUnpaid  = 4030
	CodeQuotaExhausted = 4290
	CodeNotFound     = 4040
	CodeServerError  = 5000
	CodeLLMFailed    = 5001
	CodeRenderFailed = 5002
)

// Resp 统一响应。
type Resp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// OK 成功响应。
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Resp{Code: CodeOK, Msg: "ok", Data: data})
}

// Fail 业务错误响应。
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Resp{Code: code, Msg: msg})
}

// Error 服务器内部错误响应（500 状态码）。
func Error(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, Resp{Code: CodeServerError, Msg: msg})
}

// JSON 直接返回任意 JSON 状态码（用于 webhook 等非标准响应）。
func JSON(c *gin.Context, status int, data interface{}) {
	c.JSON(status, data)
}
