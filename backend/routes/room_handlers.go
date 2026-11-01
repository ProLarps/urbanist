package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/prolarps/urbanist/backend/auth"
	"github.com/prolarps/urbanist/backend/errcodes"
	"github.com/prolarps/urbanist/backend/gameplay"
)

// roomCreateBody is the §2.4 create payload.
type roomCreateBody struct {
	Name       string `json:"name"`
	MaxPlayers int    `json:"maxPlayers"`
}

// handleRoomCreate: creates a wait-mode room; the creator becomes owner
// and enters it immediately (docs/API.md §2.4 v3).
func (s *Server) handleRoomCreate(w http.ResponseWriter, r *http.Request) {
	var body roomCreateBody
	if err := decodeOptional(r, &body); err != nil {
		sendError(w, errcodes.BadRequest, "invalid json")
		return
	}

	owner := s.memberOfClaims(claimsOf(r))
	room, code := s.rooms.CreateRoom(owner, body.Name, body.MaxPlayers)
	if code != errcodes.OK {
		sendError(w, code, code.Message())
		return
	}
	sendJSON(w, http.StatusCreated, room.View())
}

// handleRoomList: GET /api/rooms with live member counts.
func (s *Server) handleRoomList(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, s.rooms.List())
}

// handleRoomGet: GET /api/rooms/{code}.
func (s *Server) handleRoomGet(w http.ResponseWriter, r *http.Request) {
	room := s.roomOfURL(r)
	if room == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	sendJSON(w, http.StatusOK, room.View())
}

// handleRoomPatch renames a room or changes capacity (owner only, §2.4).
func (s *Server) handleRoomPatch(w http.ResponseWriter, r *http.Request) {
	room := s.roomOfURL(r)
	claims := claimsOf(r)
	if room == nil {
		sendError(w, errcodes.RoomNotFound, "room")
		return
	}
	if !room.OwnerIs(claims.PID) {
		sendError(w, errcodes.Forbidden, "only the room owner can modify it")
		return
	}
	var body roomCreateBody
	if err := decodeOptional(r, &body); err != nil {
		sendError(w, errcodes.BadRequest, "invalid json")
		return
	}
	if view, code := s.rooms.PatchRoom(room, body.Name, body.MaxPlayers); code != errcodes.OK {
		sendError(w, code, code.Message())
	} else {
		sendJSON(w, http.StatusOK, view)
	}
}

// memberOfClaims builds a gameplay member from the verified token claims.
func (s *Server) memberOfClaims(claims *auth.Claims) gameplay.Member {
	player, err := s.store.GetPlayerByID(claims.PID)
	if err != nil || player == nil {
		return gameplay.Member{ID: claims.PID, Name: claims.Name, Color: colorByAvatar(0)}
	}
	return gameplay.Member{ID: player.ID, Name: player.Name, Color: player.Color, Avatar: player.Avatar}
}

// roomOfURL resolves {code} or answers nil for unknown rooms.
func (s *Server) roomOfURL(r *http.Request) *gameplay.Room {
	return s.rooms.Get(chi.URLParam(r, "code"))
}
