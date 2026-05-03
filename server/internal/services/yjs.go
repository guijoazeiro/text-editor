package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"gorm.io/gorm"
)

type YjsService struct {
	db *gorm.DB
}

func NewYjsService(db *gorm.DB) *YjsService {
	return &YjsService{db: db}
}

func (s *YjsService) SaveUpdate(documentID uuid.UUID, update []byte, lamportTS int64, clientID int64) error {
	yjsUpdate := models.YjsUpdate{
		DocumentID: documentID,
		Update:     update,
		LamportTS:  lamportTS,
		ClientID:   clientID,
	}

	return s.db.Create(&yjsUpdate).Error
}

func (s *YjsService) GetUpdates(documentID uuid.UUID) ([]models.YjsUpdate, error) {
	var updates []models.YjsUpdate
	err := s.db.
		Where("document_id = ?", documentID).
		Order("lamport_ts ASC").
		Find(&updates).Error

	return updates, err
}

func (s *YjsService) GetUpdatesSince(documentID uuid.UUID, sinceLamport int64) ([]models.YjsUpdate, error) {
	var updates []models.YjsUpdate
	err := s.db.
		Where("document_id = ? AND lamport_ts > ?", documentID, sinceLamport).
		Order("lamport_ts ASC").
		Find(&updates).Error

	return updates, err
}

func (s *YjsService) GetDocumentUpdateCount(documentID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.
		Model(&models.YjsUpdate{}).
		Where("document_id = ?", documentID).
		Count(&count).Error

	return count, err
}

func (s *YjsService) GetDocumentUpdateSize(documentID uuid.UUID) (int64, error) {
	var totalBytes int64
	err := s.db.
		Model(&models.YjsUpdate{}).
		Where("document_id = ?", documentID).
		Select("COALESCE(SUM(octet_length(update)), 0)").
		Scan(&totalBytes).Error

	return totalBytes, err
}

func (s *YjsService) CompactUpdates(documentID uuid.UUID, beforeLamport int64, mergedUpdate []byte) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("document_id = ? AND lamport_ts < ?", documentID, beforeLamport).
			Delete(&models.YjsUpdate{}).Error; err != nil {
			return err
		}

		compacted := models.YjsUpdate{
			DocumentID: documentID,
			Update:     mergedUpdate,
			LamportTS:  beforeLamport,
			ClientID:   0,
		}

		return tx.Create(&compacted).Error
	})
}

func (s *YjsService) DeleteOldUpdates(olderThan time.Time) error {
	return s.db.
		Where("created_at < ?", olderThan).
		Delete(&models.YjsUpdate{}).Error
}

func (s *YjsService) DeleteUpdatesForDocument(documentID uuid.UUID) error {
	return s.db.
		Where("document_id = ?", documentID).
		Delete(&models.YjsUpdate{}).Error
}
