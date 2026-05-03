package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/auth"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	ws "github.com/guijoazeiro/text-editor/tree/main/server/internal/websocket"
	"github.com/guijoazeiro/text-editor/tree/main/server/pkg/response"
	"gorm.io/gorm"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WebSocketHandler struct {
	hub               *ws.Hub
	db                *gorm.DB
	permissionService *services.PermissionService
	jwtService        *auth.JWT
}

func NewWebSocketHandler(hub *ws.Hub, db *gorm.DB, jwtService *auth.JWT) *WebSocketHandler {
	return &WebSocketHandler{
		hub:               hub,
		db:                db,
		permissionService: services.NewPermissionService(db),
		jwtService:        jwtService,
	}
}

func (h *WebSocketHandler) HandleWebSocket(c *gin.Context) {
	protocolHeader := c.Request.Header.Get("Sec-WebSocket-Protocol")
	protocols := strings.Split(protocolHeader, ", ")

	var token string
	if len(protocols) >= 2 && protocols[0] == "access_token" {
		token = protocols[1]
	}

	if token == "" {
		log.Println("No token in WebSocket subprotocol")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	claims, err := h.jwtService.ValidateToken(token)
	if err != nil {
		log.Printf("Invalid token: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	userUUID := claims.UserID

	documentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid document ID"})
		return
	}

	if !h.permissionService.CanView(documentID, userUUID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var user models.User
	if err := h.db.First(&user, userUUID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, http.Header{
		"Sec-WebSocket-Protocol": []string{"access_token"},
	})
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := ws.NewClient(h.hub, conn, userUUID, documentID, user.Name, token)

	h.hub.Register <- client

	// goroutines
	go client.WritePump()
	go client.ReadPump()
}

func (h *WebSocketHandler) GetActiveUsers(c *gin.Context) {
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

	activeUsers := h.hub.GetActiveUsers(documentID)

	response.Success(c, http.StatusOK, "Active users fetched", gin.H{
		"count": activeUsers,
	})
}
