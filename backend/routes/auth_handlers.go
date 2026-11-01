package routes

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/prolarps/urbanist/backend/db"
	"github.com/prolarps/urbanist/backend/errcodes"
	"github.com/prolarps/urbanist/backend/gameplay"
)

// authRequest is the §2.1 body. The room field is gone: membership is now
// an explicit room action (POST /api/rooms/{code}/join), not part of login.
type authRequest struct {
	Name   string `json:"name"`
	Avatar *int   `json:"avatar"`
}

// handleAuth: name -> profile + player JWT (docs/API.md §2.1).
func (s *Server) handleAuth(w http.ResponseWriter, r *http.Request) {
	var body authRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		sendError(w, errcodes.BadRequest, "invalid json")
		return
	}
	name := strings.TrimSpace(body.Name)
	if len(name) < 1 || len(name) > 24 {
		sendError(w, errcodes.BadRequest, "имя должно быть от 1 до 24 символов")
		return
	}
	avatar := 0
	if body.Avatar != nil {
		avatar = clampInt(*body.Avatar, 0, 7)
	}

	player, err := s.store.CreatePlayer(name, avatar, colorByAvatar(avatar))
	if err != nil {
		sendError(w, errcodes.Internal, err.Error())
		return
	}
	token, err := s.auth.IssuePlayer(player.ID, player.Name)
	if err != nil {
		sendError(w, errcodes.Internal, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, authResponseOf(s, player, token))
}

func authResponseOf(s *Server, p *db.Player, token string) map[string]any {
	return map[string]any{
		"token":    token,
		"playerId": p.ID,
		"name":     p.Name,
		"color":    p.Color,
		"world":    gameplay.WorldInfoOf(gameplay.GetGlobalConfig()),
		"protocol": 2,
	}
}
