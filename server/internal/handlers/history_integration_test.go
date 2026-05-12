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

var histCols = []string{"id", "document_id", "user_id", "action", "changes", "created_at"}

func buildHistoryRouter(db *gorm.DB) (*gin.Engine, *testutil.UserFixture) {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()

	owner := testutil.NewUser()
	ownerFixture := &testutil.UserFixture{
		ID: owner.ID, Name: owner.Name, Email: owner.Email,
		PasswordHash: owner.PasswordHash, CreatedAt: owner.CreatedAt, UpdatedAt: owner.UpdatedAt,
	}

	h := handlers.NewHistoryHandler(db)
	g := r.Group("/api/documents/:id")
	g.Use(middleware.AuthRequired(jwtSvc))
	{
		g.GET("/history", h.GetDocumentHistory)
	}
	return r, ownerFixture
}

func histRow(h models.DocumentHistory) *sqlmock.Rows {
	return sqlmock.NewRows(histCols).AddRow(
		h.ID, h.DocumentID, h.UserID, h.Action, h.Changes, h.CreatedAt,
	)
}

func TestHistoryHandler_NoAuth_Returns401(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, _ := buildHistoryRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+uuid.New().String()+"/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestHistoryHandler_InvalidDocID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildHistoryRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/not-a-uuid/history", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHistoryHandler_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildHistoryRouter(db)

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

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/history", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHistoryHandler_NoFilters_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildHistoryRouter(db)

	doc := testutil.NewDocument(owner.ID)
	entry := models.DocumentHistory{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		UserID:     owner.ID,
		Action:     models.ActionCreated,
		CreatedAt:  time.Now(),
	}

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).WillReturnRows(histRow(entry))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).WillReturnRows(
		sqlmock.NewRows(userCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		),
	)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/history", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHistoryHandler_WithActionFilter_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildHistoryRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).
		WillReturnRows(sqlmock.NewRows(histCols))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/history?action=created", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHistoryHandler_InvalidActionFilter_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildHistoryRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/history?action=invalid_action", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHistoryHandler_InvalidUserIDFilter_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildHistoryRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/history?user_id=not-a-uuid", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHistoryHandler_WithDateFilters_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildHistoryRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).
		WillReturnRows(sqlmock.NewRows(histCols))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols))

	url := "/api/documents/" + doc.ID.String() + "/history?from_date=2024-01-01&to_date=2024-12-31"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestHistoryHandler_WithValidUserIDFilter_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildHistoryRouter(db)

	doc := testutil.NewDocument(owner.ID)
	filterUID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).
		WillReturnRows(sqlmock.NewRows(histCols))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols))

	url := "/api/documents/" + doc.ID.String() + "/history?user_id=" + filterUID.String()
	req := httptest.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}
