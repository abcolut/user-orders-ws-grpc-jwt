package ws

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Интерфейс для мока в тестах
type WSConn interface {
	WriteMessage(messageType int, data []byte) error
	Close() error
}

type Client struct {
	Conn   WSConn
	UserID int
}

type Hub struct {
	rooms map[int][]*Client
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{rooms: make(map[int][]*Client)}
}

func (h *Hub) AddClient(userID int, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.rooms[userID] = append(h.rooms[userID], c)
}

func (h *Hub) RemoveClient(userID int, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[userID]
	if !ok {
		return
	}
	for i, cl := range clients {
		if cl == c {
			h.rooms[userID] = append(clients[:i], clients[i+1:]...)
			break
		}
	}
	if len(h.rooms[userID]) == 0 {
		delete(h.rooms, userID)
	}
}

func (h *Hub) SendToUser(userID int, payload map[string]interface{}) {
	h.mu.RLock()
	clients, ok := h.rooms[userID]
	h.mu.RUnlock()
	if !ok || len(clients) == 0 {
		return
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	for _, c := range clients {
		if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			c.Conn.Close()
			h.RemoveClient(userID, c)
		}
	}
}
