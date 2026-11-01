package gameplay

import (
	"github.com/prolarps/urbanist/backend/errcodes"
)

// PatchRoom applies owner edits: rename and/or new capacity. Only wait-mode
// rooms can be modified; run-mode settings are already frozen in the
// room's config snapshot.
func (m *Manager) PatchRoom(room *Room, name string, maxPlayers int) (RoomView, errcodes.Code) {
	view, code := room.Patch(name, maxPlayers)
	return view, code
}
