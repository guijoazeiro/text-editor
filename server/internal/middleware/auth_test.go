package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/middleware"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupAuthRouter() *gin.Engine {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()
	r.GET("/protected", middleware.AuthRequired(jwtSvc), func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})
	return r
}

func TestAuthRequired_NoHeader_Returns401(t *testing.T) {
	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_EmptyBearerHeader_Returns401(t *testing.T) {
	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_InvalidToken_Returns401(t *testing.T) {
	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer this-is-not-a-real-token")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestAuthRequired_ExpiredToken_Returns401(t *testing.T) {
	r := setupAuthRouter()
	userID := uuid.New()
	expiredToken, err := testutil.GenerateExpiredToken(userID, "expired@example.com")
	if err != nil {
		t.Fatalf("failed to generate expired token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+expiredToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthRequired_ValidBearerToken_Returns200(t *testing.T) {
	r := setupAuthRouter()
	userID := uuid.New()
	token, err := testutil.GenerateTestToken(userID, "alice@example.com")
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with valid token, got %d", w.Code)
	}
}

func TestAuthRequired_ValidQueryParamToken_Returns200(t *testing.T) {
	r := setupAuthRouter()
	userID := uuid.New()
	token, _ := testutil.GenerateTestToken(userID, "alice@example.com")

	req := httptest.NewRequest(http.MethodGet, "/protected?token="+token, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 with token query param, got %d", w.Code)
	}
}

func TestAuthRequired_InjectsUserIDIntoContext(t *testing.T) {
	r := setupAuthRouter()
	userID := uuid.New()
	token, _ := testutil.GenerateTestToken(userID, "alice@example.com")

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	gotID, ok := body["user_id"].(string)
	if !ok || gotID == "" {
		t.Errorf("expected user_id in response body, got: %v", body["user_id"])
	}
	if gotID != userID.String() {
		t.Errorf("expected user_id %s, got %s", userID, gotID)
	}
}

func TestAuthRequired_MalformedAuthHeader_Returns401(t *testing.T) {
	r := setupAuthRouter()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Token sometoken")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for malformed auth header, got %d", w.Code)
	}
}
