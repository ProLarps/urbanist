package gameplay

import (
	"math/rand/v2"
)

// nextGrant offers up to ChunksPerGrant neighbours of held districts (§8).
// A player holding nothing gets one fresh random chunk instead. Holds r.mu.
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
		p.granted[c] = newHeld(c)
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
		Cycle: p.cycle, Score: scoreOf(p), Alive: p.Alive,
		Skipped: p.skipped, Pending: []PendingRow{}}
	for _, held := range p.granted {
		row := PendingRow{CX: held.cx, CZ: held.cz, CyclesHeld: held.age}
		if p.assigned != nil && *p.assigned == held.pos() {
			row.Deadline = p.deadline.Unix()
		}
		st.Pending = append(st.Pending, row)
	}
	return st
}

// statsFor answers a stats request from outside the room lock (§5 "stats").
func (r *Room) statsFor(playerID string) (PlayerStateDTO, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.members[playerID]
	if !ok {
		return PlayerStateDTO{}, false
	}
	return r.stateOf(p), true
}

// events builds the standard follow-up frames after an economy change:
// stats always; gameOver when the §8 defeat conditions fire.
func (r *Room) events(p *roomPlayer, extra ...GameEvent) []GameEvent {
	out := append(extra, GameEvent{Kind: "stats", PlayerID: p.ID, State: r.stateOf(p)})
	if p.Alive && p.Money < cheapestScenarioCost() {
		p.Alive = false
		out = append(out, GameEvent{Kind: "gameOver", PlayerID: p.ID, Reason: "bankrupt"})
	} else if p.Alive && p.Rating < r.cfg.RatingMin {
		p.Alive = false
		out = append(out, GameEvent{Kind: "gameOver", PlayerID: p.ID, Reason: "low_rating"})
	}
	return out
}
