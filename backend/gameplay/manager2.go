package gameplay

import (
	"crypto/rand"
	"strings"
)

// PlayerRoom returns the room the player currently belongs to, if any.
// A player may be in at most one room — rejoining another room first
// leaves the current one (docs/API.md §2.4 v3).
func (m *Manager) PlayerRoom(playerID string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rooms {
		if r.hasMember(playerID) {
			return r
		}
	}
	return nil
}

// Delete removes the room regardless of mode and returns the removed room
// (nil if there was none). The owner holds this right after leaving, too.
func (m *Manager) Delete(code string) *Room {
	m.mu.Lock()
	room := m.rooms[code]
	delete(m.rooms, code)
	m.mu.Unlock()
	return room
}

// List snapshots every live room for the GET /api/rooms listing.
func (m *Manager) List() []RoomView {
	m.mu.Lock()
	live := make([]*Room, 0, len(m.rooms))
	for _, r := range m.rooms {
		live = append(live, r)
	}
	m.mu.Unlock()

	views := make([]RoomView, 0, len(live))
	for _, r := range live {
		views = append(views, r.View())
	}
	return views
}

// hasMember reports membership; called with the manager lock held.
func (r *Room) hasMember(playerID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.members[playerID]
	return ok
}

// randomRoomCode builds a 6-char code from the no-lookalike alphabet.
func randomRoomCode() string {
	const codeLen = 6
	b := make([]byte, codeLen)
	_, _ = rand.Read(b)
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(roomCodeAlphabet[int(c)%len(roomCodeAlphabet)])
	}
	return sb.String()
}
