package ws_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

// httpJSON performs an authorized HTTP call and returns status + decoded
// body; empty body string means "send no payload" (docs/API.md §2.4).
func httpJSON(t *testing.T, method, url, body, token string) (int, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(method, url, strings.NewReader(body))
	if err != nil {
		t.Fatalf("request %s %s: %v", method, url, err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()

	out := map[string]any{}
	_ = json.NewDecoder(resp.Body).Decode(&out) // tolerate empty bodies
	return resp.StatusCode, out
}
