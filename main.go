package main

import (
	"log"
	"os"
	"time"

	"github.com/prolarps/urbanist/backend/auth"
	"github.com/prolarps/urbanist/backend/db"
	"github.com/prolarps/urbanist/backend/gameplay"
	"github.com/prolarps/urbanist/backend/routes"
	"github.com/prolarps/urbanist/backend/ws"
)

// main assembles the whole backend: sqlite store, JWT auth, room manager
// and the WebSocket hub, then serves under one address (docs/API.md §2.5).
func main() {
	addr := envOr("ADDR", ":8080")
	dbPath := envOr("DB_PATH", "urbanist.db")
	secretPath := envOr("SECRET_FILE", "var/secret.key")
	adminPass := os.Getenv("ADMIN_PASSWORD")

	store, err := db.Open(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()

	authMan, err := auth.NewManager(secretPath, 7*24*time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	rooms := gameplay.NewManager()
	hub := ws.NewHub(authMan, rooms)
	server := routes.NewServer(store, authMan, rooms, hub, adminPass)

	log.Printf("urbanist listening on %s (db=%s, admin panel=%v)",
		addr, dbPath, adminPass != "")
	if err := server.Run(addr); err != nil {
		log.Fatal(err)
	}
}

// envOr reads an environment variable or falls back to a default.
func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
