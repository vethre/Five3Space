package upsidedown

import (
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"main/internal/data"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	TickRate          = 30
	GameDuration      = 180 // 3 minutes survival
	MaxHealth         = 100
	MaxSanity         = 100
	SanityDrainRate   = 2.0  // Per second in darkness
	HealthDrainRate   = 1.0  // Per second when sanity is 0
	LightRestoreRate  = 10.0 // Sanity restore per second near light
	DemoSpawnInterval = 15   // Seconds between demogorgon spawns
)

// Resource types
const (
	ResourceLightOrb = "light_orb"
	ResourceBattery  = "battery"
	ResourceFlare    = "flare"
)

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Player struct {
	ID       string          `json:"id"`
	UserID   string          `json:"-"`
	Nickname string          `json:"name"`
	Conn     *websocket.Conn `json:"-"`
	Send     chan []byte     `json:"-"`

	Pos             Vec2    `json:"pos"`
	Health          float64 `json:"health"`
	Sanity          float64 `json:"sanity"`
	Score           int     `json:"score"`
	Alive           bool    `json:"alive"`
	AvailableFlares int     `json:"availableFlares"`
	HasFlare        bool    `json:"hasFlare"` // Active flare
	FlareTime       float64 `json:"-"`        // Seconds remaining
	LightRadius     float64 `json:"lightRadius"`
}

type Entity struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"` // "demogorgon", "light_orb", "battery", "flare"
	Pos          Vec2    `json:"pos"`
	Active       bool    `json:"active"`
	Health       int     `json:"-"`
	StunnedUntil float64 `json:"-"`
}

type Game struct {
	mu         sync.Mutex
	store      *data.Store
	players    map[*Player]bool
	entities   []*Entity
	register   chan *Player
	unregister chan *Player

	gameActive bool
	gameTime   float64
	spawnTimer float64
	difficulty float64 // Increases over time
}

func NewGame(store *data.Store) *Game {
	g := &Game{
		store:      store,
		players:    make(map[*Player]bool),
		entities:   make([]*Entity, 0),
		register:   make(chan *Player),
		unregister: make(chan *Player),
	}
	go g.run()
	return g
}

func (g *Game) run() {
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()

	lastTime := time.Now()

	for {
		select {
		case p := <-g.register:
			g.mu.Lock()
			g.players[p] = true
			// Start game if first player or reset if needed
			if len(g.players) == 1 && !g.gameActive {
				g.startGame()
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
			// Reset game if no players
			if len(g.players) == 0 {
				g.gameActive = false
			}
			g.mu.Unlock()

		case <-ticker.C:
			now := time.Now()
			dt := now.Sub(lastTime).Seconds()
			lastTime = now
			g.update(dt)
		}
	}
}

func (g *Game) startGame() {
	g.gameActive = true
	g.gameTime = 0
	g.difficulty = 1.0
	g.spawnTimer = 5.0 // First spawn in 5 seconds
	g.entities = make([]*Entity, 0)

	// Spawn initial resources
	for i := 0; i < 10; i++ {
		g.spawnResource(ResourceLightOrb)
	}
	for i := 0; i < 5; i++ {
		g.spawnResource(ResourceBattery)
	}
	for i := 0; i < 3; i++ {
		g.spawnResource(ResourceFlare)
	}

	// Reset all players
	for p := range g.players {
		p.Health = MaxHealth
		p.Sanity = MaxSanity
		p.Score = 0
		p.Alive = true
		p.Alive = true
		p.AvailableFlares = 0
		p.HasFlare = false
		p.FlareTime = 0
		p.LightRadius = 3.0 // Base visibility
		p.Pos = Vec2{X: rand.Float64()*20 - 10, Y: rand.Float64()*20 - 10}
	}
}

func (g *Game) spawnResource(resType string) {
	e := &Entity{
		ID:     "r_" + uuid.NewString()[:8],
		Type:   resType,
		Pos:    Vec2{X: rand.Float64()*60 - 30, Y: rand.Float64()*60 - 30},
		Active: true,
	}
	g.entities = append(g.entities, e)
}

func (g *Game) spawnDemogorgon() {
	// Spawn at edge of map
	edge := rand.Intn(4)
	var pos Vec2
	switch edge {
	case 0:
		pos = Vec2{X: -35, Y: rand.Float64()*70 - 35}
	case 1:
		pos = Vec2{X: 35, Y: rand.Float64()*70 - 35}
	case 2:
		pos = Vec2{X: rand.Float64()*70 - 35, Y: -35}
	case 3:
		pos = Vec2{X: rand.Float64()*70 - 35, Y: 35}
	}

	e := &Entity{
		ID:     "d_" + uuid.NewString()[:8],
		Type:   "demogorgon",
		Pos:    pos,
		Active: true,
	}
	g.entities = append(g.entities, e)
}

func (g *Game) update(dt float64) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if !g.gameActive || len(g.players) == 0 {
		return
	}

	g.gameTime += dt
	g.difficulty = 1.0 + g.gameTime/60.0 // Increase difficulty over time

	// Check game end
	if g.gameTime >= GameDuration {
		g.endGame()
		return
	}

	// Spawn demogorgons
	g.spawnTimer -= dt
	if g.spawnTimer <= 0 {
		demoCount := int(g.difficulty) // More demos as game progresses
		for i := 0; i < demoCount; i++ {
			g.spawnDemogorgon()
		}
		g.spawnTimer = DemoSpawnInterval / g.difficulty
	}

	// Occasionally spawn resources
	if rand.Float64() < 0.01 { // ~1% chance per tick
		g.spawnResource(ResourceLightOrb)
	}

	// Update players
	aliveCount := 0
	for p := range g.players {
		if !p.Alive {
			continue
		}
		aliveCount++

		// Check if player is near any light source or has flare
		nearLight := p.HasFlare && p.FlareTime > 0
		if !nearLight {
			for _, e := range g.entities {
				if e.Active && e.Type == ResourceLightOrb {
					dist := distance(p.Pos, e.Pos)
					if dist < 5 {
						nearLight = true
						break
					}
				}
			}
		}

		// Update flare
		if p.FlareTime > 0 {
			p.FlareTime -= dt
			if p.FlareTime <= 0 {
				p.HasFlare = false
				p.LightRadius = 3.0
			}
		}

		// Sanity drain/restore
		if nearLight {
			p.Sanity = math.Min(MaxSanity, p.Sanity+LightRestoreRate*dt)
			p.LightRadius = 5.0
		} else {
			p.Sanity = math.Max(0, p.Sanity-SanityDrainRate*dt*g.difficulty)
			p.LightRadius = math.Max(1.5, 3.0*(p.Sanity/MaxSanity))
		}

		// Health drain when insane
		if p.Sanity <= 0 {
			p.Health -= HealthDrainRate * dt * g.difficulty
			if p.Health <= 0 {
				p.Alive = false
				p.Health = 0
			}
		}

		// Survival score
		p.Score += int(dt * 10 * g.difficulty)
	}

	// Game over if everyone dead
	if aliveCount == 0 && len(g.players) > 0 {
		g.endGame()
		return
	}

	// Update demogorgons - chase nearest player
	for _, e := range g.entities {
		if e.Type != "demogorgon" || !e.Active {
			continue
		}

		// Find nearest alive player
		var nearestPlayer *Player
		nearestDist := math.MaxFloat64
		for p := range g.players {
			if !p.Alive {
				continue
			}
			dist := distance(e.Pos, p.Pos)
			if dist < nearestDist {
				nearestDist = dist
				nearestPlayer = p
			}
		}

		// Skip if stunned
		if e.StunnedUntil > g.gameTime {
			continue
		}

		if nearestPlayer != nil {
			// Move towards player (slower if player has flare)
			speed := 3.0 * g.difficulty
			if nearestPlayer.HasFlare && nearestPlayer.FlareTime > 0 {
				speed = 1.0 // Demogorgons fear light
			}

			dx := nearestPlayer.Pos.X - e.Pos.X
			dy := nearestPlayer.Pos.Y - e.Pos.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			if dist > 0 {
				e.Pos.X += (dx / dist) * speed * dt
				e.Pos.Y += (dy / dist) * speed * dt
			}

			// Attack if close
			if dist < 2 {
				nearestPlayer.Health -= 20 * dt * g.difficulty
				if nearestPlayer.Health <= 0 {
					nearestPlayer.Alive = false
					nearestPlayer.Health = 0
				}
			}
		}
	}

	// Check player-resource collisions
	for p := range g.players {
		if !p.Alive {
			continue
		}
		for _, e := range g.entities {
			if !e.Active || e.Type == "demogorgon" {
				continue
			}
			if distance(p.Pos, e.Pos) < 2 {
				switch e.Type {
				case ResourceLightOrb:
					p.Sanity = math.Min(MaxSanity, p.Sanity+30)
					p.Score += 50
					e.Active = false
				case ResourceBattery:
					p.Health = math.Min(MaxHealth, p.Health+25)
					p.Score += 30
					e.Active = false
				case ResourceFlare:
					p.AvailableFlares++
					p.Score += 100
					e.Active = false
				}
			}
		}
	}

	g.broadcastState()
}

func (g *Game) endGame() {
	g.gameActive = false

	// Calculate rewards
	for p := range g.players {
		if p.UserID == "" || p.UserID == "guest" {
			continue
		}

		// Rewards based on score
		coins := p.Score / 10
		trophies := p.Score / 50
		exp := p.Score / 5

		if p.Alive {
			// Survival bonus
			coins += 100
			trophies += 20
			exp += 200
		}

		g.store.AdjustCoins(p.UserID, coins)
		g.store.AdjustTrophies(p.UserID, trophies)
		g.store.AdjustExp(p.UserID, exp)

		// Award medal for surviving full duration
		if p.Alive && g.gameTime >= GameDuration-1 {
			g.store.AwardMedals(p.UserID, "upside_down_survivor")
		}
	}

	// Send game over
	g.broadcastJSON(map[string]interface{}{
		"type":     "game_over",
		"survived": g.gameTime,
	})
}

func (g *Game) sendWelcome(p *Player) {
	g.sendTo(p, map[string]interface{}{
		"type":     "welcome",
		"id":       p.ID,
		"nickname": p.Nickname,
		"active":   g.gameActive,
	})
}

func (g *Game) broadcastState() {
	// Build player list with sanity-based visibility
	players := make([]map[string]interface{}, 0)
	for p := range g.players {
		players = append(players, map[string]interface{}{
			"id":          p.ID,
			"name":        p.Nickname,
			"pos":         p.Pos,
			"health":      p.Health,
			"sanity":      p.Sanity,
			"score":       p.Score,
			"alive":       p.Alive,
			"hasFlare":    p.HasFlare,
			"flares":      p.AvailableFlares,
			"lightRadius": p.LightRadius,
		})
	}

	// Only send active entities
	entities := make([]map[string]interface{}, 0)
	for _, e := range g.entities {
		if e.Active {
			entities = append(entities, map[string]interface{}{
				"id":   e.ID,
				"type": e.Type,
				"pos":  e.Pos,
			})
		}
	}

	g.broadcastJSON(map[string]interface{}{
		"type":       "state",
		"time":       g.gameTime,
		"maxTime":    GameDuration,
		"difficulty": g.difficulty,
		"players":    players,
		"entities":   entities,
	})
}

func (g *Game) broadcastJSON(v interface{}) {
	data, _ := json.Marshal(v)
	for p := range g.players {
		select {
		case p.Send <- data:
		default:
		}
	}
}

func (g *Game) sendTo(p *Player, v interface{}) {
	data, _ := json.Marshal(v)
	select {
	case p.Send <- data:
	default:
	}
}

func distance(a, b Vec2) float64 {
	dx := b.X - a.X
	dy := b.Y - a.Y
	return math.Sqrt(dx*dx + dy*dy)
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (g *Game) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	userID := r.URL.Query().Get("userID")
	nick := "Stranger"
	if userID != "" {
		if u, ok := g.store.GetUser(userID); ok {
			nick = u.Nickname
		}
	}

	p := &Player{
		ID:          "u_" + uuid.NewString()[:8],
		UserID:      userID,
		Nickname:    nick,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		Pos:         Vec2{X: rand.Float64()*20 - 10, Y: rand.Float64()*20 - 10},
		Health:      MaxHealth,
		Sanity:      MaxSanity,
		Alive:       true,
		LightRadius: 3.0,
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

		g.mu.Lock()
		switch msg["type"] {
		case "move":
			if p.Alive {
				if pos, ok := msg["pos"].(map[string]interface{}); ok {
					p.Pos.X = pos["x"].(float64)
					p.Pos.Y = pos["y"].(float64)
				}
			}
		case "restart":
			if !g.gameActive {
				g.startGame()
			}
		case "use_flare":
			g.handleFlareUse(p)
		case "attack":
			if angle, ok := msg["angle"].(float64); ok {
				g.handleAttack(p, angle)
			}
		}
		g.mu.Unlock()
	}
}

func (g *Game) handleFlareUse(p *Player) {
	if p.Alive && p.AvailableFlares > 0 && !p.HasFlare {
		p.AvailableFlares--
		p.HasFlare = true
		p.FlareTime = 15 // 15 seconds duration
		p.LightRadius = 10.0

		// Stun/Pushback nearby enemies
		for _, e := range g.entities {
			if e.Type == "demogorgon" && e.Active {
				dist := distance(p.Pos, e.Pos)
				if dist < 12 { // Wide range
					e.StunnedUntil = g.gameTime + 5.0 // 5 seconds stun
					// Push back
					dx := e.Pos.X - p.Pos.X
					dy := e.Pos.Y - p.Pos.Y
					e.Pos.X += dx * 2
					e.Pos.Y += dy * 2
				}
			}
		}
	}
}

func (g *Game) handleAttack(p *Player, angle float64) {
	if !p.Alive {
		return
	}

	// Raycast parameters
	reach := 20.0
	ex := p.Pos.X + math.Cos(angle)*reach
	ey := p.Pos.Y + math.Sin(angle)*reach

	for _, e := range g.entities {
		if !e.Active || e.Type != "demogorgon" {
			continue
		}

		// Circle-Line collision check
		dx := ex - p.Pos.X
		dy := ey - p.Pos.Y
		lenSq := dx*dx + dy*dy

		epx := e.Pos.X - p.Pos.X
		epy := e.Pos.Y - p.Pos.Y

		t := (epx*dx + epy*dy) / lenSq
		t = math.Max(0, math.Min(1, t))

		cx := p.Pos.X + t*dx
		cy := p.Pos.Y + t*dy

		// Distance to entity (radius ~1.5)
		distX := e.Pos.X - cx
		distY := e.Pos.Y - cy

		if (distX*distX + distY*distY) < (1.5 * 1.5) {
			// Hit!
			e.Health -= 34 // 3 hits to kill
			if e.Health <= 0 {
				e.Active = false
				p.Score += 100
			}
			break
		}
	}
}
