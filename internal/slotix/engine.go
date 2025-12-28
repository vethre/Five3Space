package slotix

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"main/internal/data"

	"github.com/gorilla/websocket"
)

// Symbols for the slot machine
const (
	SymbolCherry  = "🍒"
	SymbolLemon   = "🍋"
	SymbolOrange  = "🍊"
	SymbolPlum    = "🍇"
	SymbolBell    = "🔔"
	SymbolBar     = "📊"
	SymbolSeven   = "7️⃣"
	SymbolDiamond = "💎"
	SymbolWild    = "⭐"
	SymbolJackpot = "👑"
)

// Symbol weights (higher = more common)
var symbolWeights = []struct {
	Symbol string
	Weight int
}{
	{SymbolCherry, 20},
	{SymbolLemon, 18},
	{SymbolOrange, 16},
	{SymbolPlum, 14},
	{SymbolBell, 10},
	{SymbolBar, 8},
	{SymbolSeven, 6},
	{SymbolDiamond, 4},
	{SymbolWild, 3},
	{SymbolJackpot, 1},
}

// Payouts for 3 matching symbols (multiplier of bet)
var payouts = map[string]int{
	SymbolCherry:  2,
	SymbolLemon:   3,
	SymbolOrange:  4,
	SymbolPlum:    5,
	SymbolBell:    8,
	SymbolBar:     12,
	SymbolSeven:   20,
	SymbolDiamond: 50,
	SymbolWild:    25,
	SymbolJackpot: 100,
}

type Player struct {
	UserID   string
	Nickname string
	Conn     *websocket.Conn
	Send     chan []byte
	mu       sync.Mutex
}

type Game struct {
	mu           sync.Mutex
	store        *data.Store
	players      map[*Player]bool
	register     chan *Player
	unregister   chan *Player
	jackpot      int
	lastSpinTime map[string]time.Time
}

func NewGame(store *data.Store) *Game {
	g := &Game{
		store:        store,
		players:      make(map[*Player]bool),
		register:     make(chan *Player),
		unregister:   make(chan *Player),
		jackpot:      1000, // Starting jackpot
		lastSpinTime: make(map[string]time.Time),
	}
	go g.run()
	return g
}

func (g *Game) run() {
	for {
		select {
		case p := <-g.register:
			g.mu.Lock()
			g.players[p] = true
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
		}
	}
}

func (g *Game) sendWelcome(p *Player) {
	coins := 0
	if p.UserID != "" && p.UserID != "guest" {
		if u, ok := g.store.GetUser(p.UserID); ok {
			coins = u.Coins
		}
	}

	g.sendTo(p, map[string]interface{}{
		"type":     "welcome",
		"coins":    coins,
		"jackpot":  g.jackpot,
		"nickname": p.Nickname,
	})
}

func (g *Game) spin(p *Player, bet int) {
	// Anti-spam: minimum 500ms between spins
	g.mu.Lock()
	lastSpin, exists := g.lastSpinTime[p.UserID]
	if exists && time.Since(lastSpin) < 500*time.Millisecond {
		g.mu.Unlock()
		g.sendTo(p, map[string]interface{}{"type": "error", "msg": "Too fast! Wait a moment."})
		return
	}
	g.lastSpinTime[p.UserID] = time.Now()
	currentJackpot := g.jackpot
	g.mu.Unlock()

	// Validate bet
	if bet < 10 || bet > 1000 {
		g.sendTo(p, map[string]interface{}{"type": "error", "msg": "Bet must be 10-1000"})
		return
	}

	// Check player has enough coins
	if p.UserID == "" || p.UserID == "guest" {
		g.sendTo(p, map[string]interface{}{"type": "error", "msg": "Must be logged in to play"})
		return
	}

	user, ok := g.store.GetUser(p.UserID)
	if !ok || user.Coins < bet {
		g.sendTo(p, map[string]interface{}{"type": "error", "msg": "Not enough coins"})
		return
	}

	// Deduct bet
	g.store.AdjustCoins(p.UserID, -bet)

	// Add 5% of bet to jackpot
	g.mu.Lock()
	g.jackpot += bet / 20
	g.mu.Unlock()

	// Spin the reels (3x3 grid)
	reels := make([][]string, 3)
	for i := 0; i < 3; i++ {
		reels[i] = make([]string, 3)
		for j := 0; j < 3; j++ {
			reels[i][j] = randomSymbol()
		}
	}

	// Calculate winnings
	winAmount := 0
	winLines := []string{}

	// Check middle row (main line)
	if reels[0][1] == reels[1][1] && reels[1][1] == reels[2][1] {
		mult := payouts[reels[0][1]]
		winAmount += bet * mult
		winLines = append(winLines, "middle")
	}

	// Check top row
	if reels[0][0] == reels[1][0] && reels[1][0] == reels[2][0] {
		mult := payouts[reels[0][0]]
		winAmount += bet * mult / 2 // Secondary lines pay half
		winLines = append(winLines, "top")
	}

	// Check bottom row
	if reels[0][2] == reels[1][2] && reels[1][2] == reels[2][2] {
		mult := payouts[reels[0][2]]
		winAmount += bet * mult / 2
		winLines = append(winLines, "bottom")
	}

	// Check diagonals
	if reels[0][0] == reels[1][1] && reels[1][1] == reels[2][2] {
		mult := payouts[reels[0][0]]
		winAmount += bet * mult / 2
		winLines = append(winLines, "diagonal1")
	}
	if reels[0][2] == reels[1][1] && reels[1][1] == reels[2][0] {
		mult := payouts[reels[0][2]]
		winAmount += bet * mult / 2
		winLines = append(winLines, "diagonal2")
	}

	// Check for jackpot (3 jackpot symbols in middle row)
	jackpotWon := false
	if reels[0][1] == SymbolJackpot && reels[1][1] == SymbolJackpot && reels[2][1] == SymbolJackpot {
		winAmount += currentJackpot
		jackpotWon = true
		g.mu.Lock()
		g.jackpot = 1000 // Reset jackpot
		g.mu.Unlock()
	}

	// Wild substitutions - wilds match anything
	// Check middle row with wilds
	if !contains(winLines, "middle") {
		symbols := []string{reels[0][1], reels[1][1], reels[2][1]}
		if matchedSymbol := checkWildMatch(symbols); matchedSymbol != "" {
			mult := payouts[matchedSymbol]
			winAmount += bet * mult / 2 // Wild matches pay half
			winLines = append(winLines, "middle-wild")
		}
	}

	// Award winnings
	if winAmount > 0 {
		g.store.AdjustCoins(p.UserID, winAmount)
	}

	// Get updated balance
	newBalance := 0
	if u, ok := g.store.GetUser(p.UserID); ok {
		newBalance = u.Coins
	}

	g.mu.Lock()
	newJackpot := g.jackpot
	g.mu.Unlock()

	g.sendTo(p, map[string]interface{}{
		"type":       "spin_result",
		"reels":      reels,
		"winAmount":  winAmount,
		"winLines":   winLines,
		"jackpotWon": jackpotWon,
		"newBalance": newBalance,
		"jackpot":    newJackpot,
	})
}

func randomSymbol() string {
	totalWeight := 0
	for _, sw := range symbolWeights {
		totalWeight += sw.Weight
	}

	r := rand.Intn(totalWeight)
	for _, sw := range symbolWeights {
		r -= sw.Weight
		if r < 0 {
			return sw.Symbol
		}
	}
	return SymbolCherry
}

func checkWildMatch(symbols []string) string {
	nonWild := ""
	wildCount := 0
	for _, s := range symbols {
		if s == SymbolWild {
			wildCount++
		} else if nonWild == "" {
			nonWild = s
		} else if s != nonWild {
			return "" // No match
		}
	}
	if wildCount >= 1 && nonWild != "" {
		return nonWild
	}
	return ""
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func (g *Game) sendTo(p *Player, v interface{}) {
	data, _ := json.Marshal(v)
	select {
	case p.Send <- data:
	default:
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func (g *Game) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	userID := r.URL.Query().Get("userID")
	nick := "Guest"
	if userID != "" {
		if u, ok := g.store.GetUser(userID); ok {
			nick = u.Nickname
		}
	}

	p := &Player{
		UserID:   userID,
		Nickname: nick,
		Conn:     conn,
		Send:     make(chan []byte, 256),
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
		case "spin":
			bet := int(msg["bet"].(float64))
			g.spin(p, bet)
		}
	}
}
