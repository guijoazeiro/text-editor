package handlers_test

import (
	"encoding/base64"
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
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
	"gorm.io/gorm"
)

var yjsCols = []string{"id", "document_id", "lamport_ts", "client_id", "update", "created_at"}

func buildYjsRouter(db *gorm.DB) (*gin.Engine, *testutil.UserFixture) {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()

	owner := testutil.NewUser()
	ownerFixture := &testutil.UserFixture{
		ID: owner.ID, Name: owner.Name, Email: owner.Email,
		PasswordHash: owner.PasswordHash, CreatedAt: owner.CreatedAt, UpdatedAt: owner.UpdatedAt,
	}

	permSvc := services.NewPermissionService(db)
	yjsSvc := services.NewYjsService(db)
	h := handlers.NewYjsHandler(yjsSvc, permSvc, nil, nil)

	g := r.Group("/api/documents/:id/yjs")
	g.Use(middleware.AuthRequired(jwtSvc))
	{
		g.GET("/updates", h.GetUpdates)
		g.GET("/state-vector", h.GetStateVector)
		g.GET("/diff", h.GetDiff)
	}
	return r, ownerFixture
}

func TestYjsGetUpdates_NoAuth_Returns401(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, _ := buildYjsRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+uuid.New().String()+"/yjs/updates", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestYjsGetUpdates_InvalidDocID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildYjsRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/not-uuid/yjs/updates", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestYjsGetUpdates_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildYjsRouter(db)

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

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/yjs/updates", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestYjsGetUpdates_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildYjsRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "yjs_updates"`).
		WillReturnRows(sqlmock.NewRows(yjsCols).AddRow(
			uuid.New(), doc.ID, int64(1), int64(42), []byte("update-data"), time.Now(),
		))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/yjs/updates", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestYjsGetStateVector_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildYjsRouter(db)

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

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/yjs/state-vector", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestYjsGetStateVector_NilServices_Returns204(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildYjsRouter(db)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/yjs/state-vector", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 with nil snapshot service, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestYjsGetDiff_MissingSV_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildYjsRouter(db)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/yjs/diff", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing sv param, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestYjsGetDiff_InvalidBase64SV_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildYjsRouter(db)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/yjs/diff?sv=!!!not-base64!!!", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid base64, got %d", w.Code)
	}
}

func TestYjsGetDiff_NilCompactorService_Returns503(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildYjsRouter(db)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	sv := base64.StdEncoding.EncodeToString([]byte("fake-state-vector"))
	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/yjs/diff?sv="+sv, nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 with nil compactor, got %d — body: %s", w.Code, w.Body.String())
	}
}
