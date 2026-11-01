package gameplay

import (
	"math/rand/v2"
	"time"

	"github.com/prolarps/urbanist/backend/config"
	"github.com/prolarps/urbanist/backend/errcodes"
)

// GameEvent is one step of the §8 loop, produced by Pick/Renovate/Tick
// and translated into WS frames by the hub.
type GameEvent struct {
	Kind     string // assigned / stats / grant / gameOver / toast
	PlayerID string
	Text     string // toast payload (§6)
	Chunk    *ChunkDTO
	Options  []ChunkDTO
	State    PlayerStateDTO
	Deadline *time.Time // assigned event
	Cycle    int        // grant event
	AgentID  string     // agentSay event
	Reason   string     // gameOver reason: bankrupt / low_rating
}

// nowFunc is swappable in tests so round timers need no real waiting.
var nowFunc = time.Now

// Pick implements chunkPick (§5): put one held district under the round
// timer. Deadbeats are clamped to at least one second from "now".
func (r *Room) Pick(playerID string, cx, cz int) ([]GameEvent, errcodes.Code) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.members[playerID]
	if !ok {
		return nil, errcodes.NotMember
	}
	if r.mode != ModeRun || !p.Alive {
		return nil, errcodes.RoomBusy
	}
	c := chunkPos{cx, cz}
	if _, ok := p.granted[c]; !ok || p.assigned != nil {
		return nil, errcodes.BadRequest // not yours, or a round already runs
	}
	p.assigned = &c
	secs := r.cfg.RoundSeconds
	if secs < 1 {
		secs = 1
	}
	p.deadline = nowFunc().Add(time.Duration(secs) * time.Second)
	d, _ := r.world.Chunk(cx, cz)
	ev := GameEvent{Kind: "assigned", PlayerID: playerID, Chunk: &d, Deadline: &p.deadline}
	return r.events(p, ev), errcodes.OK
}

// seedLoop hands every alive member one random chunk (first grant).
// Callers must hold r.mu.
func (r *Room) seedLoop() {
	for _, p := range r.members {
		if len(p.granted) > 0 || !p.Alive {
			continue
		}
		if c, ok := r.randomFreeChunk(); ok {
			p.granted[c] = newHeld(c)
			r.owner[c] = p.ID
		}
	}
}

// randomFreeChunk picks any unowned district; used for the first grant.
func (r *Room) randomFreeChunk() (chunkPos, bool) {
	var free []chunkPos
	for cx := range config.ChunksPerAxis {
		for cz := range config.ChunksPerAxis {
			c := chunkPos{cx, cz}
			if _, taken := r.owner[c]; !taken {
				free = append(free, c)
			}
		}
	}
	if len(free) == 0 {
		return chunkPos{}, false
	}
	return free[rand.IntN(len(free))], true
}
