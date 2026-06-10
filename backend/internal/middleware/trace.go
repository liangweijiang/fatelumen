package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"fatelumen/backend/internal/pkg/logger"

	"github.com/gin-gonic/gin"
)

const headerTraceID = "X-Trace-Id"

// Trace 每个请求注入 trace_id 并回传给前端。
// 优先取请求头 X-Trace-Id（便于外部链路透传），没有则生成新的。
func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(headerTraceID)
		if traceID == "" {
			traceID = genTraceID()
		}
		c.Request = c.Request.WithContext(logger.WithTraceID(c.Request.Context(), traceID))
		c.Header(headerTraceID, traceID)
		c.Next()
	}
}

func genTraceID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}
