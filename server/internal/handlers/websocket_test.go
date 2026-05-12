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
	ws "github.com/guijoazeiro/text-editor/tree/main/server/internal/websocket"
	"gorm.io/gorm"
)

func buildWSRouter(db *gorm.DB, hub *ws.Hub) (*gin.Engine, *testutil.UserFixture) {
	r := gin.New()
	jwtSvc := testutil.NewTestJWT()

	owner := testutil.NewUser()
	ownerFixture := &testutil.UserFixture{
		ID: owner.ID, Name: owner.Name, Email: owner.Email,
		PasswordHash: owner.PasswordHash, CreatedAt: owner.CreatedAt, UpdatedAt: owner.UpdatedAt,
	}

	h := handlers.NewWebSocketHandler(hub, db, jwtSvc)
	g := r.Group("/api/documents/:id")
	g.Use(middleware.AuthRequired(jwtSvc))
	{
		g.GET("/active-users", h.GetActiveUsers)
	}
	return r, ownerFixture
}

func newIdleHub() *ws.Hub {
	return ws.NewHub(nil, nil, nil)
}

func TestWSGetActiveUsers_NoAuth_Returns401(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, _ := buildWSRouter(db, newIdleHub())

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+uuid.New().String()+"/active-users", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestWSGetActiveUsers_InvalidDocID_Returns400(t *testing.T) {
	db, _ := newDocTestDB(t)
	r, owner := buildWSRouter(db, newIdleHub())

	req := httptest.NewRequest(http.MethodGet, "/api/documents/not-a-uuid/active-users", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestWSGetActiveUsers_NoPermission_Returns403(t *testing.T) {
	db, mock := newDocTestDB(t)
	r, _ := buildWSRouter(db, newIdleHub())

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

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+docID.String()+"/active-users", nil)
	req.Header.Set("Authorization", authHeader(t, strangerID, "s@s.com"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d — body: %s", w.Code, w.Body.String())
	}
}

func TestWSGetActiveUsers_EmptyHub_Returns200WithZero(t *testing.T) {
	db, mock := newDocTestDB(t)
	hub := newIdleHub()
	r, owner := buildWSRouter(db, hub)

	doc := testutil.NewDocument(owner.ID)
	mock.ExpectQuery(`SELECT .+ FROM "documents"`).WillReturnRows(docRow(doc))

	req := httptest.NewRequest(http.MethodGet, "/api/documents/"+doc.ID.String()+"/active-users", nil)
	req.Header.Set("Authorization", authHeader(t, owner.ID, owner.Email))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}
	if !containsStr(w.Body.String(), "\"count\":0") {
		t.Errorf("expected count=0 in body, got: %s", w.Body.String())
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStrHelper(s, sub))
}

func containsStrHelper(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestHubGetActiveUsers_NoClientsForDoc(t *testing.T) {
	hub := newIdleHub()
	docID := uuid.New()

	count := hub.GetActiveUsers(docID)
	if count != 0 {
		t.Errorf("expected 0 for unknown doc, got %d", count)
	}
}

func TestHubGetActiveUsers_MultipleDocuments_IsolatedCounts(t *testing.T) {
	hub := ws.NewHub(nil, nil, nil)

	docA := uuid.New()
	docB := uuid.New()
	docC := uuid.New()

	var fake1, fake2, fake3 ws.Client
	hub.Clients[docA] = map[*ws.Client]bool{&fake1: true, &fake2: true}
	hub.Clients[docB] = map[*ws.Client]bool{&fake3: true}

	if got := hub.GetActiveUsers(docA); got != 2 {
		t.Errorf("docA: expected 2, got %d", got)
	}
	if got := hub.GetActiveUsers(docB); got != 1 {
		t.Errorf("docB: expected 1, got %d", got)
	}
	if got := hub.GetActiveUsers(docC); got != 0 {
		t.Errorf("unknown doc: expected 0, got %d", got)
	}
}
