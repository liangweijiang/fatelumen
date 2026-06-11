package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

type fakeRoleStore struct {
	role string
	err  error
}

func (f *fakeRoleStore) GetRole(userID uint64) (string, error) {
	return f.role, f.err
}

func TestAdminOnly_AllowsAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(1))
		c.Next()
	})
	r.Use(adminOnly(&fakeRoleStore{role: model.RoleAdmin}))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for admin, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAdminOnly_RejectsUser(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(2))
		c.Next()
	})
	r.Use(adminOnly(&fakeRoleStore{role: model.RoleUser}))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Resp
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != response.CodeForbidden {
		t.Fatalf("expected %d for non-admin, got %d", response.CodeForbidden, resp.Code)
	}
}

func TestAdminOnly_NoUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(adminOnly(&fakeRoleStore{role: model.RoleAdmin}))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Resp
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected %d for no user_id, got %d", response.CodeUnauthorized, resp.Code)
	}
}

func TestAdminOnly_UserNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", uint64(99))
		c.Next()
	})
	r.Use(adminOnly(&fakeRoleStore{err: errors.New("not found")}))
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Resp
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != response.CodeUnauthorized {
		t.Fatalf("expected %d for user not found, got %d", response.CodeUnauthorized, resp.Code)
	}
}
