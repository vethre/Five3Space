package chibiki

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

func NewWebsocketHandler(g *GameInstance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println(err)
			return
		}

		userID := r.URL.Query().Get("userID")
		if userID == "" {
			userID = "guest" // Default user ID
		}

		playerID := fmt.Sprintf("p-%d", time.Now().UnixNano())

		player := &Player{
			ID:     playerID,
			UserID: userID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
		}

		g.Register <- player
		go writePump(player)
		go readPump(player, g)
	}
}

func readPump(p *Player, g *GameInstance) {
	defer func() {
		g.Unregister <- p
		p.Conn.Close()
	}()

	for {
		_, message, err := p.Conn.ReadMessage()
		if err != nil {
			break
		}

		var input struct {
			Type string  `json:"type"`
			Key  string  `json:"key"`
			X    float64 `json:"x"`
			Y    float64 `json:"y"`
		}

		if err := json.Unmarshal(message, &input); err == nil {
			if input.Type == "spawn" {
				g.SpawnUnit(p, input.Key, input.X, input.Y)
			} else if input.Type == "reset" {
				g.Reset()
			}
		}
	}
}

func writePump(p *Player) {
	defer func() { p.Conn.Close() }()
	for {
		select {
		case message, ok := <-p.Send:
			if !ok {
				p.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			p.Conn.WriteMessage(websocket.TextMessage, message)
		}
	}
}
