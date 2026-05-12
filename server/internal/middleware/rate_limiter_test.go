package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/middleware"
	"golang.org/x/time/rate"
)

func buildRateLimitedRouter(r rate.Limit, burst int) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	rl := middleware.NewRateLimiter(r, burst)
	router.Use(rl.Middleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return router
}

func TestRateLimiter_AllowsWithinBurst(t *testing.T) {
	router := buildRateLimitedRouter(rate.Every(0), 5)

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
	}
}

func TestRateLimiter_BlocksAfterBurstExhausted(t *testing.T) {
	router := buildRateLimitedRouter(0, 2)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "10.0.0.2:1234"
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 after burst, got %d", w.Code)
	}
}

func TestRateLimiter_DifferentIPsAreIndependent(t *testing.T) {
	router := buildRateLimitedRouter(0, 1)

	for _, ip := range []string{"192.168.1.1:0", "192.168.1.2:0", "192.168.1.3:0"} {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("IP %s: expected 200 on first request, got %d", ip, w.Code)
		}
	}

	for _, ip := range []string{"192.168.1.1:0", "192.168.1.2:0", "192.168.1.3:0"} {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusTooManyRequests {
			t.Errorf("IP %s: expected 429 on second request, got %d", ip, w.Code)
		}
	}
}

func TestRateLimiter_NewRateLimiter_ReturnsNonNil(t *testing.T) {
	rl := middleware.NewRateLimiter(rate.Limit(10), 20)
	if rl == nil {
		t.Fatal("NewRateLimiter returned nil")
	}
}

func TestRateLimiter_Middleware_ReturnsHandlerFunc(t *testing.T) {
	rl := middleware.NewRateLimiter(rate.Limit(100), 100)
	fn := rl.Middleware()
	if fn == nil {
		t.Fatal("Middleware() returned nil HandlerFunc")
	}
}
