package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Hub owns per-room game hubs and the WS handshake (docs/API.md §3).
type Hub struct {
	cfg   Config
	srv   *Server
	mu    sync.Mutex
	rooms map[string]*RoomHub
}

func NewHub(cfg Config) *Hub {
	return &Hub{cfg: cfg, rooms: map[string]*RoomHub{}}
}

func (h *Hub) attach(s *Server) { h.srv = s }

// CloseRoom closes all sockets in a deleted room with code 4404 (§2.4).
func (h *Hub) CloseRoom(code string) {
	h.mu.Lock()
	rh := h.rooms[code]
	delete(h.rooms, code)
	h.mu.Unlock()
	if rh != nil {
		rh.closeRoom()
	}
}

func (h *Hub) room(code string) *RoomHub {
	h.mu.Lock()
	defer h.mu.Unlock()
	if rh, ok := h.rooms[code]; ok {
		return rh
	}
	rh := newRoomHub(h, code)
	h.rooms[code] = rh
	return rh
}

// ---- RoomHub: per-room connection set + world ----------------------------

type RoomHub struct {
	hub  *Hub
	code string

	mu        sync.Mutex
	clients   map[string]*Client // playerId -> conn
	world     *World
	startedAt time.Time
}

func newRoomHub(hub *Hub, code string) *RoomHub {
	return &RoomHub{hub: hub, code: code, clients: map[string]*Client{}, startedAt: time.Now()}
}

func (rh *RoomHub) closeRoom() {
	rh.mu.Lock()
	clients := make([]*Client, 0, len(rh.clients))
	for _, c := range rh.clients {
		clients = append(clients, c)
	}
	rh.mu.Unlock()
	for _, c := range clients {
		c.close(websocket.StatusCode(4404), "room deleted")
	}
}

// broadcast sends a raw frame to every live client of the room.
func (rh *RoomHub) broadcast(frame []byte) {
	rh.mu.Lock()
	clients := make([]*Client, 0, len(rh.clients))
	for _, c := range rh.clients {
		clients = append(clients, c)
	}
	rh.mu.Unlock()
	for _, c := range clients {
		c.sendBytes(frame)
	}
}

// sendTo sends a raw frame to one live client if present.
func (rh *RoomHub) sendTo(playerID string, frame []byte) {
	rh.mu.Lock()
	c := rh.clients[playerID]
	rh.mu.Unlock()
	if c != nil {
		c.sendBytes(frame)
	}
}

// frame builds a flat {"t": ..., ...payload} message per docs/API.md §2.
// JSON field order does not matter to clients.
func frame(t string, payload any) []byte {
	merged := map[string]any{}
	if payload != nil {
		raw := mustJSON(payload)
		_ = json.Unmarshal(raw, &merged)
	}
	merged["t"] = t
	return mustJSON(merged)
}

// mustJSON marshals any payload; nil for empty.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// ---- WS handshake -----------------------------------------------------------

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		reject(w, r, websocket.StatusCode(4400), "token missing")
		return
	}
	payload, err := h.srv.auth.Verify(token)
	if err != nil {
		reject(w, r, websocket.StatusCode(4401), "token invalid or expired")
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	c := &Client{
		hub:  h,
		conn: conn,
		send: make(chan []byte, sendBufSize),
		id:   payload.PID,
		name: payload.Name,
		room: payload.Room,
	}

	room := h.room(payload.Room)
	room.mu.Lock()
	room.clients[c.id] = c
	if room.world == nil {
		// TODO: load from store when the store-backed world lands (§13); the
		// seed is registered in the rooms table.
		room.world = NewWorld(time.Now().UnixNano(), h.cfg)
	}
	room.mu.Unlock()

	c.sendBytes(room.authOK(c)) // spec §3: authOk immediately after token handshake
	go c.writePump()
	c.readPump(room) // blocks until disconnect; cleanup on return
}

// reject completes the WS handshake and immediately sends the requested
// close code so clients observe the spec codes (§3) instead of bare 4xx.
func reject(w http.ResponseWriter, r *http.Request, code websocket.StatusCode, reason string) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	_ = conn.Close(code, reason)
}

// ---- Client ------------------------------------------------------------------

const sendBufSize = 64

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte

	id   string
	name string
	room string

	closeOnce sync.Once
}

func (c *Client) readPump(room *RoomHub) {
	defer func() {
		room.mu.Lock()
		delete(room.clients, c.id)
		room.mu.Unlock()
		c.close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(1 << 16)
	for {
		// Liveness: each message must arrive within pongWait.
		ctx, cancel := context.WithTimeout(context.Background(), pongWait)
		_, payload, err := c.conn.Read(ctx)
		cancel()
		if err != nil {
			return
		}
		room.handleFrame(c, payload)
	}
}

func (c *Client) writePump() {
	defer c.close(websocket.StatusNormalClosure, "closing")
	for data := range c.send {
		ctx, cancel := context.WithTimeout(context.Background(), writeWait)
		err := c.conn.Write(ctx, websocket.MessageText, data)
		cancel()
		if err != nil {
			return
		}
	}
}

func (c *Client) sendBytes(frame []byte) {
	select {
	case c.send <- frame:
	default:
		go c.close(websocket.StatusCode(4002), "slow client")
	}
}

func (c *Client) close(status websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		close(c.send)
		_ = c.conn.Close(status, reason)
	})
}

// handleFrame: stage-1 subset of §5. authOk / ping-pong live here; movement,
// chunking, act/rounds and chat arrive with the game loop stage.
func (rh *RoomHub) handleFrame(c *Client, payload []byte) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(payload, &probe); err != nil {
		rh.sendTo(c.id, frame(MsgToast, Toast{Kind: ToastError, Text: "malformed frame"}))
		return
	}

	t := ""
	if raw, ok := probe["t"]; ok {
		_ = json.Unmarshal(raw, &t)
	}

	switch t {
	case MsgPing:
		rh.sendTo(c.id, frame(MsgPong, map[string]any{"n": frameNumber(probe)}))
	case MsgAuth:
		// Already-authenticated (token was validated in the handshake):
		// re-confirm the identity.
		rh.sendTo(c.id, rh.authOK(c))
	default:
		rh.sendTo(c.id, frame(MsgToast, Toast{Kind: ToastInfo, Text: "не реализовано на этой стадии"}))
	}
}

// frameNumber extracts the numeric "n" field: ping's payload used to be
// named "t" per the draft, which collided with the frame-type key — any
// JSON decoder keeps only one of them. Renamed to "n" (docs/API.md §0/§5).
func frameNumber(probe map[string]json.RawMessage) float64 {
	n := 0.0
	if raw, ok := probe["n"]; ok {
		_ = json.Unmarshal(raw, &n)
	}
	return n
}

// authOK composes the §6 authOk envelope.
func (rh *RoomHub) authOK(c *Client) []byte {
	cfg := rh.hub.cfg
	total := cfg.ChunksPerAxis * cfg.ChunksPerAxis * cfg.LotsPerChunk * cfg.LotsPerChunk
	you := YouDTO{
		ID: c.id, Name: c.name, Color: paletteByAvatar(0),
		Money: rh.hub.cfg.StartMoney, Rating: rh.hub.cfg.RatingStart,
		Alive: true, Room: rh.code,
	}
	return frame(MsgAuthOk, AuthOK{
		PlayerID:   c.id,
		You:        you,
		World:      WorldInfo{Chunks: cfg.ChunksPerAxis * cfg.ChunksPerAxis, LotsPerChunk: cfg.LotsPerChunk * cfg.LotsPerChunk, TotalLots: total, ViewChunks: cfg.ViewChunks * cfg.ViewChunks},
		ServerTime: time.Now().Unix(),
		Protocol:   2,
	})
}

// ---- small DTOs referenced above ---------------------------------------------

type YouDTO struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Color  string  `json:"color,omitempty"`
	Avatar int     `json:"avatar,omitempty"`
	Money  float64 `json:"money"`
	Rating float64 `json:"rating"`
	Alive  bool    `json:"alive"`
	Room   string  `json:"room"`
}

type WorldInfo struct {
	Chunks       int `json:"chunks"`
	LotsPerChunk int `json:"lotsPerChunk"`
	TotalLots    int `json:"totalLots"`
	ViewChunks   int `json:"viewChunks"`
}

type AuthOK struct {
	PlayerID   string    `json:"playerId"`
	You        YouDTO    `json:"you"`
	World      WorldInfo `json:"world"`
	ServerTime int64     `json:"serverTime"`
	Protocol   int       `json:"protocol"`
}

type Toast struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// ---- time helpers ------------------------------------------------------------

const (
	writeWait = 10 * time.Second
	pongWait  = 60 * time.Second
)

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
