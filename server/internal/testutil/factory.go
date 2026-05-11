package testutil

import (
	"time"

	"github.com/google/uuid"
	internalauth "github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
)

type UserModel = models.User

type UserFixture struct {
	ID               uuid.UUID
	Name             string
	Email            string
	PasswordHash     string
	RefreshTokenHash string
	RefreshTokenExp  *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewUser(overrides ...func(*models.User)) models.User {
	u := models.User{
		ID:           uuid.New(),
		Name:         "Test User",
		Email:        "test@example.com",
		PasswordHash: mustHash("password123"),
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	for _, fn := range overrides {
		fn(&u)
	}
	return u
}

func NewDocument(ownerID uuid.UUID, overrides ...func(*models.Document)) models.Document {
	d := models.Document{
		ID:            uuid.New(),
		Title:         "Test Document",
		Content:       "Test content",
		ContentFormat: models.ContentFormatTipTap,
		UserID:        ownerID,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	for _, fn := range overrides {
		fn(&d)
	}
	return d
}

func NewCollaborator(documentID, userID uuid.UUID, perm models.PermissionType) models.DocumentCollaborator {
	return models.DocumentCollaborator{
		ID:         uuid.New(),
		DocumentID: documentID,
		UserID:     userID,
		Permission: perm,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func mustHash(password string) string {
	hash, err := internalauth.HashPassword(password)
	if err != nil {
		panic("testutil: failed to hash password: " + err.Error())
	}
	return hash
}
