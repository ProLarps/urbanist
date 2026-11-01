package db

import "database/sql"

// LeaderboardRow is one entry of the global top (docs/API.md §2.3).
type LeaderboardRow struct {
	PlayerID  string  `json:"playerId"`
	Name      string  `json:"name"`
	BestScore float64 `json:"bestScore"`
}

// Leaderboard returns up to 100 profiles sorted by their best session score.
func (s *Store) Leaderboard() ([]LeaderboardRow, error) {
	rows, err := s.db.Query(`SELECT id,name,best_score FROM players
		WHERE best_score>0 ORDER BY best_score DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLeaderboard(rows)
}

func scanLeaderboard(rows *sql.Rows) ([]LeaderboardRow, error) {
	var out []LeaderboardRow
	for rows.Next() {
		var r LeaderboardRow
		if err := rows.Scan(&r.PlayerID, &r.Name, &r.BestScore); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListPlayers returns every profile (docs/API.md §2.3, service endpoint).
func (s *Store) ListPlayers() ([]Player, error) {
	rows, err := s.db.Query(`SELECT id,name,avatar,color,created_at,last_seen,best_score
		FROM players ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		var p Player
		err := rows.Scan(&p.ID, &p.Name, &p.Avatar, &p.Color, &p.CreatedAt, &p.LastSeen, &p.BestScore)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
