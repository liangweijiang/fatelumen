package middleware

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

// allowedOrigins 返回允许的来源白名单。
// 读环境变量 CORS_ORIGINS（逗号分隔），未配置时回落到本地开发常用端口。
func allowedOrigins() map[string]struct{} {
	raw := os.Getenv("CORS_ORIGINS")
	if strings.TrimSpace(raw) == "" {
		raw = "http://localhost:3000,http://127.0.0.1:3000"
	}
	set := make(map[string]struct{})
	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			set[o] = struct{}{}
		}
	}
	return set
}

// CORS 跨域中间件。
// 带凭证（Allow-Credentials: true）时，Allow-Origin 不能用通配符 "*"，
// 必须回显具体且在白名单内的请求 Origin，否则浏览器会拒绝该响应。
func CORS() gin.HandlerFunc {
	origins := allowedOrigins()
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin != "" {
			if _, ok := origins[origin]; ok {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				c.Writer.Header().Add("Vary", "Origin")
			}
		}
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, PATCH, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
