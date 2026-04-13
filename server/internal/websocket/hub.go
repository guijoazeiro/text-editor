package websocket

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
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
	yjsService *services.YjsService
}

func NewHub(yjsService *services.YjsService) *Hub {
	return &Hub{
		Clients:    make(map[uuid.UUID]map[*Client]bool),
		Broadcast:  make(chan *Message, 256),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		yjsService: yjsService,
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

			h.sendPersistedYjsState(client)

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
			h.tryPersistYjsUpdate(message)

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

func (h *Hub) sendPersistedYjsState(client *Client) {
	if h.yjsService == nil {
		return
	}

	updates, err := h.yjsService.GetUpdates(client.DocumentID)
	if err != nil {
		log.Printf("Failed to load persisted Yjs updates: %v", err)
		return
	}

	if len(updates) == 0 {
		return
	}

	for _, u := range updates {
		msg := models.WSMessage{
			Type: models.MessageTypeYjsSync,
			Data: map[string]interface{}{
				"update": u.Update,
			},
		}
		data, err := json.Marshal(msg)
		if err != nil {
			continue
		}
		select {
		case client.Send <- data:
		default:
			log.Printf("Client send buffer full while replaying Yjs state")
			return
		}
	}

	log.Printf("Sent %d persisted Yjs updates to user=%s doc=%s", len(updates), client.UserName, client.DocumentID)
}

func (h *Hub) tryPersistYjsUpdate(message *Message) {
	if h.yjsService == nil {
		return
	}

	var wsMsg models.WSMessage
	if err := json.Unmarshal(message.Data, &wsMsg); err != nil {
		return
	}

	if wsMsg.Type != models.MessageTypeYjsSync {
		return
	}

	updateData, ok := wsMsg.Data["update"]
	if !ok {
		return
	}

	var updateBytes []byte
	switch v := updateData.(type) {
	case []byte:
		updateBytes = v
	case []interface{}:
		updateBytes = make([]byte, len(v))
		for i, val := range v {
			if num, ok := val.(float64); ok {
				updateBytes[i] = byte(num)
			}
		}
	default:
		return
	}

	if len(updateBytes) == 0 {
		return
	}

	if err := h.yjsService.SaveUpdate(message.DocumentID, updateBytes); err != nil {
		log.Printf("Failed to persist Yjs update: %v", err)
	}
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
