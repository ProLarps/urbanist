package routes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/prolarps/urbanist/backend/errcodes"
)

// sendError answers with the §2.6 error envelope: numeric code first,
// human detail second — numbers for clients, detail for debugging.
func sendError(w http.ResponseWriter, code errcodes.Code, detail string) {
	body := map[string]any{"error": int(code)}
	if detail != "" {
		body["detail"] = detail
	}
	sendJSON(w, code.HTTP(), body)
}

// sendJSON answers with any payload and status.
func sendJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// decodeOptional reads a JSON body that may also be empty (EOF): actions
// like POST /rooms/{code}/start are called bare in most clients.
func decodeOptional(r *http.Request, into any) error {
	err := json.NewDecoder(r.Body).Decode(into)
	if errors.Is(err, io.EOF) {
		return nil
	}
	return err
}
