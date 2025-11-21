package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Document struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	Title     string    `gorm:"not null" json:"title" binding:"required"`
	Content   string    `gorm:"type:text" json:"content"`
	UserID    uuid.UUID `gorm:"type:uuid;not null" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d *Document) BeforeCreate(tx *gorm.DB) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	return nil
}

type CreateDocumentRequest struct {
	Title   string `json:"title" binding:"required"`
	Content string `json:"content"`
}

type UpdateDocumentRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}
