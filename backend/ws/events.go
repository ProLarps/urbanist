package ws

import (
	"github.com/prolarps/urbanist/backend/gameplay"
)

// gameEventFrame translates one §8 loop event into the WS frame(s) the
// docs prescribe (§6): assigned / grant / stats / gameOver go as typed
// frames, everything else degrades into a toast (plus stats when present).
func gameEventFrame(ev gameplay.GameEvent) [][]byte {
	switch ev.Kind {
	case "assigned":
		deadline := int64(0)
		if ev.Deadline != nil {
			deadline = ev.Deadline.Unix()
		}
		return [][]byte{frameAssigned(ev, deadline)}
	case "grant":
		return [][]byte{buildFrame(MsgGrant, map[string]any{
			"options": ev.Options, "cycle": ev.Cycle})}
	case "stats":
		return [][]byte{buildFrame(MsgStats, ev.State)}
	case "gameOver":
		return [][]byte{buildFrame(MsgGameOver, map[string]any{
			"reason": ev.Reason})}
	case "agentSay":
		return [][]byte{buildFrame(MsgAgentSay, map[string]any{
			"agentId": ev.AgentID, "text": ev.Text})}
	case "finished", "timeout", "renovated":
		toast := toastFrame(ToastInfo, eventToastText(ev))
		if ev.State.PlayerID != "" {
			return [][]byte{toast, buildFrame(MsgStats, ev.State)}
		}
		return [][]byte{toast}
	default: // chatPublic and unknown kinds: plain toast keeps clients sane
		return [][]byte{toastFrame(ToastInfo, ev.Text)}
	}
}

// frameAssigned builds §6 "assigned": chunk coords + unix deadline.
func frameAssigned(ev gameplay.GameEvent, deadline int64) []byte {
	cx, cz := -1, -1
	if ev.Chunk != nil {
		cx, cz = ev.Chunk.CX, ev.Chunk.CZ
	}
	return buildFrame(MsgAssigned, map[string]any{
		"cx": cx, "cz": cz, "deadline": deadline})
}

// eventToastText renders the human line for lifecycle events.
func eventToastText(ev gameplay.GameEvent) string {
	switch ev.Kind {
	case "finished":
		return "chunk renovated — new grant incoming"
	case "timeout":
		return "round timed out — rating and money penalised"
	default:
		return "project started: " + ev.Text
	}
}
