package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type DocumentVersion struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	DocumentID    uuid.UUID `gorm:"type:uuid;not null" json:"document_id"`
	VersionNumber int       `gorm:"not null" json:"version_number"`
	Title         string    `gorm:"type:varchar(255);not null" json:"title"`
	Content       string    `gorm:"type:text" json:"content"`
	YjsSnapshot   []byte    `gorm:"type:bytea" json:"yjs_snapshot,omitempty"`
	CreatedBy     uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	User          User      `gorm:"foreignKey:CreatedBy" json:"user"`
	CreatedAt     time.Time `json:"created_at"`
}

func (dv *DocumentVersion) BeforeCreate(tx *gorm.DB) error {
	if dv.ID == uuid.Nil {
		dv.ID = uuid.New()
	}
	return nil
}
