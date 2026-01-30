package services

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"gorm.io/gorm"
)

type HistoryService struct {
	db *gorm.DB
}

func NewHistoryService(db *gorm.DB) *HistoryService {
	return &HistoryService{db: db}
}

func (s *HistoryService) RecordCreation(documentID, userID uuid.UUID) error {
	history := models.DocumentHistory{
		DocumentID: documentID,
		UserID:     userID,
		Action:     models.ActionCreated,
	}
	return s.db.Create(&history).Error
}

func (s *HistoryService) RecordUpdate(documentID, userID uuid.UUID, oldDoc, newDoc *models.Document) error {
	var changes []models.HistoryChange

	if oldDoc.Title != newDoc.Title {
		changes = append(changes, models.HistoryChange{
			Field:    "title",
			OldValue: oldDoc.Title,
			NewValue: newDoc.Title,
		})
	}

	if oldDoc.Content != newDoc.Content {
		changes = append(changes, models.HistoryChange{
			Field:    "content",
			OldValue: truncateString(oldDoc.Content, 100),
			NewValue: truncateString(newDoc.Content, 100),
		})
	}

	if len(changes) == 0 {
		return nil
	}

	changesJSON, err := json.Marshal(changes)
	if err != nil {
		return err
	}

	history := models.DocumentHistory{
		DocumentID: documentID,
		UserID:     userID,
		Action:     models.ActionUpdated,
		Changes:    changesJSON,
	}

	return s.db.Create(&history).Error
}

func (s *HistoryService) GetDocumentHistory(documentID uuid.UUID, userID *uuid.UUID, action *models.ActionType, fromDate, toDate *string) ([]models.DocumentHistory, error) {
	query := s.db.Preload("User").Where("document_id = ?", documentID)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	if action != nil {
		query = query.Where("action = ?", *action)
	}

	if fromDate != nil {
		query = query.Where("created_at >= ?", *fromDate)
	}
	if toDate != nil {
		query = query.Where("created_at <= ?", *toDate)
	}

	var histories []models.DocumentHistory
	err := query.Order("created_at DESC").Find(&histories).Error
	return histories, err
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
