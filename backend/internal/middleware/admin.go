package middleware

import (
	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// adminRoleStore 获取用户 role 的抽象，便于单测注入 fake。
type adminRoleStore interface {
	GetRole(userID uint64) (string, error)
}

// dbRoleStore 基于 GORM 的实现。
type dbRoleStore struct {
	db *gorm.DB
}

func (s *dbRoleStore) GetRole(userID uint64) (string, error) {
	var user model.User
	if err := s.db.Select("role").First(&user, userID).Error; err != nil {
		return "", err
	}
	return user.Role, nil
}

// AdminOnly 返回一个 Gin 中间件，仅允许 role=admin 的用户通过。
// 前置假设：已在 authed 链上（JWT 已解析，userID 已在 ctx）。
// 采用 DB 查 role 方式（不改 JWT claims 结构，侵入最小）。
func AdminOnly(db *gorm.DB) gin.HandlerFunc {
	return adminOnly(&dbRoleStore{db: db})
}

// adminOnly 内部实现，接受接口便于单测。
func adminOnly(store adminRoleStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := GetUserID(c)
		if userID == 0 {
			response.Fail(c, response.CodeUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		role, err := store.GetRole(userID)
		if err != nil {
			response.Fail(c, response.CodeUnauthorized, "user not found")
			c.Abort()
			return
		}

		if role != model.RoleAdmin {
			response.Fail(c, response.CodeForbidden, "admin access required")
			c.Abort()
			return
		}

		c.Next()
	}
}
