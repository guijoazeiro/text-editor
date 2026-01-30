package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ActionType string

const (
	ActionCreated        ActionType = "created"
	ActionUpdated        ActionType = "updated"
	ActionTitleChanged   ActionType = "title_changed"
	ActionContentChanged ActionType = "content_changed"
)

type DocumentHistory struct {
	ID         uuid.UUID      `gorm:"type:uuid;primary_key" json:"id"`
	DocumentID uuid.UUID      `gorm:"type:uuid;not null" json:"document_id"`
	UserID     uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	User       User           `gorm:"foreignKey:UserID" json:"user"`
	Action     ActionType     `gorm:"type:action_type;not null" json:"action"`
	Changes    datatypes.JSON `gorm:"type:jsonb" json:"changes,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

func (dh *DocumentHistory) BeforeCreate(tx *gorm.DB) error {
	if dh.ID == uuid.Nil {
		dh.ID = uuid.New()
	}
	return nil
}

type HistoryChange struct {
	Field    string      `json:"field"`
	OldValue interface{} `json:"old_value"`
	NewValue interface{} `json:"new_value"`
}
