package websocket

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
)

type Message struct {
	DocumentID   uuid.UUID
	Data         []byte
	SenderID     uuid.UUID
	SenderConnID uuid.UUID
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
				if client.ID == message.SenderConnID {
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

	allUpdates := make([][]byte, 0, len(updates))
	for _, u := range updates {
		if len(u.Update) > 0 {
			allUpdates = append(allUpdates, u.Update)
		}
	}

	if len(allUpdates) == 0 {
		log.Printf("All persisted updates were invalid for doc=%s", client.DocumentID)
		return
	}

	msg := models.WSMessage{
		Type: models.MessageTypeYjsInit,
		Data: map[string]interface{}{
			"updates": allUpdates,
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal Yjs snapshot: %v", err)
		return
	}

	select {
	case client.Send <- data:
		log.Printf("Sent Yjs snapshot (%d updates, %d valid) to user=%s doc=%s",
			len(updates), len(allUpdates), client.UserName, client.DocumentID)
	default:
		log.Printf("Client send buffer full while sending Yjs snapshot")
	}
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

	msgType := updateBytes[0]
	if msgType != 2 {
		return
	}

	rawUpdate, err := stripSyncEnvelope(updateBytes)
	if err != nil {
		log.Printf("Failed to strip sync envelope: %v", err)
		return
	}

	if err := h.yjsService.SaveUpdate(message.DocumentID, rawUpdate); err != nil {
		log.Printf("Failed to persist Yjs update: %v", err)
	}
}

func stripSyncEnvelope(data []byte) ([]byte, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("data too short to contain sync envelope")
	}

	pos := 0
	_, n := binary.Uvarint(data[pos:])
	if n <= 0 {
		return nil, fmt.Errorf("invalid varint for msgType")
	}
	pos += n

	payloadLen, n := binary.Uvarint(data[pos:])
	if n <= 0 {
		return nil, fmt.Errorf("invalid varint for payload length")
	}
	pos += n

	end := pos + int(payloadLen)
	if end > len(data) {
		return nil, fmt.Errorf("payload length %d extends beyond data length %d", payloadLen, len(data))
	}

	return data[pos:end], nil
}

func tryCleanPersistedUpdate(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	firstByte := data[0]

	if firstByte == 0 {
		return nil
	}

	if firstByte == 1 || firstByte == 2 {
		raw, err := stripSyncEnvelope(data)
		if err == nil && len(raw) > 0 {
			return raw
		}
	}

	return data
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
