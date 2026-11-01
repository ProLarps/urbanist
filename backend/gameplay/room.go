package gameplay

import (
	"sync"
	"time"

	"github.com/prolarps/urbanist/backend/config"
	"github.com/prolarps/urbanist/backend/errcodes"
)

// Room modes: wait is the lobby ("Dawn of War" style), run is the game.
const (
	ModeWait = "wait"
	ModeRun  = "run"
)

// heldChunk tracks one district a player owns but has not renovated yet:
// how many cycles it survived unassigned (skip rule §8) and which of its
// renovation cells are already done.
type heldChunk struct {
	cx, cz   int
	age      int
	done     map[int]bool
	renovate map[int]string // cell index -> scenario, for profit on finish
}

func newHeld(c chunkPos) *heldChunk {
	return &heldChunk{cx: c.cx, cz: c.cz, age: 1,
		done: map[int]bool{}, renovate: map[int]string{}}
}

// pos converts back to the world-grid coordinate.
func (h *heldChunk) pos() chunkPos { return chunkPos{h.cx, h.cz} }

// roomPlayer is one participant with per-room economic state.
type roomPlayer struct {
	Member
	Money  float64
	Rating float64
	Alive  bool

	// Game loop state (§8), valid only in run mode: cycle counter, held
	// districts, the one under the round timer and its deadline.
	cycle    int
	granted  map[chunkPos]*heldChunk
	assigned *chunkPos
	deadline time.Time
	skipped  int
	best     float64 // best session score, persisted to the profile (§11)
}

func newRoomPlayer(m Member, cfg config.Config) *roomPlayer {
	return &roomPlayer{
		Member: m, Money: cfg.StartMoney, Rating: cfg.RatingStart,
		Alive: true, granted: map[chunkPos]*heldChunk{},
	}
}

// Room is one independent city lobby. It is ephemeral by design: it lives
// only in memory of the Manager and dies when the game ends (room_end.go).
type Room struct {
	mu         sync.Mutex
	Code       string
	Name       string
	OwnerID    string // owns deletion, even long after leaving
	MaxPlayers int
	CreatedAt  time.Time
	Seed       int64

	mode    string
	members map[string]*roomPlayer

	// Set when the room enters run mode; nil in wait mode.
	world *World
	cfg   config.Config
	owner map[chunkPos]string  // chunk -> player holding it (§8)
	will  map[chunkPos]float64 // citizens' will, shifted by persuasion (§8)

	pendingEvents []GameEvent // loop outbox drained by the hub (§6)
}

// View snapshots the roster for HTTP/WS consumers.
func (r *Room) View() RoomView {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.viewLocked()
}

// viewLocked builds the snapshot; callers must already hold r.mu.
func (r *Room) viewLocked() RoomView {
	members := make([]Member, 0, len(r.members))
	for _, p := range r.members {
		members = append(members, p.Member)
	}
	return RoomView{
		Code: r.Code, Name: r.Name, OwnerID: r.OwnerID,
		Mode: r.mode, MaxPlayers: r.MaxPlayers,
		Members: members, CreatedAt: r.CreatedAt.Unix(),
	}
}

// OwnerIs reports whether playerID is the room owner.
func (r *Room) OwnerIs(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.OwnerID == playerID
}

// Patch applies owner edits; only wait-mode rooms can be renamed or
// resized because run-mode settings are frozen in the start snapshot.
func (r *Room) Patch(name string, maxPlayers int) (RoomView, errcodes.Code) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mode != ModeWait {
		return RoomView{}, errcodes.RoomBusy
	}
	if name != "" {
		r.Name = name
	}
	if maxPlayers != 0 {
		r.MaxPlayers = clampRange(maxPlayers, 1, 64)
	}
	return r.viewLocked(), errcodes.OK
}
