package ws

// handleFrame routes one client frame by its "t" (docs/API.md §5).
// This stage implements connection bookkeeping: ping, re-auth and room
// binding. Gameplay actions (move, chunks, act, chat) arrive later stages.
func (h *Hub) handleFrame(c *Client, payload []byte) {
	t, body, ok := probe(payload)
	if !ok {
		c.sendLater(toastFrame(ToastError, "malformed frame"))
		return
	}
	switch t {
	case MsgPing:
		c.sendLater(buildFrame(MsgPong, map[string]any{"n": frameNumber(body)}))
	case MsgAuth:
		c.sendLater(h.authOKFrame(c))
	case MsgRoomJoin:
		h.handleRoomJoin(c)
	default:
		c.sendLater(toastFrame(ToastInfo, "not implemented at this stage"))
	}
}

// handleRoomJoin binds the socket to the room the player belongs to
// (membership is created over HTTP; see docs/API.md §2.4).
func (h *Hub) handleRoomJoin(c *Client) {
	if c.isAdmin() || c.player.ID == "" {
		c.sendLater(toastFrame(ToastError, "administrators cannot join rooms"))
		return
	}
	room := h.rooms.PlayerRoom(c.player.ID)
	if room == nil {
		c.sendLater(toastFrame(ToastError, "you are not in a room; use POST /api/rooms"))
		return
	}
	h.mu.Lock()
	h.roomOf[c.player.ID] = room.Code
	h.mu.Unlock()
	view := room.View()
	c.sendLater(roomStateFrame(view))
	h.BroadcastRoom(room.Code, roomStateFrame(view))
}

// toastFrame builds the §6 toast frame.
func toastFrame(kind, text string) []byte {
	return buildFrame(MsgToast, map[string]any{"kind": kind, "text": text})
}
