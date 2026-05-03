package websocket

import (
	"encoding/json"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
)

const (
	writeWait        = 10 * time.Second
	pongWait         = 60 * time.Second
	pingPeriod       = (pongWait * 9) / 10
	maxMessageSize   = 512 * 1024
	tokenCheckPeriod = 15 * time.Minute
)

type Client struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	UserName   string
	DocumentID uuid.UUID
	Token      string
	Hub        *Hub
	Conn       *websocket.Conn
	Send       chan []byte
}

func NewClient(hub *Hub, conn *websocket.Conn, userID, documentID uuid.UUID, userName, token string) *Client {
	return &Client{
		ID:         uuid.New(),
		UserID:     userID,
		UserName:   userName,
		DocumentID: documentID,
		Token:      token,
		Hub:        hub,
		Conn:       conn,
		Send:       make(chan []byte, 256),
	}
}

func isTokenExpired(tokenStr string) bool {
	if len(strings.Split(tokenStr, ".")) != 3 {
		return true
	}
	p, _, err := new(jwt.Parser).ParseUnverified(tokenStr, jwt.MapClaims{})
	if err != nil {
		return true
	}
	exp, err := p.Claims.GetExpirationTime()
	if err != nil || exp == nil {
		return false
	}
	return time.Now().After(exp.Time)
}

func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	c.Conn.SetReadLimit(maxMessageSize)

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Error unmarshaling message: %v", err)
			continue
		}

		wsMsg.UserID = c.UserID.String()
		wsMsg.UserName = c.UserName
		wsMsg.DocumentID = c.DocumentID.String()
		wsMsg.Timestamp = time.Now().Unix()

		messageJSON, err := json.Marshal(wsMsg)
		if err != nil {
			log.Printf("Error marshaling message: %v", err)
			continue
		}

		c.Hub.Broadcast <- &Message{
			DocumentID:   c.DocumentID,
			Data:         messageJSON,
			SenderID:     c.UserID,
			SenderConnID: c.ID,
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	tokenTicker := time.NewTicker(tokenCheckPeriod)
	defer func() {
		ticker.Stop()
		tokenTicker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

			n := len(c.Send)
			for i := 0; i < n; i++ {
				c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.Conn.WriteMessage(websocket.TextMessage, <-c.Send); err != nil {
					return
				}
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-tokenTicker.C:
			if c.Token != "" && isTokenExpired(c.Token) {
				log.Printf("[WS] token expired for user=%s, closing with 4001", c.UserName)
				c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				c.Conn.WriteMessage(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(4001, "token expired"),
				)
				return
			}
		}
	}
}
