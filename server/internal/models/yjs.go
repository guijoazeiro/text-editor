package models

import (
	"time"

	"github.com/google/uuid"
)

type YjsUpdate struct {
	ID         uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	DocumentID uuid.UUID `gorm:"type:uuid;not null;index"                         json:"document_id"`
	Update     []byte    `gorm:"type:bytea;not null"                              json:"update"`

	LamportTS int64 `gorm:"column:lamport_ts;not null;default:0;index" json:"lamport_ts"`

	ClientID int64 `gorm:"column:client_id;not null;default:0" json:"client_id"`

	CreatedAt time.Time `gorm:"not null;default:current_timestamp" json:"created_at"`

	Document Document `gorm:"foreignKey:DocumentID;constraint:OnDelete:CASCADE" json:"-"`
}

func (YjsUpdate) TableName() string {
	return "yjs_updates"
}
