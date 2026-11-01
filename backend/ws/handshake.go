package ws

import (
	"net/http"

	"github.com/coder/websocket"

	"github.com/prolarps/urbanist/backend/gameplay"
)

// ServeWS is the HTTP upgrade endpoint /ws?token=<JWT> (docs/API.md §3).
// The token is verified before the connection is accepted.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		reject(w, r, CloseTokenMissing, "token missing")
		return
	}
	claims, err := h.auth.Verify(token)
	if err != nil {
		reject(w, r, CloseTokenInvalid, "token invalid or expired")
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	c := h.register(conn, claims)
	c.sendLater(h.authOKFrame(c))
	go c.writePump()
	c.readPump()
}

// authOKFrame composes §6 authOk: identity plus the room the player is in,
// if any (membership is re-resolved so reconnects rebind automatically).
func (h *Hub) authOKFrame(c *Client) []byte {
	h.mu.Lock()
	delete(h.roomOf, c.player.ID)
	h.mu.Unlock()

	var room *gameplay.RoomView
	if !c.isAdmin() {
		if r := h.rooms.PlayerRoom(c.player.ID); r != nil {
			h.mu.Lock()
			h.roomOf[c.player.ID] = r.Code
			h.mu.Unlock()
			view := r.View()
			room = &view
		}
	}
	return buildFrame(MsgAuthOk, map[string]any{
		"playerId": c.player.ID,
		"name":     c.player.Name,
		"room":     room,
	})
}

// RoomStateFrame is the canonical roster broadcast after any room change.
func RoomStateFrame(v gameplay.RoomView) []byte {
	return roomStateFrame(v)
}

// roomStateFrame is the internal alias used by the handlers.
func roomStateFrame(v gameplay.RoomView) []byte {
	return buildFrame(MsgRoomSt, map[string]any{"room": v})
}

// RoomStateBroadcast pushes the current roster to all sockets of the room;
// routes calls it right after every membership or mode mutation.
func (h *Hub) RoomStateBroadcast(code string) {
	if room := h.rooms.Get(code); room != nil {
		h.BroadcastRoom(code, roomStateFrame(room.View()))
	}
}
