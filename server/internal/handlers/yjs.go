package handlers

import (
	"encoding/base64"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
)

type YjsHandler struct {
	yjsService        *services.YjsService
	permissionService *services.PermissionService
	snapshotService   *services.SnapshotService
	compactorService  *services.CompactorService
}

func NewYjsHandler(
	yjsService *services.YjsService,
	permissionService *services.PermissionService,
	snapshotService *services.SnapshotService,
	compactorService *services.CompactorService,
) *YjsHandler {
	return &YjsHandler{
		yjsService:        yjsService,
		permissionService: permissionService,
		snapshotService:   snapshotService,
		compactorService:  compactorService,
	}
}

func (h *YjsHandler) GetUpdates(c *gin.Context) {
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

	updates, err := h.yjsService.GetUpdates(documentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch updates", err)
		return
	}

	updateList := make([]map[string]interface{}, len(updates))
	for i, update := range updates {
		updateList[i] = map[string]interface{}{
			"id":         update.ID,
			"lamport_ts": update.LamportTS,
			"client_id":  update.ClientID,
			"update":     update.Update,
		}
	}

	response.Success(c, http.StatusOK, "Updates fetched successfully", gin.H{
		"updates": updateList,
		"count":   len(updates),
	})
}

func (h *YjsHandler) GetStateVector(c *gin.Context) {
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

	if h.snapshotService == nil || h.compactorService == nil {
		c.Status(http.StatusNoContent)
		return
	}

	snap, err := h.snapshotService.GetSnapshot(documentID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch snapshot", err)
		return
	}

	if snap == nil || len(snap.Snapshot) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	sv, err := h.compactorService.GetStateVector(snap.Snapshot)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "State vector service unavailable", err)
		return
	}

	response.Success(c, http.StatusOK, "State vector fetched successfully", gin.H{
		"state_vector": base64.StdEncoding.EncodeToString(sv),
		"lamport_ts":   snap.LamportTS,
	})
}
