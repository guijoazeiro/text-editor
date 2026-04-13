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

func (s *YjsService) SaveUpdate(documentID uuid.UUID, update []byte) error {
	yjsUpdate := models.YjsUpdate{
		DocumentID: documentID,
		Update:     update,
		Clock:      time.Now().UnixNano(),
	}

	return s.db.Create(&yjsUpdate).Error
}

func (s *YjsService) GetUpdates(documentID uuid.UUID) ([]models.YjsUpdate, error) {
	var updates []models.YjsUpdate
	err := s.db.
		Where("document_id = ?", documentID).
		Order("clock ASC").
		Find(&updates).Error

	return updates, err
}

func (s *YjsService) GetUpdatesSince(documentID uuid.UUID, sinceClock int64) ([]models.YjsUpdate, error) {
	var updates []models.YjsUpdate
	err := s.db.
		Where("document_id = ? AND clock > ?", documentID, sinceClock).
		Order("clock ASC").
		Find(&updates).Error

	return updates, err
}

func (s *YjsService) CompactUpdates(documentID uuid.UUID, beforeClock int64, mergedUpdate []byte) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.
			Where("document_id = ? AND clock < ?", documentID, beforeClock).
			Delete(&models.YjsUpdate{}).Error; err != nil {
			return err
		}

		compactedUpdate := models.YjsUpdate{
			DocumentID: documentID,
			Update:     mergedUpdate,
			Clock:      beforeClock,
		}

		return tx.Create(&compactedUpdate).Error
	})
}

func (s *YjsService) DeleteOldUpdates(olderThan time.Time) error {
	return s.db.
		Where("created_at < ?", olderThan).
		Delete(&models.YjsUpdate{}).Error
}

func (s *YjsService) GetDocumentUpdateCount(documentID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.
		Model(&models.YjsUpdate{}).
		Where("document_id = ?", documentID).
		Count(&count).Error

	return count, err
}
