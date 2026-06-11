package middleware

import (
	"fmt"
	"strconv"

	"fatelumen/backend/internal/pkg/logger"
	"fatelumen/backend/internal/pkg/ratelimit"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// KeyByUser returns a rate-limit key scoped to the authenticated user.
// Falls back to client IP when no user is authenticated.
func KeyByUser(c *gin.Context) string {
	uid := GetUserID(c)
	if uid > 0 {
		return fmt.Sprintf("uid:%d", uid)
	}
	return KeyByIP(c)
}

// KeyByIP returns a rate-limit key scoped to the client IP.
func KeyByIP(c *gin.Context) string {
	return fmt.Sprintf("ip:%s", c.ClientIP())
}

// RateLimit returns a Gin middleware that uses the provided Limiter.
// keyFunc determines the dimension (e.g. per-user or per-IP).
// Admin users are exempt from rate limiting.
func RateLimit(limiter ratelimit.Limiter, keyFunc func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Admins are exempt
		if IsAdmin(c) {
			c.Next()
			return
		}

		key := keyFunc(c)
		allowed, retryAfter := limiter.Allow(key)
		if !allowed {
			c.Header("Retry-After", strconv.Itoa(int(retryAfter.Seconds())+1))
			logger.FromCtx(c.Request.Context()).Warn("rate limit exceeded",
				"key", key,
				"path", c.Request.URL.Path,
			)
			response.Fail(c, response.CodeTooManyRequests, "too many requests")
			c.Abort()
			return
		}
		c.Next()
	}
}
