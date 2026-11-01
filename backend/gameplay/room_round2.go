package gameplay

// finishRound closes a successfully renovated round (§8): payout, cycle++,
// release of the chunk back to the city and the next neighbour grant.
// Callers hold r.mu.
func (r *Room) finishRound(p *roomPlayer) {
	c := *p.assigned
	held := p.granted[c]
	r.applyProjectPayout(p, held)
	delete(p.granted, c)
	delete(r.owner, c) // renovated districts return to the common pool
	p.assigned, p.deadline = nil, nowFunc()
	p.cycle++
	options := r.nextGrant(p)
	r.pushEvent(GameEvent{Kind: "grant", PlayerID: p.ID, Options: options})
}

// timeoutRound punishes an expired round timer (§8): fixed rating hit and
// a fraction of money; the chunk stays pending and simply ages one cycle.
// Callers hold r.mu.
func (r *Room) timeoutRound(p *roomPlayer) {
	p.Rating -= r.cfg.TimeoutRatingLoss
	p.Money *= 1 - r.cfg.TimeoutMoneyRate
	p.assigned, p.deadline = nil, nowFunc()
	p.cycle++
}

// skipExpired removes districts held for MaxCyclesHeld cycles without any
// renovation attempt (§8 "skipped"): one-time penalty, chunk goes public.
// Callers hold r.mu.
func (r *Room) skipExpired(p *roomPlayer) {
	limit := r.cfg.MaxCyclesHeld
	if limit < 1 {
		limit = 1
	}
	for c, held := range p.granted {
		if held.age <= limit || (p.assigned != nil && *p.assigned == c) {
			held.age++
			continue
		}
		p.Rating -= r.cfg.SkipRatingLoss
		p.Money *= 1 - r.cfg.SkipMoneyRate
		p.skipped++
		delete(p.granted, c)
		delete(r.owner, c)
	}
}
