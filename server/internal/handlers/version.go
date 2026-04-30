package handlers

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/websocket"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type VersionHandler struct {
	db                *gorm.DB
	versionService    *services.VersionService
	permissionService *services.PermissionService
	historyService    *services.HistoryService
	yjsService        *services.YjsService
	hub               *websocket.Hub
}

func NewVersionHandler(db *gorm.DB, snapshotService *services.SnapshotService, yjsService *services.YjsService, hub *websocket.Hub) *VersionHandler {
	return &VersionHandler{
		db:                db,
		versionService:    services.NewVersionService(db, snapshotService),
		permissionService: services.NewPermissionService(db),
		historyService:    services.NewHistoryService(db),
		yjsService:        yjsService,
		hub:               hub,
	}
}

func (h *VersionHandler) GetVersions(c *gin.Context) {
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

	versions, err := h.versionService.GetVersions(documentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch versions", err)
		return
	}

	response.Success(c, http.StatusOK, "Versions fetched successfully", versions)
}

func (h *VersionHandler) GetVersion(c *gin.Context) {
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

	versionNumber, err := strconv.Atoi(c.Param("version_number"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid version number", err)
		return
	}

	if !h.permissionService.CanView(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	version, err := h.versionService.GetVersion(documentID, versionNumber)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Version not found", err)
		return
	}

	response.Success(c, http.StatusOK, "Version fetched successfully", version)
}

func (h *VersionHandler) RestoreVersion(c *gin.Context) {
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

	versionNumber, err := strconv.Atoi(c.Param("version_number"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid version number", err)
		return
	}

	if !h.permissionService.CanEdit(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "You don't have permission to restore versions", nil)
		return
	}

	if h.hub != nil {
		h.hub.WaitForDrain(documentID)
	}

	if h.hub != nil {
		h.hub.SetRestoring(documentID, true)
		defer h.hub.SetRestoring(documentID, false)
	}

	result, err := h.versionService.RestoreVersion(documentID, versionNumber, userUUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to restore version", err)
		return
	}

	if h.yjsService != nil {
		if err := h.yjsService.DeleteUpdatesForDocument(documentID); err != nil {
			log.Printf("[Version] failed to clear yjs_updates after restore doc=%s: %v", documentID, err)
		}
	}

	h.historyService.RecordCreation(documentID, userUUID)

	if h.hub != nil && result.Document != nil {
		go func() {
			h.hub.BroadcastDocumentContentReset(documentID, result.Document.Content, result.Document.Title)
			log.Printf("[Version] document-content-reset broadcast for doc=%s version=%d", documentID, versionNumber)
		}()
	}

	response.Success(c, http.StatusOK, "Version restored successfully", result.Document)
}

func (h *VersionHandler) CompareVersions(c *gin.Context) {
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

	v1Str := c.Query("v1")
	v2Str := c.Query("v2")

	if v1Str == "" || v2Str == "" {
		response.Error(c, http.StatusBadRequest, "Both v1 and v2 query params are required", nil)
		return
	}

	v1, err := strconv.Atoi(v1Str)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid v1 parameter", err)
		return
	}

	v2, err := strconv.Atoi(v2Str)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid v2 parameter", err)
		return
	}

	if !h.permissionService.CanView(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	diff, err := h.versionService.CompareVersions(documentID, v1, v2)
	if err != nil {
		response.Error(c, http.StatusNotFound, "Failed to compare versions", err)
		return
	}

	response.Success(c, http.StatusOK, "Versions compared successfully", diff)
}
