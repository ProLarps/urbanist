package ws

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// newTestServer boots a Server on a temp sqlite file (no secret conflicts).
func newTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	cfg := DefaultConfig()
	cfg.SecretDir = t.TempDir()

	s := NewServer(cfg)
	mux := http.NewServeMux()

	store, err := OpenStore(":memory:")
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	s.store = store

	auth, err := NewAuthenticator(cfg.SecretDir+"/secret.key", cfg.TokenTTL)
	if err != nil {
		t.Fatalf("auth: %v", err)
	}
	s.auth = auth
	s.rooms.attach(s)

	// default room must exist before /api/auth (mirrors Run()).
	if r, _ := store.GetRoom("default"); r == nil {
		if _, err := store.CreateRoomWithCode("default", "", "Default room", 8); err != nil {
			t.Fatalf("default room: %v", err)
		}
	}
	s.routes(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return s, srv.URL
}

func post(t *testing.T, url, body string, header map[string]string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp.StatusCode, decoded
}

func get(t *testing.T, url, token string) (int, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp.StatusCode, decoded
}

// TestStage1 walks: rooms CRUD -> auth -> profile reuse -> WS authOk/ping.
func TestStage1RoomsAuthWS(t *testing.T) {
	_, baseURL := newTestServer(t)

	// --- rooms CRUD
	status, roomCreate := post(t, baseURL+"/api/rooms", `{"name":"alpha"}`, nil)
	if status != http.StatusUnauthorized {
		t.Fatalf("room create without token: %d", status)
	}

	// auth first
	status, authResp := post(t, baseURL+"/api/auth", `{"name":"Аня","avatar":3,"room":"default"}`, nil)
	if status != http.StatusOK || authResp["token"] == "" {
		t.Fatalf("auth: %d %v", status, authResp)
	}
	token, _ := authResp["token"].(string)
	playerID, _ := authResp["playerId"].(string)

	status, roomCreate = post(t, baseURL+"/api/rooms", `{"name":"alpha"}`, map[string]string{"Authorization": "Bearer " + token})
	if status != http.StatusCreated {
		t.Fatalf("room create: %d %v", status, roomCreate)
	}
	code, _ := roomCreate["code"].(string)
	if len(code) != 6 {
		t.Fatalf("room code: %v", roomCreate)
	}

	// owner can patch, others not
	status, _ = get(t, baseURL+"/api/rooms/"+code, token)
	if status != http.StatusOK {
		t.Fatalf("room get: %d", status)
	}

	status, patchResp := patchJSON(t, baseURL+"/api/rooms/"+code, `{"name":"beta","maxPlayers":4}`, token)
	if status != http.StatusOK || patchResp["ok"] != true {
		t.Fatalf("room patch: %d %v", status, patchResp)
	}

	// second player cannot patch owner's room
	_, auth2 := post(t, baseURL+"/api/auth", `{"name":"Боб","room":"default"}`, nil)
	token2, _ := auth2["token"].(string)
	status, _ = patchJSON(t, baseURL+"/api/rooms/"+code, `{"name":"hijack"}`, token2)
	if status != http.StatusForbidden {
		t.Fatalf("foreign patch allowed: %d", status)
	}

	// auth re-uses the profile by name
	_, auth3 := post(t, baseURL+"/api/auth", `{"name":"Аня","room":"default"}`, nil)
	if auth3["playerId"] != playerID {
		t.Fatalf("profile not reused: %v vs %v", auth3["playerId"], playerID)
	}

	// profile endpoint
	status, profile := get(t, baseURL+"/api/players/"+playerID, "")
	if status != http.StatusOK || profile["bestScore"] != 0.0 {
		t.Fatalf("profile: %d %v", status, profile)
	}

	// --- WS handshake bad token
	wsURL := "ws" + strings.TrimPrefix(baseURL, "http") + "/ws"
	connect := func(url string) (*websocket.Conn, error) {
		ctx, cancel := contextWithTimeout(5 * time.Second)
		defer cancel()
		conn, _, err := websocket.Dial(ctx, url, nil)
		return conn, err
	}
	// --- WS handshake bad token: handshake succeeds (WS-protocol
	// limitation), but the first frame is a close with the spec code.
	badConn, err := connect(wsURL + "?token=garbage")
	if err == nil {
		defer badConn.Close(websocket.StatusNormalClosure, "")
		readCtx, readCancel := contextWithTimeout(5 * time.Second)
		defer readCancel()
		_, _, readErr := badConn.Read(readCtx)
		if readErr == nil {
			t.Fatal("garbage token: no close frame")
		}
		var closeErr websocket.CloseError
		if !strings.Contains(readErr.Error(), "4401") {
			t.Fatalf("garbage token: wrong close: %v", readErr)
		}
		_ = closeErr
	}

	// --- good token: authOk then ping/pong
	conn, err := connect(wsURL + "?token=" + token)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()
	_, authOKRaw, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("recv authOk: %v", err)
	}
	var authOK map[string]any
	if err := json.Unmarshal(authOKRaw, &authOK); err != nil {
		t.Fatalf("authOk parse: %v", err)
	}
	if authOK["t"] != MsgAuthOk {
		t.Fatalf("expected authOk, got %v", authOK)
	}

	// ping -> pong; NOTE: the draft's ping payload also used key "t", which
	// collides with the frame-type key, so it is renamed to "n".
	sendCtx, sendCancel := contextWithTimeout(5 * time.Second)
	defer sendCancel()
	if err := conn.Write(sendCtx, websocket.MessageText, []byte(`{"t":"ping","n":42}`)); err != nil {
		t.Fatalf("send ping: %v", err)
	}
	pongCtx, pongCancel := contextWithTimeout(5 * time.Second)
	defer pongCancel()
	_, pongRaw, err := conn.Read(pongCtx)
	if err != nil {
		t.Fatalf("recv pong: %v", err)
	}
	var pong map[string]any
	_ = json.Unmarshal(pongRaw, &pong)
	if pong["t"] != MsgPong {
		t.Fatalf("expected pong, got %v", pong)
	}
	if n, ok := pong["n"].(float64); !ok || n != 42 {
		t.Fatalf("pong n: %v", pong)
	}
}

func patchJSON(t *testing.T, url, body, token string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp.StatusCode, decoded
}
