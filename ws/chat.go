package ws

// Chat is a room-scoped channel with a small history ring for latecomers.
type Chat struct {
	room    *Room
	history []ChatMsg
}

const maxHistory = 50

func NewChat(room *Room) *Chat {
	return &Chat{room: room}
}

func (ch *Chat) handleSend(c *Client, msg ChatMsg) error {
	if msg.Text == "" || len(msg.Text) > 500 {
		c.sendError("chat text must be 1..500 chars")
		return nil
	}
	out := ChatMsg{AuthorID: c.ID, Name: c.Name, Text: msg.Text}

	ch.addToHistory(out)
	ch.room.broadcast(Envelope{Type: TypeChatMsg, Data: mustJSON(out)})
	return nil
}

func (ch *Chat) sendHistory(c *Client) {
	ch.room.mu.RLock()
	n := len(ch.history)
	snapshot := make([]ChatMsg, n)
	copy(snapshot, ch.history)
	ch.room.mu.RUnlock()

	c.sendEnvelope(Envelope{Type: TypeChatHist, Data: mustJSON(ChatHistory{Messages: snapshot})})
}

func (ch *Chat) addToHistory(m ChatMsg) {
	ch.room.mu.Lock()
	defer ch.room.mu.Unlock()
	ch.history = append(ch.history, m)
	if len(ch.history) > maxHistory {
		ch.history = ch.history[len(ch.history)-maxHistory:]
	}
}
