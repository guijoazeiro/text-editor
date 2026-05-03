package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type DocumentHandler struct {
	db                  *gorm.DB
	permissionService   *services.PermissionService
	historyService      *services.HistoryService
	notificationService *services.NotificationService
	versionService      *services.VersionService
}

func NewDocumentHandler(db *gorm.DB, snapshotService *services.SnapshotService) *DocumentHandler {
	h := &DocumentHandler{
		db:                  db,
		permissionService:   services.NewPermissionService(db),
		historyService:      services.NewHistoryService(db),
		notificationService: services.NewNotificationService(db),
		versionService:      services.NewVersionService(db, snapshotService),
	}
	go h.startPurgeLoop()
	return h
}

// startPurgeLoop permanently deletes documents that have been in the trash for
// more than 30 days. Runs daily via an infinite loop with a 24h sleep.
func (h *DocumentHandler) startPurgeLoop() {
	for {
		time.Sleep(24 * time.Hour)
		deadline := time.Now().Add(-30 * 24 * time.Hour)
		result := h.db.Unscoped().
			Where("deleted_at IS NOT NULL AND deleted_at < ?", deadline).
			Delete(&models.Document{})
		if result.Error != nil {
			log.Printf("[Purge] Error purging old documents: %v", result.Error)
		} else if result.RowsAffected > 0 {
			log.Printf("[Purge] Permanently deleted %d document(s) older than 30 days", result.RowsAffected)
		}
	}
}

func parseUserUUID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "User not authenticated", nil)
		return uuid.Nil, false
	}
	switch v := userID.(type) {
	case uuid.UUID:
		return v, true
	case string:
		parsed, err := uuid.Parse(v)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "Invalid user ID format", err)
			return uuid.Nil, false
		}
		return parsed, true
	default:
		response.Error(c, http.StatusBadRequest, "Invalid user ID type", nil)
		return uuid.Nil, false
	}
}

func (h *DocumentHandler) Create(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	var req models.CreateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	if req.ContentFormat == "" {
		req.ContentFormat = models.ContentFormatTipTap
	}

	document := models.Document{
		Title:         req.Title,
		Content:       req.Content,
		ContentFormat: req.ContentFormat,
		UserID:        userUUID,
	}

	if err := h.db.Create(&document).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create document", err)
		return
	}

	h.historyService.RecordCreation(document.ID, userUUID)
	h.versionService.CreateVersion(document.ID, userUUID, document.Title, document.Content)

	if err := h.db.Preload("User").First(&document, document.ID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusCreated, "Document created successfully", document)
}

func (h *DocumentHandler) List(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	page := 1
	if v, err := strconv.Atoi(c.Query("page")); err == nil && v > 0 {
		page = v
	}
	limit := 20
	if v, err := strconv.Atoi(c.Query("limit")); err == nil && v > 0 && v <= 100 {
		limit = v
	}
	offset := (page - 1) * limit
	q := c.Query("q")

	ownedQ := h.db.Model(&models.Document{}).Where("user_id = ?", userUUID)
	if q != "" {
		ownedQ = ownedQ.Where("title ILIKE ? OR content ILIKE ?", "%"+q+"%", "%"+q+"%")
	}

	var total int64
	ownedQ.Count(&total)

	ownedQuery := h.db.Preload("User").Where("user_id = ?", userUUID)
	if q != "" {
		ownedQuery = ownedQuery.Where("title ILIKE ? OR content ILIKE ?", "%"+q+"%", "%"+q+"%")
	}
	var ownedDocs []models.Document
	if err := ownedQuery.Order("created_at DESC").Limit(limit).Offset(offset).Find(&ownedDocs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch documents", err)
		return
	}

	var collaborations []models.DocumentCollaborator
	if err := h.db.Where("user_id = ?", userUUID).Find(&collaborations).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch collaborations", err)
		return
	}

	var sharedDocs []models.Document
	for _, collab := range collaborations {
		var doc models.Document
		query := h.db.Preload("User").First(&doc, collab.DocumentID)
		if query.Error != nil {
			continue
		}
		if q != "" {
			matches := containsInsensitive(doc.Title, q) || containsInsensitive(doc.Content, q)
			if !matches {
				continue
			}
		}
		sharedDocs = append(sharedDocs, doc)
	}

	allDocs := append(ownedDocs, sharedDocs...)

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	response.Success(c, http.StatusOK, "Documents fetched successfully", gin.H{
		"documents": allDocs,
		"total":     total,
		"page":      page,
		"limit":     limit,
		"pages":     totalPages,
	})
}

func (h *DocumentHandler) GetByID(c *gin.Context) {
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

	var document models.Document
	if err := h.db.Preload("User").First(&document, "id = ?", documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	permission, _ := h.permissionService.GetUserPermission(documentID, userUUID)

	response.Success(c, http.StatusOK, "Document fetched successfully", gin.H{
		"document":   document,
		"permission": permission,
	})
}

func (h *DocumentHandler) Update(c *gin.Context) {
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
		response.Error(c, http.StatusForbidden, "You don't have permission to edit this document", nil)
		return
	}

	var req models.UpdateDocumentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request body", err)
		return
	}

	var doc models.Document
	if err := h.db.Preload("User").First(&doc, "id = ?", documentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.Error(c, http.StatusNotFound, "Document not found", err)
			return
		}
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	before := doc

	if req.Title != "" {
		doc.Title = req.Title
	}
	if req.Content != "" {
		doc.Content = req.Content
	}
	if req.ContentFormat != "" {
		doc.ContentFormat = req.ContentFormat
	}

	if err := h.db.Save(&doc).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to update document", err)
		return
	}

	h.historyService.RecordUpdate(documentID, userUUID, &before, &doc)
	h.versionService.CreateVersion(documentID, userUUID, doc.Title, doc.Content)
	h.notificationService.NotifyDocumentEdited(documentID, userUUID)

	if err := h.db.Preload("User").First(&doc, doc.ID).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch document", err)
		return
	}

	response.Success(c, http.StatusOK, "Document updated successfully", doc)
}

func (h *DocumentHandler) Delete(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	if !h.permissionService.IsOwner(documentID, userUUID) {
		response.Error(c, http.StatusForbidden, "Only the owner can delete this document", nil)
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

	response.Success(c, http.StatusOK, "Document moved to trash", nil)
}

func (h *DocumentHandler) Trash(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	var docs []models.Document
	if err := h.db.Unscoped().
		Preload("User").
		Where("user_id = ? AND deleted_at IS NOT NULL", userUUID).
		Order("deleted_at DESC").
		Find(&docs).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch trash", err)
		return
	}

	response.Success(c, http.StatusOK, "Trash fetched successfully", docs)
}

// Restore undeletes a soft-deleted document (sets deleted_at = NULL).
func (h *DocumentHandler) Restore(c *gin.Context) {
	userUUID, ok := parseUserUUID(c)
	if !ok {
		return
	}

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid document ID", err)
		return
	}

	var doc models.Document
	if err := h.db.Unscoped().First(&doc, "id = ? AND user_id = ?", documentID, userUUID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Document not found in trash", err)
		return
	}

	if err := h.db.Unscoped().Model(&doc).Update("deleted_at", nil).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to restore document", err)
		return
	}

	response.Success(c, http.StatusOK, "Document restored successfully", doc)
}

func containsInsensitive(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	sl, subl := []rune(s), []rune(sub)
	for i := range sl {
		if i+len(subl) > len(sl) {
			break
		}
		match := true
		for j, r := range subl {
			if toLower(sl[i+j]) != toLower(r) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(r rune) rune {
	if r >= 'A' && r <= 'Z' {
		return r + 32
	}
	return r
}
