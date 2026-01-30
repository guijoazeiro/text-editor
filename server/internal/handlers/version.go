package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type VersionHandler struct {
	db                *gorm.DB
	versionService    *services.VersionService
	permissionService *services.PermissionService
	historyService    *services.HistoryService
}

func NewVersionHandler(db *gorm.DB) *VersionHandler {
	return &VersionHandler{
		db:                db,
		versionService:    services.NewVersionService(db),
		permissionService: services.NewPermissionService(db),
		historyService:    services.NewHistoryService(db),
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

	document, err := h.versionService.RestoreVersion(documentID, versionNumber, userUUID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to restore version", err)
		return
	}

	h.historyService.RecordCreation(documentID, userUUID)

	response.Success(c, http.StatusOK, "Version restored successfully", document)
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
