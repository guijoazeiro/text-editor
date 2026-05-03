package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type CollaboratorHandler struct {
	db                  *gorm.DB
	permissionService   *services.PermissionService
	notificationService *services.NotificationService
}

func NewCollaboratorHandler(db *gorm.DB, hub services.NotificationHub) *CollaboratorHandler {
	return &CollaboratorHandler{
		db:                  db,
		permissionService:   services.NewPermissionService(db),
		notificationService: services.NewNotificationServiceWithHub(db, hub),
	}
}

func (h *CollaboratorHandler) AddCollaborator(c *gin.Context) {
	userID, _ := c.Get("user_id")
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
			return
		}
		userUUID = parsed
	}

	if !h.permissionService.IsOwner(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Only owner can add collaborators", nil)
		return
	}

	var req models.AddCollaboratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var targetUser models.User
	if err := h.db.Where("email = ?", req.Email).First(&targetUser).Error; err != nil {
		response.Error(c, http.StatusNotFound, "User not found", err)
		return
	}

	var existing models.DocumentCollaborator
	if err := h.db.First(&existing, "document_id = ? AND user_id = ?", documentID, targetUser.ID).Error; err == nil {
		response.Error(c, http.StatusConflict, "User is already a collaborator", nil)
		return
	}

	collaborator := models.DocumentCollaborator{
		DocumentID: documentID,
		UserID:     targetUser.ID,
		Permission: req.Permission,
	}

	if err := h.db.Create(&collaborator).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to add collaborator", err)
		return
	}

	h.notificationService.NotifyCollaboratorAdded(documentID, targetUser.ID, userUUID, req.Permission)

	h.db.Preload("User").First(&collaborator, collaborator.ID)

	response.Success(c, http.StatusCreated, "Collaborator added successfully", collaborator)
}

func (h *CollaboratorHandler) ListCollaborators(c *gin.Context) {
	userID, _ := c.Get("user_id")
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
			return
		}
		userUUID = parsed
	}

	if !h.permissionService.CanView(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	var collaborators []models.DocumentCollaborator
	if err := h.db.Preload("User").Where("document_id = ?", documentID).Find(&collaborators).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch collaborators", err)
		return
	}

	response.Success(c, http.StatusOK, "Collaborators fetched successfully", collaborators)
}

func (h *CollaboratorHandler) UpdateCollaborator(c *gin.Context) {
	userID, _ := c.Get("user_id")
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
			return
		}
		userUUID = parsed
	}

	if !h.permissionService.IsOwner(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Only owner can update permissions", nil)
		return
	}

	var req models.UpdateCollaboratorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var collaborator models.DocumentCollaborator
	if err := h.db.First(&collaborator, "document_id = ? AND user_id = ?", documentID, targetUserID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Collaborator not found", err)
		return
	}

	collaborator.Permission = req.Permission
	if err := h.db.Save(&collaborator).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update permission", err)
		return
	}

	h.notificationService.NotifyPermissionChanged(documentID, targetUserID, userUUID, req.Permission)

	h.db.Preload("User").First(&collaborator, collaborator.ID)
	response.Success(c, http.StatusOK, "Permission updated successfully", collaborator)
}

func (h *CollaboratorHandler) RemoveCollaborator(c *gin.Context) {
	userID, _ := c.Get("user_id")
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	targetUserID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
			return
		}
		userUUID = parsed
	}

	if !h.permissionService.IsOwner(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Only owner can remove collaborators", nil)
		return
	}

	result := h.db.Delete(&models.DocumentCollaborator{}, "document_id = ? AND user_id = ?", documentID, targetUserID)
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to remove collaborator", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Collaborator not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Collaborator removed successfully", nil)
}

func (h *CollaboratorHandler) CreateShareLink(c *gin.Context) {
	userID, _ := c.Get("user_id")
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
			return
		}
		userUUID = parsed
	}

	if !h.permissionService.IsOwner(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Only owner can create share links", nil)
		return
	}

	var req models.CreateShareLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	h.db.Delete(&models.DocumentShareLink{}, "document_id = ?", documentID)

	shareLink := models.DocumentShareLink{
		DocumentID: documentID,
		Permission: req.Permission,
		CreatedBy:  userUUID,
	}

	if req.ExpiresIn != nil && *req.ExpiresIn > 0 {
		expiresAt := time.Now().Add(time.Duration(*req.ExpiresIn) * time.Hour)
		shareLink.ExpiresAt = &expiresAt
	}

	if err := h.db.Create(&shareLink).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create share link", err)
		return
	}

	response.Success(c, http.StatusCreated, "Share link created successfully", shareLink)
}

func (h *CollaboratorHandler) GetByShareLink(c *gin.Context) {
	token := c.Param("token")

	var shareLink models.DocumentShareLink
	if err := h.db.First(&shareLink, "token = ?", token).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Share link not found", err)
		return
	}

	if shareLink.IsExpired() {
		response.Error(c, http.StatusGone, "Share link has expired", nil)
		return
	}

	var document models.Document
	if err := h.db.Preload("User").First(&document, shareLink.DocumentID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Document not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Document fetched successfully", gin.H{
		"document":   document,
		"permission": shareLink.Permission,
	})
}

func (h *CollaboratorHandler) DeleteShareLink(c *gin.Context) {
	userID, _ := c.Get("user_id")
	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
			return
		}
		userUUID = parsed
	}

	if !h.permissionService.IsOwner(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Only owner can delete share links", nil)
		return
	}

	result := h.db.Delete(&models.DocumentShareLink{}, "document_id = ?", documentID)
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete share link", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Share link not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Share link deleted successfully", nil)
}
