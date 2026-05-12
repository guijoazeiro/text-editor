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

func newVersionDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
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

var versionServiceCols = []string{
	"id", "document_id", "version_number", "title", "content",
	"yjs_snapshot", "created_by", "created_at",
}
var userVersionCols = []string{
	"id", "name", "email", "password_hash",
	"refresh_token_hash", "refresh_token_exp", "created_at", "updated_at",
}

func makeVersionRow(v models.DocumentVersion) *sqlmock.Rows {
	return sqlmock.NewRows(versionServiceCols).AddRow(
		v.ID, v.DocumentID, v.VersionNumber, v.Title, v.Content,
		v.YjsSnapshot, v.CreatedBy, v.CreatedAt,
	)
}

func TestVersionCreateVersion_Success(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)

	mock.ExpectQuery(`SELECT COALESCE`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(2))

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_versions"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.CreateVersion(doc.ID, user.ID, "My Title", "My Content"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCreateVersion_FirstVersion(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)

	mock.ExpectQuery(`SELECT COALESCE`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_versions"`).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	if err := svc.CreateVersion(doc.ID, user.ID, "Title", "Content"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVersionCreateVersion_InsertError(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	mock.ExpectQuery(`SELECT COALESCE`).
		WillReturnRows(sqlmock.NewRows([]string{"coalesce"}).AddRow(0))
	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO "document_versions"`).
		WillReturnError(gorm.ErrInvalidDB)
	mock.ExpectRollback()

	err := svc.CreateVersion(uuid.New(), uuid.New(), "T", "C")
	if err == nil {
		t.Fatal("expected error on INSERT failure")
	}
}

func TestVersionGetVersionsPaginated_Success(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)
	ver := models.DocumentVersion{
		ID:            uuid.New(),
		DocumentID:    doc.ID,
		VersionNumber: 1,
		Title:         "v1",
		Content:       "content",
		CreatedBy:     user.ID,
		CreatedAt:     time.Now(),
	}

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).
		WillReturnRows(makeVersionRow(ver))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userVersionCols).AddRow(
			user.ID, user.Name, user.Email, user.PasswordHash, "", nil, user.CreatedAt, user.UpdatedAt,
		))

	versions, total, err := svc.GetVersionsPaginated(doc.ID, 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1, got %d", total)
	}
	if len(versions) != 1 {
		t.Errorf("expected 1 version, got %d", len(versions))
	}
}

func TestVersionGetVersionsPaginated_EmptyResult(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	mock.ExpectQuery(`SELECT count`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).
		WillReturnRows(sqlmock.NewRows(versionServiceCols))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userVersionCols))

	versions, total, err := svc.GetVersionsPaginated(uuid.New(), 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions, got %d", len(versions))
	}
}

func TestVersionGetVersionsPaginated_CountError(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	mock.ExpectQuery(`SELECT count`).
		WillReturnError(gorm.ErrInvalidDB)

	_, _, err := svc.GetVersionsPaginated(uuid.New(), 20, 0)
	if err == nil {
		t.Fatal("expected error on COUNT failure")
	}
}

func TestVersionGetVersion_Found(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)
	ver := models.DocumentVersion{
		ID:            uuid.New(),
		DocumentID:    doc.ID,
		VersionNumber: 3,
		Title:         "v3",
		Content:       "content v3",
		CreatedBy:     user.ID,
		CreatedAt:     time.Now(),
	}

	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).
		WillReturnRows(makeVersionRow(ver))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userVersionCols).AddRow(
			user.ID, user.Name, user.Email, user.PasswordHash, "", nil, user.CreatedAt, user.UpdatedAt,
		))

	result, err := svc.GetVersion(doc.ID, 3)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.VersionNumber != 3 {
		t.Errorf("expected version_number=3, got %d", result.VersionNumber)
	}
	if result.Title != "v3" {
		t.Errorf("expected title='v3', got %q", result.Title)
	}
}

func TestVersionGetVersion_NotFound(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := svc.GetVersion(uuid.New(), 99)
	if err == nil {
		t.Fatal("expected error when version not found")
	}
}

func TestVersionSnapshotBase64_NonEmpty(t *testing.T) {
	input := []byte("hello")
	result := services.SnapshotBase64(input)
	if result == "" {
		t.Error("expected non-empty base64 string")
	}

	if result != "aGVsbG8=" {
		t.Errorf("unexpected base64: %q", result)
	}
}

func TestVersionSnapshotBase64_Empty(t *testing.T) {
	result := services.SnapshotBase64(nil)
	if result != "" {
		t.Errorf("expected empty string for nil snapshot, got %q", result)
	}

	result2 := services.SnapshotBase64([]byte{})
	if result2 != "" {
		t.Errorf("expected empty string for empty snapshot, got %q", result2)
	}
}

func TestVersionCompareVersions_DifferentContent(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)
	now := time.Now()

	v1 := models.DocumentVersion{
		ID: uuid.New(), DocumentID: doc.ID, VersionNumber: 1,
		Title: "Same Title", Content: "old content", CreatedBy: user.ID, CreatedAt: now,
	}
	v2 := models.DocumentVersion{
		ID: uuid.New(), DocumentID: doc.ID, VersionNumber: 2,
		Title: "Same Title", Content: "new content", CreatedBy: user.ID, CreatedAt: now,
	}

	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).WillReturnRows(makeVersionRow(v1))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userVersionCols).AddRow(
			user.ID, user.Name, user.Email, user.PasswordHash, "", nil, user.CreatedAt, user.UpdatedAt,
		))

	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).WillReturnRows(makeVersionRow(v2))
	mock.ExpectQuery(`SELECT .+ FROM "users"`).
		WillReturnRows(sqlmock.NewRows(userVersionCols).AddRow(
			user.ID, user.Name, user.Email, user.PasswordHash, "", nil, user.CreatedAt, user.UpdatedAt,
		))

	diff, err := svc.CompareVersions(doc.ID, 1, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	changes := diff["changes"].(map[string]interface{})
	if changes["content_changed"] != true {
		t.Error("expected content_changed=true")
	}
	if changes["title_changed"] != false {
		t.Error("expected title_changed=false")
	}
}

func TestVersionCompareVersions_SameContent(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	user := testutil.NewUser()
	doc := testutil.NewDocument(user.ID)
	now := time.Now()

	ver := models.DocumentVersion{
		ID: uuid.New(), DocumentID: doc.ID, VersionNumber: 1,
		Title: "Title", Content: "same", CreatedBy: user.ID, CreatedAt: now,
	}

	for i := 0; i < 2; i++ {
		mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).WillReturnRows(makeVersionRow(ver))
		mock.ExpectQuery(`SELECT .+ FROM "users"`).
			WillReturnRows(sqlmock.NewRows(userVersionCols).AddRow(
				user.ID, user.Name, user.Email, user.PasswordHash, "", nil, user.CreatedAt, user.UpdatedAt,
			))
	}

	diff, err := svc.CompareVersions(doc.ID, 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	changes := diff["changes"].(map[string]interface{})
	if changes["content_changed"] != false {
		t.Error("expected content_changed=false for identical versions")
	}
	if changes["title_changed"] != false {
		t.Error("expected title_changed=false for identical versions")
	}
}

func TestVersionCompareVersions_V1NotFound(t *testing.T) {
	db, mock := newVersionDB(t)
	svc := services.NewVersionService(db, nil)

	mock.ExpectQuery(`SELECT .+ FROM "document_versions"`).
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := svc.CompareVersions(uuid.New(), 1, 2)
	if err == nil {
		t.Fatal("expected error when v1 not found")
	}
}
