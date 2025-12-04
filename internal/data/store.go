package data

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

// Medal represents metadata for an achievement.
type Medal struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

// UserData is the public-facing user payload.
type UserData struct {
	ID       string   `json:"id"`
	Nickname string   `json:"nickname"`
	Tag      int      `json:"tag"`
	Level    int      `json:"level"`
	Exp      int      `json:"exp"`
	MaxExp   int      `json:"max_exp"`
	Coins    int      `json:"coins"`
	Trophies int      `json:"trophies"`
	Status   string   `json:"status"`
	Medals   []string `json:"medals"`
	Language string   `json:"language"`
}

// Store persists user progress and medal metadata in Postgres.
type Store struct {
	mu     sync.Mutex
	db     *sql.DB
	medals map[string]Medal
}

// NewStore accepts an existing DB handle.
func NewStore(db *sql.DB, medalsPath string) (*Store, error) {
	s := &Store{
		db:     db,
		medals: make(map[string]Medal),
	}
	if err := s.loadMedals(medalsPath); err != nil {
		return nil, err
	}
	return s, nil
}

// NewStoreFromDB builds the store from a connection string (e.g. os.Getenv("DATABASE_URL")).
func NewStoreFromDB(connStr, medalsPath string) (*Store, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return NewStore(db, medalsPath)
}

// loadMedals keeps medal metadata in memory and mirrors it into the DB table.
func (s *Store) loadMedals(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var list []Medal
	if err := json.Unmarshal(raw, &list); err != nil {
		return err
	}
	for _, m := range list {
		s.medals[m.ID] = m
	}

	// Best-effort upsert into DB to keep table in sync with JSON source.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, m := range list {
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO medals (id, name, description, icon)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name,
			    description = EXCLUDED.description,
			    icon = EXCLUDED.icon
		`, m.ID, m.Name, m.Description, m.Icon)
	}
	return nil
}

// FirstUser is a convenience helper used by the lobby when no ID is provided.
func (s *Store) FirstUser() (UserData, bool) {
	row := s.db.QueryRow(`
        SELECT id, nickname, tag, level, exp, max_exp, coins, trophies, COALESCE(status, 'offline'), COALESCE(language, 'en')
        FROM users
        ORDER BY created_at ASC
        LIMIT 1
    `)

	var u UserData
	if err := row.Scan(&u.ID, &u.Nickname, &u.Tag, &u.Level, &u.Exp, &u.MaxExp, &u.Coins, &u.Trophies, &u.Status, &u.Language); err != nil {
		return UserData{}, false
	}

	u.Medals = s.getUserMedalIDs(u.ID)
	return u, true
}

// GetUser returns a single user by ID.
func (s *Store) GetUser(id string) (UserData, bool) {
	row := s.db.QueryRow(`
        SELECT id, nickname, tag, level, exp, max_exp, coins, trophies, COALESCE(status, 'offline'), COALESCE(language, 'en')
        FROM users
        WHERE id = $1
    `, id)

	var u UserData
	if err := row.Scan(&u.ID, &u.Nickname, &u.Tag, &u.Level, &u.Exp, &u.MaxExp, &u.Coins, &u.Trophies, &u.Status, &u.Language); err != nil {
		return UserData{}, false
	}

	u.Medals = s.getUserMedalIDs(id)
	return u, true
}

func (s *Store) getUserMedalIDs(userID string) []string {
	rows, err := s.db.Query(`SELECT medal_id FROM user_medals WHERE user_id = $1`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// MedalCount returns how many medals the user currently holds.
func (s *Store) MedalCount(id string) int {
	user, ok := s.GetUser(id)
	if !ok {
		return 0
	}
	return len(user.Medals)
}

// AwardMedals inserts new medals for a user, ignoring duplicates.
func (s *Store) AwardMedals(userID string, medalIDs ...string) (UserData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return UserData{}, err
	}
	defer tx.Rollback()

	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)`, userID).Scan(&exists); err != nil {
		return UserData{}, err
	}
	if !exists {
		return UserData{}, errors.New("user not found")
	}

	for _, id := range medalIDs {
		if _, ok := s.medals[id]; !ok {
			continue // Ignore unknown medals
		}
		if _, err := tx.Exec(`
            INSERT INTO user_medals (user_id, medal_id)
            VALUES ($1, $2)
            ON CONFLICT (user_id, medal_id) DO NOTHING
        `, userID, id); err != nil {
			return UserData{}, fmt.Errorf("insert medal %s: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return UserData{}, err
	}

	user, ok := s.GetUser(userID)
	if !ok {
		return UserData{}, errors.New("user not found after awarding medals")
	}
	return user, nil
}

// MedalDetails returns metadata for the provided medal IDs.
func (s *Store) MedalDetails(ids []string) []Medal {
	out := make([]Medal, 0, len(ids))
	for _, id := range ids {
		if m, ok := s.medals[id]; ok {
			out = append(out, m)
		}
	}
	return out
}

// AdjustTrophies adds/subtracts and clamps to zero.
func (s *Store) AdjustTrophies(userID string, delta int) error {
	_, err := s.db.Exec(`
		UPDATE users
		SET trophies = GREATEST(0, trophies + $1),
		    updated_at = NOW()
		WHERE id = $2
	`, delta, userID)
	return err
}

// Friend represents a basic friend profile.
type Friend struct {
	ID       string
	Nickname string
	Tag      int
	Level    int
	Presence string
}

// ListFriends returns accepted friends for a user.
func (s *Store) ListFriends(userID string) ([]Friend, error) {
	rows, err := s.db.Query(`
		SELECT
			u.id,
			u.nickname,
			u.tag,
			u.level,
			CASE
				WHEN u.status = 'offline' THEN 'offline'
				WHEN NOW() - u.last_seen <= INTERVAL '60 seconds' THEN u.status
				WHEN NOW() - u.last_seen <= INTERVAL '5 minutes' THEN 'away'
				ELSE 'offline'
			END AS presence
		FROM friendships f
		JOIN users u ON (
			(u.id = f.requester_id AND f.addressee_id = $1)
			OR (u.id = f.addressee_id AND f.requester_id = $1)
		)
		WHERE f.status = 'accepted' AND u.id <> $1
	`, userID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var friends []Friend
	for rows.Next() {
		var fr Friend
		if err := rows.Scan(&fr.ID, &fr.Nickname, &fr.Tag, &fr.Level, &fr.Presence); err != nil {
			continue
		}
		friends = append(friends, fr)
	}

	return friends, nil
}

func (s *Store) AdjustCoins(userID string, amount int) error {
	_, err := s.db.Exec(`
		UPDATE users SET coins = coins + $1 WHERE id = $2
	`, amount, userID)
	return err
}

// HasItem checks inventory.
func (s *Store) HasItem(userID, itemID string) bool {
	var exists bool
	_ = s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM inventory WHERE user_id=$1 AND item_id=$2)`, userID, itemID).Scan(&exists)
	return exists
}

// Transaction for buying items.
func (s *Store) DeductCoinsAndAddItem(userID, itemID string, cost int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Deduct
	res, err := tx.Exec(`UPDATE users SET coins = coins - $1 WHERE id = $2 AND coins >= $1`, cost, userID)
	if err != nil {
		return err
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient funds")
	}

	// Add Item
	_, err = tx.Exec(`INSERT INTO inventory (user_id, item_id) VALUES ($1, $2)`, userID, itemID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
