package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/prolarps/urbanist/backend/errcodes"
)

// handleAdminRooms: GET /api/admin/rooms — the live room table (§2.7).
func (s *Server) handleAdminRooms(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, s.rooms.List())
}

// handleAdminCloseRoom: DELETE /api/admin/rooms/{code} — force-closes any
// room regardless of owner; sockets drop with 4404 (§2.7).
func (s *Server) handleAdminCloseRoom(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if s.rooms.Delete(code) == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	s.hub.CloseRoom(code)
	sendJSON(w, http.StatusOK, map[string]any{"ok": true})
}
