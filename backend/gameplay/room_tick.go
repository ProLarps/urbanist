package gameplay

import "time"

// Tick advances every alive member's round state (§9): timeouts, skips
// and defeat checks. Routes call it from the room's periodic ticker; the
// injected clock keeps it testable without sleeping. Returns the events
// clients must receive (stats / gameOver per affected player).
func (r *Room) Tick(now time.Time) []GameEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.mode != ModeRun {
		return nil
	}
	var out []GameEvent
	for _, p := range r.members {
		if !p.Alive {
			continue
		}
		if p.assigned != nil && !now.Before(p.deadline) {
			r.timeoutRound(p)
			out = append(out, GameEvent{Kind: "timeout", PlayerID: p.ID})
		}
		r.skipExpired(p)
		out = append(out, r.events(p)...)
	}
	return out
}

// pushEvent appends a loop event to the room's outbox; the hub drains it
// via DrainEvents on the next broadcast tick (or immediately in tests).
// Callers hold r.mu.
func (r *Room) pushEvent(ev GameEvent) { r.pendingEvents = append(r.pendingEvents, ev) }

// DrainEvents returns and clears the queued outbox events.
func (r *Room) DrainEvents() []GameEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.pendingEvents
	r.pendingEvents = nil
	return out
}
