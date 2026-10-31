package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type Hub struct {
	roomsMu sync.Mutex
	rooms   map[string]*Room

	npcResolver NPCResolver
}

func NewHub(npc NPCResolver) *Hub {
	return &Hub{rooms: map[string]*Room{}, npcResolver: npc}
}

// ServeWS is the HTTP handler upgrading /ws?room=<name> connections.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	roomName := r.URL.Query().Get("room")
	if roomName == "" {
		http.Error(w, "missing ?room=", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("accept: %v", err)
		return
	}

	id := uuid.NewString()
	c := &Client{
		hub:    h,
		conn:   conn,
		send:   make(chan Envelope, sendBufSiz),
		ID:     id,
		Name:   "player-" + shortID(id),
		chunks: map[[2]int]bool{},
		ready:  make(chan struct{}),
	}

	h.register(c)
	go c.WritePump()
	go c.ReadPump()

	h.room(roomName).join(c)
}

// register / unregister are the client lifecycle hooks; closing `send`
// makes the WritePump loop terminate.
func (h *Hub) register(c *Client)   {}
func (h *Hub) unregister(c *Client) { close(c.send) }

// room returns the named room, creating it on first use.
func (h *Hub) room(name string) *Room {
	h.roomsMu.Lock()
	defer h.roomsMu.Unlock()
	if r, ok := h.rooms[name]; ok {
		return r
	}
	r := &Room{name: name, members: map[string]*Client{}}
	r.world = NewWorld()
	r.chat = NewChat(r)
	r.npc = h.npcResolver
	r.vote = NewVote(r)
	h.rooms[name] = r
	return r
}

// dispatch routes one inbound envelope from a client. It first waits until
// the join phase has been fully enqueued, keeping per-socket message order
// deterministic even when the client sends messages during its own join.
func (h *Hub) dispatch(c *Client, env Envelope) error {
	<-c.ready
	if c.room == nil {
		return errors.New("not in a room yet")
	}
	switch env.Type {
	case TypeSetName:
		var p MemberInfo
		if err := decode(env, &p); err != nil {
			return err
		}
		c.room.rename(c, p.Name)
		return nil
	case TypeChatSend:
		var p ChatMsg
		if err := decode(env, &p); err != nil {
			return err
		}
		return c.room.chat.handleSend(c, p)
	case TypeChunkReq:
		var p ChunkReq
		if err := decode(env, &p); err != nil {
			return err
		}
		c.room.sendChunks(c, [][2]int{{p.X, p.Y}})
		return nil
	case TypeVoteStart:
		var p VoteStart
		if err := decode(env, &p); err != nil {
			return err
		}
		c.room.vote.OpenVote(nil, time.Duration(p.TimeoutSec)*time.Second)
		return nil
	case TypeVoteCast:
		var p VoteCast
		if err := decode(env, &p); err != nil {
			return err
		}
		return c.room.vote.cast(c, p.ProposalID)
	case TypeNpcTalk:
		var p NpcTalk
		if err := decode(env, &p); err != nil {
			return err
		}
		return c.room.npc.Talk(c.room, c, p)
	default:
		return fmt.Errorf("unknown message type %q", env.Type)
	}
}

func decode(env Envelope, into any) error {
	return json.Unmarshal(env.Data, into)
}

func shortID(id string) string { return id[:6] }
