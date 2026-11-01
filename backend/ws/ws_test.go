package ws_test

import (
	"testing"

	"github.com/coder/websocket"

	"github.com/prolarps/urbanist/backend/auth"
)

// TestWSTokenHandshake covers docs/API.md §3: rejected garbage tokens close
// with 4401, valid tokens accept the socket and answer authOk with the
// player's current room membership (nil here).
func TestWSTokenHandshake(t *testing.T) {
	base, issue := bootServer(t)
	wsBase := "ws" + stripScheme(base) + "/ws"

	bad := dial(t, wsBase+"?token=garbage")
	readClose(t, bad, websocket.StatusCode(4401))

	conn := dial(t, wsBase+"?token="+issue("Аня", auth.RolePlayer))
	defer conn.Close(websocket.StatusNormalClosure, "")
	authOK := readJSON(t, conn)
	if authOK["t"] != "authOk" || authOK["room"] != nil {
		t.Fatalf("authOk: %v", authOK)
	}
}

// TestWSPingPongAndRoomBinding covers ping/pong routing without a room and
// re-auth: both must answer even in the lobby (§5 ping, §6 pong).
func TestWSPingPongAndRoomBinding(t *testing.T) {
	base, issue := bootServer(t)
	conn := dial(t, "ws"+stripScheme(base)+"/ws"+"?token="+issue("Боб", auth.RolePlayer))
	defer conn.Close(websocket.StatusNormalClosure, "")
	readJSON(t, conn) // authOk

	sendJSON(t, conn, `{"t":"ping","n":42}`)
	pong := readJSON(t, conn)
	if pong["t"] != "pong" || pong["n"] != float64(42) {
		t.Fatalf("pong: %v", pong)
	}

	sendJSON(t, conn, `{"t":"roomJoin"}`)
	toast := readJSON(t, conn)
	if toast["t"] != "toast" {
		t.Fatalf("roomJoin without membership: %v", toast)
	}
}

// TestUnknownFrameToasts checks the catch-all behaviour of handlers.go.
func TestUnknownFrameToasts(t *testing.T) {
	base, issue := bootServer(t)
	conn := dial(t, "ws"+stripScheme(base)+"/ws"+"?token="+issue("Вера", auth.RolePlayer))
	defer conn.Close(websocket.StatusNormalClosure, "")
	readJSON(t, conn)

	sendJSON(t, conn, `{"t":"move","x":1,"z":2}`)
	if frame := readJSON(t, conn); frame["t"] != "toast" {
		t.Fatalf("expected toast for unimplemented: %v", frame)
	}
}

// TestRoomActState covers roomAct{action:"state"}: without a room binding
// the socket gets an error toast; after joining over HTTP it receives the
// authoritative roomState frame (docs/API.md §5 roomAct).
func TestRoomActState(t *testing.T) {
	base, issue := bootServer(t)
	token := issue("Гриша", auth.RolePlayer)

	status, room := httpJSON(t, "POST", base+"/api/rooms", `{"name":"act","maxPlayers":2}`, token)
	if status != 201 {
		t.Fatalf("create room: %d %v", status, room)
	}
	code, _ := room["code"].(string)
	if status, _ := httpJSON(t, "GET", base+"/api/rooms/"+code+"/start", "", token); status != 409 {
		t.Fatalf("offline start must be rejected: %d", status)
	}

	conn := dial(t, "ws"+stripScheme(base)+"/ws"+"?token="+token)
	defer conn.Close(websocket.StatusNormalClosure, "")
	readJSON(t, conn) // authOk rebinds the membership automatically

	sendJSON(t, conn, `{"t":"roomAct","action":"state"}`)
	frame := readJSON(t, conn)
	if frame["t"] != "roomState" {
		t.Fatalf("roomAct state: %v", frame)
	}
}

// stripScheme turns an http test URL into its host part for ws:// URLs.
func stripScheme(url string) string {
	const prefixLen = len("http")
	return url[prefixLen:]
}
