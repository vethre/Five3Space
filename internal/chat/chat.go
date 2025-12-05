package chat

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// Message represents a chat message
type Message struct {
	Type string `json:"type"`           // "dm"
	To   string `json:"to,omitempty"`   // Target UserID
	From string `json:"from,omitempty"` // Sender UserID (filled by server)
	Text string `json:"text"`
}

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients    map[string]*Client // UserID -> Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message // For global (optional, unused for DMs)
	mu         sync.Mutex
}

var MainHub = &Hub{
	clients:    make(map[string]*Client),
	register:   make(chan *Client),
	unregister: make(chan *Client),
	broadcast:  make(chan Message),
}

func init() {
	go MainHub.run()
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.UserID] = client
			h.mu.Unlock()
			log.Printf("[CHAT] User connected: %s", client.UserID)

		case client := <-h.unregister:
			h.mu.Lock()
			if c, ok := h.clients[client.UserID]; ok && c == client {
				delete(h.clients, client.UserID)
				close(client.Send)
			}
			h.mu.Unlock()
			log.Printf("[CHAT] User disconnected: %s", client.UserID)
		}
	}
}

// SendDirectMessage sends a message to a specific user if they are online
func (h *Hub) SendDirectMessage(toUserID string, msg Message) {
	h.mu.Lock()
	target, ok := h.clients[toUserID]
	h.mu.Unlock()

	if ok {
		data, _ := json.Marshal(msg)
		select {
		case target.Send <- data:
		default:
			close(target.Send)
			h.mu.Lock()
			delete(h.clients, target.UserID)
			h.mu.Unlock()
		}
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func HandleWS(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("userID")
	if userID == "" {
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Chat upgrade error:", err)
		return
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	MainHub.register <- client

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		MainHub.unregister <- c
		c.Conn.Close()
	}()

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			msg.From = c.UserID
			// Routing logic
			if msg.Type == "dm" && msg.To != "" {
				MainHub.SendDirectMessage(msg.To, msg)
			}
		}
	}
}

func (c *Client) writePump() {
	defer c.Conn.Close()
	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}
