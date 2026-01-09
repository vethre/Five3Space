package warthunder

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Country represents a nation with expanded attributes
type Country struct {
	ID             string             `json:"id"`
	Name           string             `json:"name"`
	Color          string             `json:"color"`
	Population     int64              `json:"population"`
	Economy        float64            `json:"economy"`        // GDP in billions
	Military       float64            `json:"military"`       // Strength index
	Stability      float64            `json:"stability"`      // 0-100%
	ApprovalRating float64            `json:"approvalRating"` // 0-100%
	TechLevel      float64            `json:"techLevel"`      // 0-100
	Corruption     float64            `json:"corruption"`     // 0-100%
	Resources      map[string]float64 `json:"resources"`      // oil, food, tech, etc
	Relations      map[string]float64 `json:"relations"`      // -100 to 100
	Alliances      []string           `json:"alliances"`
	Sanctions      []string           `json:"sanctions"` // Countries sanctioning this one
	IsPlayer       bool               `json:"isPlayer"`
	IsEliminated   bool               `json:"isEliminated"`
	Government     string             `json:"government"` // democracy, autocracy, etc
	Ideology       string             `json:"ideology"`   // liberal, conservative, etc
}

// GameState with enhanced features
type GameState struct {
	PlayerID      string              `json:"playerId"`
	PlayerCountry string              `json:"playerCountry"`
	Countries     map[string]*Country `json:"countries"`
	Turn          int                 `json:"turn"`
	Events        []string            `json:"events"`
	GameOver      bool                `json:"gameOver"`
	VictoryType   string              `json:"victoryType"`
	GlobalTension float64             `json:"globalTension"` // 0-100 (higher = more conflict)
	UNSanctions   map[string]int      `json:"unSanctions"`   // Country ID -> severity
	TradeDeals    []TradeDeal         `json:"tradeDeals"`
	Treaties      []Treaty            `json:"treaties"`
	Mutex         sync.RWMutex        `json:"-"`
}

type TradeDeal struct {
	ID        string  `json:"id"`
	Country1  string  `json:"country1"`
	Country2  string  `json:"country2"`
	Resource  string  `json:"resource"`
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"`
	TurnsLeft int     `json:"turnsLeft"`
}

type Treaty struct {
	ID        string   `json:"id"`
	Type      string   `json:"type"` // alliance, non-aggression, trade
	Members   []string `json:"members"`
	TurnsLeft int      `json:"turnsLeft"`
}

var activeGames = make(map[string]*GameState)
var gamesMutex sync.RWMutex

// Enhanced base countries
var baseCountries = []Country{
	{ID: "us", Name: "United States", Color: "#2E86AB", Population: 331000000, Economy: 23000, Military: 1000, Stability: 80, ApprovalRating: 55, TechLevel: 95, Corruption: 30, Government: "democracy", Ideology: "liberal"},
	{ID: "cn", Name: "China", Color: "#D90429", Population: 1400000000, Economy: 18000, Military: 900, Stability: 70, ApprovalRating: 70, TechLevel: 85, Corruption: 50, Government: "autocracy", Ideology: "communist"},
	{ID: "ru", Name: "Russia", Color: "#8D99AE", Population: 144000000, Economy: 1700, Military: 800, Stability: 60, ApprovalRating: 65, TechLevel: 75, Corruption: 70, Government: "autocracy", Ideology: "conservative"},
	{ID: "ua", Name: "Ukraine", Color: "#FFD700", Population: 40000000, Economy: 200, Military: 600, Stability: 75, ApprovalRating: 60, TechLevel: 60, Corruption: 55, Government: "democracy", Ideology: "liberal"},
	{ID: "de", Name: "Germany", Color: "#2B2D42", Population: 83000000, Economy: 4000, Military: 400, Stability: 90, ApprovalRating: 65, TechLevel: 90, Corruption: 20, Government: "democracy", Ideology: "centrist"},
	{ID: "fr", Name: "France", Color: "#EF233C", Population: 67000000, Economy: 2900, Military: 450, Stability: 85, ApprovalRating: 50, TechLevel: 88, Corruption: 25, Government: "democracy", Ideology: "centrist"},
	{ID: "uk", Name: "United Kingdom", Color: "#1B263B", Population: 67000000, Economy: 3100, Military: 450, Stability: 85, ApprovalRating: 52, TechLevel: 87, Corruption: 22, Government: "democracy", Ideology: "conservative"},
	{ID: "jp", Name: "Japan", Color: "#D90429", Population: 125000000, Economy: 5000, Military: 300, Stability: 95, ApprovalRating: 58, TechLevel: 92, Corruption: 18, Government: "democracy", Ideology: "conservative"},
	{ID: "br", Name: "Brazil", Color: "#52B788", Population: 212000000, Economy: 1400, Military: 350, Stability: 50, ApprovalRating: 45, TechLevel: 55, Corruption: 60, Government: "democracy", Ideology: "populist"},
	{ID: "mg", Name: "Madagascar", Color: "#FF9F1C", Population: 27000000, Economy: 14, Military: 50, Stability: 60, ApprovalRating: 40, TechLevel: 30, Corruption: 65, Government: "democracy", Ideology: "centrist"},
	{ID: "za", Name: "South Africa", Color: "#008000", Population: 59000000, Economy: 300, Military: 200, Stability: 45, ApprovalRating: 42, TechLevel: 50, Corruption: 58, Government: "democracy", Ideology: "liberal"},
}

func GetGame(playerID string) *GameState {
	gamesMutex.RLock()
	defer gamesMutex.RUnlock()
	return activeGames[playerID]
}

func CreateGame(playerID string, countryID string) *GameState {
	gamesMutex.Lock()
	defer gamesMutex.Unlock()

	rand.Seed(time.Now().UnixNano())

	countries := make(map[string]*Country)
	for _, c := range baseCountries {
		newC := c
		newC.Relations = make(map[string]float64)
		newC.Resources = map[string]float64{
			"oil":  rand.Float64() * 100,
			"food": rand.Float64() * 100,
			"tech": newC.TechLevel,
		}
		newC.IsPlayer = (c.ID == countryID)
		newC.Alliances = []string{}
		newC.Sanctions = []string{}
		countries[c.ID] = &newC
	}

	// Initialize relations with geopolitical biases
	for _, c1 := range countries {
		for _, c2 := range countries {
			if c1.ID != c2.ID {
				relation := 0.0
				// NATO allies start friendly
				nato := map[string]bool{"us": true, "uk": true, "fr": true, "de": true}
				if nato[c1.ID] && nato[c2.ID] {
					relation = 50.0
				}
				// Russia-West tension
				if (c1.ID == "ru" && nato[c2.ID]) || (nato[c1.ID] && c2.ID == "ru") {
					relation = -40.0
				}
				// China-US rivalry
				if (c1.ID == "us" && c2.ID == "cn") || (c1.ID == "cn" && c2.ID == "us") {
					relation = -20.0
				}
				c1.Relations[c2.ID] = relation
			}
		}
	}

	game := &GameState{
		PlayerID:      playerID,
		PlayerCountry: countryID,
		Countries:     countries,
		Turn:          1,
		Events:        []string{"🎯 Your rule begins. Shape the destiny of your nation!"},
		GlobalTension: 25.0,
		UNSanctions:   make(map[string]int),
		TradeDeals:    []TradeDeal{},
		Treaties:      []Treaty{},
	}

	activeGames[playerID] = game

	// Start AI routine
	go game.AIRoutine()

	return game
}

func (g *GameState) AddEvent(msg string) {
	g.Events = append([]string{fmt.Sprintf("📅 Turn %d: %s", g.Turn, msg)}, g.Events...)
	if len(g.Events) > 100 {
		g.Events = g.Events[:100]
	}
}

// ACTION: Attack with enhanced mechanics
func (g *GameState) Attack(targetID string) string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]
	target, ok := g.Countries[targetID]
	if !ok || target.IsEliminated {
		return "Invalid target"
	}

	// Check if allies will defend
	var defenders []*Country
	for _, allyID := range target.Alliances {
		if ally, ok := g.Countries[allyID]; ok && !ally.IsEliminated {
			defenders = append(defenders, ally)
		}
	}

	defenderNames := ""
	totalDefense := target.Military
	if len(defenders) > 0 {
		defenderNames = " (defended by"
		for i, ally := range defenders {
			totalDefense += ally.Military * 0.5 // Allies contribute 50%
			defenderNames += " " + ally.Name
			if i < len(defenders)-1 {
				defenderNames += ","
			}
		}
		defenderNames += ")"
	}

	g.AddEvent(fmt.Sprintf("⚔️ WAR! You attacked %s%s", target.Name, defenderNames))

	// Combat calculation with more factors
	attackPower := player.Military * (1 + player.TechLevel/200) * (player.Stability / 100)
	defensePower := totalDefense * (1 + target.TechLevel/200) * (target.Stability / 100)

	roll := rand.Float64()
	winChance := attackPower / (attackPower + defensePower)

	if roll < winChance {
		// Victory
		loot := target.Economy * 0.6
		resources := target.Resources["oil"] * 0.7

		player.Economy += loot
		player.Resources["oil"] += resources
		player.Military *= 0.85 // War casualties
		player.ApprovalRating += 15
		player.Stability += 10

		target.IsEliminated = true
		target.Economy = 0

		g.AddEvent(fmt.Sprintf("🏆 VICTORY! Annexed %s. Seized $%.1fB and %.0f oil units", target.Name, loot, resources))

		// World reaction - massive reputation hit
		g.GlobalTension += 30
		for _, c := range g.Countries {
			if c.ID != player.ID && !c.IsEliminated {
				penalty := 40.0
				if c.Government == "democracy" {
					penalty = 50.0 // Democracies hate aggression more
				}
				c.Relations[player.ID] -= penalty
				if c.Relations[player.ID] < -100 {
					c.Relations[player.ID] = -100
				}
			}
		}

		// UN might impose sanctions
		if rand.Float64() < 0.6 {
			g.UNSanctions[player.ID] = 3 // 3 turns
			g.AddEvent("🏛️ UN Security Council imposed sanctions on you!")
			player.Economy *= 0.7
		}

		g.CheckVictoryConditions()
		return "victory"

	} else {
		// Defeat
		loss := player.Economy * 0.3
		militaryLoss := player.Military * 0.4

		player.Economy -= loss
		player.Military -= militaryLoss
		player.ApprovalRating -= 25
		player.Stability -= 20

		g.AddEvent(fmt.Sprintf("💀 DEFEAT! Our invasion failed. Lost $%.1fB and %.0f military strength", loss, militaryLoss))

		// Risk of coup in autocracies
		if player.Government == "autocracy" && player.ApprovalRating < 30 {
			if rand.Float64() < 0.3 {
				g.AddEvent("🚨 COUP D'ÉTAT! You have been overthrown!")
				g.GameOver = true
				g.VictoryType = "defeat"
			}
		}

		target.Relations[player.ID] = -100
		return "defeat"
	}
}

// ACTION: Diplomacy
func (g *GameState) Diplomat(targetID string) string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]
	target, ok := g.Countries[targetID]
	if !ok || target.IsEliminated {
		return "Invalid target"
	}

	cost := 15.0
	if player.Economy < cost {
		return "Insufficient funds for diplomatic mission"
	}

	player.Economy -= cost
	boost := 15.0 + rand.Float64()*15.0

	// Ideology affects diplomacy
	if player.Ideology == target.Ideology {
		boost *= 1.5
	}

	target.Relations[player.ID] += boost
	player.Relations[targetID] += boost * 0.8

	if target.Relations[player.ID] > 100 {
		target.Relations[player.ID] = 100
	}
	if player.Relations[targetID] > 100 {
		player.Relations[targetID] = 100
	}

	g.GlobalTension -= 2
	g.AddEvent(fmt.Sprintf("🤝 Diplomatic mission to %s successful (+%.0f relations)", target.Name, boost))

	return "success"
}

// ACTION: Form Alliance
func (g *GameState) FormAlliance(targetID string) string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]
	target, ok := g.Countries[targetID]
	if !ok || target.IsEliminated {
		return "Invalid target"
	}

	// Check relations threshold
	if target.Relations[player.ID] < 50 {
		return "Relations too low for alliance (need 50+)"
	}

	// Check if already allied
	for _, allyID := range player.Alliances {
		if allyID == targetID {
			return "Already allied"
		}
	}

	player.Alliances = append(player.Alliances, targetID)
	target.Alliances = append(target.Alliances, player.ID)

	treaty := Treaty{
		ID:        fmt.Sprintf("alliance_%d", len(g.Treaties)),
		Type:      "alliance",
		Members:   []string{player.ID, targetID},
		TurnsLeft: -1, // Permanent until broken
	}
	g.Treaties = append(g.Treaties, treaty)

	g.AddEvent(fmt.Sprintf("🛡️ Alliance formed with %s!", target.Name))
	player.Stability += 5

	return "success"
}

// ACTION: Impose Sanctions
func (g *GameState) ImposeSanctions(targetID string) string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]
	target, ok := g.Countries[targetID]
	if !ok || target.IsEliminated {
		return "Invalid target"
	}

	target.Sanctions = append(target.Sanctions, player.ID)
	target.Economy *= 0.9
	target.Resources["tech"] *= 0.85

	player.Relations[targetID] -= 30
	target.Relations[player.ID] -= 40

	g.GlobalTension += 5
	g.AddEvent(fmt.Sprintf("📛 Imposed economic sanctions on %s", target.Name))

	return "success"
}

// ACTION: Espionage
func (g *GameState) Espionage(targetID string) string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]
	target, ok := g.Countries[targetID]
	if !ok || target.IsEliminated {
		return "Invalid target"
	}

	cost := 50.0
	if player.Economy < cost {
		return "Insufficient funds for espionage"
	}

	player.Economy -= cost

	successChance := (player.TechLevel / 100) * (1 - target.TechLevel/200)
	if rand.Float64() < successChance {
		// Success - steal tech or sabotage
		action := rand.Intn(3)
		switch action {
		case 0: // Steal technology
			stolen := target.Resources["tech"] * 0.2
			player.Resources["tech"] += stolen
			player.TechLevel = math.Min(100, player.TechLevel+3)
			g.AddEvent(fmt.Sprintf("🕵️ Espionage successful! Stole %.0f tech units from %s", stolen, target.Name))
		case 1: // Economic sabotage
			damage := target.Economy * 0.15
			target.Economy -= damage
			target.Stability -= 10
			g.AddEvent(fmt.Sprintf("🕵️ Sabotage successful! Damaged %s's economy by $%.1fB", target.Name, damage))
		case 2: // Lower stability
			target.Stability -= 15
			target.ApprovalRating -= 10
			g.AddEvent(fmt.Sprintf("🕵️ Covert ops successful! Destabilized %s's government", target.Name))
		}
		return "success"
	} else {
		// Caught!
		g.AddEvent(fmt.Sprintf("🚨 EXPOSED! Our spies were caught in %s", target.Name))
		target.Relations[player.ID] -= 50
		player.ApprovalRating -= 15
		g.GlobalTension += 10
		return "caught"
	}
}

// ACTION: Invest in Economy
func (g *GameState) InvestEconomy() string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]

	cost := player.Economy * 0.1
	if cost < 50 {
		return "Economy too small for effective investment"
	}

	player.Economy -= cost

	// Investment pays off over time
	returns := cost * 1.3
	player.Economy += returns
	player.Stability += 5
	player.ApprovalRating += 8

	g.AddEvent(fmt.Sprintf("📈 Economic investment successful! GDP increased by $%.1fB", returns-cost))

	return "success"
}

// ACTION: Military Buildup
func (g *GameState) BuildMilitary() string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]

	cost := 100.0
	if player.Economy < cost {
		return "Insufficient funds for military buildup"
	}

	player.Economy -= cost
	increase := 50 + rand.Float64()*50
	player.Military += increase

	// Democracies lose approval for militarization
	if player.Government == "democracy" {
		player.ApprovalRating -= 5
	}

	g.GlobalTension += 3
	g.AddEvent(fmt.Sprintf("🎖️ Military expanded by %.0f units", increase))

	return "success"
}

// ACTION: Propaganda Campaign
func (g *GameState) Propaganda() string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]

	cost := 30.0
	if player.Economy < cost {
		return "Insufficient funds for propaganda"
	}

	player.Economy -= cost
	boost := 10 + rand.Float64()*15
	player.ApprovalRating += boost

	if player.ApprovalRating > 100 {
		player.ApprovalRating = 100
	}

	g.AddEvent(fmt.Sprintf("📢 Propaganda campaign boosted approval by %.0f%%", boost))

	return "success"
}

// ACTION: Anti-Corruption Drive
func (g *GameState) FightCorruption() string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	player := g.Countries[g.PlayerCountry]

	cost := 50.0
	if player.Economy < cost {
		return "Insufficient funds"
	}

	player.Economy -= cost
	reduction := 10 + rand.Float64()*15
	player.Corruption -= reduction

	if player.Corruption < 0 {
		player.Corruption = 0
	}

	player.ApprovalRating += 8
	player.Stability += 6

	g.AddEvent(fmt.Sprintf("⚖️ Anti-corruption reforms reduced corruption by %.0f%%", reduction))

	return "success"
}

// AI Routine - makes AI countries take actions
func (g *GameState) AIRoutine() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		g.Mutex.Lock()

		if g.GameOver {
			g.Mutex.Unlock()
			return
		}

		// AI countries take actions
		for _, country := range g.Countries {
			if country.IsPlayer || country.IsEliminated {
				continue
			}

			action := rand.Intn(10)
			switch action {
			case 0, 1: // Economic investment
				if country.Economy > 200 {
					country.Economy *= 1.05
				}
			case 2: // Military buildup
				if country.Economy > 150 && country.Stability > 40 {
					country.Economy -= 80
					country.Military += 30 + rand.Float64()*40
					g.GlobalTension += 1
				}
			case 3: // Improve relations with random country
				for targetID := range g.Countries {
					target := g.Countries[targetID]
					if targetID != country.ID && !target.IsEliminated {
						country.Relations[targetID] += 5
						break
					}
				}
			case 4: // Form alliance
				for targetID, relation := range country.Relations {
					if relation > 60 && rand.Float64() < 0.1 {
						target := g.Countries[targetID]
						if !target.IsEliminated {
							country.Alliances = append(country.Alliances, targetID)
							target.Alliances = append(target.Alliances, country.ID)
							g.AddEvent(fmt.Sprintf("🌍 %s and %s formed an alliance", country.Name, target.Name))
							break
						}
					}
				}
			}

			// Random events affect AI countries
			if rand.Float64() < 0.05 {
				event := rand.Intn(5)
				switch event {
				case 0:
					country.Stability -= 10
					g.AddEvent(fmt.Sprintf("📰 Political crisis in %s", country.Name))
				case 1:
					country.Economy *= 1.1
					g.AddEvent(fmt.Sprintf("📰 Economic boom in %s", country.Name))
				case 2:
					country.ApprovalRating -= 15
					g.AddEvent(fmt.Sprintf("📰 Protests erupted in %s", country.Name))
				}
			}
		}

		g.Mutex.Unlock()
	}
}

// Check various victory conditions
func (g *GameState) CheckVictoryConditions() {
	player := g.Countries[g.PlayerCountry]

	// Count non-eliminated countries
	alive := 0
	for _, c := range g.Countries {
		if !c.IsEliminated {
			alive++
		}
	}

	// Domination victory - only you remain
	if alive == 1 {
		g.GameOver = true
		g.VictoryType = "domination"
		g.AddEvent("🏆 DOMINATION VICTORY! You rule the world!")
		return
	}

	// Economic victory - massive GDP
	if player.Economy > 50000 {
		g.GameOver = true
		g.VictoryType = "economic"
		g.AddEvent("🏆 ECONOMIC VICTORY! Your economy dominates the world!")
		return
	}

	// Diplomatic victory - many allies
	if len(player.Alliances) >= 6 {
		g.GameOver = true
		g.VictoryType = "diplomatic"
		g.AddEvent("🏆 DIPLOMATIC VICTORY! You united the world in alliance!")
		return
	}

	// Tech victory
	if player.TechLevel >= 100 && player.Resources["tech"] > 1000 {
		g.GameOver = true
		g.VictoryType = "technological"
		g.AddEvent("🏆 TECHNOLOGICAL VICTORY! Your advanced civilization leads humanity!")
		return
	}
}

// Advance Turn
func (g *GameState) NextTurn() string {
	g.Mutex.Lock()
	defer g.Mutex.Unlock()

	g.Turn++
	player := g.Countries[g.PlayerCountry]

	// Economic growth
	growthRate := 0.02 * (player.Stability / 100) * (1 - player.Corruption/200)
	player.Economy *= (1 + growthRate)

	// Resource production
	player.Resources["oil"] += 5 + rand.Float64()*10
	player.Resources["food"] += 8 + rand.Float64()*12

	// UN sanctions wear off
	if g.UNSanctions[player.ID] > 0 {
		g.UNSanctions[player.ID]--
		if g.UNSanctions[player.ID] == 0 {
			g.AddEvent("🏛️ UN sanctions have been lifted")
		}
	}

	// Stability effects
	if player.Stability < 30 {
		player.ApprovalRating -= 5
		if rand.Float64() < 0.1 {
			g.AddEvent("🚨 Civil unrest! Rebels causing havoc")
			player.Economy *= 0.95
		}
	}

	// Random world events
	if rand.Float64() < 0.15 {
		g.TriggerRandomEvent()
	}

	g.AddEvent(fmt.Sprintf("📅 Turn %d complete. Economy: $%.1fB, Military: %.0f", g.Turn, player.Economy, player.Military))

	return "success"
}

func (g *GameState) TriggerRandomEvent() {
	events := []func(){
		func() {
			g.AddEvent("🌍 Global economic recession! All economies drop 10%")
			for _, c := range g.Countries {
				c.Economy *= 0.9
			}
		},
		func() {
			g.AddEvent("⚡ Technological breakthrough! All tech levels increase")
			for _, c := range g.Countries {
				c.TechLevel += 2
			}
		},
		func() {
			g.AddEvent("🌾 Global food crisis! Resource production affected")
			for _, c := range g.Countries {
				c.Stability -= 5
			}
		},
		func() {
			g.AddEvent("☮️ Peace movements worldwide! Global tension decreases")
			g.GlobalTension -= 15
			if g.GlobalTension < 0 {
				g.GlobalTension = 0
			}
		},
	}

	if rand.Float64() < 0.3 {
		event := events[rand.Intn(len(events))]
		event()
	}
}
