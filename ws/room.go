package ws

import (
	"sync"
)

// Room is a lobby with its own world state (district, chat, vote, NPCs).
type Room struct {
	name string

	mu      sync.RWMutex
	members map[string]*Client

	// Subsystems scoped to this room; wired up by the Hub at creation.
	chat  *Chat
	world *World
	vote  *Vote
	npc   NPCResolver
}

func (r *Room) join(c *Client) {
	c.room = r

	r.mu.Lock()
	r.members[c.ID] = c
	r.mu.Unlock()

	// First message to the joiner confirms their identity.
	c.sendEnvelope(Envelope{Type: TypeJoined, Data: mustJSON(MemberInfo{ClientID: c.ID, Name: c.Name})})
	// Everyone else, and the joiner too, sees the join event.
	r.broadcast(Envelope{Type: TypeJoined, Data: mustJSON(MemberInfo{ClientID: c.ID, Name: c.Name})})

	// Catch the joiner up: chat history + first chunks around district origin.
	r.chat.sendHistory(c)
	r.sendChunkArea(c, 1) // 3x3 chunks around (0,0)
	// An already running vote is re-announced to the latecomer.
	r.vote.syncWith(c)

	// Join phase enqueued; unblock dispatch so inbound messages are served.
	c.markReady()
}

// rename updates a member's display name and tells the room.
func (r *Room) rename(c *Client, name string) {
	if name == "" {
		c.sendError("empty name")
		return
	}
	c.Name = name
	r.broadcast(Envelope{Type: TypeMemberUpd, Data: mustJSON(MemberInfo{ClientID: c.ID, Name: c.Name})})
}

func (r *Room) leave(c *Client) {
	r.mu.Lock()
	delete(r.members, c.ID)
	r.mu.Unlock()

	c.room = nil
	r.broadcast(Envelope{Type: TypeLeft, Data: mustJSON(MemberInfo{ClientID: c.ID, Name: c.Name})})
}

// broadcast sends env to every current member without holding mu during sends.
func (r *Room) broadcast(env Envelope) {
	r.mu.RLock()
	members := make([]*Client, 0, len(r.members))
	for _, m := range r.members {
		members = append(members, m)
	}
	r.mu.RUnlock()
	for _, m := range members {
		m.sendEnvelope(env)
	}
}

// sendChunkArea sends a (2*radius+1)^2 grid of chunks centered on origin.
func (r *Room) sendChunkArea(c *Client, radius int) {
	var coords [][2]int
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			coords = append(coords, [2]int{x, y})
		}
	}
	r.sendChunks(c, coords)
}

// sendChunks delivers chunks the client has not seen yet, deduping repeats.
func (r *Room) sendChunks(c *Client, coords [][2]int) {
	for _, xy := range coords {
		if c.chunks[xy] {
			continue
		}
		c.chunks[xy] = true
		chunk := r.world.ChunkAt(xy[0], xy[1])
		c.sendEnvelope(Envelope{Type: TypeChunk, Data: mustJSON(chunk)})
	}
}

// broadcastChunkUpdate pushes regenerated chunk data to everyone in the room.
func (r *Room) broadcastChunkUpdate(coords [][2]int, reason string) {
	for _, xy := range coords {
		chunk := r.world.ChunkAt(xy[0], xy[1])
		r.broadcast(Envelope{
			Type: TypeChunkUpd,
			Data: mustJSON(ChunkUpdate{ChunkData: chunk, Reason: reason}),
		})
	}
}
