package services_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newPermDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

var docCols = []string{"id", "title", "content", "content_format", "user_id", "created_at", "updated_at", "deleted_at"}
var collabCols = []string{"id", "document_id", "user_id", "permission", "created_at", "updated_at"}

func TestGetUserPermission_Owner(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))

	perm, err := svc.GetUserPermission(doc.ID, owner.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if perm != models.PermissionOwner {
		t.Errorf("expected %q, got %q", models.PermissionOwner, perm)
	}
}

func TestGetUserPermission_Collaborator_Editor(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	editor := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)
	collab := testutil.NewCollaborator(doc.ID, editor.ID, models.PermissionEditor)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, editor.ID, models.PermissionEditor,
			collab.CreatedAt, collab.UpdatedAt,
		))

	perm, err := svc.GetUserPermission(doc.ID, editor.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if perm != models.PermissionEditor {
		t.Errorf("expected %q, got %q", models.PermissionEditor, perm)
	}
}

func TestGetUserPermission_Collaborator_Viewer(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	viewer := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)
	collab := testutil.NewCollaborator(doc.ID, viewer.ID, models.PermissionViewer)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, viewer.ID, models.PermissionViewer,
			collab.CreatedAt, collab.UpdatedAt,
		))

	perm, err := svc.GetUserPermission(doc.ID, viewer.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if perm != models.PermissionViewer {
		t.Errorf("expected %q, got %q", models.PermissionViewer, perm)
	}
}

func TestGetUserPermission_NoAccess(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	stranger := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := svc.GetUserPermission(doc.ID, stranger.ID)
	if err == nil {
		t.Fatal("expected error for user with no permission")
	}
}

func TestGetUserPermission_DocumentNotFound(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := svc.GetUserPermission(uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected error when document doesn't exist")
	}
}

func TestCanView_Owner(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))

	if !svc.CanView(doc.ID, owner.ID) {
		t.Error("owner must be able to view their own document")
	}
}

func TestCanView_Viewer(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	viewer := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)
	collab := testutil.NewCollaborator(doc.ID, viewer.ID, models.PermissionViewer)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, viewer.ID, models.PermissionViewer,
			collab.CreatedAt, collab.UpdatedAt,
		))

	if !svc.CanView(doc.ID, viewer.ID) {
		t.Error("viewer collaborator must be able to view document")
	}
}

func TestCanView_NoAccess(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	stranger := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnError(gorm.ErrRecordNotFound)

	if svc.CanView(doc.ID, stranger.ID) {
		t.Error("stranger must not be able to view document")
	}
}

func TestCanEdit_Owner(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))

	if !svc.CanEdit(doc.ID, owner.ID) {
		t.Error("owner must be able to edit")
	}
}

func TestCanEdit_Editor(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	editor := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)
	collab := testutil.NewCollaborator(doc.ID, editor.ID, models.PermissionEditor)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, editor.ID, models.PermissionEditor,
			collab.CreatedAt, collab.UpdatedAt,
		))

	if !svc.CanEdit(doc.ID, editor.ID) {
		t.Error("editor collaborator must be able to edit")
	}
}

func TestCanEdit_ViewerCannotEdit(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	viewer := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)
	collab := testutil.NewCollaborator(doc.ID, viewer.ID, models.PermissionViewer)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, viewer.ID, models.PermissionViewer,
			collab.CreatedAt, collab.UpdatedAt,
		))

	if svc.CanEdit(doc.ID, viewer.ID) {
		t.Error("viewer must NOT be able to edit")
	}
}

func TestIsOwner_True(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))

	if !svc.IsOwner(doc.ID, owner.ID) {
		t.Error("expected IsOwner=true for actual owner")
	}
}

func TestIsOwner_False_Editor(t *testing.T) {
	db, mock := newPermDB(t)
	svc := services.NewPermissionService(db)

	owner := testutil.NewUser()
	editor := testutil.NewUser()
	doc := testutil.NewDocument(owner.ID)
	collab := testutil.NewCollaborator(doc.ID, editor.ID, models.PermissionEditor)

	mock.ExpectQuery(`SELECT .+ FROM "documents"`).
		WillReturnRows(sqlmock.NewRows(docCols).AddRow(
			doc.ID, doc.Title, doc.Content, doc.ContentFormat, owner.ID,
			doc.CreatedAt, doc.UpdatedAt, nil,
		))
	mock.ExpectQuery(`SELECT .+ FROM "document_collaborators"`).
		WillReturnRows(sqlmock.NewRows(collabCols).AddRow(
			collab.ID, doc.ID, editor.ID, models.PermissionEditor,
			collab.CreatedAt, collab.UpdatedAt,
		))

	if svc.IsOwner(doc.ID, editor.ID) {
		t.Error("editor must NOT be owner")
	}
}
