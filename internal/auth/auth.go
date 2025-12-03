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

type loginRequest struct {
	Nickname string `json:"nickname"`
	Tag      int    `json:"tag"`
}

type friendRequest struct {
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

// LoginHandler sets the cookie for an existing nickname+tag combo.
func (a *Auth) LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}

	nick := strings.TrimSpace(req.Nickname)
	if nick == "" || req.Tag <= 0 {
		http.Error(w, "invalid credentials", http.StatusBadRequest)
		return
	}

	var userID string
	err := a.DB.QueryRow(`SELECT id FROM users WHERE nickname = $1 AND tag = $2`, nick, req.Tag).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "lookup failed", http.StatusInternalServerError)
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
		Tag:      req.Tag,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// AddFriendHandler accepts a nickname+tag and creates an accepted friendship row.
func (a *Auth) AddFriendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqUserID, err := readUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req friendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Nickname) == "" || req.Tag <= 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	var targetID string
	err = a.DB.QueryRow(`
		SELECT id FROM users WHERE nickname = $1 AND tag = $2
	`, req.Nickname, req.Tag).Scan(&targetID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}
		http.Error(w, "lookup failed", http.StatusInternalServerError)
		return
	}
	if targetID == reqUserID {
		http.Error(w, "cannot add yourself", http.StatusBadRequest)
		return
	}

	_, err = a.DB.Exec(`
		INSERT INTO friendships (requester_id, addressee_id, status)
		VALUES ($1, $2, 'accepted')
		ON CONFLICT (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id))
		DO UPDATE SET status = 'accepted', updated_at = NOW()
	`, reqUserID, targetID)
	if err != nil {
		log.Println("add friend:", err)
		http.Error(w, "failed to add friend", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveFriendHandler removes a friendship row between the requester and the target nickname/tag.
func (a *Auth) RemoveFriendHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	reqUserID, err := readUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req friendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Nickname) == "" || req.Tag <= 0 {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	var targetID string
	err = a.DB.QueryRow(`
		SELECT id FROM users WHERE nickname = $1 AND tag = $2
	`, req.Nickname, req.Tag).Scan(&targetID)
	if err != nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	_, err = a.DB.Exec(`
		DELETE FROM friendships
		WHERE LEAST(requester_id, addressee_id) = LEAST($1, $2)
		AND GREATEST(requester_id, addressee_id) = GREATEST($1, $2)
	`, reqUserID, targetID)
	if err != nil {
		log.Println("remove friend:", err)
		http.Error(w, "failed to remove friend", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// insertUserWithTag retries random tag generation and inserts the user atomically.
func (a *Auth) insertUserWithTag(nickname string) (int, string, error) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < 20; i++ {
		tag := rng.Intn(9999) + 1 // 1..9999
		userID := "u_" + uuid.NewString()

		var insertedID string
		err := a.DB.QueryRow(`
			INSERT INTO users (id, nickname, tag, level, exp, max_exp, status)
			VALUES ($1, $2, $3, 1, 0, 1000, 'online')
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

// readUserID extracts the user_id cookie.
func readUserID(r *http.Request) (string, error) {
	c, err := r.Cookie("user_id")
	if err != nil || c.Value == "" {
		return "", errors.New("missing user id cookie")
	}
	return c.Value, nil
}
