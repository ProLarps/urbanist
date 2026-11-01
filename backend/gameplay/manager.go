package gameplay

import (
	"sync"
	"time"

	"github.com/prolarps/urbanist/backend/errcodes"
)

// roomCodeAlphabet avoids look-alike characters (docs/API.md §2.4).
const roomCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

// Manager owns every ephemeral room. Rooms persist only in memory: a
// finished or deleted game vanishes with all its state (§2.4).
type Manager struct {
	mu    sync.Mutex
	rooms map[string]*Room
}

func NewManager() *Manager {
	return &Manager{rooms: map[string]*Room{}}
}

// CreateRoom makes a wait-mode room and puts the owner in it.
func (m *Manager) CreateRoom(owner Member, name string, maxPlayers int) (*Room, errcodes.Code) {
	if maxPlayers < 1 {
		maxPlayers = 8
	}
	r := &Room{
		Code:       uniqueRoomCode(m),
		Name:       pickRoomName(name, owner.Name),
		OwnerID:    owner.ID,
		MaxPlayers: maxPlayers,
		CreatedAt:  time.Now(),
		Seed:       time.Now().UnixNano(),
		mode:       ModeWait,
		members:    map[string]*roomPlayer{},
	}
	if code := r.Join(owner); code != errcodes.OK {
		return nil, code
	}
	m.mu.Lock()
	m.rooms[r.Code] = r
	m.mu.Unlock()
	return r, errcodes.OK
}

func pickRoomName(name, playerName string) string {
	if name != "" {
		return name
	}
	return "Room " + playerName
}

// uniqueRoomCode retries until the alphabet space yields a free code.
func uniqueRoomCode(m *Manager) string {
	for {
		code := randomRoomCode()
		if m.Get(code) == nil {
			return code
		}
	}
}

// Get returns a room by code, or nil.
func (m *Manager) Get(code string) *Room {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.rooms[code]
}
