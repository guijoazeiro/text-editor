package services_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newHistoryDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

var historyCols = []string{"id", "document_id", "user_id", "action", "changes", "created_at"}

func TestHistoryRecordCreation_Success(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	doc := testutil.NewDocument(testutil.NewUser().ID)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_histories"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.RecordCreation(doc.ID, doc.UserID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHistoryRecordCreation_DBError(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_histories"`).
		WillReturnError(gorm.ErrInvalidDB)
	mock.ExpectRollback()

	err := svc.RecordCreation(uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error on DB failure, got nil")
	}
}

func TestHistoryRecordUpdate_TitleChange(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	ownerID := uuid.New()
	docID := uuid.New()
	old := &models.Document{ID: docID, UserID: ownerID, Title: "Old Title", Content: "same"}
	new := &models.Document{ID: docID, UserID: ownerID, Title: "New Title", Content: "same"}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_histories"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.RecordUpdate(docID, ownerID, old, new); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHistoryRecordUpdate_ContentChange(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	ownerID := uuid.New()
	docID := uuid.New()
	old := &models.Document{ID: docID, UserID: ownerID, Title: "Title", Content: "old content"}
	newDoc := &models.Document{ID: docID, UserID: ownerID, Title: "Title", Content: "new content"}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_histories"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.RecordUpdate(docID, ownerID, old, newDoc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHistoryRecordUpdate_NoChanges_NoInsert(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	ownerID := uuid.New()
	docID := uuid.New()
	same := &models.Document{ID: docID, UserID: ownerID, Title: "Title", Content: "content"}

	_ = mock

	err := svc.RecordUpdate(docID, ownerID, same, same)
	if err != nil {
		t.Fatalf("expected nil when no changes, got: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB calls: %v", err)
	}
}

func TestHistoryRecordUpdate_BothFieldsChanged(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	ownerID := uuid.New()
	docID := uuid.New()
	old := &models.Document{ID: docID, UserID: ownerID, Title: "Old", Content: "old content"}
	newDoc := &models.Document{ID: docID, UserID: ownerID, Title: "New", Content: "new content"}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_histories"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.RecordUpdate(docID, ownerID, old, newDoc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func historyRow(h models.DocumentHistory) *sqlmock.Rows {
	return sqlmock.NewRows(historyCols).AddRow(
		h.ID, h.DocumentID, h.UserID, h.Action, h.Changes, h.CreatedAt,
	)
}

func TestHistoryGetDocumentHistory_NoFilters(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)
	entry := models.DocumentHistory{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		UserID:     user.ID,
		Action:     models.ActionCreated,
		CreatedAt:  time.Now(),
	}

	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).
		WillReturnRows(historyRow(entry))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "refresh_token_hash", "refresh_token_exp", "created_at", "updated_at"}).AddRow(
			user.ID, user.Name, user.Email, user.PasswordHash, "", nil, user.CreatedAt, user.UpdatedAt,
		))

	results, err := svc.GetDocumentHistory(doc.ID, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHistoryGetDocumentHistory_WithUserFilter(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)
	uid := user.ID

	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).
		WillReturnRows(sqlmock.NewRows(historyCols))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "refresh_token_hash", "refresh_token_exp", "created_at", "updated_at"}))

	results, err := svc.GetDocumentHistory(doc.ID, &uid, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestHistoryGetDocumentHistory_WithActionFilter(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)
	action := models.ActionCreated
	entry := models.DocumentHistory{
		ID:         uuid.New(),
		DocumentID: doc.ID,
		UserID:     user.ID,
		Action:     models.ActionCreated,
		CreatedAt:  time.Now(),
	}

	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).
		WillReturnRows(historyRow(entry))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "refresh_token_hash", "refresh_token_exp", "created_at", "updated_at"}).AddRow(
			user.ID, user.Name, user.Email, user.PasswordHash, "", nil, user.CreatedAt, user.UpdatedAt,
		))

	results, err := svc.GetDocumentHistory(doc.ID, nil, &action, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}
}

func TestHistoryGetDocumentHistory_WithDateFilters(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	doc := testutil.NewDocument(testutil.NewUser().ID)
	from := "2024-01-01"
	to := "2024-12-31"

	mock.ExpectQuery(`SELECT .+ FROM "document_histories"`).
		WillReturnRows(sqlmock.NewRows(historyCols))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash", "refresh_token_hash", "refresh_token_exp", "created_at", "updated_at"}))

	results, err := svc.GetDocumentHistory(doc.ID, nil, nil, &from, &to)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Error("expected non-nil slice")
	}
}

func TestHistoryRecordUpdate_LongContent_Truncated(t *testing.T) {
	db, mock := newHistoryDB(t)
	svc := services.NewHistoryService(db)

	ownerID := uuid.New()
	docID := uuid.New()
	longStr := string(make([]byte, 200))
	old := &models.Document{ID: docID, UserID: ownerID, Title: "T", Content: longStr}
	newDoc := &models.Document{ID: docID, UserID: ownerID, Title: "T", Content: "short"}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_histories"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.RecordUpdate(docID, ownerID, old, newDoc); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
