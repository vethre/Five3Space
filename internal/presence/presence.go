package presence

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

type Service struct {
	DB *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{DB: db}
}

type pingRequest struct {
	Status string `json:"status"` // "online" | "away" | "offline"
}

func (s *Service) PingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID, err := readUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req pingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	status := strings.ToLower(strings.TrimSpace(req.Status))
	switch status {
	case "online", "away", "offline":
	default:
		status = "online"
	}

	_, err = s.DB.Exec(`
		UPDATE users
		SET status = $1,
		    last_seen = $2,
		    updated_at = $2
		WHERE id = $3
	`, status, time.Now().UTC(), userID)
	if err != nil {
		http.Error(w, "failed to update presence", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func readUserID(r *http.Request) (string, error) {
	c, err := r.Cookie("user_id")
	if err != nil || c.Value == "" {
		return "", errors.New("no user_id cookie")
	}
	return c.Value, nil
}
