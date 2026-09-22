package websocket

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// writeWait is the longest a single write to a client may block. Client.WriteJSON
// arms it before every write, so it bounds every write path in this package: the
// fan-out in Broadcast, the ERROR replies in handler.go, and anything added later.
//
// A write blocks only when the kernel's send buffer is already full - the peer
// has stopped acknowledging, and a buffer's worth of earlier frames is still
// outstanding. Room messages weigh a few hundred bytes, so that is dozens of
// frames the peer has not taken. Ten more seconds of no progress on top of that
// is a peer that is gone - a phone that lost the network, a tab the browser
// froze - not one that is slow. Without a deadline gorilla waits for TCP to give
// up on its own, about fifteen minutes on Linux defaults, and the goroutine
// doing the write waits with it; before board row 11 that goroutine also held
// the hub's read lock, and every room in the process waited with it.
//
// Too short is the dangerous direction, because it fails silently: a healthy
// player on a poor link is dropped under momentary load, in production, with
// every gate green. The cost of that false positive is bounded - the browser
// reconnects one second after any close and re-sends JOIN
// (frontend/app/room/[code]/page.tsx, socket.onclose), so the room sees a
// left/joined pair of toasts and no seat is lost - but it is a cost, and it is
// why this is ten seconds and not two. Ten is also gorilla's own choice for the
// same job (examples/chat/client.go, writeWait). The full reasoning is in the
// ledger step board row 11's DONE cell names; change the number there first,
// and the pin test in broadcast_stall_test.go will hold you to it.
//
// A var rather than a const so that broadcast_stall_test.go can shorten it.
// Nothing outside a test writes it, and no test in this package runs in
// parallel.
var writeWait = 10 * time.Second

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
// goroutine is ever inside gorilla's writer for this conn, and with a deadline
// of writeWait, so no goroutine stays inside it for longer than that. Every
// write to a client in this package goes through here: reaching for
// c.Conn.WriteJSON directly bypasses both the lock and the deadline.
//
// The deadline is armed under writeMu on purpose. gorilla's SetWriteDeadline
// stores the time in an unguarded field of the conn and applies it when the
// frame is flushed (gorilla/websocket v1.5.3 conn.go: SetWriteDeadline writes
// c.writeDeadline, flushFrame reads it), so arming it from outside the lock
// would race a write in flight on another goroutine.
//
// A write that times out is fatal to the conn, not only to that message:
// gorilla latches the error and every later write returns it at once, without
// waiting a second deadline. Broadcast relies on that to treat a timed-out
// client exactly like a dead one - close it, prune it - and it is what keeps a
// goroutine that queued on writeMu behind the stalled write from paying
// another writeWait for the same peer.
func (c *Client) WriteJSON(message Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	// Unchecked on purpose: gorilla's SetWriteDeadline only records the time
	// and always returns nil.
	c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.Conn.WriteJSON(message)
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
