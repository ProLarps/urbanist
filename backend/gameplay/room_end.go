package gameplay

// EndGame closes the room when its game finishes (docs/API.md §2.4 v3:
// rooms auto-destroy after the game ends, the owner never cleans up).
// The gameplay loop calls it with cleanup; wait-mode rooms ignore it.
func (r *Room) EndGame(onClosed func(code string)) {
	r.mu.Lock()
	if r.mode != ModeRun {
		r.mu.Unlock()
		return
	}
	r.mode = ""
	r.mu.Unlock()
	if onClosed != nil {
		onClosed(r.Code)
	}
}

// FinishGame is on the manager because destruction is a manager concern.
func (m *Manager) FinishGame(code string, notify func(closedCode string)) {
	if room := m.Delete(code); room != nil && notify != nil {
		notify(code)
	}
}
