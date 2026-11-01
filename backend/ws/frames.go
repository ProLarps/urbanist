// Package ws holds every WebSocket concern: handshake with JWT tokens,
// per-room client sets, reading/writing pumps and the frame protocol.
package ws

import (
	"encoding/json"
)

// Client -> server message types (docs/API.md §5).
const (
	MsgAuth     = "auth"
	MsgPing     = "ping"
	MsgRoomJoin = "roomJoin"
	MsgRoomAct  = "roomAct"
)

// Server -> client message types (docs/API.md §6).
const (
	MsgAuthOk  = "authOk"
	MsgToast   = "toast"
	MsgPong    = "pong"
	MsgRoomSt  = "roomState"
	MsgRoomUpd = "roomUpdate"
)

// Close codes are fixed protocol constants (§3).
const (
	CloseTokenMissing = 4400
	CloseTokenInvalid = 4401
	CloseRoomDeleted  = 4404
)

// Toast payloads use the §6 kinds.
const (
	ToastInfo  = "info"
	ToastWarn  = "warn"
	ToastError = "error"
	ToastOK    = "ok"
)

// mustJSON marshals v or falls back to an empty object.
func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// buildFrame merges payload into a {"t": ...} envelope. Client frames are
// single JSON objects on one text frame; order of keys does not matter.
func buildFrame(t string, payload any) []byte {
	body := map[string]any{"t": t}
	for k, val := range payloadAsMap(payload) {
		body[k] = val
	}
	return mustJSON(body)
}

// payloadAsMap round-trips a struct through JSON so every frame stays a
// flat object, whatever the sender's struct shape is.
func payloadAsMap(payload any) map[string]any {
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	out := map[string]any{}
	_ = json.Unmarshal(raw, &out)
	return out
}

// Ping body: the draft's "t" collided with the frame-type key, so it is
// renamed to "n" (docs/API.md §0/§5).
func frameNumber(probe map[string]json.RawMessage) float64 {
	n := 0.0
	if raw, ok := probe["n"]; ok {
		_ = json.Unmarshal(raw, &n)
	}
	return n
}

// probe decodes a client frame just enough to route by "t".
func probe(payload []byte) (t string, body map[string]json.RawMessage, ok bool) {
	body = map[string]json.RawMessage{}
	if err := json.Unmarshal(payload, &body); err != nil {
		return "", nil, false
	}
	raw, exists := body["t"]
	if !exists {
		return "", body, true
	}
	_ = json.Unmarshal(raw, &t)
	return t, body, true
}
