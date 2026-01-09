package warthunder

import (
	"fmt"
	"math/rand"
	"sync"
)

// Country represents a nation in the game
type Country struct {
	ID           string             `json:"id"`
	Name         string             `json:"name"`
	Color        string             `json:"color"`
	Population   int64              `json:"population"`
	Economy      float64            `json:"economy"`   // Billions
	Military     float64            `json:"military"`  // Strength index
	Stability    float64            `json:"stability"` // 0-100%
	Relations    map[string]float64 `json:"relations"` // -100 to 100
	IsPlayer     bool               `json:"isPlayer"`
	IsEliminated bool               `json:"isEliminated"`
}

// GameState holds the current session data
type GameState struct {
	PlayerID      string              `json:"playerId"`
	PlayerCountry string              `json:"playerCountry"`
	Countries     map[string]*Country `json:"countries"`
	Turn          int                 `json:"turn"`
	Events        []string            `json:"events"`
	GameOver      bool                `json:"gameOver"`
	Mutex         sync.RWMutex        `json:"-"`
}

var activeGames = make(map[string]*GameState)
var gamesMutex sync.RWMutex

// Initial Countries Data
var baseCountries = []Country{
	{ID: "us", Name: "United States", Color: "#2E86AB", Population: 331000000, Economy: 23000, Military: 1000, Stability: 80},
	{ID: "cn", Name: "China", Color: "#D90429", Population: 1400000000, Economy: 18000, Military: 900, Stability: 70},
	{ID: "ru", Name: "Russia", Color: "#8D99AE", Population: 144000000, Economy: 1700, Military: 800, Stability: 60},
	{ID: "ua", Name: "Ukraine", Color: "#FFD700", Population: 40000000, Economy: 200, Military: 600, Stability: 75}, // Boosted military for game balance/fun
	{ID: "de", Name: "Germany", Color: "#2B2D42", Population: 83000000, Economy: 4000, Military: 400, Stability: 90},
	{ID: "fr", Name: "France", Color: "#EF233C", Population: 67000000, Economy: 2900, Military: 450, Stability: 85},
	{ID: "uk", Name: "United Kingdom", Color: "#1B263B", Population: 67000000, Economy: 3100, Military: 450, Stability: 85},
	{ID: "jp", Name: "Japan", Color: "#D90429", Population: 125000000, Economy: 5000, Military: 300, Stability: 95},
	{ID: "br", Name: "Brazil", Color: "#52B788", Population: 212000000, Economy: 1400, Military: 350, Stability: 50},
	{ID: "mg", Name: "Madagascar", Color: "#FF9F1C", Population: 27000000, Economy: 14, Military: 50, Stability: 60}, // Requested
	{ID: "za", Name: "South Africa", Color: "#008000", Population: 59000000, Economy: 300, Military: 200, Stability: 45},
}

func GetGame(playerID string) *GameState {
	gamesMutex.RLock()
	defer gamesMutex.RUnlock()
	return activeGames[playerID]
}

func CreateGame(playerID string, countryID string) *GameState {
	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	countries := make(map[string]*Country)
	for _, c := range baseCountries {
		newC := c
		newC.Relations = make(map[string]float64)
		newC.IsPlayer = (c.ID == countryID)
		countries[c.ID] = &newC
	}

	// Init relations (neutral default)
	for _, c1 := range countries {
		for _, c2 := range countries {
			if c1.ID != c2.ID {
				c1.Relations[c2.ID] = 0
			}
		}
	}

	game := &GameState{
		PlayerID:      playerID,
		PlayerCountry: countryID,
		Countries:     countries,
		Turn:          1,
		Events:        []string{"Welcome, Leader. Your rule begins now."},
	}
	activeGames[playerID] = game
	return game
}

func (g *GameState) AddEvent(msg string) {
	g.Events = append([]string{fmt.Sprintf("Turn %d: %s", g.Turn, msg)}, g.Events...)
	if len(g.Events) > 50 {
		g.Events = g.Events[:50]
	}
}

// Action: Attack
func (g *GameState) Attack(targetID string) string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]
	target, ok := g.Countries[targetID]
	if !ok || target.IsEliminated {
		return "Target not found or already eliminated."
	}

	g.AddEvent(fmt.Sprintf("You declared WAR on %s!", target.Name))

	// Simple Combat Logic
	// Random factor + Military Strength difference
	roll := rand.Float64() * (player.Military + target.Military)

	if roll < player.Military {
		// Player Wins
		loot := target.Economy * 0.5
		player.Economy += loot
		player.Stability += 5
		target.IsEliminated = true
		target.Economy = 0
		g.AddEvent(fmt.Sprintf("Victory! You annexed %s and seized $%.1fB.", target.Name, loot))

		// World Reaction
		for _, c := range g.Countries {
			if c.ID != player.ID && !c.IsEliminated {
				c.Relations[player.ID] -= 30
				if c.Relations[player.ID] < -100 {
					c.Relations[player.ID] = -100
				}
			}
		}
		return "victory"
	} else {
		// Player Loses
		loss := player.Economy * 0.2
		player.Economy -= loss
		player.Military *= 0.8
		player.Stability -= 15
		g.AddEvent(fmt.Sprintf("Defeat! Our forces were repelled by %s. Lost $%.1fB and military strength.", target.Name, loss))

		// Relations Worsen
		target.Relations[player.ID] = -100
		return "defeat"
	}
}

// Action: Improve Relations
func (g *GameState) Diplomat(targetID string) string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]
	target, ok := g.Countries[targetID]
	if !ok {
		return "Invalid target"
	}

	cost := 10.0
	if player.Economy < cost {
		return "Insufficient funds"
	}
	player.Economy -= cost

	boost := 10.0 + rand.Float64()*10.0
	target.Relations[player.ID] += boost
	if target.Relations[player.ID] > 100 {
		target.Relations[player.ID] = 100
	}
	g.AddEvent(fmt.Sprintf("Diplomatic mission to %s successful. Relations improved by %.1f.", target.Name, boost))
	return "success"
}
