package ws

import (
	"context"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Client is one live WebSocket connection, authenticated by a JWT.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	player    MemberLite
	closeOnce sync.Once
}

// MemberLite is the socket identity: a player, or a pseudo-id for admins.
type MemberLite struct {
	ID   string
	Name string
}

const (
	sendBufSize = 64
	writeWait   = 10 * time.Second
	pongWait    = 60 * time.Second
)

// newClient builds a client with a buffered send queue.
func newClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{hub: hub, conn: conn, send: make(chan []byte, sendBufSize)}
}

// isAdmin separates the admin path in frame handlers for readability.
func (c *Client) isAdmin() bool { return c.player.Name == "" }

// readPump drains frames until disconnect; cleanup runs on return.
func (c *Client) readPump() {
	defer func() {
		c.hub.drop(c)
		c.close(websocket.StatusNormalClosure, "")
	}()
	c.conn.SetReadLimit(1 << 16)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), pongWait)
		_, payload, err := c.conn.Read(ctx)
		cancel()
		if err != nil {
			return
		}
		c.hub.handleFrame(c, payload)
	}
}

// writePump serialises outgoing frames; closing c.send ends the pump.
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

// sendLater enqueues a frame; a slow client is dropped instead of blocking
// the room's broadcast fan-out.
func (c *Client) sendLater(frame []byte) {
	select {
	case c.send <- frame:
	default:
		go c.close(websocket.StatusInternalError, "slow client")
	}
}

func (c *Client) close(status websocket.StatusCode, reason string) {
	c.closeOnce.Do(func() {
		close(c.send)
		_ = c.conn.Close(status, reason)
	})
}
