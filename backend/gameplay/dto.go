package gameplay

import (
	"github.com/prolarps/urbanist/backend/config"
)

// Member is a player as seen from a room's roster.
type Member struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Color  string `json:"color"`
	Avatar int    `json:"avatar"`
}

// RoomView is the HTTP/WS representation of a room (docs/API.md §2.4).
type RoomView struct {
	Code       string   `json:"code"`
	Name       string   `json:"name"`
	OwnerID    string   `json:"ownerId"`
	Mode       string   `json:"mode"` // "wait" or "run"
	MaxPlayers int      `json:"maxPlayers"`
	Members    []Member `json:"players"`
	CreatedAt  int64    `json:"createdAt"`
}

// ScoreRow is one scoreboard line; score = money x rating (§2.3).
type ScoreRow struct {
	PlayerID string  `json:"playerId"`
	Name     string  `json:"name"`
	Money    float64 `json:"money"`
	Rating   float64 `json:"rating"`
	Score    float64 `json:"score"`
	Online   bool    `json:"online"`
}

// ChunkDTO describes one generated district: the renovation type playing
// the chunk's theme plus the flat array of cells (index = y*size + x,
// values are config.BUILDING_*).
type ChunkDTO struct {
	CX          int     `json:"cx"`
	CZ          int     `json:"cz"`
	Rennovation int     `json:"rennovation"`
	Will        float64 `json:"will"` // citizens' will 0..1 (§8)
	Size        int     `json:"size"`
	Cells       []uint8 `json:"cells"`
}

// PendingRow is one held district in stats.pending (docs/API.md §6).
type PendingRow struct {
	CX         int   `json:"cx"`
	CZ         int   `json:"cz"`
	CyclesHeld int   `json:"cyclesHeld"`
	Deadline   int64 `json:"deadline,omitempty"` // unix seconds, 0 if none
}

// PlayerStateDTO is the "stats" frame body (§6): economy + held districts.
type PlayerStateDTO struct {
	PlayerID string       `json:"playerId"`
	Money    float64      `json:"money"`
	Rating   float64      `json:"rating"`
	Cycle    int          `json:"cycle"`
	Score    float64      `json:"score"`
	Alive    bool         `json:"alive"`
	Pending  []PendingRow `json:"pending"`
	Skipped  int          `json:"skipped"`
}

// WorldInfo mirrors docs/API.md §2.1 "world" block.
type WorldInfo struct {
	Chunks       int `json:"chunks"`
	LotsPerChunk int `json:"lotsPerChunk"`
	TotalLots    int `json:"totalLots"`
	ViewChunks   int `json:"viewChunks"`
}

// WorldInfoOf computes the world meta block from a config.
func WorldInfoOf(cfg config.Config) WorldInfo {
	perChunk := cfg.ChunkSize * cfg.ChunkSize
	total := config.ChunksPerAxis * config.ChunksPerAxis * perChunk
	return WorldInfo{
		Chunks:       config.ChunksPerAxis * config.ChunksPerAxis,
		LotsPerChunk: perChunk,
		TotalLots:    total,
		ViewChunks:   config.ViewChunks * config.ViewChunks,
	}
}
