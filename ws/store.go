package ws

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no cgo
)

// Store is the SQLite persistence layer (docs/API.md §12).
type Store struct {
	db *sql.DB
}

func OpenStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Modernc sqlite benefits from explicit pragmas for a game server.
	for _, pragma := range []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma: %w", err)
		}
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS players(
  id TEXT PRIMARY KEY, name TEXT NOT NULL, avatar INTEGER NOT NULL DEFAULT 0,
  color TEXT NOT NULL, token_hash TEXT, created_at INTEGER NOT NULL,
  last_seen INTEGER NOT NULL, best_score REAL NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS rooms(
  code TEXT PRIMARY KEY, name TEXT, owner_id TEXT, max_players INTEGER NOT NULL DEFAULT 8,
  created_at INTEGER NOT NULL, seed INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS room_players(
  room_id TEXT, player_id TEXT, joined_at INTEGER, money REAL, rating REAL,
  alive INTEGER NOT NULL DEFAULT 1, score REAL NOT NULL DEFAULT 0,
  x REAL NOT NULL DEFAULT 0, z REAL NOT NULL DEFAULT 0,
  PRIMARY KEY(room_id, player_id));
CREATE TABLE IF NOT EXISTS buildings(
  room_id TEXT, lot_id INTEGER, cx INTEGER, cz INTEGER, ix INTEGER, iz INTEGER,
  type INTEGER, floors INTEGER, condition REAL, heritage REAL, residents INTEGER,
  status TEXT, progress REAL, needs INTEGER, will_base REAL,
  PRIMARY KEY(room_id, lot_id));
CREATE INDEX IF NOT EXISTS idx_buildings_room_chunk ON buildings(room_id, cx, cz);
CREATE TABLE IF NOT EXISTS grants(
  room_id TEXT, player_id TEXT, cx INTEGER, cz INTEGER, cycles_held INTEGER,
  deadline INTEGER, status TEXT,
  PRIMARY KEY(room_id, player_id, cx, cz));
CREATE INDEX IF NOT EXISTS idx_grants_room_player ON grants(room_id, player_id);
CREATE TABLE IF NOT EXISTS npc_agents(
  id TEXT PRIMARY KEY, room_id TEXT, archetype TEXT, name TEXT, color TEXT,
  home_lot INTEGER, will REAL, route_json TEXT);
CREATE TABLE IF NOT EXISTS agent_state(
  agent_id TEXT, player_id TEXT, topics_mask INTEGER NOT NULL DEFAULT 0,
  said_count INTEGER NOT NULL DEFAULT 0, trust INTEGER NOT NULL DEFAULT 0,
  last_topic TEXT, last_ts INTEGER, PRIMARY KEY(agent_id, player_id));
CREATE TABLE IF NOT EXISTS agent_memory(
  id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT, player_id TEXT,
  topic TEXT, said_count INTEGER, player_text TEXT, agent_text TEXT, ts INTEGER);
CREATE INDEX IF NOT EXISTS idx_agent_memory ON agent_memory(agent_id, player_id);
CREATE TABLE IF NOT EXISTS chat_log(
  id INTEGER PRIMARY KEY AUTOINCREMENT, room_id TEXT, chunk_x INTEGER, chunk_z INTEGER,
  player_id TEXT, agent_id TEXT, role TEXT, text TEXT, ts INTEGER);
CREATE INDEX IF NOT EXISTS idx_chat_room_chunk ON chat_log(room_id, chunk_x, chunk_z);
CREATE TABLE IF NOT EXISTS journal(
  id INTEGER PRIMARY KEY AUTOINCREMENT, room_id TEXT, ts INTEGER, kind TEXT,
  author TEXT, text TEXT, meta_json TEXT);
CREATE INDEX IF NOT EXISTS idx_journal_room ON journal(room_id, ts);
`
	_, err := s.db.Exec(schema)
	return err
}

func newID(prefix string) string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func nowUnix() int64 { return time.Now().Unix() }

// ---- players ---------------------------------------------------------

type Player struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Avatar    int     `json:"avatar"`
	Color     string  `json:"color"`
	CreatedAt int64   `json:"createdAt"`
	LastSeen  int64   `json:"lastSeen"`
	BestScore float64 `json:"bestScore"`
}

func (s *Store) CreatePlayer(name string, avatar int, color string) (*Player, error) {
	// Profile identity is the name: re-auth returns the same profile.
	if existing, err := s.GetPlayerByName(name); err == nil && existing != nil {
		_ = s.UpdatePlayerProfile(existing.ID, existing.Name, avatar)
		if err := s.TouchPlayer(existing.ID); err != nil {
			return nil, err
		}
		return existing, nil
	}

	p := &Player{ID: newID("p_"), Name: name, Avatar: avatar, Color: color}
	p.CreatedAt = nowUnix()
	p.LastSeen = p.CreatedAt
	_, err := s.db.Exec(`INSERT INTO players(id,name,avatar,color,created_at,last_seen,best_score)
		VALUES(?,?,?,?,?,?,0)`, p.ID, p.Name, p.Avatar, p.Color, p.CreatedAt, p.LastSeen)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *Store) GetPlayer(id string) (*Player, error) {
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

func (s *Store) ListPlayers() ([]Player, error) {
	rows, err := s.db.Query(`SELECT id,name,avatar,color,created_at,last_seen,best_score FROM players`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Player
	for rows.Next() {
		var p Player
		if err := rows.Scan(&p.ID, &p.Name, &p.Avatar, &p.Color, &p.CreatedAt, &p.LastSeen, &p.BestScore); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) TouchPlayer(id string) error {
	_, err := s.db.Exec(`UPDATE players SET last_seen=? WHERE id=?`, nowUnix(), id)
	return err
}

func (s *Store) SaveBestScore(playerID string, score float64) error {
	_, err := s.db.Exec(`UPDATE players SET best_score=MAX(best_score,?) WHERE id=?`, score, playerID)
	return err
}

func (s *Store) UpdatePlayerProfile(id string, name string, avatar int) error {
	_, err := s.db.Exec(`UPDATE players SET name=?, avatar=? WHERE id=?`, name, avatar, id)
	return err
}

// ---- rooms -----------------------------------------------------------

type Room struct {
	Code       string `json:"code"`
	Name       string `json:"name"`
	OwnerID    string `json:"ownerId"`
	MaxPlayers int    `json:"maxPlayers"`
	CreatedAt  int64  `json:"createdAt"`
	Seed       int64  `json:"-"`
}

func (s *Store) CreateRoom(ownerID, name string, maxPlayers int) (*Room, error) {
	const roomCodeLen = 6
	code, err := newRoomCode(s, roomCodeLen)
	if err != nil {
		return nil, err
	}
	return s.CreateRoomWithCode(code, ownerID, name, maxPlayers)
}

// CreateRoomWithCode stores a room under an explicit code ("default" etc.)
// and is the base for the auto-generated-code variant above.
func (s *Store) CreateRoomWithCode(code, ownerID, name string, maxPlayers int) (*Room, error) {
	r := &Room{Code: code, Name: name, OwnerID: ownerID, MaxPlayers: maxPlayers, CreatedAt: nowUnix(), Seed: nowUnix()%1_000_000 + 1}
	_, err := s.db.Exec(`INSERT INTO rooms(code,name,owner_id,max_players,created_at,seed)
		VALUES(?,?,?,?,?,?)`, r.Code, r.Name, r.OwnerID, r.MaxPlayers, r.CreatedAt, r.Seed)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func newRoomCode(s *Store, n int) (string, error) {
	for range 20 {
		b := make([]byte, n)
		_, _ = rand.Read(b)
		code := ""
		for _, c := range b {
			code += string("ABCDEFGHJKLMNPQRSTUVWXYZ23456789"[int(c)%32])
		}
		var exists int
		if err := s.db.QueryRow(`SELECT COUNT(*) FROM rooms WHERE code=?`, code).Scan(&exists); err == nil && exists == 0 {
			return code, nil
		}
	}
	return "", fmt.Errorf("room code space exhausted")
}

func (s *Store) GetRoom(code string) (*Room, error) {
	row := s.db.QueryRow(`SELECT code,name,owner_id,max_players,created_at,seed FROM rooms WHERE code=?`, code)
	var r Room
	err := row.Scan(&r.Code, &r.Name, &r.OwnerID, &r.MaxPlayers, &r.CreatedAt, &r.Seed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *Store) ListRooms() ([]Room, error) {
	rows, err := s.db.Query(`SELECT code,name,owner_id,max_players,created_at,seed FROM rooms ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.Code, &r.Name, &r.OwnerID, &r.MaxPlayers, &r.CreatedAt, &r.Seed); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) UpdateRoom(code string, name *string, maxPlayers *int) error {
	if name != nil {
		if _, err := s.db.Exec(`UPDATE rooms SET name=? WHERE code=?`, *name, code); err != nil {
			return err
		}
	}
	if maxPlayers != nil {
		if _, err := s.db.Exec(`UPDATE rooms SET max_players=? WHERE code=?`, *maxPlayers, code); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) DeleteRoom(code string) error {
	if _, err := s.db.Exec(`DELETE FROM room_players WHERE room_id=?`, code); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM buildings WHERE room_id=?`, code); err != nil {
		return err
	}
	if _, err := s.db.Exec(`DELETE FROM grants WHERE room_id=?`, code); err != nil {
		return err
	}
	_, err := s.db.Exec(`DELETE FROM rooms WHERE code=?`, code)
	return err
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

func (s *Store) CountRoomPlayers(code string) (int, error) {
	if code == "" {
		var n int
		err := s.db.QueryRow(`SELECT COUNT(*) FROM room_players WHERE alive=1`).Scan(&n)
		return n, err
	}
	var n int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM room_players WHERE room_id=? AND alive=1`, code).Scan(&n)
	return n, err
}

// JoinRoom upserts a player into a room's session, resetting per-room state.
func (s *Store) JoinRoom(code, playerID string, money, rating float64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO room_players(room_id,player_id,joined_at,money,rating,alive,score)
		VALUES(?,?,?,?,?,1,0)`, code, playerID, nowUnix(), money, rating)
	return err
}

// ---- room session rows -----------------------------------------------

type RoomPlayer struct {
	RoomID   string
	PlayerID string
	Money    float64
	Rating   float64
	Alive    bool
	Score    float64
	X, Z     float64
}

func (s *Store) GetRoomPlayer(roomID, playerID string) (*RoomPlayer, error) {
	row := s.db.QueryRow(`SELECT money,rating,alive,score,x,z FROM room_players WHERE room_id=? AND player_id=?`, roomID, playerID)
	var rp RoomPlayer
	rp.RoomID, rp.PlayerID = roomID, playerID
	err := row.Scan(&rp.Money, &rp.Rating, &rp.Alive, &rp.Score, &rp.X, &rp.Z)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rp, nil
}

// ---- leaderboard ------------------------------------------------------

type BestScoreRow struct {
	PlayerID  string  `json:"playerId"`
	Name      string  `json:"name"`
	BestScore float64 `json:"bestScore"`
}

func (s *Store) Leaderboard() ([]BestScoreRow, error) {
	rows, err := s.db.Query(`SELECT id,name,best_score FROM players WHERE best_score>0 ORDER BY best_score DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BestScoreRow
	for rows.Next() {
		var r BestScoreRow
		if err := rows.Scan(&r.PlayerID, &r.Name, &r.BestScore); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
