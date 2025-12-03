package main

import (
	"log"
	"main/internal/chibiki"
	"main/internal/data"
	"main/internal/lobby"
	"net/http"
	"os"
)

func main() {
	store, err := data.NewStore("internal/data/users.json", "internal/data/medals.json")
	if err != nil {
		log.Fatalf("failed to load data store: %v", err)
	}

	// 1. Initialize the Game Engine
	gameInstance := chibiki.NewGame()
	gameInstance.OnGameOver = func(winnerTeam int, players map[*chibiki.Player]bool) {
		for p := range players {
			if p.Team == winnerTeam && p.UserID != "" && p.UserID != "guest" {
				if _, err := store.AwardMedals(p.UserID, "first_win"); err != nil {
					log.Printf("failed to award medal to %s: %v", p.UserID, err)
				}
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

	// 5. Configure Routes
	http.HandleFunc("/ws", chibiki.NewWebsocketHandler(gameInstance))

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
