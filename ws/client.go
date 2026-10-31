package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	sendBufSiz = 64
	maxMsgSize = 1 << 16 // 64 KiB
)

// Client is one connected player.
type Client struct {
	hub  *Hub
	room *Room

	conn *websocket.Conn
	send chan Envelope

	ID   string
	Name string

	// Chunks this client has already been shown, keyed by "x,y".
	chunks map[[2]int]bool

	// ready is closed once the join phase (welcome, history, initial chunks)
	// has been fully enqueued; dispatch waits for it so ordering is stable.
	ready     chan struct{}
	readyOnce sync.Once
}

// markReady unblocks dispatch without hanging if the client died pre-join.
func (c *Client) markReady() { c.readyOnce.Do(func() { close(c.ready) }) }

func (c *Client) ReadPump() {
	defer func() {
		c.markReady() // never let a pre-join client hang in dispatch
		if c.room != nil {
			c.room.leave(c)
		}
		c.hub.unregister(c)
		c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(maxMsgSize)

	for {
		// Each message must arrive within pongWait of the read start, which
		// doubles as the liveness check.
		ctx, cancel := context.WithTimeout(context.Background(), pongWait)
		_, payload, err := c.conn.Read(ctx)
		cancel()
		if err != nil {
			return
		}

		var env Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			c.sendError("malformed message")
			continue
		}
		if err := c.hub.dispatch(c, env); err != nil {
			c.sendError(err.Error())
		}
	}
}

func (c *Client) WritePump() {
	defer c.conn.Close(websocket.StatusNormalClosure, "")
	for env := range c.send {
		payload, err := json.Marshal(env)
		if err != nil {
			log.Printf("marshal %s: %v", env.Type, err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), writeWait)
		err = c.conn.Write(ctx, websocket.MessageText, payload)
		cancel()
		if err != nil {
			return
		}
	}
}

func (c *Client) sendEnvelope(env Envelope) {
	select {
	case c.send <- env:
	default:
		// Slow client: close it rather than stalling the room.
		go c.conn.Close(websocket.StatusPolicyViolation, "send buffer full")
	}
}

func (c *Client) sendError(msg string) {
	c.sendEnvelope(Envelope{Type: TypeError, Data: mustJSON(ErrorMsg{Message: msg})})
}

// mustJSON is a helper for payload structs.
func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}
