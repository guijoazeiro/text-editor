package services

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"gorm.io/gorm"
)

type NotificationHub interface {
	SendToUser(userID uuid.UUID, msg models.WSMessage)
}

type NotificationService struct {
	db  *gorm.DB
	hub NotificationHub
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{db: db}
}

func NewNotificationServiceWithHub(db *gorm.DB, hub NotificationHub) *NotificationService {
	return &NotificationService{db: db, hub: hub}
}

func (s *NotificationService) push(n *models.Notification) {
	if s.hub == nil {
		return
	}
	payload, _ := json.Marshal(n)
	s.hub.SendToUser(n.UserID, models.WSMessage{
		Type: "notification:new",
		Data: map[string]interface{}{"notification": json.RawMessage(payload)},
	})
}

func (s *NotificationService) NotifyCollaboratorAdded(documentID, targetUserID, fromUserID uuid.UUID, permission models.PermissionType) error {
	var document models.Document
	if err := s.db.First(&document, documentID).Error; err != nil {
		return err
	}

	var fromUser models.User
	if err := s.db.First(&fromUser, fromUserID).Error; err != nil {
		return err
	}

	notification := models.Notification{
		UserID:     targetUserID,
		Type:       models.NotificationCollaboratorAdded,
		Title:      "You've been added as a collaborator",
		Message:    fmt.Sprintf("%s added you as %s to '%s'", fromUser.Name, permission, document.Title),
		DocumentID: &documentID,
		FromUserID: &fromUserID,
		Read:       false,
	}

	if err := s.db.Create(&notification).Error; err != nil {
		return err
	}
	s.push(&notification)
	return nil
}

func (s *NotificationService) NotifyDocumentEdited(documentID, editorUserID uuid.UUID) error {
	var document models.Document
	if err := s.db.First(&document, documentID).Error; err != nil {
		return err
	}

	if document.UserID == editorUserID {
		return nil
	}

	var editor models.User
	if err := s.db.First(&editor, editorUserID).Error; err != nil {
		return err
	}

	notification := models.Notification{
		UserID:     document.UserID,
		Type:       models.NotificationDocumentEdited,
		Title:      "Document edited",
		Message:    fmt.Sprintf("%s edited '%s'", editor.Name, document.Title),
		DocumentID: &documentID,
		FromUserID: &editorUserID,
		Read:       false,
	}

	if err := s.db.Create(&notification).Error; err != nil {
		return err
	}
	s.push(&notification)
	return nil
}

func (s *NotificationService) NotifyPermissionChanged(documentID, targetUserID, fromUserID uuid.UUID, newPermission models.PermissionType) error {
	var document models.Document
	if err := s.db.First(&document, documentID).Error; err != nil {
		return err
	}

	var fromUser models.User
	if err := s.db.First(&fromUser, fromUserID).Error; err != nil {
		return err
	}

	notification := models.Notification{
		UserID:     targetUserID,
		Type:       models.NotificationPermissionChanged,
		Title:      "Permission updated",
		Message:    fmt.Sprintf("%s changed your permission to %s on '%s'", fromUser.Name, newPermission, document.Title),
		DocumentID: &documentID,
		FromUserID: &fromUserID,
		Read:       false,
	}

	if err := s.db.Create(&notification).Error; err != nil {
		return err
	}
	s.push(&notification)
	return nil
}
