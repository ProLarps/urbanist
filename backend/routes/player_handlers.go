package routes

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/prolarps/urbanist/backend/db"
	"github.com/prolarps/urbanist/backend/errcodes"
)

// handlePlayers returns the profile list (§2.3, service endpoint).
func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := s.store.ListPlayers()
	if err != nil {
		sendError(w, errcodes.Internal, err.Error())
		return
	}
	sendJSON(w, http.StatusOK, players)
}

// handlePlayerProfile returns one profile with best score (§2.3).
func (s *Server) handlePlayerProfile(w http.ResponseWriter, r *http.Request) {
	player, err := s.store.GetPlayerByID(chi.URLParam(r, "id"))
	if err != nil {
		sendError(w, errcodes.Internal, err.Error())
		return
	}
	if player == nil {
		sendError(w, errcodes.NotFound, "player")
		return
	}
	sendJSON(w, http.StatusOK, player)
}

// handleLeaderboard returns the global best scores (§2.3).
func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := s.store.Leaderboard()
	if err != nil {
		sendError(w, errcodes.Internal, err.Error())
		return
	}
	if rows == nil {
		rows = []db.LeaderboardRow{}
	}
	sendJSON(w, http.StatusOK, rows)
}
