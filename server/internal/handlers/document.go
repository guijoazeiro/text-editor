package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db                  *gorm.DB
	permissionService   *services.PermissionService
	historyService      *services.HistoryService
	notificationService *services.NotificationService
	versionService      *services.VersionService
}

func NewDocumentHandler(db *gorm.DB) *DocumentHandler {
	return &DocumentHandler{
		db:                  db,
		permissionService:   services.NewPermissionService(db),
		historyService:      services.NewHistoryService(db),
		notificationService: services.NewNotificationService(db),
		versionService:      services.NewVersionService(db, nil),
	}
}

func parseUserUUID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return uuid.Nil, false
	}
	switch v := userID.(type) {
	case uuid.UUID:
		return v, true
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID format", err)
			return uuid.Nil, false
		}
		return parsed, true
	default:
		response.Error(c, http.StatusBadRequest, "Invalid user ID type", nil)
		return uuid.Nil, false
	}
}

func (h *DocumentHandler) Create(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.ContentFormat == "" {
		req.ContentFormat = models.ContentFormatTipTap
	}

	document := models.Document{
		Title:         req.Title,
		Content:       req.Content,
		ContentFormat: req.ContentFormat,
		UserID:        userUUID,
	}

	if err := h.db.Create(&document).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create document", err)
		return
	}

	h.historyService.RecordCreation(document.ID, userUUID)
	h.versionService.CreateVersion(document.ID, userUUID, document.Title, document.Content)

	if err := h.db.Preload("User").First(&document, document.ID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusCreated, "Document created successfully", document)
}

func (h *DocumentHandler) List(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	var ownedDocs []models.Document
	if err := h.db.Preload("User").Where("user_id = ?", userUUID).Order("created_at DESC").Find(&ownedDocs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch documents", err)
		return
	}

	var collaborations []models.DocumentCollaborator
	if err := h.db.Where("user_id = ?", userUUID).Find(&collaborations).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch collaborations", err)
		return
	}

	var sharedDocs []models.Document
	for _, collab := range collaborations {
		var doc models.Document
		if err := h.db.Preload("User").First(&doc, collab.DocumentID).Error; err == nil {
			sharedDocs = append(sharedDocs, doc)
		}
	}

	allDocs := append(ownedDocs, sharedDocs...)
	response.Success(c, http.StatusOK, "Documents fetched successfully", allDocs)
}

func (h *DocumentHandler) GetByID(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	if !h.permissionService.CanView(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	var document models.Document
	if err := h.db.Preload("User").First(&document, "id = ?", documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	permission, _ := h.permissionService.GetUserPermission(documentID, userUUID)

	response.Success(c, http.StatusOK, "Document fetched successfully", gin.H{
		"document":   document,
		"permission": permission,
	})
}

func (h *DocumentHandler) Update(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	if !h.permissionService.CanEdit(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "You don't have permission to edit this document", nil)
		return
	}

	var req models.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var doc models.Document
	if err := h.db.Preload("User").First(&doc, "id = ?", documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	before := doc

	if req.Title != "" {
		doc.Title = req.Title
	}
	if req.Content != "" {
		doc.Content = req.Content
	}
	if req.ContentFormat != "" {
		doc.ContentFormat = req.ContentFormat
	}

	if err := h.db.Save(&doc).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update document", err)
		return
	}

	h.historyService.RecordUpdate(documentID, userUUID, &before, &doc)
	h.versionService.CreateVersion(documentID, userUUID, doc.Title, doc.Content)
	h.notificationService.NotifyDocumentEdited(documentID, userUUID)

	if err := h.db.Preload("User").First(&doc, doc.ID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusOK, "Document updated successfully", doc)
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	if !h.permissionService.IsOwner(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Only the owner can delete this document", nil)
		return
	}

	result := h.db.Delete(&models.Document{}, "id = ?", documentID)
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete document", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Document not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Document deleted successfully", nil)
}
