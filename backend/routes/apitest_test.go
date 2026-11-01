package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prolarps/urbanist/backend/auth"
	"github.com/prolarps/urbanist/backend/db"
	"github.com/prolarps/urbanist/backend/gameplay"
	"github.com/prolarps/urbanist/backend/routes"
	"github.com/prolarps/urbanist/backend/ws"
)

// newTestServer boots the whole stack on a temp sqlite file and secret dir.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store, err := db.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	authMan, err := auth.NewManager(t.TempDir()+"/secret.key", 7*24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	rooms := gameplay.NewManager()
	hub := ws.NewHub(authMan, rooms)
	server := routes.NewServer(store, authMan, rooms, hub, "admin-pass")
	ts := httptest.NewServer(server.Router())
	t.Cleanup(ts.Close)
	return ts
}

// request is the one HTTP helper for tests: status + decoded JSON body.
func request(t *testing.T, method, url, body, token string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader([]byte(body)))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var decoded map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&decoded)
	return resp.StatusCode, decoded
}

// authPlayer registers a player and returns a Bearer-ready token.
func authPlayer(t *testing.T, base, name string) (url string, token, id string) {
	t.Helper()
	status, body := request(t, http.MethodPost, base+"/api/auth", `{"name":"`+name+`"}`, "")
	if status != http.StatusOK || body["token"] == "" {
		t.Fatalf("auth %s: %d %v", name, status, body)
	}
	token, _ = body["token"].(string)
	id, _ = body["playerId"].(string)
	return strings.TrimRight(base, "/") + baseSuffix(), token, id
}

// baseSuffix exists so helpers stay readable when the base URL grows.
func baseSuffix() string { return "" }

// roomJSON makes compact room bodies for tests.
func roomJSON(name string, max int) string {
	return `{"name":"` + name + `","maxPlayers":` + itoa(max) + `}`
}

func itoa(n int) string {
	return json.Number(intDigits(n)).String()
}

func intDigits(n int) string {
	buf, _ := json.Marshal(n)
	return string(buf)
}
