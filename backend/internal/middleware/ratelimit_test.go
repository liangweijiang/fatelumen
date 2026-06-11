package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"fatelumen/backend/internal/model"
	"fatelumen/backend/internal/pkg/ratelimit"
	"fatelumen/backend/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// ------- fake Limiter -------

type fakeLimiter struct {
	mu         sync.Mutex
	results    map[string]fakeResult
	callCount  map[string]int
	defaultAllowed bool
	defaultRetry   time.Duration
}

type fakeResult struct {
	allowed    bool
	retryAfter time.Duration
}

func newFakeLimiter() *fakeLimiter {
	return &fakeLimiter{
		results:    make(map[string]fakeResult),
		callCount:  make(map[string]int),
		defaultAllowed: true,
	}
}

func (f *fakeLimiter) setResult(key string, allowed bool, retry time.Duration) {
	f.results[key] = fakeResult{allowed: allowed, retryAfter: retry}
}

func (f *fakeLimiter) Allow(key string) (bool, time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callCount[key]++
	r, ok := f.results[key]
	if ok {
		return r.allowed, r.retryAfter
	}
	return f.defaultAllowed, f.defaultRetry
}

func (f *fakeLimiter) callsFor(key string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.callCount[key]
}

// ------- helpers -------

func setupRouter(limiter ratelimit.Limiter, keyFunc func(*gin.Context) string, injectUser uint64, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		if injectUser > 0 {
			c.Set("user_id", injectUser)
		}
		if role != "" {
			c.Set("role", role)
		}
		c.Next()
	})
	r.Use(RateLimit(limiter, keyFunc))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func setupNoMiddlewareRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	return r
}

func mustParseResp(t *testing.T, w *httptest.ResponseRecorder) response.Resp {
	t.Helper()
	var resp response.Resp
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v (body=%s)", err, w.Body.String())
	}
	return resp
}

// ------- tests -------

func TestRateLimit_AllowsWhenUnderLimit(t *testing.T) {
	lim := newFakeLimiter()
	lim.defaultAllowed = true
	router := setupRouter(lim, KeyByIP, 0, "")

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := mustParseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestRateLimit_Returns429WhenExceeded(t *testing.T) {
	lim := newFakeLimiter()
	lim.setResult("ip:192.0.2.1", false, 2*time.Second)
	router := setupRouter(lim, KeyByIP, 0, "")

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeTooManyRequests {
		t.Errorf("expected code %d, got %d", response.CodeTooManyRequests, resp.Code)
	}
	if resp.Msg != "too many requests" {
		t.Errorf("expected msg 'too many requests', got '%s'", resp.Msg)
	}

	retryHeader := w.Header().Get("Retry-After")
	if retryHeader == "" {
		t.Error("expected Retry-After header to be present")
	} else {
		// Implementation: int(2s.Seconds()) + 1 = int(2) + 1 = 3
		expected := strconv.Itoa(int(2*time.Second.Seconds()) + 1)
		if retryHeader != expected {
			t.Errorf("expected Retry-After '%s' (2s→2+1), got '%s'", expected, retryHeader)
		}
	}
}

func TestRateLimit_AdminExempted(t *testing.T) {
	lim := newFakeLimiter()
	lim.setResult("ip:192.0.2.1", false, 2*time.Second) // would reject
	router := setupRouter(lim, KeyByIP, 0, model.RoleAdmin)

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("admin should be exempt, expected 200, got %d", w.Code)
	}
	resp := mustParseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Errorf("expected code 0 for admin, got %d", resp.Code)
	}

	// Limiter.Allow must NOT be called (admin bypasses before Allow)
	if c := lim.callsFor("ip:192.0.2.1"); c != 0 {
		t.Errorf("limiter.Allow should not be called for admin, got %d calls", c)
	}
}

func TestRateLimit_DisabledPassthrough(t *testing.T) {
	// When rate-limit is disabled, the middleware is simply not mounted.
	// router.App sets the handler func to nil when RateLimitEnabled=false.
	// This test verifies requests pass through normally with no middleware.
	router := setupNoMiddlewareRouter()

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 without middleware, got %d", w.Code)
	}
	resp := mustParseResp(t, w)
	if resp.Code != response.CodeOK {
		t.Errorf("expected code 0, got %d", resp.Code)
	}
}

func TestRateLimit_KeyIsolation(t *testing.T) {
	lim := newFakeLimiter()
	// User 1 is rate-limited (denied)
	lim.setResult("uid:1", false, 1*time.Second)
	// User 2 is allowed
	lim.setResult("uid:2", true, 0)

	// Test user 1 (rate-limited)
	router1 := setupRouter(lim, KeyByUser, 1, "")
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	router1.ServeHTTP(w1, req1)
	resp1 := mustParseResp(t, w1)
	if resp1.Code != response.CodeTooManyRequests {
		t.Errorf("user 1 should be rate-limited, got code %d", resp1.Code)
	}

	// Test user 2 (allowed)
	router2 := setupRouter(lim, KeyByUser, 2, "")
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	router2.ServeHTTP(w2, req2)
	resp2 := mustParseResp(t, w2)
	if resp2.Code != response.CodeOK {
		t.Errorf("user 2 should be allowed, got code %d", resp2.Code)
	}

	// Verify limiter was called for each key exactly once
	if c := lim.callsFor("uid:1"); c != 1 {
		t.Errorf("expected 1 call for uid:1, got %d", c)
	}
	if c := lim.callsFor("uid:2"); c != 1 {
		t.Errorf("expected 1 call for uid:2, got %d", c)
	}
}

func TestRateLimit_NonAdminUserStillLimited(t *testing.T) {
	lim := newFakeLimiter()
	lim.setResult("uid:3", false, 1*time.Second)
	router := setupRouter(lim, KeyByUser, 3, model.RoleUser)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeTooManyRequests {
		t.Errorf("non-admin user should be rate-limited, got code %d", resp.Code)
	}
}

func TestRateLimit_KeyByUser_FallsBackToIP(t *testing.T) {
	// When no user_id is set, KeyByUser falls back to client IP
	lim := newFakeLimiter()
	lim.setResult("ip:192.0.2.99", false, 1*time.Second)
	router := setupRouter(lim, KeyByUser, 0, "") // no user_id injected

	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.0.2.99:12345"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	resp := mustParseResp(t, w)
	if resp.Code != response.CodeTooManyRequests {
		t.Errorf("expected 429 for IP fallback, got code %d", resp.Code)
	}
	if c := lim.callsFor("ip:192.0.2.99"); c != 1 {
		t.Errorf("expected KeyByUser to fall back to IP key, got %d calls for ip:192.0.2.99; all calls: %v", c, lim.callCount)
	}
}

func TestRateLimit_RetryAfterHeaderFormat(t *testing.T) {
	tests := []struct {
		name       string
		retryIn    time.Duration
		wantHeader string
	}{
		{"0s", 0, "1"},        // int(0) + 1 = 1
		{"1s", 1 * time.Second, "2"},  // int(1) + 1 = 2
		{"3s", 3 * time.Second, "4"},  // int(3) + 1 = 4
		{"500ms", 500 * time.Millisecond, "1"}, // int(0) + 1 = 1
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lim := newFakeLimiter()
			lim.setResult("ip:10.0.0.1", false, tc.retryIn)
			router := setupRouter(lim, KeyByIP, 0, "")

			req := httptest.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "10.0.0.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			got := w.Header().Get("Retry-After")
			if got != tc.wantHeader {
				t.Errorf("Retry-After: want '%s', got '%s'", tc.wantHeader, got)
			}
		})
	}
}
