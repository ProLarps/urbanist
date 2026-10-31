package ws

import (
	"fmt"
	"strings"
	"time"
)

// NPCResolver answers player dialogue with an NPC. The production
// implementation will delegate to a local LLM; for now the stub keeps the
// message loop working end to end.
type NPCResolver interface {
	Talk(room *Room, from *Client, msg NpcTalk) error
}

// StubResolver echoes canned replies, one per known NPC, with a small delay
// to simulate thinking. Replace with an LLM-backed resolver later.
type StubResolver struct{}

func (StubResolver) Talk(room *Room, from *Client, msg NpcTalk) error {
	if strings.TrimSpace(msg.Text) == "" {
		return fmt.Errorf("empty npc talk")
	}

	reply := StubResolver{}.replyFor(msg.NpcID, msg.Text)
	// Answer asynchronously so the room loop is not blocked by "thinking".
	go func() {
		time.Sleep(400 * time.Millisecond)
		// Rooms can outlive the client: re-check membership before sending.
		room.mu.RLock()
		_, stillThere := room.members[from.ID]
		room.mu.RUnlock()
		if stillThere {
			from.sendEnvelope(Envelope{Type: TypeNpcReply, Data: mustJSON(NpcReply{NpcID: msg.NpcID, Text: reply})})
		}
	}()
	return nil
}

func (StubResolver) replyFor(npcID, text string) string {
	switch npcID {
	case "architect":
		return "The residential complex would cover block 0,0; see the tall plans on the board."
	case "resident":
		return "We vote for the park renovation — it was beautiful once, it can be again."
	default:
		return fmt.Sprintf("I have nothing on that yet (%s).", text)
	}
}
