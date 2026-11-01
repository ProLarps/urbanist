package routes

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMW "github.com/go-chi/chi/v5/middleware"

	"github.com/prolarps/urbanist/backend/auth"
	"github.com/prolarps/urbanist/backend/db"
	"github.com/prolarps/urbanist/backend/gameplay"
	"github.com/prolarps/urbanist/backend/ws"
)

// Server wires the dependencies and owns the chi router + all HTTP handlers.
type Server struct {
	store     *db.Store
	auth      *auth.Manager
	rooms     *gameplay.Manager
	hub       *ws.Hub
	adminPass string
	started   time.Time
}

// NewServer takes already-constructed dependencies (main assembles them).
func NewServer(store *db.Store, authMan *auth.Manager,
	rooms *gameplay.Manager, hub *ws.Hub, adminPass string) *Server {
	return &Server{
		store: store, auth: authMan,
		rooms: rooms, hub: hub,
		adminPass: adminPass, started: time.Now(),
	}
}

// Run starts serving with the built router and blocks until shutdown.
func (s *Server) Run(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Router(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

// serveStatic mounts ./static for everything that is not API/WS (§2.5).
func (s *Server) serveStatic(r chi.Router) {
	r.Handle("/*", http.FileServer(http.Dir("static")))
}

// useMiddleware is the shared middleware stack for every request.
func useMiddleware(r chi.Router) {
	// Recoverer keeps panics from handlers fatal for the whole server.
	r.Use(chiMW.Recoverer)
}
