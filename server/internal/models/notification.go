package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationType string

const (
	NotificationCollaboratorAdded NotificationType = "collaborator_added"
	NotificationDocumentShared    NotificationType = "document_shared"
	NotificationDocumentEdited    NotificationType = "document_edited"
	NotificationPermissionChanged NotificationType = "permission_changed"
)

type Notification struct {
	ID         uuid.UUID        `gorm:"type:uuid;primary_key" json:"id"`
	UserID     uuid.UUID        `gorm:"type:uuid;not null" json:"user_id"`
	Type       NotificationType `gorm:"type:notification_type;not null" json:"type"`
	Title      string           `gorm:"type:varchar(255);not null" json:"title"`
	Message    string           `gorm:"type:text;not null" json:"message"`
	DocumentID *uuid.UUID       `gorm:"type:uuid" json:"document_id,omitempty"`
	FromUserID *uuid.UUID       `gorm:"type:uuid" json:"from_user_id,omitempty"`
	FromUser   *User            `gorm:"foreignKey:FromUserID" json:"from_user,omitempty"`
	Document   *Document        `gorm:"foreignKey:DocumentID" json:"document,omitempty"`
	Read       bool             `gorm:"default:false" json:"read"`
	CreatedAt  time.Time        `json:"created_at"`
}

func (n *Notification) BeforeCreate(tx *gorm.DB) error {
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	return nil
}
