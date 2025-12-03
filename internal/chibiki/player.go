package chibiki

import "github.com/gorilla/websocket"

type Player struct {
	ID     string
	UserID string
	Team   int
	Conn   *websocket.Conn
	Send   chan []byte
}
