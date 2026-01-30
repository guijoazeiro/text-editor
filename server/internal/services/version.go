package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"gorm.io/gorm"
)

type VersionService struct {
	db *gorm.DB
}

func NewVersionService(db *gorm.DB) *VersionService {
	return &VersionService{db: db}
}

func (s *VersionService) CreateVersion(documentID, userID uuid.UUID, title, content string) error {
	var maxVersion int
	s.db.Model(&models.DocumentVersion{}).
		Where("document_id = ?", documentID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion)

	version := models.DocumentVersion{
		DocumentID:    documentID,
		VersionNumber: maxVersion + 1,
		Title:         title,
		Content:       content,
		CreatedBy:     userID,
	}

	return s.db.Create(&version).Error
}

func (s *VersionService) GetVersions(documentID uuid.UUID) ([]models.DocumentVersion, error) {
	var versions []models.DocumentVersion
	err := s.db.Preload("User").
		Where("document_id = ?", documentID).
		Order("version_number DESC").
		Find(&versions).Error
	return versions, err
}

func (s *VersionService) GetVersion(documentID uuid.UUID, versionNumber int) (*models.DocumentVersion, error) {
	var version models.DocumentVersion
	err := s.db.Preload("User").
		Where("document_id = ? AND version_number = ?", documentID, versionNumber).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (s *VersionService) RestoreVersion(documentID uuid.UUID, versionNumber int, userID uuid.UUID) (*models.Document, error) {
	version, err := s.GetVersion(documentID, versionNumber)
	if err != nil {
		return nil, errors.New("version not found")
	}

	var document models.Document
	if err := s.db.First(&document, documentID).Error; err != nil {
		return nil, err
	}

	s.CreateVersion(documentID, userID, document.Title, document.Content)

	document.Title = version.Title
	document.Content = version.Content

	if err := s.db.Save(&document).Error; err != nil {
		return nil, err
	}

	s.db.Preload("User").First(&document, documentID)

	return &document, nil
}

func (s *VersionService) CompareVersions(documentID uuid.UUID, version1, version2 int) (map[string]interface{}, error) {
	v1, err := s.GetVersion(documentID, version1)
	if err != nil {
		return nil, errors.New("version 1 not found")
	}

	v2, err := s.GetVersion(documentID, version2)
	if err != nil {
		return nil, errors.New("version 2 not found")
	}

	diff := map[string]interface{}{
		"version_1": map[string]interface{}{
			"number":     v1.VersionNumber,
			"title":      v1.Title,
			"content":    v1.Content,
			"created_by": v1.User.Name,
			"created_at": v1.CreatedAt,
		},
		"version_2": map[string]interface{}{
			"number":     v2.VersionNumber,
			"title":      v2.Title,
			"content":    v2.Content,
			"created_by": v2.User.Name,
			"created_at": v2.CreatedAt,
		},
		"changes": map[string]interface{}{
			"title_changed":   v1.Title != v2.Title,
			"content_changed": v1.Content != v2.Content,
		},
	}

	return diff, nil
}
