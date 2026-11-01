package routes

import (
	"net/http"
	"time"

	"github.com/prolarps/urbanist/backend/gameplay"
)

// handleHealth answers §2.2 with live server statistics.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	cfg := gameplay.GetGlobalConfig()
	world := gameplay.WorldInfoOf(cfg)
	sendJSON(w, http.StatusOK, map[string]any{
		"ok":        true,
		"uptime":    time.Since(s.started).Seconds(),
		"players":   s.hub.LivePlayers(),
		"chunks":    world.Chunks,
		"buildings": world.TotalLots,
		"tps":       10,
	})
}

// clampInt limits v into [lo, hi] — request parameters never crash tables.
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// colorByAvatar maps an avatar slot onto one of eight fixed player colors.
func colorByAvatar(n int) string {
	palette := [8]string{
		"#4f9dff", "#ffb64f", "#8bd450", "#ff6f6f",
		"#b28ad6", "#ff8bd1", "#4fd1c9", "#c9b881",
	}
	return palette[n%len(palette)]
}
