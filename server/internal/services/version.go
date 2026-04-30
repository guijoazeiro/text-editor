package services

import (
	"encoding/base64"
	"errors"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"gorm.io/gorm"
)

type VersionService struct {
	db              *gorm.DB
	snapshotService *SnapshotService
}

func NewVersionService(db *gorm.DB, snapshotService *SnapshotService) *VersionService {
	return &VersionService{db: db, snapshotService: snapshotService}
}

func (s *VersionService) CreateVersion(documentID, userID uuid.UUID, title, content string) error {
	var maxVersion int
	s.db.Model(&models.DocumentVersion{}).
		Where("document_id = ?", documentID).
		Select("COALESCE(MAX(version_number), 0)").
		Scan(&maxVersion)

	var yjsSnapshot []byte
	if s.snapshotService != nil {
		if snap, err := s.snapshotService.GetSnapshot(documentID); err == nil && snap != nil {
			yjsSnapshot = snap.Snapshot
		}
	}

	version := models.DocumentVersion{
		DocumentID:    documentID,
		VersionNumber: maxVersion + 1,
		Title:         title,
		Content:       content,
		YjsSnapshot:   yjsSnapshot,
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

func (s *VersionService) GetVersionsPaginated(documentID uuid.UUID, limit, offset int) ([]models.DocumentVersion, int64, error) {
	var versions []models.DocumentVersion
	var total int64

	if err := s.db.Model(&models.DocumentVersion{}).
		Where("document_id = ?", documentID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := s.db.Preload("User").
		Where("document_id = ?", documentID).
		Order("version_number DESC").
		Limit(limit).Offset(offset).
		Find(&versions).Error

	return versions, total, err
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

type RestoreResult struct {
	Document    *models.Document
	YjsSnapshot []byte
}

func (s *VersionService) RestoreVersion(documentID uuid.UUID, versionNumber int, userID uuid.UUID) (*RestoreResult, error) {
	version, err := s.GetVersion(documentID, versionNumber)
	if err != nil {
		return nil, errors.New("version not found")
	}

	var document models.Document
	if err := s.db.First(&document, documentID).Error; err != nil {
		return nil, err
	}

	_ = s.CreateVersion(documentID, userID, document.Title, document.Content)

	document.Title = version.Title
	document.Content = version.Content

	if err := s.db.Save(&document).Error; err != nil {
		return nil, err
	}

	s.db.Preload("User").First(&document, documentID)

	result := &RestoreResult{Document: &document}

	if s.snapshotService != nil {
		_ = s.snapshotService.ClearDocumentCRDTState(documentID)
	}

	return result, nil
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
			"number":       v1.VersionNumber,
			"title":        v1.Title,
			"content":      v1.Content,
			"created_by":   v1.User.Name,
			"created_at":   v1.CreatedAt,
			"has_snapshot": len(v1.YjsSnapshot) > 0,
		},
		"version_2": map[string]interface{}{
			"number":       v2.VersionNumber,
			"title":        v2.Title,
			"content":      v2.Content,
			"created_by":   v2.User.Name,
			"created_at":   v2.CreatedAt,
			"has_snapshot": len(v2.YjsSnapshot) > 0,
		},
		"changes": map[string]interface{}{
			"title_changed":   v1.Title != v2.Title,
			"content_changed": v1.Content != v2.Content,
		},
	}

	return diff, nil
}

func SnapshotBase64(snapshot []byte) string {
	if len(snapshot) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(snapshot)
}
