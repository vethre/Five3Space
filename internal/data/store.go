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

type Medal struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type UserData struct {
	ID           string   `json:"id"`
	Nickname     string   `json:"nickname"`
	Tag          int      `json:"tag"`
	Level        int      `json:"level"`
	Exp          int      `json:"exp"`
	MaxExp       int      `json:"max_exp"`
	Coins        int      `json:"coins"`
	Trophies     int      `json:"trophies"`
	Status       string   `json:"status"`
	Medals       []string `json:"medals"`
	Language     string   `json:"language"`
	NameColor    string   `json:"name_color"`
	BannerColor  string   `json:"banner_color"`
	CustomAvatar string   `json:"custom_avatar"` // Base64 data or empty
}

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

func (s *Store) GetUser(id string) (UserData, bool) {
	row := s.db.QueryRow(`
        SELECT id, nickname, tag, level, exp, max_exp, coins, trophies, 
		       COALESCE(status, 'offline'), COALESCE(language, 'en'),
			   COALESCE(name_color, 'white'), COALESCE(banner_color, 'default'),
			   COALESCE(custom_avatar, '')
        FROM users
        WHERE id = $1
    `, id)

	var u UserData
	if err := row.Scan(&u.ID, &u.Nickname, &u.Tag, &u.Level, &u.Exp, &u.MaxExp, &u.Coins, &u.Trophies, &u.Status, &u.Language, &u.NameColor, &u.BannerColor, &u.CustomAvatar); err != nil {
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

func (s *Store) AwardMedals(userID string, medalIDs ...string) (UserData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

func (s *Store) AdjustExp(userID string, delta int) error {
	_, err := s.db.Exec(`UPDATE users SET exp = exp + $1 WHERE id = $2`, delta, userID)
	return err
}

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
	AvatarURL string // Final URL to display
}

func (s *Store) ListFriends(userID string) ([]Friend, error) {
	rows, err := s.db.Query(`
		SELECT
			u.id, u.nickname, u.tag, u.level, u.exp, u.max_exp, u.trophies,
			COALESCE(u.name_color, 'white'),
			COALESCE(u.custom_avatar, ''),
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
		var customAvatar string
		if err := rows.Scan(&fr.ID, &fr.Nickname, &fr.Tag, &fr.Level, &fr.Exp, &fr.MaxExp, &fr.Trophies, &fr.NameColor, &customAvatar, &fr.Presence); err != nil {
			continue
		}
		// Determine Avatar
		if customAvatar != "" {
			fr.AvatarURL = customAvatar
		} else {
			fr.AvatarURL = fmt.Sprintf("https://api.dicebear.com/7.x/avataaars/svg?seed=%s&backgroundColor=ffdfbf", fr.Nickname)
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

func (s *Store) UpdateProfileLook(userID, nameColor, bannerColor, avatarBase64 string) error {
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
	if avatarBase64 != "" {
		// Save custom avatar
		_, err := s.db.Exec(`UPDATE users SET custom_avatar = $1 WHERE id = $2`, avatarBase64, userID)
		if err != nil {
			return err
		}
	}
	return nil
}
