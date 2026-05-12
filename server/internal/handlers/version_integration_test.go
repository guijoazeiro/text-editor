package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/handlers"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/middleware"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
	"gorm.io/gorm"
)

var versionCols = []string{"id", "document_id", "version_number", "title", "content", "yjs_snapshot", "created_by", "created_at"}

func buildVersionRouter(db *gorm.DB) (*gin.Engine, *testutil.UserFixture) {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()

	owner := testutil.NewUser()
	ownerFixture := &testutil.UserFixture{
		ID: owner.ID, Name: owner.Name, Email: owner.Email,
		PasswordHash: owner.PasswordHash, CreatedAt: owner.CreatedAt, UpdatedAt: owner.UpdatedAt,
	}

	h := handlers.NewVersionHandler(db, nil, nil, nil)
	g := r.Group("/api/documents/:id/versions")
	g.Use(middleware.AuthRequired(jwtSvc))
	{
		g.GET("", h.GetVersions)
		g.GET("/:version_number", h.GetVersion)
		g.POST("/:version_number/restore", h.RestoreVersion)
		g.GET("/compare", h.CompareVersions)
	}
	return r, ownerFixture
}

func versionRow(v models.DocumentVersion) *sqlmock.Rows {
	return sqlmock.NewRows(versionCols).AddRow(
		v.ID, v.DocumentID, v.VersionNumber, v.Title, v.Content, v.YjsSnapshot, v.CreatedBy, v.CreatedAt,
	)
}

func TestVersionGetVersions_NoAuth_Returns401(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, _ := buildVersionRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+uuid.New().String()+"/versions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestVersionGetVersions_InvalidDocID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildVersionRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/not-a-uuid/versions", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestVersionGetVersions_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildVersionRouter(db)

	ownerID := uuid.New()
	strangerID := uuid.New()
	docID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/versions", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestVersionGetVersions_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildVersionRouter(db)

	doc := testutil.NewDocument(owner.ID)
	ver := models.DocumentVersion{
		ID:            uuid.New(),
		DocumentID:    doc.ID,
		VersionNumber: 1,
		Title:         "v1",
		Content:       "content",
		CreatedBy:     owner.ID,
		CreatedAt:     time.Now(),
	}

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).WillReturnRows(versionRow(ver))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/versions", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestVersionGetVersion_InvalidVersionNumber_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildVersionRouter(db)

	docID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/versions/abc", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestVersionGetVersion_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildVersionRouter(db)

	ownerID := uuid.New()
	strangerID := uuid.New()
	docID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/versions/1", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestVersionRestoreVersion_NoEdit_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildVersionRouter(db)

	ownerID := uuid.New()
	strangerID := uuid.New()
	docID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+docID.String()+"/versions/1/restore", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestVersionCompareVersions_MissingParams_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildVersionRouter(db)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/versions/compare", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing v1/v2 params, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestVersionCompareVersions_InvalidParams_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildVersionRouter(db)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/versions/compare?v1=abc&v2=xyz", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-numeric params, got %d", w.Code)
	}
}

func TestParseIntQuery_ValidInt(t *testing.T) {
	cases := []struct {
		query    string
		key      string
		fallback int
		want     int
	}{
		{"page=3", "page", 1, 3},
		{"limit=25", "limit", 10, 25},
		{"limit=abc", "limit", 10, 10},
		{"", "page", 1, 1},
	}
	for _, tc := range cases {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest(http.MethodGet, "/?"+tc.query, nil)
		got := handlers.ParseIntQuery(c, tc.key, tc.fallback)
		if got != tc.want {
			t.Errorf("ParseIntQuery(%q, %q, %d) = %d, want %d", tc.query, tc.key, tc.fallback, got, tc.want)
		}
	}
}

func TestClamp(t *testing.T) {
	cases := []struct {
		v, lo, hi, want int
	}{
		{5, 1, 10, 5},
		{0, 1, 10, 1},
		{15, 1, 10, 10},
		{1, 1, 10, 1},
		{10, 1, 10, 10},
	}
	for _, tc := range cases {
		got := handlers.Clamp(tc.v, tc.lo, tc.hi)
		if got != tc.want {
			t.Errorf("Clamp(%d,%d,%d) = %d, want %d", tc.v, tc.lo, tc.hi, got, tc.want)
		}
	}
}
