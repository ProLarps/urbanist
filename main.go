package main

import (
	"log"
	"os"

	"github.com/prolarps/urbanist/ws"
)

func main() {
	cfg := ws.DefaultConfig()

	addr := envOr("ADDR", ":8080")
	dbPath := envOr("DB_PATH", "urbanist.db")

	log.Printf("starting urbanist on %s (db=%s)", addr, dbPath)
	if err := ws.NewServer(cfg).Run(addr, dbPath); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
