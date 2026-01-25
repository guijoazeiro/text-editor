package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"gorm.io/gorm"
)

type PermissionService struct {
	db *gorm.DB
}

func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

func (s *PermissionService) GetUserPermission(documentID, userID uuid.UUID) (models.PermissionType, error) {
	var document models.Document
	if err := s.db.First(&document, "id = ?", documentID).Error; err != nil {
		return "", err
	}

	if document.UserID == userID {
		return models.PermissionOwner, nil
	}

	var collaborator models.DocumentCollaborator
	if err := s.db.First(&collaborator, "document_id = ? AND user_id = ?", documentID, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", errors.New("no permission")
		}
		return "", err
	}

	return collaborator.Permission, nil
}

func (s *PermissionService) CanView(documentID, userID uuid.UUID) bool {
	permission, err := s.GetUserPermission(documentID, userID)
	if err != nil {
		return false
	}
	return permission == models.PermissionOwner || permission == models.PermissionEditor || permission == models.PermissionViewer
}

func (s *PermissionService) CanEdit(documentID, userID uuid.UUID) bool {
	permission, err := s.GetUserPermission(documentID, userID)
	if err != nil {
		return false
	}
	return permission == models.PermissionOwner || permission == models.PermissionEditor
}

func (s *PermissionService) IsOwner(documentID, userID uuid.UUID) bool {
	permission, err := s.GetUserPermission(documentID, userID)
	if err != nil {
		return false
	}
	return permission == models.PermissionOwner
}

func (s *PermissionService) GetAccessibleDocuments(userID uuid.UUID) ([]models.Document, error) {
	var documents []models.Document

	var ownedDocs []models.Document
	if err := s.db.Where("user_id = ?", userID).Find(&ownedDocs).Error; err != nil {
		return nil, err
	}
	documents = append(documents, ownedDocs...)

	var collaborations []models.DocumentCollaborator
	if err := s.db.Preload("User").Where("user_id = ?", userID).Find(&collaborations).Error; err != nil {
		return nil, err
	}

	for _, collab := range collaborations {
		var doc models.Document
		if err := s.db.Preload("User").First(&doc, collab.DocumentID).Error; err == nil {
			documents = append(documents, doc)
		}
	}

	return documents, nil
}
