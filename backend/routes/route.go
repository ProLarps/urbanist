package routes

import (
	"github.com/go-chi/chi/v5"
)

// Router builds the full chi tree: §2.5 static files, §2.1–2.4 API and
// the §2.7 administrator panel. WS shares the token namespace of §3.
func (s *Server) Router() chi.Router {
	r := chi.NewRouter()
	useMiddleware(r)

	r.Get("/ws", s.hub.ServeWS)
	r.Route("/api", func(r chi.Router) {
		r.Post("/auth", s.handleAuth)      // §2.1
		r.Get("/health", s.handleHealth)   // §2.2
		r.Get("/players", s.handlePlayers) // §2.3
		s.routeProfiles(r)                 // §2.3 profiles + leaderboard
		s.routeRooms(r)                    // §2.4 rooms lifecycle
		s.routeAdmin(r)                    // §2.7 administrator panel
	})
	s.serveStatic(r)
	return r
}

// routeProfiles mounts the §2.3 read endpoints.
func (s *Server) routeProfiles(r chi.Router) {
	r.Get("/players/{id}", s.handlePlayerProfile)
	r.Get("/leaderboard", s.handleLeaderboard)
}

// routeRooms mounts the whole §2.4 subtree behind the player JWT guard.
func (s *Server) routeRooms(r chi.Router) {
	r.Route("/rooms", func(r chi.Router) {
		r.Use(requireAuth(s.auth))
		r.Post("/", s.handleRoomCreate)
		r.Get("/", s.handleRoomList)
		r.Route("/{code}", func(r chi.Router) {
			r.Get("/", s.handleRoomGet)
			r.Patch("/", s.handleRoomPatch)
			r.Delete("/", s.handleRoomDelete)
			r.Post("/join", s.handleRoomJoin)
			r.Post("/leave", s.handleRoomLeave)
			r.Post("/start", s.handleRoomStart)
			r.Get("/scoreboard", s.handleRoomScoreboard)
		})
	})
}

// routeAdmin mounts §2.7: login is public, everything else needs a token
// with role=admin, issued only after the ADMIN_PASSWORD check.
func (s *Server) routeAdmin(r chi.Router) {
	r.Route("/admin", func(r chi.Router) {
		r.Post("/login", s.handleAdminLogin)
		r.Group(func(r chi.Router) {
			r.Use(requireAdmin(s.auth))
			r.Get("/config", s.handleAdminGetConfig)
			r.Put("/config", s.handleAdminSetConfig)
			r.Get("/rooms", s.handleAdminRooms)
			r.Delete("/rooms/{code}", s.handleAdminCloseRoom)
		})
	})
}
