package routes

import (
	"net/http"

	"github.com/prolarps/urbanist/backend/errcodes"
	"github.com/prolarps/urbanist/backend/gameplay"
)

// adminLoginBody is POST /api/admin/login (§2.7).
type adminLoginBody struct {
	Password string `json:"password"`
}

// handleAdminLogin exchanges the admin password for an admin JWT. Without
// a configured ADMIN_PASSWORD the panel stays closed.
func (s *Server) handleAdminLogin(w http.ResponseWriter, r *http.Request) {
	var body adminLoginBody
	if err := decodeJSON(r, &body); err != nil {
		sendError(w, errcodes.BadRequest, "invalid json")
		return
	}
	if s.adminPass == "" || body.Password != s.adminPass {
		sendError(w, errcodes.Unauthorized, "bad admin password")
		return
	}
	token, err := s.auth.IssueAdmin()
	if err != nil {
		sendError(w, errcodes.Internal, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, map[string]any{"token": token})
}

// handleAdminGetConfig: GET /api/admin/config — the current global config.
func (s *Server) handleAdminGetConfig(w http.ResponseWriter, r *http.Request) {
	sendJSON(w, http.StatusOK, gameplay.GetGlobalConfig())
}
