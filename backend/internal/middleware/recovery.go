package middleware

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Recovery 全局 panic 恢复 + 统一错误响应。
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[Recovery] panic recovered: %v", r)
				c.AbortWithStatusJSON(http.StatusOK, gin.H{
					"code": 5000,
					"msg":  "internal server error",
				})
			}
		}()
		c.Next()
	}
}
