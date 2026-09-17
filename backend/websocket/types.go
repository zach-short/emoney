package websocket

import (
	"sync"

	"github.com/gorilla/websocket"
)

// Client is one player's connection to one room.
//
// A *websocket.Conn permits exactly one concurrent writer: gorilla's
// flushFrame panics "concurrent write to websocket connection" the moment a
// second goroutine reaches it, and c.writer / c.writeBuf / c.isWriting are
// ordinary unguarded struct fields, so there is no internal lock to fall back
// on. This package writes to a single client's conn from more than one
// goroutine as a matter of course. Every connected player has their own reader
// goroutine in handler.go, and a money action on any of them fans out through
// RoomManager.Broadcast to every conn in the room - including the conns owned
// by the other players' goroutines. Add the ERROR replies handler.go writes
// back on a player's own conn and two players acting at the same instant put
// two goroutines inside one conn's writer.
//
// Gin's Recovery middleware catches that panic, so the cost is one dropped
// connection rather than the process, but the dropped connection is a player
// silently falling out of a live game. writeMu is the serialization, and
// WriteJSON below is the only write path in this package.
//
// A Client carries a mutex and must never be copied; hold it by pointer, which
// is what RoomManager's map keys already are.
type Client struct {
	Conn       *websocket.Conn
	PlayerID   string
	Room       string
	PlayerName string

	writeMu sync.Mutex
}

// WriteJSON sends message on the client's conn with writeMu held, so only one
// goroutine is ever inside gorilla's writer for this conn. Every write to a
// client in this package goes through here: reaching for c.Conn.WriteJSON
// directly bypasses the lock and puts back the panic it exists to prevent.
func (c *Client) WriteJSON(message Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.Conn.WriteJSON(message)
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
