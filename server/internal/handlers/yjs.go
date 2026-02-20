package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
)

type YjsHandler struct {
	yjsService        *services.YjsService
	permissionService *services.PermissionService
}

func NewYjsHandler(yjsService *services.YjsService, permissionService *services.PermissionService) *YjsHandler {
	return &YjsHandler{
		yjsService:        yjsService,
		permissionService: permissionService,
	}
}

// GetUpdates returns all Yjs updates for a document
func (h *YjsHandler) GetUpdates(c *gin.Context) {
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

	// Check permissions
	if !h.permissionService.CanView(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	// Get updates
	updates, err := h.yjsService.GetUpdates(documentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch updates", err)
		return
	}

	// Convert updates to response format
	updateList := make([]map[string]interface{}, len(updates))
	for i, update := range updates {
		updateList[i] = map[string]interface{}{
			"id":     update.ID,
			"clock":  update.Clock,
			"update": update.Update,
		}
	}

	response.Success(c, http.StatusOK, "Updates fetched successfully", gin.H{
		"updates": updateList,
		"count":   len(updates),
	})
}
