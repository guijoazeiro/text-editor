package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type YjsSnapshot struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null;uniqueIndex"                  json:"document_id"`

	Snapshot []byte `gorm:"type:bytea;not null" json:"snapshot"`

	LamportTS int64 `gorm:"column:lamport_ts;not null;default:0" json:"lamport_ts"`

	CreatedAt time.Time `gorm:"not null;default:current_timestamp" json:"created_at"`

	Document Document `gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE" json:"-"`
}

func (YjsSnapshot) TableName() string {
	return "yjs_snapshots"
}

func (s *YjsSnapshot) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
