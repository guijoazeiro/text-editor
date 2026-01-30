package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

type NotificationHandler struct {
	db *gorm.DB
}

func NewNotificationHandler(db *gorm.DB) *NotificationHandler {
	return &NotificationHandler{db: db}
}

func (h *NotificationHandler) List(c *gin.Context) {
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

	var notifications []models.Notification
	if err := h.db.Preload("FromUser").
		Preload("Document.User").
		Where("user_id = ?", userUUID).
		Order("created_at DESC").
		Find(&notifications).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to fetch notifications", err)
		return
	}

	var unreadCount int64
	h.db.Model(&models.Notification{}).Where("user_id = ? AND read = false", userUUID).Count(&unreadCount)

	response.Success(c, http.StatusOK, "Notifications fetched successfully", gin.H{
		"notifications": notifications,
		"unread_count":  unreadCount,
	})
}

func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
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

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid notification ID", err)
		return
	}

	var notification models.Notification
	if err := h.db.First(&notification, "id = ? AND user_id = ?", notificationID, userUUID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "Notification not found", err)
		return
	}

	notification.Read = true
	if err := h.db.Save(&notification).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to mark as read", err)
		return
	}

	response.Success(c, http.StatusOK, "Notification marked as read", notification)
}

func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
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

	if err := h.db.Model(&models.Notification{}).
		Where("user_id = ? AND read = false", userUUID).
		Update("read", true).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to mark all as read", err)
		return
	}

	response.Success(c, http.StatusOK, "All notifications marked as read", nil)
}

func (h *NotificationHandler) Delete(c *gin.Context) {
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

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid notification ID", err)
		return
	}

	result := h.db.Delete(&models.Notification{}, "id = ? AND user_id = ?", notificationID, userUUID)
	if result.Error != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to delete notification", result.Error)
		return
	}

	if result.RowsAffected == 0 {
		response.Error(c, http.StatusNotFound, "Notification not found", nil)
		return
	}

	response.Success(c, http.StatusOK, "Notification deleted successfully", nil)
}
