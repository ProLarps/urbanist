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
Kind     string // assigned / stats / grant / gameOver
PlayerID string
Chunk    *ChunkDTO
Options  []ChunkDTO
State    PlayerStateDTO
Reason   string // gameOver reason: bankrupt / low_rating
}

// seedLoop hands every alive member one random chunk (first grant).
// Callers must hold r.mu.
func (r *Room) seedLoop() {
for _, p := range r.members {
if len(p.granted) > 0 || !p.Alive {
continue
}
if c, ok := r.randomFreeChunk(); ok {
p.granted[c] = true
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

// nextGrant offers up to ChunksPerGrant neighbours of held districts (§8).
// A player holding nothing gets one fresh random chunk instead.
func (r *Room) nextGrant(p *roomPlayer) []ChunkDTO {
var cands []chunkPos
if len(p.granted) == 0 {
if c, ok := r.randomFreeChunk(); ok {
cands = []chunkPos{c}
}
} else {
cands = r.world.freeNeighbours(p.granted, func(c chunkPos) bool {
_, taken := r.owner[c]
return taken
})
}
rand.Shuffle(len(cands), func(i, j int) { cands[i], cands[j] = cands[j], cands[i] })
out := []ChunkDTO{}
for _, c := range cands {
if len(out) >= r.cfg.ChunksPerGrant {
break
}
p.granted[c] = true
r.owner[c] = p.ID
if d, ok := r.world.Chunk(c.cx, c.cz); ok {
out = append(out, d)
}
}
return out
}

// stateOf snapshots the §6 "stats" body for one member. Holds r.mu.
func (r *Room) stateOf(p *roomPlayer) PlayerStateDTO {
st := PlayerStateDTO{PlayerID: p.ID, Money: p.Money, Rating: p.Rating,
Cycle: p.cycle, Score: p.Money * p.Rating, Alive: p.Alive,
Skipped: p.skipped, Pending: []PendingRow{}}
for c := range p.granted {
row := PendingRow{CX: c.cx, CZ: c.cz, CyclesHeld: 1}
if p.assigned != nil && *p.assigned == c {
row.Deadline = p.deadline.Unix()
}
st.Pending = append(st.Pending, row)
}
return st
}

// events builds the standard follow-up frames after an economy change:
// stats always; gameOver when the §8 defeat conditions fire.
func (r *Room) events(p *roomPlayer, extra ...GameEvent) []GameEvent {
out := append(extra, GameEvent{Kind: "stats", PlayerID: p.ID, State: r.stateOf(p)})
if p.Alive && !r.canAffordAny() {
p.Alive = false
out = append(out, GameEvent{Kind: "gameOver", PlayerID: p.ID, Reason: "bankrupt"})
} else if p.Alive && p.Rating < r.cfg.RatingMin {
p.Alive = false
out = append(out, GameEvent{Kind: "gameOver", PlayerID: p.ID, Reason: "low_rating"})
}
return out
}

// canAffordAny: cheapest possible project = BASE_COST * cheapest scenario.
func (r *Room) canAffordAny() bool {
return p_minCost(r) <= r.minMoney()
}

func (r *Room) minMoney() float64 { return minScenarioCost }

var _ = errcodes.OK
var _ = time.Now
