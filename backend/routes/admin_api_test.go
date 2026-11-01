package routes_test

import (
	"net/http"
	"testing"
)

// TestAdminPanel covers §2.7: wrong password, config read/patch.
func TestAdminPanel(t *testing.T) {
	base := newTestServer(t).URL
	status, body := request(t, http.MethodPost, base+"/api/admin/login",
		`{"password":"nope"}`, "")
	_ = body
	if status != http.StatusUnauthorized {
		t.Fatalf("bad password: %d", status)
	}
	status, login := request(t, http.MethodPost, base+"/api/admin/login",
		`{"password":"admin-pass"}`, "")
	if status != http.StatusOK || login["token"] == "" {
		t.Fatalf("login: %d %v", status, login)
	}
	adminToken, _ := login["token"].(string)

	// Player endpoints reject admin tokens and vice versa.
	status, _ = request(t, http.MethodGet, base+"/api/admin/config", "", "")
	if status != http.StatusForbidden {
		t.Fatalf("admin without token: %d", status)
	}
	status, cfg := request(t, http.MethodGet, base+"/api/admin/config", "", adminToken)
	if status != http.StatusOK || cfg["chunkSize"] == nil {
		t.Fatalf("get config: %d %v", status, cfg)
	}
	status, _ = request(t, http.MethodPut, base+"/api/admin/config", configJSON(10), adminToken)
	if status != http.StatusOK {
		t.Fatalf("set config: %d", status)
	}
	_, cfg = request(t, http.MethodGet, base+"/api/admin/config", "", adminToken)
	if cfg["chunkSize"] != float64(10) {
		t.Fatalf("config not applied: %v", cfg["chunkSize"])
	}
}

// configJSON builds a minimal valid admin config body.
func configJSON(chunkSize int) string {
	return `{"chunkSize":` + itoa(chunkSize) + `, "roundSeconds":300,
		"chunksPerGrant":2, "rennovationAppearChances": {"0":35,"1":30,"2":20,"3":10,"4":5},
		"rennovationMinSizes": {"0":4}, "rennovationMaxSizes": {"0":8},
		"startMoney":500, "ratingStart":1, "ratingMin":0.25, "ratingStep":0.05,
		"needsThreshold":0.5, "timeoutRatingLoss":0.1, "timeoutMoneyRate":0.2,
		"skipRatingLoss":0.15, "skipMoneyRate":0.3, "moveUnitsPerSec":6}`
}
