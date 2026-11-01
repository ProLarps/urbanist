package ws

import (
	"net/http"
	"sync"

	"github.com/coder/websocket"

	"github.com/prolarps/urbanist/backend/auth"
	"github.com/prolarps/urbanist/backend/gameplay"
)

// Hub owns all sockets: playerID -> client, plus the player's current
// room binding. It is the WebSocket layer's only stateful type.
type Hub struct {
	auth    *auth.Manager
	rooms   *gameplay.Manager
	mu      sync.Mutex
	clients map[string]*Client // playerID -> socket (admins get pseudo ids)
	roomOf  map[string]string  // playerID -> room code (lobby when absent)
}

// NewHub wires the hub to token issuing and the room manager.
func NewHub(authMan *auth.Manager, rooms *gameplay.Manager) *Hub {
	return &Hub{
		auth:    authMan,
		rooms:   rooms,
		clients: map[string]*Client{},
		roomOf:  map[string]string{},
	}
}

// CloseRoom closes every socket bound to the room with code 4404
// (room deleted by its owner or by an administrator).
func (h *Hub) CloseRoom(code string) {
	for _, c := range h.roomClients(code) {
		c.close(websocket.StatusCode(CloseRoomDeleted), "room deleted")
	}
}

// BroadcastRoom fans a frame out to every socket bound to the room.
func (h *Hub) BroadcastRoom(code string, data []byte) {
	for _, c := range h.roomClients(code) {
		c.sendLater(data)
	}
}

// roomClients snapshots the sockets currently bound to a room.
func (h *Hub) roomClients(code string) []*Client {
	h.mu.Lock()
	defer h.mu.Unlock()
	var targets []*Client
	for id, roomCode := range h.roomOf {
		if roomCode == code {
			targets = append(targets, h.clients[id])
		}
	}
	return targets
}

// LivePlayers counts live player sockets (health endpoint, §2.2).
// Admin sockets carry an empty player name, so they are not counted.
func (h *Hub) LivePlayers() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, c := range h.clients {
		if c.player.Name != "" {
			n++
		}
	}
	return n
}

// reject accepts the handshake and immediately closes with the requested
// spec code, so clients see 4400/4401 instead of a bare 4xx HTTP error.
func reject(w http.ResponseWriter, r *http.Request, code int, reason string) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	_ = conn.Close(websocket.StatusCode(code), reason)
}
