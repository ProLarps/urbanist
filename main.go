package main

import (
	"log"
	"net/http"

	"github.com/prolarps/urbanist/ws"
)

func main() {
	hub := ws.NewHub(ws.StubResolver{})

	http.Handle("/", http.FileServer(http.Dir("static")))
	http.HandleFunc("/ws", hub.ServeWS)

	log.Println("listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}
