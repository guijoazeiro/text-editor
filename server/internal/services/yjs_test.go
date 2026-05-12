package services_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newYjsDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

var yjsUpdateCols = []string{"id", "document_id", "lamport_ts", "client_id", "update", "created_at"}

func TestYjsSaveUpdate_Success(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	doc := testutil.NewDocument(testutil.NewUser().ID)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO "yjs_updates"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New(), time.Now()),
	)
	mock.ExpectCommit()

	if err := svc.SaveUpdate(doc.ID, []byte("update-bytes"), int64(1), int64(42)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestYjsSaveUpdate_DBError(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "yjs_updates"`).
		WillReturnError(gorm.ErrInvalidDB)
	mock.ExpectRollback()

	err := svc.SaveUpdate(uuid.New(), []byte("data"), int64(1), int64(0))
	if err == nil {
		t.Fatal("expected error on DB failure")
	}
}

func TestYjsGetUpdates_ReturnsOrderedList(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	docID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT .+ FROM "yjs_updates"`).
		WillReturnRows(sqlmock.NewRows(yjsUpdateCols).
			AddRow(uuid.New(), docID, int64(1), int64(10), []byte("a"), now).
			AddRow(uuid.New(), docID, int64(2), int64(10), []byte("b"), now),
		)

	updates, err := svc.GetUpdates(docID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 2 {
		t.Errorf("expected 2 updates, got %d", len(updates))
	}
}

func TestYjsGetUpdates_EmptyResult(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	docID := uuid.New()
	mock.ExpectQuery(`SELECT .+ FROM "yjs_updates"`).
		WillReturnRows(sqlmock.NewRows(yjsUpdateCols))

	updates, err := svc.GetUpdates(docID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("expected empty slice, got %d items", len(updates))
	}
}

func TestYjsGetUpdatesSince_FiltersCorrectly(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	docID := uuid.New()
	now := time.Now()

	mock.ExpectQuery(`SELECT .+ FROM "yjs_updates"`).
		WillReturnRows(sqlmock.NewRows(yjsUpdateCols).
			AddRow(uuid.New(), docID, int64(6), int64(10), []byte("c"), now),
		)

	updates, err := svc.GetUpdatesSince(docID, int64(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 1 {
		t.Errorf("expected 1 update, got %d", len(updates))
	}
	if updates[0].LamportTS != 6 {
		t.Errorf("expected lamport_ts=6, got %d", updates[0].LamportTS)
	}
}

func TestYjsGetUpdatesSince_EmptyResult(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	mock.ExpectQuery(`SELECT .+ FROM "yjs_updates"`).
		WillReturnRows(sqlmock.NewRows(yjsUpdateCols))

	updates, err := svc.GetUpdatesSince(uuid.New(), int64(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("expected empty, got %d", len(updates))
	}
}

func TestYjsGetDocumentUpdateCount_ReturnsCount(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	docID := uuid.New()
	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(7))

	count, err := svc.GetDocumentUpdateCount(docID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 7 {
		t.Errorf("expected count=7, got %d", count)
	}
}

func TestYjsGetDocumentUpdateCount_Zero(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	count, err := svc.GetDocumentUpdateCount(uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestYjsCompactUpdates_Success(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	docID := uuid.New()
	merged := []byte("merged-update")

	mock.ExpectBegin()

	mock.ExpectExec(`DELETE FROM "yjs_updates"`).
		WillReturnResult(sqlmock.NewResult(3, 3))

	mock.ExpectQuery(`INSERT INTO "yjs_updates"`).WillReturnRows(
		sqlmock.NewRows([]string{"id", "created_at"}).AddRow(uuid.New(), time.Now()),
	)
	mock.ExpectCommit()

	if err := svc.CompactUpdates(docID, int64(10), merged); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestYjsCompactUpdates_DeleteFails_Rollback(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "yjs_updates"`).
		WillReturnError(gorm.ErrInvalidDB)
	mock.ExpectRollback()

	err := svc.CompactUpdates(uuid.New(), int64(5), []byte("data"))
	if err == nil {
		t.Fatal("expected error when DELETE fails")
	}
}

func TestYjsDeleteUpdatesForDocument_Success(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	docID := uuid.New()
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "yjs_updates"`).
		WillReturnResult(sqlmock.NewResult(5, 5))
	mock.ExpectCommit()

	if err := svc.DeleteUpdatesForDocument(docID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestYjsDeleteUpdatesForDocument_AlreadyEmpty(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "yjs_updates"`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	if err := svc.DeleteUpdatesForDocument(uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestYjsDeleteOldUpdates_Success(t *testing.T) {
	db, mock := newYjsDB(t)
	svc := services.NewYjsService(db)

	cutoff := time.Now().Add(-24 * time.Hour)
	mock.ExpectBegin()
	mock.ExpectExec(`DELETE FROM "yjs_updates"`).
		WillReturnResult(sqlmock.NewResult(10, 10))
	mock.ExpectCommit()

	if err := svc.DeleteOldUpdates(cutoff); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
