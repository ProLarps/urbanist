package db

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"time"
)

// Player is a persistent profile (docs/API.md §12 table players).
type Player struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Avatar    int     `json:"avatar"`
	Color     string  `json:"color"`
	CreatedAt int64   `json:"createdAt"`
	LastSeen  int64   `json:"lastSeen"`
	BestScore float64 `json:"bestScore"`
}

func newID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func nowUnix() int64 { return time.Now().Unix() }

// CreatePlayer creates a profile or reuses the one with the same name
// (profile identity is the name: re-auth must return the same profile).
func (s *Store) CreatePlayer(name string, avatar int, color string) (*Player, error) {
	if p, err := s.GetPlayerByName(name); err == nil && p != nil {
		p.Avatar = avatar
		if err := s.UpdatePlayerProfile(p.ID, p.Name, avatar); err != nil {
			return nil, err
		}
		return p, s.TouchPlayer(p.ID)
	}

	p := &Player{ID: newID("p_"), Name: name, Avatar: avatar, Color: color, CreatedAt: nowUnix()}
	p.LastSeen = p.CreatedAt
	_, err := s.db.Exec(`INSERT INTO players(id,name,avatar,color,created_at,last_seen,best_score)
		VALUES(?,?,?,?,?,?,0)`, p.ID, p.Name, p.Avatar, p.Color, p.CreatedAt, p.LastSeen)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) GetPlayerByID(id string) (*Player, error) {
	row := s.db.QueryRow(`SELECT id,name,avatar,color,created_at,last_seen,best_score
		FROM players WHERE id=?`, id)
	var p Player
	err := row.Scan(&p.ID, &p.Name, &p.Avatar, &p.Color, &p.CreatedAt, &p.LastSeen, &p.BestScore)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) GetPlayerByName(name string) (*Player, error) {
	row := s.db.QueryRow(`SELECT id,name,avatar,color,created_at,last_seen,best_score
		FROM players WHERE name=? LIMIT 1`, name)
	var p Player
	err := row.Scan(&p.ID, &p.Name, &p.Avatar, &p.Color, &p.CreatedAt, &p.LastSeen, &p.BestScore)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *Store) UpdatePlayerProfile(id, name string, avatar int) error {
	_, err := s.db.Exec(`UPDATE players SET name=?, avatar=? WHERE id=?`, name, avatar, id)
	return err
}

func (s *Store) TouchPlayer(id string) error {
	_, err := s.db.Exec(`UPDATE players SET last_seen=? WHERE id=?`, nowUnix(), id)
	return err
}

// SaveBestScore keeps the best session score (money x rating) in the profile.
func (s *Store) SaveBestScore(playerID string, score float64) error {
	_, err := s.db.Exec(`UPDATE players SET best_score=MAX(best_score,?) WHERE id=?`, score, playerID)
	return err
}
