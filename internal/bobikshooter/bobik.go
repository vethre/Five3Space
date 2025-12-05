package bobikshooter

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"main/internal/data"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	roundDuration = 180 * time.Second
	maxHealth     = 100
)

type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type Player struct {
	ID       string
	UserID   string
	Nickname string
	Tag      int
	Conn     *websocket.Conn
	Send     chan []byte

	Pos    Vec3
	RotY   float64
	Health int
	Kills  int
	Deaths int
	Score  int
}

type Game struct {
	mu          sync.Mutex
	store       *data.Store
	players     map[*Player]bool
	register    chan *Player
	unregister  chan *Player
	broadcast   chan []byte
	roundActive bool
	roundEnds   time.Time
}

func NewGame(store *data.Store) *Game {
	g := &Game{
		store:      store,
		players:    make(map[*Player]bool),
		register:   make(chan *Player),
		unregister: make(chan *Player),
		broadcast:  make(chan []byte, 64),
	}
	go g.run()
	go g.stateLoop()
	return g
}

func (g *Game) run() {
	for {
		select {
		case p := <-g.register:
			g.mu.Lock()
			g.players[p] = true
			if len(g.players) >= 2 && !g.roundActive {
				g.startRound()
			}
			g.mu.Unlock()
			g.sendWelcome(p)
		case p := <-g.unregister:
			g.mu.Lock()
			if _, ok := g.players[p]; ok {
				delete(g.players, p)
				close(p.Send)
				p.Conn.Close()
			}
			g.mu.Unlock()
		case msg := <-g.broadcast:
			g.mu.Lock()
			for p := range g.players {
				select {
				case p.Send <- msg:
				default:
					close(p.Send)
					delete(g.players, p)
				}
			}
			g.mu.Unlock()
		}
	}
}

func (g *Game) stateLoop() {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		g.mu.Lock()
		if g.roundActive && time.Now().After(g.roundEnds) {
			g.roundActive = false
			g.endRound()
		}
		state := g.buildState()
		g.mu.Unlock()
		g.broadcastJSON(state)
	}
}

func (g *Game) startRound() {
	g.roundActive = true
	g.roundEnds = time.Now().Add(roundDuration)
	for p := range g.players {
		p.Kills, p.Deaths = 0, 0
		p.Score = 800
		p.Health = maxHealth
		p.Pos = randomSpawn()
	}
}

func (g *Game) endRound() {
	var winner *Player
	maxKills := -1
	scoreboard := make([]map[string]interface{}, 0, len(g.players))

	for p := range g.players {
		if p.Kills > maxKills {
			maxKills = p.Kills
			winner = p
		}
		scoreboard = append(scoreboard, map[string]interface{}{
			"id": p.ID, "name": p.Nickname, "kills": p.Kills, "deaths": p.Deaths,
		})
	}

	if winner != nil && winner.UserID != "" && winner.UserID != "guest" {
		g.store.AdjustCoins(winner.UserID, 100)
		g.store.AdjustTrophies(winner.UserID, 25)
		g.store.AwardMedals(winner.UserID, "ten_wins")
	}

	g.broadcastJSON(map[string]interface{}{
		"type": "game_over", "scoreboard": scoreboard, "winnerId": winner.ID,
	})
}

func (g *Game) sendWelcome(p *Player) {
	g.mu.Lock()
	timeLeft := int(time.Until(g.roundEnds).Seconds())
	if !g.roundActive {
		timeLeft = 0
	} else if timeLeft < 0 {
		timeLeft = 0
	}
	g.mu.Unlock()

	g.sendTo(p, map[string]interface{}{
		"type": "welcome", "id": p.ID, "roundActive": g.roundActive, "timeLeft": timeLeft,
	})
}

func (g *Game) buildState() map[string]interface{} {
	timeLeft := 0
	if g.roundActive {
		timeLeft = int(time.Until(g.roundEnds).Seconds())
		if timeLeft < 0 {
			timeLeft = 0
		}
	}
	plist := make([]map[string]interface{}, 0, len(g.players))
	for p := range g.players {
		plist = append(plist, map[string]interface{}{
			"id": p.ID, "name": p.Nickname, "pos": p.Pos, "rotY": p.RotY,
			"kills": p.Kills, "deaths": p.Deaths, "health": p.Health, "score": p.Score,
		})
	}
	return map[string]interface{}{
		"type": "state", "roundActive": g.roundActive,
		"playerCount": len(g.players), "timeLeft": timeLeft, "players": plist,
	}
}

func (g *Game) broadcastJSON(v interface{}) {
	data, _ := json.Marshal(v)
	g.broadcast <- data
}

func (g *Game) sendTo(p *Player, v interface{}) {
	data, _ := json.Marshal(v)
	select {
	case p.Send <- data:
	default:
	}
}

func randomSpawn() Vec3 {
	return Vec3{X: rand.Float64()*160 - 80, Y: 15, Z: rand.Float64()*160 - 80}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (g *Game) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	userID := r.URL.Query().Get("userID")
	// 1. Fetch real nickname from DB
	nick := "Guest"
	tag := 0
	if userID != "" {
		if u, ok := g.store.GetUser(userID); ok {
			nick = u.Nickname
			tag = u.Tag
		}
	}

	p := &Player{
		ID: "b_" + uuid.NewString(), UserID: userID, Nickname: nick, Tag: tag,
		Conn: conn, Send: make(chan []byte, 256),
		Pos: randomSpawn(), Health: maxHealth, Score: 800,
	}

	g.register <- p
	go g.writePump(p)
	g.readPump(p)
}

func (g *Game) writePump(p *Player) {
	defer p.Conn.Close()
	for msg := range p.Send {
		if err := p.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			break
		}
	}
}

func (g *Game) readPump(p *Player) {
	defer func() { g.unregister <- p; p.Conn.Close() }()
	for {
		_, data, err := p.Conn.ReadMessage()
		if err != nil {
			break
		}
		var msg map[string]interface{}
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}

		switch msg["type"] {
		case "update":
			g.handleUpdate(p, msg)
		case "hit":
			g.handleHit(p, msg)
		case "buy":
			g.handleBuy(p, msg)
		}
	}
}

func (g *Game) handleUpdate(p *Player, msg map[string]interface{}) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if posRaw, ok := msg["pos"].(map[string]interface{}); ok {
		p.Pos = Vec3{X: toFloat(posRaw["x"]), Y: toFloat(posRaw["y"]), Z: toFloat(posRaw["z"])}
	}
	if ry, ok := msg["rotY"].(float64); ok {
		p.RotY = ry
	}
}

func (g *Game) handleHit(attacker *Player, msg map[string]interface{}) {
	targetID, _ := msg["target"].(string)
	damage := int(toFloat(msg["damage"]))
	if damage <= 0 {
		damage = 20
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	var target *Player
	for p := range g.players {
		if p.ID == targetID {
			target = p
			break
		}
	}
	// Revive logic is implicit: if health <= 0, we reset health and position immediately
	if target == nil || target == attacker || !g.roundActive {
		return
	}

	target.Health -= damage
	if target.Health <= 0 {
		target.Deaths++
		attacker.Kills++
		attacker.Score += 300
		// IMMEDIATE RESPAWN
		target.Health = maxHealth
		target.Pos = randomSpawn()
		target.Score += 100
	}
}

func (g *Game) handleBuy(p *Player, msg map[string]interface{}) {
	item, _ := msg["item"].(string)
	g.mu.Lock()
	defer g.mu.Unlock()

	cost := 0
	switch item {
	case "ammo":
		cost = 200
	case "awp":
		cost = 2500 // Expensive!
	}

	if cost > 0 && p.Score >= cost {
		p.Score -= cost
		g.sendTo(p, map[string]interface{}{"type": "buy_ack", "item": item, "success": true})
	}
}

func toFloat(v interface{}) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return 0
	}
}
