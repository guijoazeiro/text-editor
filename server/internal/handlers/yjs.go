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

func (h *YjsHandler) GetDiff(c *gin.Context) {
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

	svParam := c.Query("sv")
	if svParam == "" {
		response.Error(c, http.StatusBadRequest, "Missing state vector (sv query param)", nil)
		return
	}

	clientSV, err := base64.StdEncoding.DecodeString(svParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid state vector encoding (expected base64)", err)
		return
	}

	if h.compactorService == nil {
		response.Error(c, http.StatusServiceUnavailable, "Compactor service unavailable", nil)
		return
	}

	diff, err := h.compactorService.GetDiff(documentID, clientSV)
	if err != nil {
		response.Error(c, http.StatusServiceUnavailable, "Diff service unavailable", err)
		return
	}

	if len(diff) == 0 {
		c.Status(http.StatusNoContent)
		return
	}

	response.Success(c, http.StatusOK, "Diff computed successfully", gin.H{
		"diff": base64.StdEncoding.EncodeToString(diff),
	})
}

func (h *YjsHandler) ResetState(c *gin.Context) {
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
		response.Error(c, http.StatusForbidden, "Access denied", nil)
		return
	}

	var req struct {
		Update []int `json:"update"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if len(req.Update) < 2 {
		response.Error(c, http.StatusBadRequest, "Update payload too short to be a valid Yjs update", nil)
		return
	}

	rawUpdate := make([]byte, len(req.Update))
	for i, v := range req.Update {
		rawUpdate[i] = byte(v)
	}

	if h.snapshotService != nil {
		if err := h.snapshotService.ClearDocumentCRDTState(documentID); err != nil {
			response.Error(c, http.StatusInternalServerError, "Failed to clear CRDT state", err)
			return
		}
	}

	if err := h.yjsService.SaveUpdate(documentID, rawUpdate, 1, 0); err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to save canonical CRDT state", err)
		return
	}

	response.Success(c, http.StatusOK, "CRDT state reset successfully", nil)
}
