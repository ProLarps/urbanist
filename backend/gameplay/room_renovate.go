package gameplay

import (
	"github.com/prolarps/urbanist/backend/config"
	"github.com/prolarps/urbanist/backend/errcodes"
)

// Renovate implements act{action:"renovate"} (§5): start one renovation
// project on a needs-cell of the player's ASSIGNED chunk. The full price
// is charged immediately; profit and rating land when Tick completes it.
// Instantaneous here: construction timers belong to the 10 Hz tick stage.
func (r *Room) Renovate(playerID string, cx, cz, cell int, scen string) ([]GameEvent, errcodes.Code) {
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
	if p.assigned == nil || *p.assigned != c { // §8: only the assigned chunk
		return nil, errcodes.Forbidden
	}
	held, ok := p.granted[c]
	if !ok {
		return nil, errcodes.NotFound
	}
	d, ok := r.world.chunks[c]
	if !ok || cell < 0 || cell >= len(d.Data) {
		return nil, errcodes.BadRequest
	}
	if d.Data[cell] != config.BUILDING_RENNOVATION || held.done[cell] {
		return nil, errcodes.BadRequest // not a needs-cell, or already done
	}
	if _, ok := scenarios[scen]; !ok {
		return nil, errcodes.BadRequest
	}
	cost, _ := projectCost(scen)
	if p.Money < cost {
		return nil, errcodes.BadRequest // §8: toast "not enough money"
	}
	p.Money -= cost
	held.done[cell] = true
	held.renovate[cell] = scen
	ev := GameEvent{Kind: "renovated", PlayerID: playerID,
		Text: scen}
	if allNeedsDone(d, held.done) {
		ev.Kind = "finished"
		r.finishRound(p)
	}
	return r.events(p, ev), errcodes.OK
}
