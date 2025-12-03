package data

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// Medal represents an achievement that can be earned.
type Medal struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

type UserData struct {
	ID       string   `json:"id"`
	Nickname string   `json:"nickname"`
	Level    int      `json:"level"`
	Exp      int      `json:"exp"`
	MaxExp   int      `json:"max_exp"`
	Medals   []string `json:"medals"`
}

type UsersFile struct {
	Users []UserData `json:"users"`
}

// Store keeps user progress and medal metadata in memory and flushes updates back to disk.
type Store struct {
	mu        sync.Mutex
	usersPath string
	users     UsersFile
	medals    map[string]Medal
}

func NewStore(usersPath, medalsPath string) (*Store, error) {
	store := &Store{
		usersPath: usersPath,
		medals:    make(map[string]Medal),
	}

	if err := store.loadUsers(); err != nil {
		return nil, err
	}
	if err := store.loadMedals(medalsPath); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) loadUsers() error {
	raw, err := os.ReadFile(s.usersPath)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, &s.users)
}

func (s *Store) loadMedals(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var medals []Medal
	if err := json.Unmarshal(raw, &medals); err != nil {
		return err
	}
	for _, m := range medals {
		s.medals[m.ID] = m
	}
	return nil
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.usersPath, data, 0644)
}

// FirstUser returns the first user for quick demo purposes.
func (s *Store) FirstUser() (UserData, bool) {
	if len(s.users.Users) == 0 {
		return UserData{}, false
	}
	return s.users.Users[0], true
}

// GetUser returns a copy of the user by id.
func (s *Store) GetUser(id string) (UserData, bool) {
	for _, u := range s.users.Users {
		if u.ID == id {
			return u, true
		}
	}
	return UserData{}, false
}

// MedalCount returns how many medals the user currently has.
func (s *Store) MedalCount(id string) int {
	user, ok := s.GetUser(id)
	if !ok {
		return 0
	}
	return len(user.Medals)
}

// AwardMedals grants the provided medals to the user, avoiding duplicates and persisting the change.
func (s *Store) AwardMedals(userID string, medalIDs ...string) (UserData, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	for i, u := range s.users.Users {
		if u.ID == userID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return UserData{}, errors.New("user not found")
	}

	user := s.users.Users[idx]
	existing := make(map[string]bool, len(user.Medals))
	for _, id := range user.Medals {
		existing[id] = true
	}

	changed := false
	for _, id := range medalIDs {
		if !existing[id] {
			if _, known := s.medals[id]; known {
				user.Medals = append(user.Medals, id)
				existing[id] = true
				changed = true
			}
		}
	}

	if changed {
		s.users.Users[idx] = user
		if err := s.save(); err != nil {
			return UserData{}, err
		}
	}
	return user, nil
}

// MedalDetails returns a slice with the medal metadata for the provided IDs.
func (s *Store) MedalDetails(ids []string) []Medal {
	out := make([]Medal, 0, len(ids))
	for _, id := range ids {
		if medal, ok := s.medals[id]; ok {
			out = append(out, medal)
		}
	}
	return out
}
