package handlers_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	internalauth "github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/handlers"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/middleware"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newAuthTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return gormDB, mock
}

func buildAuthRouter(db *gorm.DB) *gin.Engine {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()
	h := handlers.NewAuthHandler(db, jwtSvc)

	g := r.Group("/api/auth")
	g.POST("/signup", h.Signup)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.Refresh)
	g.GET("/me", middleware.AuthRequired(jwtSvc), h.Me)
	g.PATCH("/me", middleware.AuthRequired(jwtSvc), h.UpdateMe)
	g.POST("/logout", middleware.AuthRequired(jwtSvc), h.Logout)
	return r
}

func toJSON(t *testing.T, v interface{}) *bytes.Buffer {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return bytes.NewBuffer(b)
}

var userCols = []string{"id", "name", "email", "password_hash", "refresh_token_hash", "refresh_token_exp", "created_at", "updated_at"}

func userRow(u testutil.UserFixture) *sqlmock.Rows {
	return sqlmock.NewRows(userCols).AddRow(
		u.ID, u.Name, u.Email, u.PasswordHash, u.RefreshTokenHash, u.RefreshTokenExp, u.CreatedAt, u.UpdatedAt,
	)
}

func TestSignup_Success(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := toJSON(t, map[string]string{"name": "Alice", "email": "alice@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestSignup_EmailAlreadyRegistered(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	existing := testutil.NewUser()
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(userRow(testutil.UserFixture{
			ID: existing.ID, Name: existing.Name, Email: existing.Email,
			PasswordHash: existing.PasswordHash, CreatedAt: existing.CreatedAt, UpdatedAt: existing.UpdatedAt,
		}))

	body := toJSON(t, map[string]string{"name": "Alice", "email": "alice@example.com", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestSignup_InvalidBody(t *testing.T) {
	db, _ := newAuthTestDB(t)
	r := buildAuthRouter(db)

	body := toJSON(t, map[string]string{"email": "not-an-email"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/signup", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_Success(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	password := "password123"
	hash, _ := internalauth.HashPassword(password)
	u := testutil.NewUser(func(u *testutil.UserModel) { u.Email = "alice@example.com"; u.PasswordHash = hash })

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(userRow(testutil.UserFixture{
			ID: u.ID, Name: u.Name, Email: u.Email,
			PasswordHash: hash, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		}))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := toJSON(t, map[string]string{"email": "alice@example.com", "password": password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	data, _ := resp["data"].(map[string]interface{})
	if data["token"] == "" || data["token"] == nil {
		t.Error("expected non-empty token in response data")
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	hash, _ := internalauth.HashPassword("correctpassword")
	u := testutil.NewUser()

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(userRow(testutil.UserFixture{
			ID: u.ID, Name: u.Name, Email: u.Email,
			PasswordHash: hash, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		}))

	body := toJSON(t, map[string]string{"email": "alice@example.com", "password": "wrongpassword"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_EmailNotFound(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)

	body := toJSON(t, map[string]string{"email": "nobody@example.com", "password": "any"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestLogin_InvalidBody(t *testing.T) {
	db, _ := newAuthTestDB(t)
	r := buildAuthRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewBufferString("{bad json}"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestMe_NoToken_Returns401(t *testing.T) {
	db, _ := newAuthTestDB(t)
	r := buildAuthRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMe_ValidToken_Returns200(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	u := testutil.NewUser()
	token, _ := testutil.GenerateTestToken(u.ID, u.Email)

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(userRow(testutil.UserFixture{
			ID: u.ID, Name: u.Name, Email: u.Email,
			PasswordHash: u.PasswordHash, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		}))

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestRefresh_NoCookie_Returns401(t *testing.T) {
	db, _ := newAuthTestDB(t)
	r := buildAuthRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestRefresh_ValidCookie_Returns200(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	jwtSvc := testutil.NewTestJWT()
	u := testutil.NewUser()
	refreshToken, _ := jwtSvc.GenerateRefreshToken(u.ID)
	tokenHash := internalauth.HashRefreshToken(refreshToken)
	exp := time.Now().Add(30 * 24 * time.Hour)

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(userRow(testutil.UserFixture{
			ID: u.ID, Name: u.Name, Email: u.Email,
			PasswordHash:     u.PasswordHash,
			RefreshTokenHash: tokenHash,
			RefreshTokenExp:  &exp,
			CreatedAt:        u.CreatedAt,
			UpdatedAt:        u.UpdatedAt,
		}))

	req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestLogout_ValidToken_Returns200(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	u := testutil.NewUser()
	token, _ := testutil.GenerateTestToken(u.ID, u.Email)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestLogout_NoToken_Returns401(t *testing.T) {
	db, _ := newAuthTestDB(t)
	r := buildAuthRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpdateMe_ValidToken_Returns200(t *testing.T) {
	db, mock := newAuthTestDB(t)
	r := buildAuthRouter(db)

	u := testutil.NewUser()
	token, _ := testutil.GenerateTestToken(u.ID, u.Email)

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(userRow(testutil.UserFixture{
			ID: u.ID, Name: u.Name, Email: u.Email,
			PasswordHash: u.PasswordHash, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt,
		}))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "users"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := toJSON(t, map[string]string{"name": "New Name"})
	req := httptest.NewRequest(http.MethodPatch, "/api/auth/me", body)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}
