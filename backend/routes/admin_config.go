package routes

import (
	"encoding/json"
	"net/http"

	"github.com/prolarps/urbanist/backend/config"
	"github.com/prolarps/urbanist/backend/errcodes"
	"github.com/prolarps/urbanist/backend/gameplay"
)

// handleAdminSetConfig: PUT /api/admin/config — replaces the global
// gameplay configuration atomically. Running rooms keep the snapshot they
// took on start, so only NEW rooms see these values (docs/API.md §2.7).
func (s *Server) handleAdminSetConfig(w http.ResponseWriter, r *http.Request) {
	var cfg config.Config
	if err := decodeJSON(r, &cfg); err != nil {
		sendError(w, errcodes.BadRequest, "invalid json")
		return
	}
	if err := validateAdminConfig(cfg); err != "" {
		sendError(w, errcodes.BadRequest, err)
		return
	}
	gameplay.SetGlobalConfig(cfg)
	sendJSON(w, http.StatusOK, gameplay.GetGlobalConfig())
}

// validateAdminConfig refuses obviously broken configurations: districts
// must still fit into maps and chances must stay positive integers.
func validateAdminConfig(cfg config.Config) string {
	if cfg.ChunkSize < config.ChunkMin || cfg.ChunkSize > config.ChunkMax {
		return "ChunkSize must be between 8 and 64"
	}
	if cfg.ChunksPerGrant < 1 || cfg.RoundSeconds < 10 {
		return "round settings out of range"
	}
	total := 0
	for rennovation := range config.RENNOVATIONS_TOTAL {
		total += cfg.RennovationAppearChances[rennovation]
	}
	if total <= 0 {
		return "rennovation appear chances must sum to a positive number"
	}
	return ""
}

// decodeJSON is the shared strict body decoder for handlers.
func decodeJSON(r *http.Request, into any) error {
	return json.NewDecoder(r.Body).Decode(into)
}
