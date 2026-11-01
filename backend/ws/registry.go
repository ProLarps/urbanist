package ws

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/prolarps/urbanist/backend/auth"
)

// adminSeq numbers admin pseudo-ids; players always have their profile id.
var adminSeq atomic.Uint64

// register stores a fresh connection. Admins get an anonymised pseudo-id
// because they are not players and must never appear in rosters.
func (h *Hub) register(conn *websocket.Conn, claims *auth.Claims) *Client {
	c := newClient(h, conn)
	c.player = identityOf(claims)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.player.ID] = c
	delete(h.roomOf, c.player.ID)
	return c
}

// identityOf derives the socket identity from the token claims.
func identityOf(claims *auth.Claims) MemberLite {
	if claims.Role != auth.RolePlayer {
		return MemberLite{ID: adminPseudoID()}
	}
	return MemberLite{ID: claims.PID, Name: claims.Name}
}

// adminPseudoID builds a unique, non-profile key for admin sockets.
func adminPseudoID() string {
	return fmt.Sprintf("admin-%d-%d", adminSeq.Add(1), time.Now().UnixNano())
}

// drop removes the socket on disconnect (readPump defer).
func (h *Hub) drop(c *Client) {
	h.mu.Lock()
	delete(h.clients, c.player.ID)
	delete(h.roomOf, c.player.ID)
	h.mu.Unlock()
}

// IsOnline reports a live socket for the everyone-connected start check.
func (h *Hub) IsOnline(playerID string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.clients[playerID]
	return ok
}
