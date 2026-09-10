package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
	GameID string
	Player string // "player" or "spectator"
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	mu      sync.RWMutex
	games   map[string]map[*Client]bool
	handler MessageHandler
}

type MessageHandler func(gameID string, player string, msg IncomingMessage)

type IncomingMessage struct {
	Type        string `json:"type"`
	Action      string `json:"action,omitempty"`
	CardUID     string `json:"cardUid,omitempty"`
	TargetUID   string `json:"targetUid,omitempty"`
	AttackIndex int    `json:"attackIndex,omitempty"`
	Position    string `json:"position,omitempty"`
	BenchIndex  int    `json:"benchIndex,omitempty"`

	// Setup-specific
	ActiveUID string   `json:"activeUid,omitempty"`
	BenchUIDs []string `json:"benchUids,omitempty"`
}

type OutgoingMessage struct {
	Type    string `json:"type"`
	Payload any    `json:"payload,omitempty"`
}

func NewHub(handler MessageHandler) *Hub {
	return &Hub{
		games:   make(map[string]map[*Client]bool),
		handler: handler,
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request, gameID string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade error: %v", err)
		return
	}

	client := &Client{
		GameID: gameID,
		Player: "player",
		Conn:   conn,
		Send:   make(chan []byte, 64),
	}

	h.register(client)
	go h.writePump(client)
	go h.readPump(client)
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.games[c.GameID] == nil {
		h.games[c.GameID] = make(map[*Client]bool)
	}
	h.games[c.GameID][c] = true
	log.Printf("[ws] client connected to game %s", c.GameID)
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.games[c.GameID]; ok {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.games, c.GameID)
		}
	}
	close(c.Send)
	c.Conn.Close()
	log.Printf("[ws] client disconnected from game %s", c.GameID)
}

func (h *Hub) BroadcastToGame(gameID string, msg OutgoingMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[ws] marshal error: %v", err)
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := h.games[gameID]
	for c := range clients {
		select {
		case c.Send <- data:
		default:
			log.Printf("[ws] client send buffer full, dropping message")
		}
	}
}

func (h *Hub) SendToGame(gameID string, msg OutgoingMessage) {
	h.BroadcastToGame(gameID, msg)
}

func (h *Hub) readPump(c *Client) {
	defer h.unregister(c)

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[ws] read error: %v", err)
			}
			break
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("[ws] invalid message: %v", err)
			continue
		}

		if h.handler != nil {
			h.handler(c.GameID, c.Player, msg)
		}
	}
}

func (h *Hub) writePump(c *Client) {
	for data := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("[ws] write error: %v", err)
			return
		}
	}
}

func (h *Hub) HasGame(gameID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.games[gameID]
	return ok
}
