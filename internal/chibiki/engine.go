package chibiki

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	TickRate   = 30
	LaneLeftX  = 3.5
	LaneRightX = 14.5
	BridgeY    = 16.0

	// Game Duration Settings
	DurationNormal   = 120.0 // 2 Minutes
	DurationOvertime = 90.0  // 1:30 Minutes
)

type PlayerState struct {
	Elixir float64  `json:"elixir"`
	Hand   []string `json:"hand"`
	Next   string   `json:"next"`
	Deck   []string `json:"-"`
}

type GameInstance struct {
	Entities     []*Entity
	UnitData     map[string]UnitStats
	PlayerStates map[string]*PlayerState
	GameTime     float64
	Mutex        sync.RWMutex
	Register     chan *Player
	Unregister   chan *Player
	nextTeamID   int
	Players      map[*Player]bool

	OnGameOver func(winnerTeam int, players map[*Player]bool)

	// Game State Flags
	GameOver     bool
	WinnerTeam   int
	IsOvertime   bool
	IsTiebreaker bool

	resultSent bool
}

func NewGame() *GameInstance {
	return &GameInstance{
		Entities:     make([]*Entity, 0),
		UnitData:     make(map[string]UnitStats),
		PlayerStates: make(map[string]*PlayerState),
		Register:     make(chan *Player),
		Unregister:   make(chan *Player),
		Players:      make(map[*Player]bool),
		nextTeamID:   0,
		GameTime:     0,
		GameOver:     false,
		WinnerTeam:   -1,
		resultSent:   false,
	}
}

// --- NEW: Reset Function for "Play Again" ---
func (g *GameInstance) Reset() {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	// Reset Game State
	g.Entities = make([]*Entity, 0)
	g.GameTime = 0
	g.GameOver = false
	g.WinnerTeam = -1
	g.IsOvertime = false
	g.IsTiebreaker = false
	g.resultSent = false

	// Reset Players (Elixir, Hands)
	for pID := range g.PlayerStates {
		deck := []string{"morphilina", "dangerlyoha", "yuuechka", "morphe", "classic_morphe", "classic_yuu", "sasavot", "murzik"}
		rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
		g.PlayerStates[pID] = &PlayerState{5.0, deck[:4], deck[4], deck[5:]}
	}

	// Respawn Towers
	g.InitTowersInternal()
}

// Separated InitTowers so we can call it from Reset (Internal use only, assumes lock held if called from Reset)
func (g *GameInstance) InitTowersInternal() {
	g.spawnEntityInternal("king_tower", "server", 0, 9, 29)
	g.spawnEntityInternal("princess_tower", "server", 0, 3.5, 26)
	g.spawnEntityInternal("princess_tower", "server", 0, 14.5, 26)
	g.spawnEntityInternal("king_tower", "server", 1, 9, 3)
	g.spawnEntityInternal("princess_tower", "server", 1, 3.5, 6)
	g.spawnEntityInternal("princess_tower", "server", 1, 14.5, 6)
}

// Public version (locks mutex)
func (g *GameInstance) InitTowers() {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()
	g.InitTowersInternal()
}

func (g *GameInstance) InitPlayer(playerID string) {
	deck := []string{"morphilina", "dangerlyoha", "yuuechka", "morphe", "classic_morphe", "classic_yuu", "sasavot", "murzik"}
	rand.Shuffle(len(deck), func(i, j int) { deck[i], deck[j] = deck[j], deck[i] })
	g.PlayerStates[playerID] = &PlayerState{5.0, deck[:4], deck[4], deck[5:]}
}

func (g *GameInstance) LoadUnits(path string) error {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var data struct {
		Units map[string]UnitStats `json:"units"`
	}
	json.Unmarshal(bytes, &data)
	g.UnitData = data.Units
	for k, v := range g.UnitData {
		v.Key = k
		g.UnitData[k] = v
	}
	g.UnitData["king_tower"] = UnitStats{Key: "king_tower", HP: 4000, Range: 7, Damage: 100, HitSpeed: 1, Speed: 0, Target: "ground"}
	g.UnitData["princess_tower"] = UnitStats{Key: "princess_tower", HP: 2500, Range: 7.5, Damage: 80, HitSpeed: 0.8, Speed: 0, Target: "ground"}
	return nil
}

func (g *GameInstance) StartLoop() {
	go g.handleConnections()
	ticker := time.NewTicker(time.Second / TickRate)
	defer ticker.Stop()
	for range ticker.C {
		dt := 1.0 / float64(TickRate)
		g.Update(dt)
		g.BroadcastCustomState()
	}
}

func (g *GameInstance) handleConnections() {
	for {
		select {
		case player := <-g.Register:
			g.Mutex.Lock()
			g.Players[player] = true
			if _, exists := g.PlayerStates[player.ID]; !exists {
				player.Team = g.nextTeamID
				g.nextTeamID = (g.nextTeamID + 1) % 2
				g.InitPlayer(player.ID)
			}
			g.Mutex.Unlock()
			fmt.Println("Player joined:", player.ID)
		case player := <-g.Unregister:
			g.Mutex.Lock()
			delete(g.Players, player)
			close(player.Send)
			g.Mutex.Unlock()
			fmt.Println("Player left")
		}
	}
}

func (g *GameInstance) Update(dt float64) {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()
	if g.GameOver {
		return
	}

	g.GameTime += dt

	if !g.IsOvertime && !g.IsTiebreaker {
		if g.GameTime >= DurationNormal {
			g.IsOvertime = true
		}
	} else if g.IsOvertime && !g.IsTiebreaker {
		if g.GameTime >= DurationNormal+DurationOvertime {
			g.IsTiebreaker = true
		}
	}

	if g.IsTiebreaker {
		drain := 50.0 * dt
		for _, e := range g.Entities {
			if e.Key == "king_tower" || e.Key == "princess_tower" {
				e.HP -= drain
				if e.HP <= 0 {
					e.HP = 0
					g.finishGame((e.Team + 1) % 2)
				}
			}
		}
		return
	}

	rate := 1.0 / 2.8
	if g.GameTime > 120 {
		rate *= 2
	}
	for _, pState := range g.PlayerStates {
		if pState.Elixir < 10 {
			pState.Elixir += rate * dt
			if pState.Elixir > 10 {
				pState.Elixir = 10
			}
		}
	}

	// Count Towers for King Activation Logic
	activeEntities := g.Entities[:0]
	towersTeam0 := 0
	towersTeam1 := 0

	// Mark if a tower drops during overtime/tiebreaker for sudden death.
	suddenDeath := g.IsOvertime || g.IsTiebreaker
	for _, e := range g.Entities {
		if e.HP > 0 {
			activeEntities = append(activeEntities, e)
			if e.Key == "princess_tower" {
				if e.Team == 0 {
					towersTeam0++
				} else {
					towersTeam1++
				}
			}
		} else {
			if e.Key == "king_tower" {
				g.finishGame((e.Team + 1) % 2)
			} else if suddenDeath && (e.Key == "princess_tower") {
				g.finishGame((e.Team + 1) % 2)
			}
		}
	}
	g.Entities = activeEntities
	if g.GameOver {
		return
	}

	now := float64(time.Now().UnixMilli()) / 1000.0
	for _, e := range g.Entities {
		if e.StunnedUntil > now {
			continue
		}

		// --- UPDATED: Tower Attack Logic ---
		// Towers are buildings (Speed 0) but have Damage > 0
		isTower := e.Key == "king_tower" || e.Key == "princess_tower"

		// King Activation Check
		if e.Key == "king_tower" {
			// King activates if: HP < Max OR Friend Princess Tower is dead (count < 2)
			friendTowers := towersTeam0
			if e.Team == 1 {
				friendTowers = towersTeam1
			}

			if e.HP >= e.MaxHP && friendTowers >= 2 {
				continue // King sleeps
			}
		}

		// Skip movement for buildings, but allow attacking
		if e.Stats.Speed == 0 && !isTower {
			continue
		}

		target := g.FindTarget(e)
		if target != nil {
			dist := g.Distance(e, target)
			if dist <= e.Stats.Range+0.5 {
				if now-e.LastAttack >= e.Stats.HitSpeed {
					g.Attack(e, target)
					e.LastAttack = now
				}
			} else if e.Stats.Speed > 0 {
				g.MoveTowards(e, target.X, target.Y, dt)
			}
		} else if e.Stats.Speed > 0 {
			g.MoveDownLane(e, dt)
		}
	}
}

func (g *GameInstance) BroadcastCustomState() {
	g.Mutex.RLock()
	defer g.Mutex.RUnlock()

	type stateMessage struct {
		Type       string       `json:"type"`
		Entities   []*Entity    `json:"entities"`
		Time       float64      `json:"time"`
		GameOver   bool         `json:"gameOver"`
		Winner     int          `json:"winner"`
		Overtime   bool         `json:"overtime"`
		Tiebreaker bool         `json:"tiebreaker"`
		Me         *PlayerState `json:"me,omitempty"`
		MyTeam     int          `json:"myTeam,omitempty"`
	}

	base := stateMessage{
		Type:       "state",
		Entities:   g.Entities,
		Time:       g.GameTime,
		GameOver:   g.GameOver,
		Winner:     g.WinnerTeam,
		Overtime:   g.IsOvertime,
		Tiebreaker: g.IsTiebreaker,
	}

	for player := range g.Players {
		pState := g.PlayerStates[player.ID]
		msg := base
		msg.Me = pState
		msg.MyTeam = player.Team
		data, _ := json.Marshal(msg)
		select {
		case player.Send <- data:
		default:
			close(player.Send)
			delete(g.Players, player)
		}
	}
}

func (g *GameInstance) SpawnUnit(player *Player, key string, x, y float64) {
	// Anti-Cheat: Validation
	if (player.Team == 0 && y < BridgeY) || (player.Team == 1 && y > BridgeY) {
		return
	}

	ownerID := player.ID
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	stats, ok := g.UnitData[key]
	if !ok {
		return
	}
	pState, ok := g.PlayerStates[player.ID]
	if !ok {
		return
	}

	cost := float64(stats.Elixir)
	if pState.Elixir < cost {
		return
	}

	cardIdx := -1
	for i, card := range pState.Hand {
		if card == key {
			cardIdx = i
			break
		}
	}
	if cardIdx == -1 {
		return
	}

	pState.Elixir -= cost
	pState.Hand[cardIdx] = pState.Next
	if len(pState.Deck) > 0 {
		pState.Next = pState.Deck[0]
		pState.Deck = pState.Deck[1:]
		pState.Deck = append(pState.Deck, key)
	}
	g.SpawnEntity(key, ownerID, player.Team, x, y)
}

// --- Helper Functions ---
// Internal spawn used by Reset/InitTowers
func (g *GameInstance) spawnEntityInternal(key, ownerID string, team int, x, y float64) {
	stats := g.UnitData[key]
	e := &Entity{
		ID:  fmt.Sprintf("%d-%f", time.Now().UnixNano(), rand.Float64()),
		Key: key, OwnerID: ownerID, Team: team, X: x, Y: y,
		HP: stats.HP, MaxHP: stats.HP, Stats: stats, LastAttack: 0,
	}
	g.Entities = append(g.Entities, e)
}

func (g *GameInstance) SpawnEntity(key, ownerID string, team int, x, y float64) {
	g.spawnEntityInternal(key, ownerID, team, x, y)
}

func (g *GameInstance) FindTarget(e *Entity) *Entity {
	var closest *Entity
	minDist := 1000.0
	sightRange := 6.5
	if e.Key == "princess_tower" || e.Key == "king_tower" {
		sightRange = e.Stats.Range
	} // Towers see exactly their range

	for _, other := range g.Entities {
		if other.Team == e.Team || other.HP <= 0 {
			continue
		}
		dist := g.Distance(e, other)
		if dist < sightRange && dist < minDist {
			minDist = dist
			closest = other
		}
	}
	return closest
}
func (g *GameInstance) Distance(e1, e2 *Entity) float64 { return math.Hypot(e2.X-e1.X, e2.Y-e1.Y) }
func (g *GameInstance) Attack(attacker, target *Entity) { target.HP -= attacker.Stats.Damage }
func (g *GameInstance) MoveTowards(e *Entity, tx, ty, dt float64) {
	dx := tx - e.X
	dy := ty - e.Y
	dist := math.Hypot(dx, dy)
	if dist > 0.1 {
		move := e.Stats.Speed * dt
		e.X += (dx / dist) * move
		e.Y += (dy / dist) * move
	}
}
func (g *GameInstance) MoveDownLane(e *Entity, dt float64) {
	targetX := LaneRightX
	if e.X < 9 {
		targetX = LaneLeftX
	}
	towerY := 3.0
	if e.Team == 1 {
		towerY = 29.0
	}
	onMySide := (e.Team == 0 && e.Y > BridgeY) || (e.Team == 1 && e.Y < BridgeY)
	if onMySide && math.Abs(e.Y-BridgeY) > 1.0 {
		g.MoveTowards(e, targetX, BridgeY, dt)
	} else {
		g.MoveTowards(e, targetX, towerY, dt)
	}
}

func (g *GameInstance) finishGame(winningTeam int) {
	if g.GameOver || g.resultSent {
		return
	}
	g.GameOver = true
	g.WinnerTeam = winningTeam
	g.resultSent = true

	if g.OnGameOver != nil {
		playersCopy := make(map[*Player]bool, len(g.Players))
		for p := range g.Players {
			playersCopy[p] = true
		}
		go g.OnGameOver(winningTeam, playersCopy)
	}
}
