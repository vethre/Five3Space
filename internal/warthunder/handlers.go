package warthunder

import (
	"encoding/json"
	"html/template"
	"net/http"
	"path/filepath"

	"main/internal/data"
)

// NewHandler renders the main game page
func NewHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := "guest"
		if c, err := r.Cookie("user_id"); err == nil && c.Value != "" {
			userID = c.Value
		} else if q := r.URL.Query().Get("userID"); q != "" {
			userID = q
		}

		lang := r.URL.Query().Get("lang")
		if lang == "" {
			lang = "en"
		}

		data := struct {
			UserID string
			Lang   string
		}{UserID: userID, Lang: lang}

		tmplPath := filepath.Join("web", "templates", "warthunder.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			http.Error(w, "Could not load War Thunder template: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, data)
	}
}

// API Handler for game actions
func NewAPIHandler(store *data.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("userID")
		if userID == "" {
			// fallback to cookie
			if c, err := r.Cookie("user_id"); err == nil {
				userID = c.Value
			}
		}
		if userID == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if r.Method == "GET" {
			// Get State
			game := GetGame(userID)
			if game == nil {
				// Return list of countries for selection if no game exists
				json.NewEncoder(w).Encode(map[string]interface{}{
					"status":    "selection",
					"countries": baseCountries,
				})
				return
			}
			game.Mutex.RLock()
			defer game.Mutex.RUnlock()
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "playing",
				"game":   game,
			})
			return
		}

		if r.Method == "POST" {
			var req struct {
				Action  string `json:"action"`  // start, attack, diplomat
				Payload string `json:"payload"` // countryID
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			if req.Action == "start" {
				game := CreateGame(userID, req.Payload)
				json.NewEncoder(w).Encode(map[string]interface{}{"status": "started", "game": game})
				return
			}

			game := GetGame(userID)
			if game == nil {
				http.Error(w, "No active game", http.StatusBadRequest)
				return
			}

			msg := ""
			switch req.Action {
			case "attack":
				msg = game.Attack(req.Payload)
			case "diplomat":
				msg = game.Diplomat(req.Payload)
			}

			// Return updated state
			game.Mutex.RLock()
			defer game.Mutex.RUnlock()
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "ok",
				"message": msg,
				"game":    game,
			})
		}
	}
}
