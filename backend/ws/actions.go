package ws

import (
	"github.com/prolarps/urbanist/backend/gameplay"
)

// handleRoomAct is the entry point for in-game actions in run mode
// (docs/API.md §5). The full economy loop (move/chunks/act/chat) belongs
// to later stages; this dispatcher keeps the frame routing honest about
// what the current build supports.
func (h *Hub) handleRoomAct(c *Client, body map[string]any) {
	action, _ := body["action"].(string)
	switch action {
	case "state": // client asks for the roster of its bound room
		h.replyRoomState(c)
	default:
		c.sendLater(toastFrame(ToastInfo, gameplay.NotYetImplemented))
	}
}

// replyRoomState answers with the current roomState frame, or a toast when
// the socket has no room binding yet (lobby players must join first).
func (h *Hub) replyRoomState(c *Client) {
	room := h.rooms.PlayerRoom(c.player.ID)
	if room == nil {
		c.sendLater(toastFrame(ToastError, "you are not in a room"))
		return
	}
	c.sendLater(roomStateFrame(room.View()))
}
