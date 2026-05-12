package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
	"gorm.io/gorm"
)

func TestCollabUpdateCollaborator_InvalidDocID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	req := httptest.NewRequest(http.MethodPatch,
		"/api/documents/not-uuid/collaborators/"+uuid.New().String(),
		toJSON(t, map[string]string{"permission": "viewer"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCollabUpdateCollaborator_InvalidTargetUserID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	docID := uuid.New()
	req := httptest.NewRequest(http.MethodPatch,
		"/api/documents/"+docID.String()+"/collaborators/not-a-uuid",
		toJSON(t, map[string]string{"permission": "viewer"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCollabUpdateCollaborator_NotOwner_Returns403(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPatch,
		"/api/documents/"+docID.String()+"/collaborators/"+targetID.String(),
		toJSON(t, map[string]string{"permission": "viewer"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabUpdateCollaborator_CollaboratorNotFound_Returns404(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)
	targetID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodPatch,
		"/api/documents/"+doc.ID.String()+"/collaborators/"+targetID.String(),
		toJSON(t, map[string]string{"permission": "viewer"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabUpdateCollaborator_InvalidBody_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)
	targetID := uuid.New()

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodPatch,
		"/api/documents/"+doc.ID.String()+"/collaborators/"+targetID.String(),
		toJSON(t, map[string]string{"permission": "superadmin"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabUpdateCollaborator_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)
	target := testutil.NewUser()
	collab := testutil.NewCollaborator(doc.ID, target.ID, models.PermissionViewer)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, target.ID, models.PermissionViewer, time.Now(), time.Now(),
		))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "document_collaborators"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, target.ID, models.PermissionEditor, time.Now(), time.Now(),
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(
			target.ID, target.Name, target.Email, target.PasswordHash, "", nil, target.CreatedAt, target.UpdatedAt,
		))

	req := httptest.NewRequest(http.MethodPatch,
		"/api/documents/"+doc.ID.String()+"/collaborators/"+target.ID.String(),
		toJSON(t, map[string]string{"permission": "editor"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabCreateShareLink_InvalidDocID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/documents/bad-uuid/share-link",
		toJSON(t, map[string]string{"permission": "viewer"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCollabCreateShareLink_NotOwner_Returns403(t *testing.T) {
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

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+docID.String()+"/share-link",
		toJSON(t, map[string]string{"permission": "viewer"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCollabCreateShareLink_InvalidBody_Returns400(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+doc.ID.String()+"/share-link",
		toJSON(t, map[string]string{"permission": "owner"})) // "owner" not allowed
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabCreateShareLink_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "document_share_links"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_share_links"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+doc.ID.String()+"/share-link",
		toJSON(t, map[string]string{"permission": "viewer"}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestCollabCreateShareLink_WithExpiry_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildCollabRouter(db)

	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "document_share_links"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_share_links"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	expiresIn := 24
	req := httptest.NewRequest(http.MethodPost, "/api/documents/"+doc.ID.String()+"/share-link",
		toJSON(t, map[string]interface{}{"permission": "editor", "expires_in": expiresIn}))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d — body: %s", w.Code, w.Body.String())
	}
}
