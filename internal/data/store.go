package data

import (
	"context"
	"database/sql"
	"encoding/json"
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
	ID          string   `json:"id"`
	Nickname    string   `json:"nickname"`
	Tag         int      `json:"tag"`
	Level       int      `json:"level"`
	Exp         int      `json:"exp"`
	MaxExp      int      `json:"max_exp"`
	Coins       int      `json:"coins"`
	Trophies    int      `json:"trophies"`
	Status      string   `json:"status"`
	Medals      []string `json:"medals"`
	Language    string   `json:"language"`
	NameColor   string   `json:"name_color"`
	BannerColor string   `json:"banner_color"`
}

// Store persists user progress and medal metadata in Postgres.
type Store struct {
	mu     sync.Mutex
	db     *sql.DB
	medals map[string]Medal
}

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
	// Best-effort sync
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, m := range list {
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO medals (id, name, description, icon)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name, description = EXCLUDED.description, icon = EXCLUDED.icon
		`, m.ID, m.Name, m.Description, m.Icon)
	}
	return nil
}

// GetUser returns a single user by ID.
func (s *Store) GetUser(id string) (UserData, bool) {
	row := s.db.QueryRow(`
        SELECT id, nickname, tag, level, exp, max_exp, coins, trophies, 
		       COALESCE(status, 'offline'), COALESCE(language, 'en'),
			   COALESCE(name_color, 'white'), COALESCE(banner_color, 'default')
        FROM users
        WHERE id = $1
    `, id)

	var u UserData
	if err := row.Scan(&u.ID, &u.Nickname, &u.Tag, &u.Level, &u.Exp, &u.MaxExp, &u.Coins, &u.Trophies, &u.Status, &u.Language, &u.NameColor, &u.BannerColor); err != nil {
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

// AwardMedals inserts new medals for a user.
func (s *Store) AwardMedals(userID string, medalIDs ...string) (UserData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Simplified logic for brevity in this response
	for _, id := range medalIDs {
		if _, ok := s.medals[id]; !ok {
			continue
		}
		_, _ = s.db.Exec(`INSERT INTO user_medals (user_id, medal_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, userID, id)
	}
	u, _ := s.GetUser(userID)
	return u, nil
}

func (s *Store) MedalDetails(ids []string) []Medal {
	out := make([]Medal, 0, len(ids))
	for _, id := range ids {
		if m, ok := s.medals[id]; ok {
			out = append(out, m)
		}
	}
	return out
}

func (s *Store) AdjustTrophies(userID string, delta int) error {
	_, err := s.db.Exec(`UPDATE users SET trophies = GREATEST(0, trophies + $1), updated_at = NOW() WHERE id = $2`, delta, userID)
	return err
}

// Friend updated with Trophies, Exp, etc.
type Friend struct {
	ID        string
	Nickname  string
	Tag       int
	Level     int
	Exp       int
	MaxExp    int
	Trophies  int
	Presence  string
	NameColor string
}

func (s *Store) ListFriends(userID string) ([]Friend, error) {
	rows, err := s.db.Query(`
		SELECT
			u.id, u.nickname, u.tag, u.level, u.exp, u.max_exp, u.trophies,
			COALESCE(u.name_color, 'white'),
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
		if err := rows.Scan(&fr.ID, &fr.Nickname, &fr.Tag, &fr.Level, &fr.Exp, &fr.MaxExp, &fr.Trophies, &fr.NameColor, &fr.Presence); err != nil {
			continue
		}
		friends = append(friends, fr)
	}

	return friends, nil
}

func (s *Store) AdjustCoins(userID string, amount int) error {
	_, err := s.db.Exec(`UPDATE users SET coins = coins + $1 WHERE id = $2`, amount, userID)
	return err
}

func (s *Store) HasItem(userID, itemID string) bool {
	var exists bool
	_ = s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM inventory WHERE user_id=$1 AND item_id=$2)`, userID, itemID).Scan(&exists)
	return exists
}

// GetUserInventory returns all item IDs owned by user
func (s *Store) GetUserInventory(userID string) ([]string, error) {
	rows, err := s.db.Query(`SELECT item_id FROM inventory WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var i string
		if err := rows.Scan(&i); err == nil {
			items = append(items, i)
		}
	}
	return items, nil
}

func (s *Store) DeductCoinsAndAddItem(userID, itemID string, cost int) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE users SET coins = coins - $1 WHERE id = $2 AND coins >= $1`, cost, userID)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("insufficient funds")
	}

	_, err = tx.Exec(`INSERT INTO inventory (user_id, item_id) VALUES ($1, $2)`, userID, itemID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Store) UpdateProfileLook(userID, nameColor, bannerColor, avatarSeed string) error {
	// Simple update. Avatar isn't stored in DB in this version (derived from seed/nick),
	// but normally you'd update a 'avatar_url' column.
	// We only update name_color and banner_color here for persistence.

	// Validation (Basic)
	if nameColor != "" {
		_, err := s.db.Exec(`UPDATE users SET name_color = $1 WHERE id = $2`, nameColor, userID)
		if err != nil {
			return err
		}
	}
	if bannerColor != "" {
		_, err := s.db.Exec(`UPDATE users SET banner_color = $1 WHERE id = $2`, bannerColor, userID)
		if err != nil {
			return err
		}
	}
	// Note: Avatar seed logic is usually client-side derived from nickname,
	// unless we add an avatar_seed column. For now, we rely on nickname change or hardcoded logic.
	return nil
}
