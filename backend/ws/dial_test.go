package ws_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// contextWithTimeout keeps the test timeouts obvious.
func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// dial opens a socket and fails the test on transport errors.
func dial(t *testing.T, url string) *websocket.Conn {
	t.Helper()
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", url, err)
	}
	return conn
}

// readClose expects the very next frame to be a close carrying code.
// coder/websocket wraps close frames into normal Read errors, so the code
// is matched textually against the printed "StatusCode(NNNN)" form.
func readClose(t *testing.T, conn *websocket.Conn, code websocket.StatusCode) {
	t.Helper()
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()
	_, _, err := conn.Read(ctx)
	if err == nil || !strings.Contains(err.Error(), "StatusCode("+itoa(int(code))+")") {
		t.Fatalf("expected close %d, got %v", code, err)
	}
}

func itoa(n int) string {
	buf, _ := json.Marshal(n)
	return string(buf)
}

// readJSON reads one frame and decodes it into a map.
func readJSON(t *testing.T, conn *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()
	_, raw, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var frame map[string]any
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatalf("frame parse: %v", err)
	}
	return frame
}

// sendJSON writes one text frame.
func sendJSON(t *testing.T, conn *websocket.Conn, body string) {
	t.Helper()
	ctx, cancel := contextWithTimeout(5 * time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, []byte(body)); err != nil {
		t.Fatalf("write: %v", err)
	}
}
