package routes_test

import (
	"net/http"
	"testing"
)

// TestAuthAndRooms walks: login -> rooms CRUD -> join/leave/start rules.
func TestAuthAndRooms(t *testing.T) {
	base := newTestServer(t).URL
	_, tokenA, idA := authPlayer(t, base, "Аня")
	_, tokenB, _ := authPlayer(t, base, "Боб")

	status, room := request(t, http.MethodPost, base+"/api/rooms", roomJSON("alpha", 2), tokenA)
	if status != http.StatusCreated || room["code"] == "" || room["mode"] != "wait" {
		t.Fatalf("create: %d %v", status, room)
	}
	code, _ := room["code"].(string)
	have := func(t *testing.T) []any {
		_, view := request(t, http.MethodGet, base+"/api/rooms/"+code, "", tokenA)
		members, _ := view["players"].([]any)
		return members
	}

	// Боб joins; capacity 2 is now full, third player would fail.
	if status, _ := request(t, http.MethodPost, base+"/api/rooms/"+code+"/join", "", tokenB); status != http.StatusOK {
		t.Fatalf("join: %d", status)
	}
	if len(have(t)) != 2 {
		t.Fatalf("members after join: %v", have(t))
	}
	_, tokenC, _ := authPlayer(t, base, "Вера")
	if status, body := request(t, http.MethodPost, base+"/api/rooms/"+code+"/join", "", tokenC); status != http.StatusConflict {
		t.Fatalf("full room join: %d %v", status, body)
	}

	// Start requires everyone's socket: none connected yet -> 409.
	status, body := request(t, http.MethodPost, base+"/api/rooms/"+code+"/start", "", tokenA)
	if status != http.StatusConflict || body["error"] == nil {
		t.Fatalf("start offline: %d %v", status, body)
	}

	// Patch is owner-only; Боб gets 403.
	status, _ = request(t, http.MethodPatch, base+"/api/rooms/"+code, roomJSON("beta", 4), tokenB)
	if status != http.StatusForbidden {
		t.Fatalf("foreign patch: %d", status)
	}

	// Owner may delete even Боб stays; afterwards the room is gone.
	status, _ = request(t, http.MethodDelete, base+"/api/rooms/"+code, "", tokenA)
	if status != http.StatusOK {
		t.Fatalf("delete: %d", status)
	}
	status, _ = request(t, http.MethodGet, base+"/api/rooms/"+code, "", tokenA)
	if status != http.StatusNotFound {
		t.Fatalf("room after delete: %d", status)
	}
	if idA == "" || len(idA) < 4 {
		t.Fatalf("player id: %q", idA)
	}
}

// TestProfilesAndLeaderboard covers §2.3 read endpoints.
func TestProfilesAndLeaderboard(t *testing.T) {
	base := newTestServer(t).URL
	_, token, id := authPlayer(t, base, "Аня")

	status, profile := request(t, http.MethodGet, base+"/api/players/"+id, "", "")
	if status != http.StatusOK || profile["bestScore"] == nil {
		t.Fatalf("profile: %d %v", status, profile)
	}
	status, list := request(t, http.MethodGet, base+"/api/leaderboard", "", "")
	if status != http.StatusOK {
		t.Fatalf("leaderboard empty ok: %d %v", status, list)
	}
	status, _ = request(t, http.MethodGet, base+"/api/players", "", token)
	if status != http.StatusOK {
		t.Fatalf("players: %d", status)
	}
}
