package chat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var DB *sql.DB

// Message represents a chat message
type Message struct {
	Type string `json:"type"`           // "dm"
	To   string `json:"to,omitempty"`   // Target UserID
	From string `json:"from,omitempty"` // Sender UserID (filled by server)
	Text string `json:"text"`
}

// MessageRow is used for fetching history from DB
type MessageRow struct {
	Sender string    `json:"sender_id"`
	Text   string    `json:"text"`
	Time   time.Time `json:"created_at"`
}

type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type Hub struct {
	clients    map[string]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan Message
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
				// Save to DB
				_, err := DB.Exec(`
					INSERT INTO messages (sender_id, receiver_id, text, delivered, seen)
					VALUES ($1, $2, $3, FALSE, FALSE)
				`, msg.From, msg.To, msg.Text)

				if err != nil {
					log.Println("DB insert error:", err)
				}

				// Send to receiver via WebSocket
				MainHub.SendDirectMessage(msg.To, msg)
			}
			// Forward seen status
			if msg.Type == "seen" {
				MainHub.SendDirectMessage(msg.From, Message{
					Type: "seen",
					From: c.UserID,
				})
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

// --- HTTP Handlers ---

// Helper to get ID from cookie
func readUserID(r *http.Request) (string, error) {
	c, err := r.Cookie("user_id")
	if err != nil || c.Value == "" {
		return "", errors.New("no user_id cookie")
	}
	return c.Value, nil
}

func DeliveredHandler(w http.ResponseWriter, r *http.Request) {
	currentUserID, err := readUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var data struct {
		From string `json:"from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		return
	}

	// Mark messages FROM the sender TO me as delivered
	DB.Exec(`UPDATE messages SET delivered = TRUE 
             WHERE sender_id = $1 AND receiver_id = $2 AND delivered = FALSE`,
		data.From, currentUserID)

	w.WriteHeader(http.StatusOK)
}

func SeenHandler(w http.ResponseWriter, r *http.Request) {
	currentUserID, err := readUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var data struct {
		From string `json:"from"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		return
	}

	// Mark messages FROM the sender TO me as seen
	DB.Exec(`UPDATE messages SET seen = TRUE 
         WHERE sender_id = $1 AND receiver_id = $2 AND seen = FALSE`,
		data.From, currentUserID)

	w.WriteHeader(http.StatusOK)
}

func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	userID, err := readUserID(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	with := r.URL.Query().Get("with")
	if with == "" {
		http.Error(w, "Missing 'with' param", http.StatusBadRequest)
		return
	}

	rows, err := DB.Query(`
        SELECT sender_id, text, created_at
        FROM messages
        WHERE (sender_id = $1 AND receiver_id = $2)
           OR (sender_id = $2 AND receiver_id = $1)
        ORDER BY created_at ASC
        LIMIT 50
    `, userID, with)

	if err != nil {
		http.Error(w, "DB Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var msgs []MessageRow
	for rows.Next() {
		var m MessageRow
		if err := rows.Scan(&m.Sender, &m.Text, &m.Time); err == nil {
			msgs = append(msgs, m)
		}
	}

	if msgs == nil {
		msgs = []MessageRow{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(msgs)
}
