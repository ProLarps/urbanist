package gameplay

// AliveCount reports how many members can still play; when it drops to 1
// (or 0) the run-mode game is over and the room must be destroyed.
func (r *Room) AliveCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, p := range r.members {
		if p.Alive {
			n++
		}
	}
	return n
}

// Persuade shifts the citizens' will of one held chunk after an LLM
// persuasion attempt (§8 "Воля жителей"). Clamped to [0,1]; only chunks
// the player actually holds can be influenced. Callers hold nothing.
func (r *Room) Persuade(playerID string, c chunkPos, shift float64) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.members[playerID]
	if !ok || r.mode != ModeRun {
		return false
	}
	if _, held := p.granted[c]; !held {
		return false // cannot lobby districts that are not yours
	}
	will := clampFloat(r.willOf(c)+shift, 0, 1)
	r.will[c] = will
	return true
}

// WillOf exposes the current will value of a chunk (tests, chat replies).
func (r *Room) WillOf(c chunkPos) float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.willOf(c)
}

// AssignedChunk returns the chunk currently under the round timer.
func (r *Room) AssignedChunk(playerID string) (chunkPos, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.members[playerID]
	if !ok || p.assigned == nil {
		return chunkPos{}, false
	}
	return *p.assigned, true
}

// BestScores settles every member's session-best score (money x rating,
// §11) and returns the final table for persistence into profiles.
func (r *Room) BestScores() map[string]float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.bestScores()
}

// endChecked is the manager hook used by routes: after any event batch,
// destroy the room when the game ended (all but one player defeated).
func (m *Manager) endChecked(code string, notify func(string)) {
	m.FinishGame(code, notify)
}
