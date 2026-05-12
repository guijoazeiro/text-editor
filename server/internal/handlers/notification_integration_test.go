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

var notifCols = []string{
	"id", "user_id", "type", "title", "message",
	"document_id", "from_user_id", "read", "created_at",
}

func buildNotifRouter(db *gorm.DB) (*gin.Engine, *testutil.UserFixture) {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()

	owner := testutil.NewUser()
	ownerFixture := &testutil.UserFixture{
		ID: owner.ID, Name: owner.Name, Email: owner.Email,
		PasswordHash: owner.PasswordHash, CreatedAt: owner.CreatedAt, UpdatedAt: owner.UpdatedAt,
	}

	h := handlers.NewNotificationHandler(db)
	g := r.Group("/api/notifications")
	g.Use(middleware.AuthRequired(jwtSvc))
	{
		g.GET("", h.List)
		g.POST("/:id/read", h.MarkAsRead)
		g.POST("/read-all", h.MarkAllAsRead)
		g.DELETE("/:id", h.Delete)
	}
	return r, ownerFixture
}

func notifRow(n models.Notification) *sqlmock.Rows {
	return sqlmock.NewRows(notifCols).AddRow(
		n.ID, n.UserID, n.Type, n.Title, n.Message,
		n.DocumentID, n.FromUserID, n.Read, n.CreatedAt,
	)
}

func TestNotifList_NoAuth_Returns401(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, _ := buildNotifRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestNotifList_Returns200(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	mock.ExpectQuery(`SELECT .+ FROM "notifications"`).
		WillReturnRows(sqlmock.NewRows(notifCols))
	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	req := httptest.NewRequest(http.MethodGet, "/api/notifications", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestNotifMarkAsRead_InvalidID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/not-a-uuid/read", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestNotifMarkAsRead_NotFound_Returns404(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	nid := uuid.New()
	mock.ExpectQuery(`SELECT .+ FROM "notifications"`).
		WillReturnError(gorm.ErrRecordNotFound)

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/"+nid.String()+"/read", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestNotifMarkAsRead_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	docID := uuid.New()
	fromID := uuid.New()
	nid := uuid.New()
	notif := models.Notification{
		ID: nid, UserID: owner.ID,
		Type: models.NotificationCollaboratorAdded, Title: "T", Message: "M",
		DocumentID: &docID, FromUserID: &fromID, Read: false, CreatedAt: time.Now(),
	}

	mock.ExpectQuery(`SELECT .+ FROM "notifications"`).WillReturnRows(notifRow(notif))
	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notifications"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/"+nid.String()+"/read", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestNotifMarkAllAsRead_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	mock.ExpectBegin()
	mock.ExpectExec(`UPDATE "notifications"`).WillReturnResult(sqlmock.NewResult(3, 3))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/notifications/read-all", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestNotifDelete_InvalidID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	req := httptest.NewRequest(http.MethodDelete, "/api/notifications/bad-id", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestNotifDelete_NotFound_Returns404(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	nid := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "notifications"`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodDelete, "/api/notifications/"+nid.String(), nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing notification, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestNotifDelete_Success(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, owner := buildNotifRouter(db)

	nid := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "notifications"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodDelete, "/api/notifications/"+nid.String(), nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
}
