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
	var req models.CreateDocumentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	document := models.Document{
		Title:   req.Title,
		Content: req.Content,
	}

	if err := h.db.Create(&document).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create document", err)
		return
	}

	response.Success(c, http.StatusCreated, "Document created successfully", document)
}

func (h *DocumentHandler) List(c *gin.Context) {
	var documents []models.Document

	if err := h.db.Order("created_at desc").Find(&documents).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to list documents", err)
		return
	}

	response.Success(c, http.StatusOK, "Documents fetched successfully", documents)
}

func (h *DocumentHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	documentID, err := uuid.Parse(id)

	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var document models.Document

	if err := h.db.First(&document, "id = ?", documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusOK, "Document fetched successfully", document)
}

func (h *DocumentHandler) Update(c *gin.Context) {
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

	if err := h.db.First(&document, "id = ?", documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found", err)
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

	response.Success(c, http.StatusOK, "Document updated successfully", document)
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	documentID, err := uuid.Parse(id)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
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
