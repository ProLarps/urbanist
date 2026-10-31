package ws

import "github.com/coder/websocket"

type Room struct {
}

type Client struct {
	conn *websocket.Conn
	send chan []byte
}
