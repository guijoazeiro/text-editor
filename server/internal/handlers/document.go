package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db *gorm.DB
}

func NewDocumentHandler(db *gorm.DB) *DocumentHandler {
	return &DocumentHandler{db: db}
}

func (h *DocumentHandler) Create(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID format", err)
			return
		}
		userUUID = parsed
	default:
		response.Error(c, http.StatusBadRequest, "Invalid user ID type", nil)
		return
	}

	document := models.Document{
		Title:   req.Title,
		Content: req.Content,
		UserID:  userUUID,
	}

	if err := h.db.Create(&document).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create document", err)
		return
	}

	if err := h.db.Preload("User").First(&document, document.ID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusCreated, "Document created successfully", document)
}

func (h *DocumentHandler) List(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID format", err)
			return
		}
		userUUID = parsed
	default:
		response.Error(c, http.StatusBadRequest, "Invalid user ID type", nil)
		return
	}

	var documents []models.Document

	if err := h.db.Preload("User").Where("user_id = ?", userUUID).Order("created_at DESC").Find(&documents).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch documents", err)
		return
	}

	response.Success(c, http.StatusOK, "Documents fetched successfully", documents)
}

func (h *DocumentHandler) GetByID(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID format", err)
			return
		}
		userUUID = parsed
	default:
		response.Error(c, http.StatusBadRequest, "Invalid user ID type", nil)
		return
	}

	id := c.Param("id")
	documentID, err := uuid.Parse(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var document models.Document
	if err := h.db.Preload("User").First(&document, "id = ? AND user_id = ?", documentID, userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found or access denied", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusOK, "Document fetched successfully", document)
}

func (h *DocumentHandler) Update(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID format", err)
			return
		}
		userUUID = parsed
	default:
		response.Error(c, http.StatusBadRequest, "Invalid user ID type", nil)
		return
	}

	id := c.Param("id")
	documentID, err := uuid.Parse(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var req models.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var document models.Document
	if err := h.db.Preload("User").First(&document, "id = ? AND user_id = ?", documentID, userUUID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found or access denied", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	if req.Title != "" {
		document.Title = req.Title
	}
	if req.Content != "" {
		document.Content = req.Content
	}

	if err := h.db.Save(&document).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update document", err)
		return
	}

	if err := h.db.Preload("User").First(&document, document.ID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusOK, "Document updated successfully", document)
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var userUUID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		userUUID = v
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID format", err)
			return
		}
		userUUID = parsed
	default:
		response.Error(c, http.StatusBadRequest, "Invalid user ID type", nil)
		return
	}

	id := c.Param("id")
	documentID, err := uuid.Parse(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	result := h.db.Where("user_id = ?", userUUID).Delete(&models.Document{}, "id = ?", documentID)
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete document", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Document not found or access denied", nil)
		return
	}

	response.Success(c, http.StatusOK, "Document deleted successfully", nil)
}
