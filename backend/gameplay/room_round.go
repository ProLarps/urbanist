package gameplay

import "github.com/prolarps/urbanist/backend/config"

// needsCells lists renovation-marker cells of one district (§8: all of
// them must reach "done" before the round closes successfully).
func needsCells(d *District) []int {
	var out []int
	for i, v := range d.Data {
		if v == config.BUILDING_RENNOVATION {
			out = append(out, i)
		}
	}
	return out
}

// allNeedsDone reports whether every needs-cell of the chunk is renovated.
func allNeedsDone(d *District, done map[int]bool) bool {
	for _, cell := range needsCells(d) {
		if !done[cell] {
			return false
		}
	}
	return true
}

// applyProjectPayout settles one finished project (§8 "Начисление"):
// one-time profit plus the citizens'-will rating shift of its scenario.
func (r *Room) applyProjectPayout(p *roomPlayer, held *heldChunk) {
	will := r.willOf(held.pos())
	for _, scen := range held.renovate {
		s := scenarios[scen]
		cost, _ := projectCost(scen)
		p.Money += cost * s.profitRate
		p.Rating += r.cfg.RatingStep*(will-0.5)*2 + s.willShift*0.5
	}
	held.renovate = map[int]string{}
}

// willOf reads the (persuasion-shifted) citizens' will of a chunk; the
// district base applies until the chat/persuasion stage stores overrides.
func (r *Room) willOf(c chunkPos) float64 {
	if w, ok := r.will[c]; ok {
		return w
	}
	if d, ok := r.world.chunks[c]; ok {
		return d.Will
	}
	return 0.5
}
