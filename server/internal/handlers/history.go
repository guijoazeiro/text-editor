package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type HistoryHandler struct {
	db                *gorm.DB
	historyService    *services.HistoryService
	permissionService *services.PermissionService
}

func NewHistoryHandler(db *gorm.DB) *HistoryHandler {
	return &HistoryHandler{
		db:                db,
		historyService:    services.NewHistoryService(db),
		permissionService: services.NewPermissionService(db),
	}
}

func (h *HistoryHandler) GetDocumentHistory(c *gin.Context) {
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
			response.Error(c, http.StatusBadRequest, "Invalid user ID", err)
			return
		}
		userUUID = parsed
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

	histories, err := h.historyService.GetDocumentHistory(documentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch history", err)
		return
	}

	response.Success(c, http.StatusOK, "History fetched successfully", histories)
}
