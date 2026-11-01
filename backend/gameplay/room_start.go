package gameplay

import (
	"github.com/prolarps/urbanist/backend/errcodes"
)

// startCheck reports whether a player has a live socket. Start requires
// everyone to be connected, like in classic RTS lobbies.
type startCheck func(playerID string) bool

// Start switches the room into run mode. Any member may press start, but
// only when every member is connected. The room takes its config snapshot
// here, so later admin edits of GlobalConfig cannot touch this game.
func (r *Room) Start(online startCheck) errcodes.Code {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mode == ModeRun {
		return errcodes.RoomBusy
	}
	if len(r.members) == 0 {
		return errcodes.NotFound
	}
	for id := range r.members {
		if !online(id) {
			return errcodes.NotEveryoneConnected
		}
	}
	r.cfg = GetGlobalConfig() // frozen for the whole run
	r.world = NewWorld(r.Seed, r.cfg)
	r.owner = map[chunkPos]string{}
	r.will = map[chunkPos]float64{}
	r.mode = ModeRun
	r.seedLoop() // first grant: 1 random chunk per player (§8 step 1)
	return errcodes.OK
}

// GrantOptions returns the districts currently offered to a member:
// their held-but-unassigned chunks (the "grant" frame of §6).
func (r *Room) GrantOptions(playerID string) ([]ChunkDTO, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.members[playerID]
	if !ok {
		return nil, false
	}
	out := []ChunkDTO{}
	for _, held := range p.granted {
		c := held.pos()
		if p.assigned != nil && *p.assigned == c {
			continue // already picked, shown via assigned instead
		}
		if d, ok := r.world.Chunk(c.cx, c.cz); ok {
			out = append(out, d)
		}
	}
	return out, true
}

// bestScores reports every member's session-best score (money x rating),
// which routes persist into player profiles (§2.3, §11). Holds r.mu.
func (r *Room) bestScores() map[string]float64 {
	out := make(map[string]float64, len(r.members))
	for id, p := range r.members {
		if p.best < scoreOf(p) {
			p.best = scoreOf(p)
		}
		out[id] = p.best
	}
	return out
}

// Get returns one member's economic state; ok=false for non-members.
func (r *Room) Get(playerID string) (Member, float64, float64, int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.members[playerID]
	if !ok {
		return Member{}, 0, 0, 0, false
	}
	return p.Member, p.Money, p.Rating, p.cycle, true
}

// Chunk exposes one district of the world with copied cells, so callers
// cannot mutate the live grid. Works only after the room started.
func (r *Room) Chunk(cx, cz int) (ChunkDTO, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mode != ModeRun || r.world == nil {
		return ChunkDTO{}, false
	}
	return r.world.Chunk(cx, cz)
}

// Scoreboard lists the members sorted by score = money x rating (§2.3).
// online keeps the room lock scope short: the socket probe runs unlocked.
func (r *Room) Scoreboard(online startCheck) []ScoreRow {
	r.mu.Lock()
	rows := make([]ScoreRow, 0, len(r.members))
	for id, p := range r.members {
		rows = append(rows, ScoreRow{
			PlayerID: p.ID, Name: p.Name,
			Money: p.Money, Rating: p.Rating,
			Score:  p.Money * p.Rating,
			Online: online(id),
		})
	}
	r.mu.Unlock()
	return sortScores(rows)
}
