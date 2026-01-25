package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PermissionType string

const (
	PermissionOwner  PermissionType = "owner"
	PermissionEditor PermissionType = "editor"
	PermissionViewer PermissionType = "viewer"
)

type DocumentCollaborator struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	DocumentID uuid.UUID      `gorm:"type:uuid;not null" json:"document_id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	Permission PermissionType `gorm:"type:permission_type;not null;default:'viewer'" json:"permission"`
	User       User           `gorm:"foreignKey:UserID" json:"user"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

func (dc *DocumentCollaborator) BeforeCreate(tx *gorm.DB) error {
	if dc.ID == uuid.Nil {
		dc.ID = uuid.New()
	}
	return nil
}

type DocumentShareLink struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	DocumentID uuid.UUID      `gorm:"type:uuid;not null;uniqueIndex" json:"document_id"`
	Token      string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"token"`
	Permission PermissionType `gorm:"type:permission_type;not null;default:'viewer'" json:"permission"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty"`
	CreatedBy  uuid.UUID      `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt  time.Time      `json:"created_at"`
}

func (dsl *DocumentShareLink) BeforeCreate(tx *gorm.DB) error {
	if dsl.ID == uuid.Nil {
		dsl.ID = uuid.New()
	}
	if dsl.Token == "" {
		dsl.Token = uuid.New().String()
	}
	return nil
}

func (dsl *DocumentShareLink) IsExpired() bool {
	if dsl.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*dsl.ExpiresAt)
}

type AddCollaboratorRequest struct {
	Email      string         `json:"email" binding:"required,email"`
	Permission PermissionType `json:"permission" binding:"required,oneof=viewer editor"`
}

type UpdateCollaboratorRequest struct {
	Permission PermissionType `json:"permission" binding:"required,oneof=viewer editor owner"`
}

type CreateShareLinkRequest struct {
	Permission PermissionType `json:"permission" binding:"required,oneof=viewer editor"`
	ExpiresIn  *int           `json:"expires_in"`
}
