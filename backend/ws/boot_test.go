package ws_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prolarps/urbanist/backend/auth"
	"github.com/prolarps/urbanist/backend/db"
	"github.com/prolarps/urbanist/backend/gameplay"
	"github.com/prolarps/urbanist/backend/routes"
	"github.com/prolarps/urbanist/backend/ws"
)

// bootServer builds the full stack on temp storage; returns base URL +
// issuing helpers already wired to the SAME auth manager as the routes.
func bootServer(t *testing.T) (baseURL string, issueToken func(name, role string) string) {
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
	ts := httptest.NewServer(routes.NewServer(store, authMan, rooms, hub, "").Router())
	t.Cleanup(ts.Close)

	issue := func(name, role string) string {
		if role == "admin" {
			if tok, err := authMan.IssueAdmin(); err == nil {
				return tok
			}
			t.Fatal("issue admin token")
			return ""
		}
		player, err := store.CreatePlayer(name, 0, "#4f9dff")
		if err != nil {
			t.Fatal(err)
		}
		tok, _ := authMan.IssuePlayer(player.ID, player.Name)
		return tok
	}
	return ts.URL, issue
}
