package party

import (
	"encoding/json"
	"fmt"
	"main/internal/data"
	"math/rand"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	MinPlayers    = 2
	MaxPlayers    = 8
	RoundDuration = 30 // Seconds to write answer
	VoteDuration  = 15 // Seconds to vote
	TotalRounds   = 3
)

var Prompts = []string{
	"Самое худшее, что можно сказать на похоронах.",
	"Отвергнутое название для нового цвета карандаша.",
	"Что на самом деле убило динозавров?",
	"Плохой ледокол для первого свидания.",
	"Самый бесполезный супергерой: Человек-...",
	"Что нельзя говорить полицейскому?",
	"Лучший способ расстаться с девушкой.",
	"Надпись на твоем надгробии.",
	"Причина, по которой тебя уволили с работы мечты.",
}

type Player struct {
	ID       string
	UserID   string
	Nickname string
	Score    int
	Conn     *websocket.Conn
	Send     chan []byte
	Answer   string
	Voted    bool
}

type Game struct {
	mu         sync.Mutex
	store      *data.Store
	players    map[string]*Player
	register   chan *Player
	unregister chan *Player
	broadcast  chan []byte

	state         string // "LOBBY", "INPUT", "VOTING", "RESULT", "GAME_OVER"
	round         int
	timer         int
	currentPrompt string

	// Voting Logic
	answers    []*Player // List of players who answered
	matchIndex int       // Current pair index being voted on
	matchA     *Player
	matchB     *Player
	votesA     int
	votesB     int
}

func NewGame(store *data.Store) *Game {
	g := &Game{
		store:      store,
		players:    make(map[string]*Player),
		register:   make(chan *Player),
		unregister: make(chan *Player),
		broadcast:  make(chan []byte),
		state:      "LOBBY",
	}
	go g.run()
	return g
}

func (g *Game) run() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case p := <-g.register:
			g.mu.Lock()
			if g.state != "LOBBY" || len(g.players) >= MaxPlayers {
				p.Conn.Close() // Reject if game started or full
			} else {
				g.players[p.ID] = p
				g.broadcastState()
			}
			g.mu.Unlock()

		case p := <-g.unregister:
			g.mu.Lock()
			if _, ok := g.players[p.ID]; ok {
				delete(g.players, p.ID)
				close(p.Send)
				// If game is running and players drop below min, reset
				if len(g.players) < MinPlayers && g.state != "LOBBY" {
					g.resetGame()
				} else {
					g.broadcastState()
				}
			}
			g.mu.Unlock()

		case msg := <-g.broadcast:
			for _, p := range g.players {
				select {
				case p.Send <- msg:
				default:
					close(p.Send)
					delete(g.players, p.ID)
				}
			}

		case <-ticker.C:
			g.tick()
		}
	}
}

func (g *Game) tick() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.timer > 0 {
		g.timer--
		if g.timer == 0 {
			g.nextPhase()
		}
	}
}

func (g *Game) nextPhase() {
	switch g.state {
	case "INPUT":
		g.state = "VOTING"
		g.startVotingPhase()
	case "VOTING":
		g.resolveVote()
	case "RESULT":
		g.round++
		if g.round > TotalRounds {
			g.endGame()
		} else {
			g.startRound()
		}
	}
	g.broadcastState()
}

func (g *Game) startRound() {
	g.state = "INPUT"
	g.timer = RoundDuration
	g.currentPrompt = Prompts[rand.Intn(len(Prompts))]
	for _, p := range g.players {
		p.Answer = ""
		p.Voted = false
	}
}

func (g *Game) startVotingPhase() {
	g.answers = make([]*Player, 0)
	for _, p := range g.players {
		if p.Answer != "" {
			g.answers = append(g.answers, p)
		}
	}

	rand.Shuffle(len(g.answers), func(i, j int) {
		g.answers[i], g.answers[j] = g.answers[j], g.answers[i]
	})

	g.matchIndex = 0
	g.nextMatch()
}

func (g *Game) nextMatch() {
	if g.matchIndex+1 < len(g.answers) {
		g.state = "VOTING"
		g.matchA = g.answers[g.matchIndex]
		g.matchB = g.answers[g.matchIndex+1]
		g.votesA = 0
		g.votesB = 0
		g.timer = VoteDuration
		g.matchIndex += 2

		for _, p := range g.players {
			p.Voted = false
		}
	} else {
		g.state = "RESULT"
		g.timer = 10
	}
}

func (g *Game) resolveVote() {
	pointsA := g.votesA * 100
	pointsB := g.votesB * 100

	if g.votesA > g.votesB {
		pointsA += 250
	} else if g.votesB > g.votesA {
		pointsB += 250
	}

	g.matchA.Score += pointsA
	g.matchB.Score += pointsB

	g.nextMatch()
}

func (g *Game) endGame() {
	g.state = "GAME_OVER"
	g.timer = 0

	ranking := make([]*Player, 0, len(g.players))
	for _, p := range g.players {
		ranking = append(ranking, p)
	}
	sort.Slice(ranking, func(i, j int) bool {
		return ranking[i].Score > ranking[j].Score
	})

	playerCount := len(ranking)

	for rank, p := range ranking {
		if p.UserID == "guest" || p.UserID == "" {
			continue
		}

		// Default: "For others -Trophies and +Some Coins +XP"
		trophies := -5
		coins := 20
		exp := 50

		if playerCount <= 3 {
			// "if players 2-3 then only TOP-1 will receive reward"
			if rank == 0 { // Top 1
				trophies = 30
				coins = 200
				exp = 300
				g.store.AwardMedals(p.UserID, "party_king")
			}
		} else {
			// "For TOP-1, TOP-2 and TOP-3 players there will be +Trophies, +Coins and +XP"
			if rank == 0 { // Top 1
				trophies = 50
				coins = 300
				exp = 500
				g.store.AwardMedals(p.UserID, "party_king")
			} else if rank == 1 { // Top 2
				trophies = 25
				coins = 150
				exp = 250
			} else if rank == 2 { // Top 3
				trophies = 10
				coins = 75
				exp = 150
			}
		}

		g.store.AdjustTrophies(p.UserID, trophies)
		g.store.AdjustCoins(p.UserID, coins)
		g.store.AdjustExp(p.UserID, exp)
	}
}

func (g *Game) resetGame() {
	g.state = "LOBBY"
	g.round = 0
	g.timer = 0
	for _, p := range g.players {
		p.Score = 0
		p.Answer = ""
	}
	g.broadcastState()
}

func (g *Game) broadcastState() {
	type PlayerView struct {
		ID          string `json:"id"`
		Nickname    string `json:"name"`
		Score       int    `json:"score"`
		HasAnswered bool   `json:"answered"`
	}

	pList := make([]PlayerView, 0)
	for _, p := range g.players {
		pList = append(pList, PlayerView{p.ID, p.Nickname, p.Score, p.Answer != ""})
	}
	sort.Slice(pList, func(i, j int) bool { return pList[i].Score > pList[j].Score })

	state := map[string]interface{}{
		"type":    "state",
		"status":  g.state,
		"timer":   g.timer,
		"round":   g.round,
		"players": pList,
		"prompt":  g.currentPrompt,
	}

	if g.state == "VOTING" && g.matchA != nil && g.matchB != nil {
		state["match"] = map[string]interface{}{
			"a_id": g.matchA.ID, "a_text": g.matchA.Answer,
			"b_id": g.matchB.ID, "b_text": g.matchB.Answer,
		}
	}

	msg, _ := json.Marshal(state)
	g.broadcast <- msg
}

func (g *Game) HandleMsg(p *Player, msg []byte) {
	var input struct {
		Type string `json:"type"`
		Text string `json:"text"`
		Vote string `json:"vote"` // "A" or "B"
	}
	if err := json.Unmarshal(msg, &input); err != nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	if input.Type == "start" && g.state == "LOBBY" && len(g.players) >= MinPlayers {
		g.round = 1
		g.startRound()
		g.broadcastState()
		return
	}

	if input.Type == "answer" && g.state == "INPUT" {
		p.Answer = input.Text
		allAnswered := true
		for _, pl := range g.players {
			if pl.Answer == "" {
				allAnswered = false
				break
			}
		}
		if allAnswered {
			g.timer = 3
		}
		g.broadcastState()
	}

	if input.Type == "vote" && g.state == "VOTING" && !p.Voted {
		if input.Vote == "A" {
			g.votesA++
		}
		if input.Vote == "B" {
			g.votesB++
		}
		p.Voted = true
	}
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

func HandleWS(g *Game, w http.ResponseWriter, r *http.Request, store *data.Store) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	userID := r.URL.Query().Get("userID")
	nick := "Guest"
	if userID != "" {
		if u, ok := store.GetUser(userID); ok {
			nick = u.Nickname
		}
	}

	p := &Player{
		ID:     fmt.Sprintf("p%d", rand.Int()),
		UserID: userID, Nickname: nick,
		Conn: conn, Send: make(chan []byte, 256),
	}

	g.register <- p

	go func() {
		for msg := range p.Send {
			conn.WriteMessage(websocket.TextMessage, msg)
		}
		conn.Close()
	}()

	go func() {
		defer func() { g.unregister <- p; conn.Close() }()
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				break
			}
			g.HandleMsg(p, msg)
		}
	}()
}
