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

var collabCols = []string{"id", "document_id", "user_id", "permission", "created_at", "updated_at"}

func buildCollabRouter(db *gorm.DB) (*gin.Engine, *testutil.UserFixture) {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()

	owner := testutil.NewUser()
	ownerFixture := &testutil.UserFixture{
		ID: owner.ID, Name: owner.Name, Email: owner.Email,
		PasswordHash: owner.PasswordHash, CreatedAt: owner.CreatedAt, UpdatedAt: owner.UpdatedAt,
	}

	h := handlers.NewCollaboratorHandler(db, nil)
	g := r.Group("/api/documents/:id")
	g.Use(middleware.AuthRequired(jwtSvc))
	{
		g.POST("/collaborators", h.AddCollaborator)
		g.GET("/collaborators", h.ListCollaborators)
		g.PATCH("/collaborators/:user_id", h.UpdateCollaborator)
		g.DELETE("/collaborators/:user_id", h.RemoveCollaborator)
		g.POST("/share-link", h.CreateShareLink)
		g.DELETE("/share-link", h.DeleteShareLink)
	}
	r.GET("/api/share/:token", h.GetByShareLink)
	return r, ownerFixture
}

func TestCollabAddCollaborator_InvalidDocID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/not-a-uuid/collaborators", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCollabAddCollaborator_NotOwner_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildCollabRouter(db)

	ownerID := uuid.New()
	requesterID := uuid.New()
	docID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+docID.String()+"/collaborators",
		toJSON(t, map[string]string{"email": "x@x.com", "permission": "editor"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, requesterID, "stranger@example.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabAddCollaborator_TargetUserNotFound_Returns404(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+doc.ID.String()+"/collaborators",
		toJSON(t, map[string]string{"email": "nobody@example.com", "permission": "editor"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabAddCollaborator_AlreadyExists_Returns409(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)
	target := testutil.NewUser()
	collab := testutil.NewCollaborator(doc.ID, target.ID, models.PermissionEditor)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			target.ID, target.Name, target.Email, target.PasswordHash, "", nil, target.CreatedAt, target.UpdatedAt,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, target.ID, models.PermissionEditor, time.Now(), time.Now(),
		))

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+doc.ID.String()+"/collaborators",
		toJSON(t, map[string]string{"email": target.Email, "permission": "editor"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabListCollaborators_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildCollabRouter(db)

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

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/collaborators", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCollabListCollaborators_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)
	target := testutil.NewUser()
	collab := testutil.NewCollaborator(doc.ID, target.ID, models.PermissionEditor)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, target.ID, models.PermissionEditor, time.Now(), time.Now(),
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			target.ID, target.Name, target.Email, target.PasswordHash, "", nil, target.CreatedAt, target.UpdatedAt,
		))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/collaborators", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabRemoveCollaborator_NotOwner_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildCollabRouter(db)

	ownerID := uuid.New()
	strangerID := uuid.New()
	targetID := uuid.New()
	docID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))

	req := httptest.NewRequest(http.MethodDelete, "/api/documents/"+docID.String()+"/collaborators/"+targetID.String(), nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCollabRemoveCollaborator_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)
	target := testutil.NewUser()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "document_collaborators"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodDelete, "/api/documents/"+doc.ID.String()+"/collaborators/"+target.ID.String(), nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabGetByShareLink_NotFound_Returns404(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildCollabRouter(db)

	mock.ExpectQuery(`SELECT .+ FROM "document_share_links"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodGet, "/api/share/nonexistent-token", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabDeleteShareLink_NotOwner_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildCollabRouter(db)

	ownerID := uuid.New()
	strangerID := uuid.New()
	docID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			docID, "Doc", "", models.ContentFormatTipTap, ownerID,
			time.Now(), time.Now(), nil,
		))

	req := httptest.NewRequest(http.MethodDelete, "/api/documents/"+docID.String()+"/share-link", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}
