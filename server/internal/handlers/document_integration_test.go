package handlers_test

import (
	"bytes"
	"encoding/json"
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
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDocTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

func buildDocRouter(db *gorm.DB) (*gin.Engine, *testutil.UserFixture) {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()

	owner := testutil.NewUser()
	ownerFixture := &testutil.UserFixture{
		ID:           owner.ID,
		Name:         owner.Name,
		Email:        owner.Email,
		PasswordHash: owner.PasswordHash,
		CreatedAt:    owner.CreatedAt,
		UpdatedAt:    owner.UpdatedAt,
	}

	h := handlers.NewDocumentHandler(db, nil)

	g := r.Group("/api/documents")
	g.Use(middleware.AuthRequired(jwtSvc))
	{
		g.POST("", h.Create)
		g.GET("", h.List)
		g.GET("/trash", h.Trash)
		g.GET("/:id", h.GetByID)
		g.PUT("/:id", h.Update)
		g.DELETE("/:id", h.Delete)
		g.POST("/:id/restore", h.Restore)
	}
	return r, ownerFixture
}

var docCols = []string{
	"id", "title", "content", "content_format", "user_id",
	"created_at", "updated_at", "deleted_at",
}

func docRow(d models.Document) *sqlmock.Rows {
	return sqlmock.NewRows(docCols).AddRow(
		d.ID, d.Title, d.Content, d.ContentFormat, d.UserID,
		d.CreatedAt, d.UpdatedAt, nil,
	)
}

func authHeader(t *testing.T, userID uuid.UUID, email string) string {
	t.Helper()
	token, err := testutil.GenerateTestToken(userID, email)
	if err != nil {
		t.Fatalf("GenerateTestToken: %v", err)
	}
	return "Bearer " + token
}

func TestDocCreate_NoAuth_Returns401(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, _ := buildDocRouter(db)

	body := bytes.NewBufferString(`{"title":"My Doc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/documents", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDocCreate_MissingTitle_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	body := bytes.NewBufferString(`{"content":"no title here"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/documents", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocCreate_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	newDocID := uuid.New()
	now := time.Now()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "documents"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_histories"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectQuery(`SELECT COALESCE`).WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_versions"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(sqlmock.NewRows(docCols).AddRow(
		newDocID, "My Doc", "", models.ContentFormatTipTap, owner.ID, now, now, nil,
	))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).WillReturnRows(sqlmock.NewRows(userCols).AddRow(
		owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
	))

	body := bytes.NewBufferString(`{"title":"My Doc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/documents", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocGetByID_InvalidUUID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/not-a-uuid", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocGetByID_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildDocRouter(db)

	docID := uuid.New()
	otherOwnerID := uuid.New()
	requesterID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, otherOwnerID,
			time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String(), nil)
	req.Header.Set("Authorization", authHeader(t, requesterID, "stranger@example.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocGetByID_AsOwner_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(docRow(doc))

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		))

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String(), nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocDelete_NotOwner_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildDocRouter(db)

	docID := uuid.New()
	ownerID := uuid.New()
	requesterID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))

	req := httptest.NewRequest(http.MethodDelete, "/api/documents/"+docID.String(), nil)
	req.Header.Set("Authorization", authHeader(t, requesterID, "stranger@example.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocDelete_AsOwner_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(docRow(doc))

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "documents"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodDelete, "/api/documents/"+doc.ID.String(), nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocUpdate_NotEditor_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildDocRouter(db)

	docID := uuid.New()
	ownerID := uuid.New()
	requesterID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	body := bytes.NewBufferString(`{"title":"New Title"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/documents/"+docID.String(), body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, requesterID, "stranger@example.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocTrash_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(append(docCols, "user_id")).AddRow(
			uuid.New(), "Deleted Doc", "", models.ContentFormatTipTap, owner.ID,
			time.Now(), time.Now(), time.Now(), owner.ID,
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/trash", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocRestore_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	deletedDoc := testutil.NewDocument(owner.ID)
	deletedAt := time.Now().Add(-1 * time.Hour)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(append(docCols, "deleted_at")).AddRow(
			deletedDoc.ID, deletedDoc.Title, deletedDoc.Content, deletedDoc.ContentFormat, owner.ID,
			deletedDoc.CreatedAt, deletedDoc.UpdatedAt, deletedAt, deletedAt,
		))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "documents"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+deletedDoc.ID.String()+"/restore", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestDocList_Returns200WithPagination(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildDocRouter(db)

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(docRow(testutil.NewDocument(owner.ID)))

	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		))

	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "document_id", "user_id", "permission"}))

	req := httptest.NewRequest(http.MethodGet, "/api/documents?page=1&limit=10", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	data, _ := resp["data"].(map[string]interface{})
	if data["page"] == nil {
		t.Error("expected 'page' field in response data")
	}
	if data["limit"] == nil {
		t.Error("expected 'limit' field in response data")
	}
}
