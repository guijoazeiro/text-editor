package websocket

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/models"
	"github.com/guijoazeiro/text-editor/tree/main/server/internal/services"
	yjsdecoder "github.com/guijoazeiro/text-editor/tree/main/server/internal/yjs"
)

type Message struct {
	DocumentID   uuid.UUID
	Data         []byte
	SenderID     uuid.UUID
	SenderConnID uuid.UUID
}

type Hub struct {
	Clients          map[uuid.UUID]map[*Client]bool
	Broadcast        chan *Message
	Register         chan *Client
	Unregister       chan *Client
	mu               sync.RWMutex
	yjsService       *services.YjsService
	snapshotService  *services.SnapshotService
	compactorService *services.CompactorService
	restoringDocs    map[uuid.UUID]bool
	restoreMu        sync.RWMutex
}

func NewHub(
	yjsService *services.YjsService,
	snapshotService *services.SnapshotService,
	compactorService *services.CompactorService,
) *Hub {
	return &Hub{
		Clients:          make(map[uuid.UUID]map[*Client]bool),
		Broadcast:        make(chan *Message, 256),
		Register:         make(chan *Client),
		Unregister:       make(chan *Client),
		yjsService:       yjsService,
		snapshotService:  snapshotService,
		compactorService: compactorService,
		restoringDocs:    make(map[uuid.UUID]bool),
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
			var docBecameIdle bool
			h.mu.Lock()
			if clients, ok := h.Clients[client.DocumentID]; ok {
				if _, exists := clients[client]; exists {
					delete(clients, client)
					close(client.Send)

					if len(clients) == 0 {
						delete(h.Clients, client.DocumentID)
						docBecameIdle = true
					}
				}
			}
			h.mu.Unlock()

			log.Printf("Client unregistered: user=%s doc=%s", client.UserName, client.DocumentID)

			h.notifyUserLeft(client)

			if docBecameIdle && h.compactorService != nil {
				docID := client.DocumentID
				go func() {
					time.Sleep(5 * time.Second)
					h.mu.RLock()
					_, stillActive := h.Clients[docID]
					h.mu.RUnlock()
					if stillActive {
						return
					}
					if err := h.compactorService.ForceCompact(docID); err != nil {
						log.Printf("[Compactor] idle compaction failed for doc=%s: %v", docID, err)
					}
				}()
			}

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

	awarenessOffMsg := models.WSMessage{
		Type:   models.MessageTypeYjsAwarenessOff,
		UserID: client.UserID.String(),
	}
	awarenessOffData, _ := json.Marshal(awarenessOffMsg)

	h.mu.RLock()
	clients := h.Clients[client.DocumentID]
	h.mu.RUnlock()

	for c := range clients {
		select {
		case c.Send <- data:
		default:
		}

		select {
		case c.Send <- awarenessOffData:
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

func (h *Hub) BroadcastYjsReset(documentID uuid.UUID, snapshot []byte) {
	encoded := base64.StdEncoding.EncodeToString(snapshot)

	msg := models.WSMessage{
		Type: models.MessageTypeYjsReset,
		Data: map[string]interface{}{
			"snapshot": encoded,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[YjsReset] failed to marshal reset message for doc=%s: %v", documentID, err)
		return
	}

	h.mu.RLock()
	clients := h.Clients[documentID]
	h.mu.RUnlock()

	sent := 0
	for c := range clients {
		select {
		case c.Send <- data:
			sent++
		default:
			log.Printf("[YjsReset] send buffer full for user=%s, skipping", c.UserName)
		}
	}
	log.Printf("[YjsReset] broadcast reset to %d peers for doc=%s", sent, documentID)
}

func (h *Hub) BroadcastDocumentContentReset(documentID uuid.UUID, content, title string) {
	msg := models.WSMessage{
		Type: models.MessageTypeDocumentContentReset,
		Data: map[string]interface{}{
			"content": content,
			"title":   title,
		},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ContentReset] failed to marshal message for doc=%s: %v", documentID, err)
		return
	}

	h.mu.RLock()
	clients := h.Clients[documentID]
	h.mu.RUnlock()

	sent := 0
	for c := range clients {
		select {
		case c.Send <- data:
			sent++
		default:
			log.Printf("[ContentReset] send buffer full for user=%s, skipping", c.UserName)
		}
	}
	log.Printf("[ContentReset] broadcast content reset to %d peers for doc=%s", sent, documentID)
}

func (h *Hub) sendPersistedYjsState(client *Client) {
	if h.snapshotService == nil && h.yjsService == nil {
		return
	}

	if h.snapshotService != nil {
		state, err := h.snapshotService.GetStateForClient(client.DocumentID)
		if err != nil {
			log.Printf("[Yjs] failed to load state for user=%s doc=%s: %v",
				client.UserName, client.DocumentID, err)
			return
		}

		var allUpdates [][]byte
		if len(state.Snapshot) > 0 {
			allUpdates = append(allUpdates, state.Snapshot)
		}
		allUpdates = append(allUpdates, state.DeltaUpdates...)

		if len(allUpdates) == 0 {
			return
		}

		msg := models.WSMessage{
			Type: models.MessageTypeYjsInit,
			Data: map[string]interface{}{"updates": allUpdates},
		}
		data, err := json.Marshal(msg)
		if err != nil {
			log.Printf("[Yjs] failed to marshal state for user=%s: %v", client.UserName, err)
			return
		}
		select {
		case client.Send <- data:
			log.Printf("[Yjs] sent state (snapshot=%v, delta=%d) to user=%s doc=%s",
				len(state.Snapshot) > 0, len(state.DeltaUpdates), client.UserName, client.DocumentID)
		default:
			log.Printf("[Yjs] send buffer full for user=%s", client.UserName)
		}
		return
	}

	updates, err := h.yjsService.GetUpdates(client.DocumentID)
	if err != nil {
		log.Printf("[Yjs] failed to load updates: %v", err)
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
		return
	}
	msg := models.WSMessage{
		Type: models.MessageTypeYjsInit,
		Data: map[string]interface{}{"updates": allUpdates},
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	select {
	case client.Send <- data:
		log.Printf("[Yjs] sent %d updates to user=%s doc=%s", len(allUpdates), client.UserName, client.DocumentID)
	default:
		log.Printf("[Yjs] send buffer full for user=%s", client.UserName)
	}
}

func (h *Hub) SetRestoring(docID uuid.UUID, restoring bool) {
	h.restoreMu.Lock()
	defer h.restoreMu.Unlock()
	if restoring {
		h.restoringDocs[docID] = true
	} else {
		delete(h.restoringDocs, docID)
	}
}

func (h *Hub) IsRestoring(docID uuid.UUID) bool {
	h.restoreMu.RLock()
	defer h.restoreMu.RUnlock()
	return h.restoringDocs[docID]
}

func (h *Hub) WaitForDrain(_ uuid.UUID) {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(200 * time.Millisecond)
	for {
		select {
		case <-deadline:
			return
		case <-ticker.C:
			if len(h.Broadcast) == 0 {
				return
			}
		}
	}
}

func (h *Hub) tryPersistYjsUpdate(message *Message) {
	if h.yjsService == nil {
		return
	}

	if h.IsRestoring(message.DocumentID) {
		log.Printf("[Yjs] dropping update during restore for doc=%s conn=%s",
			message.DocumentID, message.SenderConnID)
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
		log.Printf("[Yjs] unexpected update payload type: %T", updateData)
		return
	}

	if len(updateBytes) == 0 {
		return
	}

	if updateBytes[0] != 2 {
		return
	}

	rawUpdate, err := stripSyncEnvelope(updateBytes)
	if err != nil {
		log.Printf("[Yjs] failed to strip sync envelope: %v", err)
		return
	}

	meta, err := yjsdecoder.DecodeUpdateMeta(rawUpdate)
	if err != nil {
		log.Printf("[Yjs] rejecting malformed update from conn=%s doc=%s: %v",
			message.SenderConnID, message.DocumentID, err)
		return
	}

	if meta.IsEmpty {
		return
	}

	if err := h.yjsService.SaveUpdate(message.DocumentID, rawUpdate, meta.LamportTS, meta.ClientID); err != nil {
		log.Printf("[Yjs] failed to persist update (lamport=%d client=%d doc=%s): %v",
			meta.LamportTS, meta.ClientID, message.DocumentID, err)
		return
	}

	if h.compactorService != nil {
		docID := message.DocumentID
		go func() {
			if err := h.compactorService.MaybeCompactWithRetry(docID); err != nil {
				log.Printf("[Compactor] MaybeCompactWithRetry failed for doc=%s: %v", docID, err)
			}
		}()
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
