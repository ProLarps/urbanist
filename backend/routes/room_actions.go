package routes

import (
	"net/http"

	"github.com/prolarps/urbanist/backend/errcodes"
	"github.com/prolarps/urbanist/backend/gameplay"
	"github.com/prolarps/urbanist/backend/ws"
)

// handleRoomJoin: POST /api/rooms/{code}/join — enters the wait-mode room.
func (s *Server) handleRoomJoin(w http.ResponseWriter, r *http.Request) {
	room := s.roomOfURL(r)
	if room == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	if code := room.Join(s.memberOfClaims(claimsOf(r))); code != errcodes.OK {
		sendError(w, code, code.Message())
		return
	}
	s.broadcastRoomState(room)
	sendJSON(w, http.StatusOK, room.View())
}

// handleRoomLeave: POST /api/rooms/{code}/leave — anyone may leave,
// including the owner; the room survives (§2.4 v3: leave, then delete).
func (s *Server) handleRoomLeave(w http.ResponseWriter, r *http.Request) {
	room := s.roomOfURL(r)
	if room == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	if !room.Leave(claimsOf(r).PID) {
		sendError(w, errcodes.NotMember, "you are not in this room")
		return
	}
	s.broadcastRoomState(room)
	sendJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleRoomStart: POST /api/rooms/{code}/start — any member may press
// start once everyone is connected (RTS-lobby style, docs/API.md §2.4 v3).
func (s *Server) handleRoomStart(w http.ResponseWriter, r *http.Request) {
	room := s.roomOfURL(r)
	if room == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	if code := room.Start(s.hub.IsOnline); code != errcodes.OK {
		sendError(w, code, code.Message())
		return
	}
	s.broadcastRoomState(room)
	sendJSON(w, http.StatusOK, room.View())
}

// handleRoomScoreboard: GET /api/rooms/{code}/scoreboard (§2.3).
func (s *Server) handleRoomScoreboard(w http.ResponseWriter, r *http.Request) {
	room := s.roomOfURL(r)
	if room == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	sendJSON(w, http.StatusOK, room.Scoreboard(s.hub.IsOnline))
}

// handleRoomDelete: DELETE /api/rooms/{code} — owner only (§2.4); active
// sockets close with 4404. Administrators use the admin panel for this.
func (s *Server) handleRoomDelete(w http.ResponseWriter, r *http.Request) {
	room := s.roomOfURL(r)
	if room == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	if !room.OwnerIs(claimsOf(r).PID) {
		sendError(w, errcodes.Forbidden, "only the room owner can delete it")
		return
	}
	s.rooms.Delete(room.Code) // rooms are ephemeral: just vanish
	s.hub.CloseRoom(room.Code)
	sendJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// broadcastRoomState pushes the fresh roster to everyone in the room.
func (s *Server) broadcastRoomState(room *gameplay.Room) {
	s.hub.BroadcastRoom(room.Code, ws.RoomStateFrame(room.View()))
}
