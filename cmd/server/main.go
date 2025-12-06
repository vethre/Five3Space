package main

import (
	"database/sql"
	"fmt"
	"log"
	"main/internal/auth"
	"main/internal/bobikshooter"
	"main/internal/chat"
	"main/internal/chibiki"
	"main/internal/data"
	"main/internal/lobby"
	"main/internal/party"
	"main/internal/presence"
	"net/http"
	"os"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	chat.DB = db

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := applySchema(db); err != nil {
		log.Fatalf("failed to apply schema: %v", err)
	}

	store, err := data.NewStore(db, "internal/data/medals.json")
	if err != nil {
		log.Fatalf("failed to init store: %v", err)
	}

	// 1. Initialize the Game Engine
	gameInstance := chibiki.NewGame()
	gameInstance.OnGameOver = func(winnerTeam int, players map[*chibiki.Player]bool) {
		log.Printf("GAME OVER! Winner Team: %d", winnerTeam)

		for p := range players {
			if p.UserID == "" || p.UserID == "guest" {
				continue
			}

			var trophyChange, coinChange, expChange int

			if p.Team == winnerTeam {
				trophyChange = 30
				coinChange = 50
				expChange = 150 // Increased XP so we can test leveling easier
				store.AwardMedals(p.UserID, "first_win")
			} else {
				trophyChange = -15
				coinChange = 10
				expChange = 25
			}

			// USE THE NEW FUNCTION
			err := store.ProcessGameResult(p.UserID, trophyChange, coinChange, expChange)
			if err != nil {
				log.Printf("Error saving stats for %s: %v", p.UserID, err)
			}
		}
	}

	if err := gameInstance.LoadUnits("internal/data/units.json"); err != nil {
		log.Printf("Warning: Could not load units.json: %v", err)
	}
	gameInstance.InitTowers()
	go gameInstance.StartLoop()

	presenceService := presence.NewService(db)
	bobikGame := bobikshooter.NewGame(store)

	partyGame := party.NewGame(store)

	authService := auth.NewAuth(db)
	http.HandleFunc("/register", authService.RegisterHandler)
	http.HandleFunc("/login", authService.LoginHandler)
	http.HandleFunc("/logout", authService.LogoutHandler)
	http.HandleFunc("/settings/language", authService.UpdateLanguageHandler)
	http.HandleFunc("/friends/add", authService.AddFriendHandler)
	http.HandleFunc("/friends/remove", authService.RemoveFriendHandler)
	http.HandleFunc("/presence/ping", presenceService.PingHandler)

	http.HandleFunc("/ws", chibiki.NewWebsocketHandler(gameInstance))
	http.HandleFunc("/ws/bobik", bobikGame.HandleWS)

	http.HandleFunc("/ws/chat", chat.HandleWS)
	http.HandleFunc("/chat/history", chat.HistoryHandler)
	http.HandleFunc("/chat/delivered", chat.DeliveredHandler)
	http.HandleFunc("/chat/seen", chat.SeenHandler)

	// Lobby Pages
	http.HandleFunc("/friends", lobby.NewFriendsHandler(store))
	http.HandleFunc("/shop", lobby.NewShopHandler(store))
	http.HandleFunc("/shop/buy", lobby.NewBuyHandler(store))
	http.HandleFunc("/customize", lobby.NewCustomizeHandler(store))
	http.HandleFunc("/customize/save", lobby.NewCustomizeSaveHandler(store))
	http.HandleFunc("/bobik", lobby.NewBobikHandler(store))
	http.HandleFunc("/leaderboard", lobby.NewLeaderboardHandler(store))

	http.HandleFunc("/game", lobby.NewGameHandler(store))
	http.HandleFunc("/", lobby.NewHandler(store))

	http.HandleFunc("/party", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/templates/party.html")
	})
	http.HandleFunc("/ws/party", func(w http.ResponseWriter, r *http.Request) {
		party.HandleWS(partyGame, w, r, store)
	})

	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("Server starting on port " + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func applySchema(db *sql.DB) error {
	statements := []string{
		`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			nickname TEXT NOT NULL,
			tag INTEGER NOT NULL,
			level INTEGER NOT NULL DEFAULT 1,
			exp INTEGER NOT NULL DEFAULT 0,
			max_exp INTEGER NOT NULL DEFAULT 1000,
			coins INTEGER NOT NULL DEFAULT 0,
			trophies INTEGER NOT NULL DEFAULT 0,
			password_hash TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'offline',
			last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			language TEXT NOT NULL DEFAULT 'en',
			
			-- New Customization Columns
			name_color TEXT NOT NULL DEFAULT 'white',
			banner_color TEXT NOT NULL DEFAULT 'default',
			
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (nickname, tag)
		);
		`,
		// Migrations for existing DBs
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS name_color TEXT NOT NULL DEFAULT 'white';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS banner_color TEXT NOT NULL DEFAULT 'default';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS custom_avatar TEXT NOT NULL DEFAULT '';`,

		`
		CREATE TABLE IF NOT EXISTS medals (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL,
			icon TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS user_medals (
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			medal_id TEXT NOT NULL REFERENCES medals(id) ON DELETE CASCADE,
			awarded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, medal_id)
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS friendships (
			id BIGSERIAL PRIMARY KEY,
			requester_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			addressee_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','accepted','blocked')),
			CONSTRAINT friendships_not_self CHECK (requester_id <> addressee_id),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
		`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_friendships_pair ON friendships (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));`,
		`
		CREATE TABLE IF NOT EXISTS inventory (
			user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			item_id TEXT NOT NULL,
			acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, item_id)
		);
		`,
		`
		CREATE TABLE IF NOT EXISTS messages (
			id BIGSERIAL PRIMARY KEY,
			sender_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			receiver_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			text TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			delivered BOOLEAN NOT NULL DEFAULT FALSE,
			seen BOOLEAN NOT NULL DEFAULT FALSE
		);
		`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("schema exec failed: %w", err)
		}
	}
	return nil
}
