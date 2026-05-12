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

func newNotifDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

type mockHub struct {
	sent []models.WSMessage
}

func (m *mockHub) SendToUser(_ uuid.UUID, msg models.WSMessage) {
	m.sent = append(m.sent, msg)
}

var notifDocCols = []string{"id", "title", "content", "content_format", "user_id", "created_at", "updated_at", "deleted_at"}
var notifUserCols = []string{"id", "name", "email", "password_hash", "refresh_token_hash", "refresh_token_exp", "created_at", "updated_at"}
var insertNotifCols = []string{"id", "created_at"}

func TestNotifService_NotifyCollaboratorAdded_Success(t *testing.T) {
	db, mock := newNotifDB(t)
	hub := &mockHub{}
	svc := services.NewNotificationServiceWithHub(db, hub)

	owner := testutil.NewUser()
	target := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(notifDocCols).AddRow(
			doc.ID, doc.Title, doc.Content, models.ContentFormatTipTap,
			owner.ID, time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(notifUserCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		))

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "notifications"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := svc.NotifyCollaboratorAdded(doc.ID, target.ID, owner.ID, models.PermissionEditor)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hub.sent) != 1 {
		t.Errorf("expected 1 WS message pushed, got %d", len(hub.sent))
	}
}

func TestNotifService_NotifyCollaboratorAdded_DocumentNotFound(t *testing.T) {
	db, mock := newNotifDB(t)
	svc := services.NewNotificationService(db)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnError(gorm.ErrRecordNotFound)

	err := svc.NotifyCollaboratorAdded(uuid.New(), uuid.New(), uuid.New(), models.PermissionViewer)
	if err == nil {
		t.Fatal("expected error when document not found")
	}
}

func TestNotifService_NotifyCollaboratorAdded_NoHub_DoesNotPanic(t *testing.T) {
	db, mock := newNotifDB(t)
	svc := services.NewNotificationService(db)

	owner := testutil.NewUser()
	target := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(notifDocCols).AddRow(
			doc.ID, doc.Title, doc.Content, models.ContentFormatTipTap,
			owner.ID, time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(notifUserCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "notifications"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.NotifyCollaboratorAdded(doc.ID, target.ID, owner.ID, models.PermissionViewer); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNotifService_NotifyDocumentEdited_SkipsWhenEditorIsOwner(t *testing.T) {
	db, mock := newNotifDB(t)
	svc := services.NewNotificationService(db)

	owner := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(notifDocCols).AddRow(
			doc.ID, doc.Title, doc.Content, models.ContentFormatTipTap,
			owner.ID, time.Now(), time.Now(), nil,
		))

	err := svc.NotifyDocumentEdited(doc.ID, owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unexpected DB calls: %v", err)
	}
}

func TestNotifService_NotifyDocumentEdited_Success(t *testing.T) {
	db, mock := newNotifDB(t)
	hub := &mockHub{}
	svc := services.NewNotificationServiceWithHub(db, hub)

	owner := testutil.NewUser()
	editor := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(notifDocCols).AddRow(
			doc.ID, doc.Title, doc.Content, models.ContentFormatTipTap,
			owner.ID, time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(notifUserCols).AddRow(
			editor.ID, editor.Name, editor.Email, editor.PasswordHash, "", nil, editor.CreatedAt, editor.UpdatedAt,
		))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "notifications"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := svc.NotifyDocumentEdited(doc.ID, editor.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hub.sent) != 1 {
		t.Errorf("expected 1 WS push, got %d", len(hub.sent))
	}
}

func TestNotifService_NotifyDocumentEdited_DocumentNotFound(t *testing.T) {
	db, mock := newNotifDB(t)
	svc := services.NewNotificationService(db)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnError(gorm.ErrRecordNotFound)

	err := svc.NotifyDocumentEdited(uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error when document not found")
	}
}

func TestNotifService_NotifyPermissionChanged_Success(t *testing.T) {
	db, mock := newNotifDB(t)
	hub := &mockHub{}
	svc := services.NewNotificationServiceWithHub(db, hub)

	owner := testutil.NewUser()
	target := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(notifDocCols).AddRow(
			doc.ID, doc.Title, doc.Content, models.ContentFormatTipTap,
			owner.ID, time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(notifUserCols).AddRow(
			owner.ID, owner.Name, owner.Email, owner.PasswordHash, "", nil, owner.CreatedAt, owner.UpdatedAt,
		))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "notifications"`).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := svc.NotifyPermissionChanged(doc.ID, target.ID, owner.ID, models.PermissionViewer)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hub.sent) != 1 {
		t.Errorf("expected 1 WS push, got %d", len(hub.sent))
	}
}

func TestNotifService_NotifyPermissionChanged_FromUserNotFound(t *testing.T) {
	db, mock := newNotifDB(t)
	svc := services.NewNotificationService(db)

	owner := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(notifDocCols).AddRow(
			doc.ID, doc.Title, doc.Content, models.ContentFormatTipTap,
			owner.ID, time.Now(), time.Now(), nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnError(gorm.ErrRecordNotFound)

	err := svc.NotifyPermissionChanged(doc.ID, uuid.New(), uuid.New(), models.PermissionEditor)
	if err == nil {
		t.Fatal("expected error when fromUser not found")
	}
}
