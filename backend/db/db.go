package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, no cgo
)

// Store is the SQLite persistence layer (docs/API.md §12).
//
// Rooms are deliberately NOT stored here: they live in memory in the
// gameplay package (wait/run lifecycle) and are destroyed automatically
// when a game ends, so persisting them would just keep corpses.
type Store struct {
	db *sql.DB
}

// Open connects to an SQLite database, applies server pragmas and migrates.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
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

// migrate keeps the schema in sync; players are global, chat/journal/agent
// rows only reference rooms by their room code, which dies with the room.
func (s *Store) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS players(
  id TEXT PRIMARY KEY, name TEXT NOT NULL, avatar INTEGER NOT NULL DEFAULT 0,
  color TEXT NOT NULL, created_at INTEGER NOT NULL,
  last_seen INTEGER NOT NULL, best_score REAL NOT NULL DEFAULT 0);
CREATE TABLE IF NOT EXISTS rooms_cache(
  code TEXT PRIMARY KEY, name TEXT, owner_id TEXT, created_at INTEGER NOT NULL);
CREATE TABLE IF NOT EXISTS agent_state(
  agent_id TEXT, player_id TEXT, topics_mask INTEGER NOT NULL DEFAULT 0,
  said_count INTEGER NOT NULL DEFAULT 0, trust INTEGER NOT NULL DEFAULT 0,
  last_topic TEXT, last_ts INTEGER, PRIMARY KEY(agent_id, player_id));
CREATE TABLE IF NOT EXISTS agent_memory(
  id INTEGER PRIMARY KEY AUTOINCREMENT, agent_id TEXT, player_id TEXT,
  topic TEXT, said_count INTEGER, player_text TEXT, agent_text TEXT, ts INTEGER);
CREATE INDEX IF NOT EXISTS idx_agent_memory ON agent_memory(agent_id, player_id);
CREATE TABLE IF NOT EXISTS chat_log(
  id INTEGER PRIMARY KEY AUTOINCREMENT, room_id TEXT, chunk_x INTEGER,
  chunk_z INTEGER, player_id TEXT, agent_id TEXT, role TEXT, text TEXT, ts INTEGER);
CREATE INDEX IF NOT EXISTS idx_chat_room_chunk ON chat_log(room_id, chunk_x, chunk_z);
CREATE TABLE IF NOT EXISTS journal(
  id INTEGER PRIMARY KEY AUTOINCREMENT, room_id TEXT, ts INTEGER, kind TEXT,
  author TEXT, text TEXT, meta_json TEXT);
CREATE INDEX IF NOT EXISTS idx_journal_room ON journal(room_id, ts);
`
	_, err := s.db.Exec(schema)
	return err
}
