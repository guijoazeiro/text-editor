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

	var filterUserID *uuid.UUID
	if userIDParam := c.Query("user_id"); userIDParam != "" {
		parsed, err := uuid.Parse(userIDParam)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user_id filter", err)
			return
		}
		filterUserID = &parsed
	}

	var actionFilter *models.ActionType
	if actionParam := c.Query("action"); actionParam != "" {
		action := models.ActionType(actionParam)
		if action != models.ActionCreated && action != models.ActionUpdated &&
			action != models.ActionTitleChanged && action != models.ActionContentChanged {
			response.Error(c, http.StatusBadRequest, "Invalid action type", nil)
			return
		}
		actionFilter = &action
	}

	var fromDate, toDate *string
	if from := c.Query("from_date"); from != "" {
		fromDate = &from
	}
	if to := c.Query("to_date"); to != "" {
		toDate = &to
	}

	histories, err := h.historyService.GetDocumentHistory(documentID, filterUserID, actionFilter, fromDate, toDate)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch history", err)
		return
	}

	response.Success(c, http.StatusOK, "History fetched successfully", histories)
}
