package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"fatelumen/backend/internal/model"
	jwtpkg "fatelumen/backend/internal/pkg/jwt"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const testJWTSecret = "auth-test-secret-2026"

func setupAuthTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.User{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func seedUser(t *testing.T, db *gorm.DB, id uint64, sub, email, role string, active bool, tokenID string) {
	t.Helper()
	db.Create(&model.User{
		ID:             id,
		GoogleSub:      sub,
		Email:          email,
		Role:           role,
		CurrentTokenID: tokenID,
	})
	if !active {
		db.Exec("UPDATE users SET active = 0 WHERE id = ?", id)
	}
}

func makeTestToken(t *testing.T, userID uint64, tokenID string) string {
	t.Helper()
	tok, err := jwtpkg.Generate(testJWTSecret, 24, userID, tokenID)
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}
	return tok
}

// ---------- Handler tests ----------

func TestAuth_DisabledUser_Blocked(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, 1, "sub-1", "user@test", model.RoleUser, false, "tok-abc")
	token := makeTestToken(t, 1, "tok-abc")

	authMW := NewAuthMiddleware(testJWTSecret, db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMW.Handler())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Resp
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != response.CodeForbidden {
		t.Fatalf("expected %d for disabled user, got code=%d msg=%s", response.CodeForbidden, resp.Code, resp.Msg)
	}
	if resp.Msg != "account is disabled" {
		t.Errorf("expected 'account is disabled', got '%s'", resp.Msg)
	}

	// Verify user_id is NOT set in context (c.Abort prevents downstream handler)
	// The handler was never called so we know the abort happened.
}

func TestAuth_ActiveUser_Passes(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, 2, "sub-2", "active@test", model.RoleUser, true, "tok-abc")
	token := makeTestToken(t, 2, "tok-abc")

	authMW := NewAuthMiddleware(testJWTSecret, db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMW.Handler())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true, "uid": GetUserID(c)})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for active user, got %d: %s", w.Code, w.Body.String())
	}
	var resp response.Resp
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != response.CodeOK {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestAuth_DisabledAdmin_Blocked(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, 3, "sub-3", "admin@test", model.RoleAdmin, false, "tok-def")
	token := makeTestToken(t, 3, "tok-def")

	authMW := NewAuthMiddleware(testJWTSecret, db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMW.Handler())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Resp
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != response.CodeForbidden {
		t.Errorf("disabled admin should be blocked, got code=%d", resp.Code)
	}
}

func TestAuth_TokenIDMismatch_Kicked(t *testing.T) {
	db := setupAuthTestDB(t)
	// User has current_token_id "tok-old" but token claims "tok-new" → mismatch
	seedUser(t, db, 4, "sub-4", "kicked@test", model.RoleUser, true, "tok-old")
	token := makeTestToken(t, 4, "tok-new")

	authMW := NewAuthMiddleware(testJWTSecret, db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMW.Handler())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Resp
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Code != response.CodeKicked {
		t.Fatalf("expected %d for token mismatch, got code=%d msg=%s", response.CodeKicked, resp.Code, resp.Msg)
	}
}

// ---------- OptionalHandler tests ----------

func TestAuth_OptionalHandler_DisabledUser_TreatedAsAnonymous(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, 5, "sub-5", "disabled@test", model.RoleUser, false, "tok-xyz")
	token := makeTestToken(t, 5, "tok-xyz")

	authMW := NewAuthMiddleware(testJWTSecret, db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMW.OptionalHandler())
	r.GET("/test", func(c *gin.Context) {
		uid := GetUserID(c)
		if uid != 0 {
			t.Error("disabled user should not have user_id set in optional handler")
		}
		c.JSON(http.StatusOK, gin.H{"uid": uid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 (treated as anonymous), got %d", w.Code)
	}
}

func TestAuth_OptionalHandler_ActiveUser_SetsContext(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, 6, "sub-6", "active@test", model.RoleUser, true, "tok-mno")
	token := makeTestToken(t, 6, "tok-mno")

	authMW := NewAuthMiddleware(testJWTSecret, db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMW.OptionalHandler())
	r.GET("/test", func(c *gin.Context) {
		uid := GetUserID(c)
		if uid != 6 {
			t.Errorf("active user should have user_id=6, got %d", uid)
		}
		c.JSON(http.StatusOK, gin.H{"uid": uid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestAuth_OptionalHandler_NoToken_PassesAnonymous(t *testing.T) {
	db := setupAuthTestDB(t)
	seedUser(t, db, 7, "sub-7", "anon@test", model.RoleUser, true, "tok-pqr")

	authMW := NewAuthMiddleware(testJWTSecret, db)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(authMW.OptionalHandler())
	r.GET("/test", func(c *gin.Context) {
		uid := GetUserID(c)
		if uid != 0 {
			t.Errorf("no-token request should have uid=0, got %d", uid)
		}
		c.JSON(http.StatusOK, gin.H{"uid": uid})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	// No Authorization header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for anonymous, got %d", w.Code)
	}
}
