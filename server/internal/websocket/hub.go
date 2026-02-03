package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
)

type Message struct {
	DocumentID uuid.UUID
	Data       []byte
	SenderID   uuid.UUID
}

type Hub struct {
	Clients    map[uuid.UUID]map[*Client]bool
	Broadcast  chan *Message
	Register   chan *Client
	Unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[uuid.UUID]map[*Client]bool),
		Broadcast:  make(chan *Message, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.Clients[client.DocumentID] == nil {
				h.Clients[client.DocumentID] = make(map[*Client]bool)
			}
			h.Clients[client.DocumentID][client] = true
			h.mu.Unlock()

			log.Printf("Client registered: user=%s doc=%s", client.UserName, client.DocumentID)

			h.sendPresenceUpdate(client.DocumentID)

			h.notifyUserJoined(client)

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.Clients[client.DocumentID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)

					if len(clients) == 0 {
						delete(h.Clients, client.DocumentID)
					}
				}
			}
			h.mu.Unlock()

			log.Printf("Client unregistered: user=%s doc=%s", client.UserName, client.DocumentID)

			h.notifyUserLeft(client)

		case message := <-h.Broadcast:
			h.mu.RLock()
			clients := h.Clients[message.DocumentID]
			h.mu.RUnlock()

			for client := range clients {
				if client.UserID == message.SenderID {
					continue
				}

				select {
				case client.Send <- message.Data:
				default:
					h.mu.Lock()
					close(client.Send)
					delete(h.Clients[message.DocumentID], client)
					h.mu.Unlock()
				}
			}
		}
	}
}

func (h *Hub) sendPresenceUpdate(documentID uuid.UUID) {
	h.mu.RLock()
	clients := h.Clients[documentID]
	h.mu.RUnlock()

	presences := []models.UserPresence{}
	for client := range clients {
		presences = append(presences, models.UserPresence{
			UserID:   client.UserID,
			UserName: client.UserName,
			Online:   true,
			Color:    generateColor(client.UserID),
		})
	}

	presenceMsg := models.WSMessage{
		Type: models.MessageTypePresence,
		Data: map[string]interface{}{
			"users": presences,
		},
	}

	data, _ := json.Marshal(presenceMsg)

	for client := range clients {
		select {
		case client.Send <- data:
		default:
		}
	}
}

func (h *Hub) notifyUserJoined(client *Client) {
	joinMsg := models.WSMessage{
		Type:       models.MessageTypeJoin,
		UserID:     client.UserID.String(),
		UserName:   client.UserName,
		DocumentID: client.DocumentID.String(),
		Data: map[string]interface{}{
			"color": generateColor(client.UserID),
		},
	}

	data, _ := json.Marshal(joinMsg)

	h.mu.RLock()
	clients := h.Clients[client.DocumentID]
	h.mu.RUnlock()

	for c := range clients {
		if c.ID != client.ID {
			select {
			case c.Send <- data:
			default:
			}
		}
	}
}

func (h *Hub) notifyUserLeft(client *Client) {
	leaveMsg := models.WSMessage{
		Type:       models.MessageTypeLeave,
		UserID:     client.UserID.String(),
		UserName:   client.UserName,
		DocumentID: client.DocumentID.String(),
	}

	data, _ := json.Marshal(leaveMsg)

	h.mu.RLock()
	clients := h.Clients[client.DocumentID]
	h.mu.RUnlock()

	for c := range clients {
		select {
		case c.Send <- data:
		default:
		}
	}
}

func (h *Hub) GetActiveUsers(documentID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if clients, ok := h.Clients[documentID]; ok {
		return len(clients)
	}
	return 0
}

func generateColor(userID uuid.UUID) string {
	colors := []string{
		"#3B82F6",
		"#10B981",
		"#F59E0B",
		"#EF4444",
		"#8B5CF6",
		"#EC4899",
		"#14B8A6",
		"#F97316",
	}

	hash := 0
	for _, b := range userID {
		hash += int(b)
	}

	return colors[hash%len(colors)]
}
