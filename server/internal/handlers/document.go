package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
