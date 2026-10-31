package ws

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// Sock wraps a websocket client connection for smoke tests.
type Sock struct {
	t    *testing.T
	conn *websocket.Conn
}

func dialSock(t *testing.T, url string) *Sock {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	t.Cleanup(func() {
		conn.Close(websocket.StatusNormalClosure, "")
	})
	return &Sock{t: t, conn: conn}
}

// send sends one envelope, marshaling data if provided.
func (s *Sock) send(msgType string, data any) {
	s.t.Helper()
	payload := []byte("{}")
	if data != nil {
		var err error
		payload, err = json.Marshal(data)
		if err != nil {
			s.t.Fatalf("marshal: %v", err)
		}
	}
	env := Envelope{Type: msgType, Data: payload}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.conn.Write(ctx, websocket.MessageText, mustJSON(env)); err != nil {
		s.t.Fatalf("send %s: %v", msgType, err)
	}
}

// next reads the next envelope, failing the test on timeout.
func (s *Sock) next() Envelope {
	s.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, payload, err := s.conn.Read(ctx)
	if err != nil {
		s.t.Fatalf("recv: %v", err)
	}
	var env Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		s.t.Fatalf("unmarshal %s: %v", payload, err)
	}
	return env
}

// nextOfType skips envelopes that are not wanted (uninteresting broadcasts).
func (s *Sock) nextOfType(want ...string) Envelope {
	s.t.Helper()
	for range 20 {
		env := s.next()
		for _, w := range want {
			if env.Type == w {
				return env
			}
		}
	}
	s.t.Fatalf("did not receive any of %v in time", want)
	return Envelope{}
}

// welcome returns the client's own id: the first message every joiner gets
// is a private joined confirmation.
func (s *Sock) welcome() string {
	s.t.Helper()
	env := s.nextOfType(TypeJoined)
	var joined MemberInfo
	if err := json.Unmarshal(env.Data, &joined); err != nil {
		s.t.Fatalf("joined payload: %v", err)
	}
	if joined.ClientID == "" {
		s.t.Fatal("empty clientId in welcome")
	}
	return joined.ClientID
}

// drainJoinPhase marks the stream with set_name (which the server
// broadcasts back as member_update) and consumes everything up to and
// including the client's own member_update. After this the socket is synced
// past the join phase and all later reads are deterministic.
func (s *Sock) drainJoinPhase(id, name string) {
	s.t.Helper()
	s.send(TypeSetName, MemberInfo{ClientID: id, Name: name})
	for range 60 {
		env := s.next()
		if env.Type != TypeMemberUpd {
			continue
		}
		var mi MemberInfo
		if err := json.Unmarshal(env.Data, &mi); err != nil {
			s.t.Fatalf("member_update payload: %v", err)
		}
		if mi.ClientID == id {
			return
		}
	}
	s.t.Fatalf("no member_update for %s in time", id)
}

// startTestServer serves a hub on /ws and returns its http test server,
// with t.Cleanup closing it.
func startTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(NewHub(StubResolver{}).ServeWS))
	t.Cleanup(srv.Close)
	return srv
}

func wsURL(srv *httptest.Server, room string) string {
	return "ws" + srv.URL[len("http"):] + "/ws?room=" + room
}

// TestSmokeChatChunksVoteNpc walks the whole planned loop with two clients.
func TestSmokeChatChunksVoteNpc(t *testing.T) {
	srv := startTestServer(t)

	// Join: each client learns its own identity, then both mark the stream
	// with set_name so all subsequent reads are past the join phase.
	a := dialSock(t, wsURL(srv, "alpha"))
	b := dialSock(t, wsURL(srv, "alpha"))
	idA, idB := a.welcome(), b.welcome()
	a.drainJoinPhase(idA, "Player A")
	b.drainJoinPhase(idB, "Player B")

	// Chat: a sends, both must receive it.
	a.send(TypeChatSend, ChatMsg{Text: "hello district"})
	envA := a.nextOfType(TypeChatMsg)
	envB := b.nextOfType(TypeChatMsg)
	var msg ChatMsg
	if err := json.Unmarshal(envA.Data, &msg); err != nil {
		t.Fatalf("chat payload: %v", err)
	}
	if msg.Text != "hello district" {
		t.Fatalf("chat text: got %q, want \"hello district\"", msg.Text)
	}
	if envB.Data == nil {
		t.Fatal("b did not get the chat message")
	}

	// Chunk request: any unseen chunk must come back as chunk data.
	a.send(TypeChunkReq, ChunkReq{X: 5, Y: -2})
	env := a.nextOfType(TypeChunk)
	var chunk ChunkData
	if err := json.Unmarshal(env.Data, &chunk); err != nil {
		t.Fatalf("chunk payload: %v", err)
	}
	if chunk.X != 5 || chunk.Y != -2 || len(chunk.Tiles) != chunkSize*chunkSize {
		t.Fatalf("bad chunk: %+v (%d tiles)", chunk, len(chunk.Tiles))
	}

	// NPC stub reply.
	a.send(TypeNpcTalk, NpcTalk{NpcID: "resident", Text: "what do you think?"})
	envNpc := a.nextOfType(TypeNpcReply)
	var npc NpcReply
	if err := json.Unmarshal(envNpc.Data, &npc); err != nil {
		t.Fatalf("npc payload: %v", err)
	}
	if npc.NpcID != "resident" || npc.Text == "" {
		t.Fatalf("bad npc reply: %+v", npc)
	}

	// Vote: start it, cast from both, expect a tally then a result.
	b.send(TypeVoteStart, VoteStart{TimeoutSec: 1})
	a.nextOfType(TypeVoteOpen)
	b.nextOfType(TypeVoteOpen)

	b.send(TypeVoteCast, VoteCast{ProposalID: "park"})
	a.nextOfType(TypeVoteTally)
	b.nextOfType(TypeVoteTally)

	a.send(TypeVoteCast, VoteCast{ProposalID: "park"})
	nextEnv := a.nextOfType(TypeVoteResult)
	var res VoteResult
	if err := json.Unmarshal(nextEnv.Data, &res); err != nil {
		t.Fatalf("vote result payload: %v", err)
	}
	if res.Winner != "park" {
		t.Fatalf("vote winner: got %q, want \"park\" (park beats the tie-break by profitability)", res.Winner)
	}
	if res.Tally["park"] != 2 {
		t.Fatalf("vote tally: %v", res.Tally)
	}

	// Both clients must receive the district repaint next.
	a.nextOfType(TypeChunkUpd)
	b.nextOfType(TypeChunkUpd)
}
