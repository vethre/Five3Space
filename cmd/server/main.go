package main

import (
	"database/sql"
	"fmt"
	"log"
	"main/internal/auth"
	"main/internal/chibiki"
	"main/internal/data"
	"main/internal/lobby"
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
		for p := range players {
			// Skip guests
			if p.UserID == "" || p.UserID == "guest" {
				continue
			}

			var trophyChange, coinChange, expChange int

			if p.Team == winnerTeam {
				// WINNER REWARDS
				trophyChange = 30
				coinChange = 50
				expChange = 25
				// Award 'first_win' medal logic if needed (optional)
				store.AwardMedals(p.UserID, "first_win")
			} else {
				// LOSER REWARDS (Consolation)
				trophyChange = -15
				coinChange = 10
				expChange = 5
			}

			// Update DB with EVERYTHING (Trophies, Coins, XP)
			_, err := db.Exec(`
				UPDATE users
				SET trophies = GREATEST(0, trophies + $1),
					coins = coins + $2,
					exp = exp + $3,
					updated_at = NOW()
				WHERE id = $4
			`, trophyChange, coinChange, expChange, p.UserID)

			if err != nil {
				log.Printf("Failed to update rewards for %s: %v", p.UserID, err)
			}
		}
	}

	// 2. Load Unit Data FIRST (Critical fix!)
	// This ensures 'king_tower' has stats like 4000 HP before we spawn it
	if err := gameInstance.LoadUnits("internal/data/units.json"); err != nil {
		log.Printf("Warning: Could not load units.json: %v. Towers might have 0 HP.", err)
	}

	// 3. Spawn Towers NOW
	gameInstance.InitTowers()

	// 4. Start the Physics Loop
	go gameInstance.StartLoop()

	presenceService := presence.NewService(db)

	// 5. Configure Routes
	authService := auth.NewAuth(db)
	http.HandleFunc("/register", authService.RegisterHandler)
	http.HandleFunc("/login", authService.LoginHandler)
	http.HandleFunc("/logout", authService.LogoutHandler)
	http.HandleFunc("/settings/language", authService.UpdateLanguageHandler)
	http.HandleFunc("/friends/add", authService.AddFriendHandler)
	http.HandleFunc("/friends/remove", authService.RemoveFriendHandler)
	http.HandleFunc("/presence/ping", presenceService.PingHandler)
	http.HandleFunc("/ws", chibiki.NewWebsocketHandler(gameInstance))
	http.HandleFunc("/friends", lobby.NewFriendsHandler(store))
	http.HandleFunc("/shop", lobby.NewShopHandler(store))
	http.HandleFunc("/customize", lobby.NewCustomizeHandler(store))

	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/game", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/game.html")
	})

	http.HandleFunc("/", lobby.NewHandler(store))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server starting on port " + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}

}

// applySchema creates/updates the minimal tables needed for auth, medals, and friendships.
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
			password_hash TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (nickname, tag)
		);
		`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS coins INTEGER NOT NULL DEFAULT 0;`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'offline';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen TIMESTAMPTZ NOT NULL DEFAULT NOW();`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS language TEXT NOT NULL DEFAULT 'en';`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS trophies INTEGER NOT NULL DEFAULT 0;`,
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
		`CREATE INDEX IF NOT EXISTS idx_users_nickname ON users (nickname);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_friendships_pair ON friendships (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));`,
		`CREATE INDEX IF NOT EXISTS idx_user_medals_user ON user_medals (user_id);`,
	}

	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("schema exec failed: %w", err)
		}
	}
	return nil
}
