package auth

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Auth struct {
	DB *sql.DB
}

func NewAuth(db *sql.DB) *Auth {
	return &Auth{DB: db}
}

type registerRequest struct {
	Nickname string `json:"nickname"`
}

type registerResponse struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
	Tag      int    `json:"tag"`
}

// RegisterHandler creates a user with a nickname and an auto-generated tag.
func (a *Auth) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	nick := strings.TrimSpace(req.Nickname)
	if nick == "" {
		http.Error(w, "empty nickname", http.StatusBadRequest)
		return
	}

	tag, userID, err := a.insertUserWithTag(nick)
	if err != nil {
		log.Println("register:", err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "user_id",
		Value:    userID,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	resp := registerResponse{
		UserID:   userID,
		Nickname: nick,
		Tag:      tag,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// insertUserWithTag retries random tag generation and inserts the user atomically.
func (a *Auth) insertUserWithTag(nickname string) (int, string, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 20; i++ {
		tag := rng.Intn(9999) + 1 // 1..9999
		userID := "u_" + uuid.NewString()

		var insertedID string
		err := a.DB.QueryRow(`
			INSERT INTO users (id, nickname, tag, level, exp, max_exp)
			VALUES ($1, $2, $3, 1, 0, 1000)
			ON CONFLICT (nickname, tag) DO NOTHING
			RETURNING id
		`, userID, nickname, tag).Scan(&insertedID)

		if errors.Is(err, sql.ErrNoRows) {
			continue // Tag collision, retry
		}
		if err != nil {
			return 0, "", err
		}

		return tag, insertedID, nil
	}

	return 0, "", fmt.Errorf("failed to generate unique tag for %s", nickname)
}
