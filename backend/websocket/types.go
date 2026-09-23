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

// pongWait is the longest a connection may go without the server hearing from
// it - a pong, or a message - before HandleWebSocket's read loop gives up on
// it, and pingPeriod is how often Client.pingLoop asks for a pong. They are
// the read-side counterpart of writeWait, and they catch the peer writeWait
// cannot.
//
// writeWait only fires when a write blocks, and a write blocks only once the
// kernel's send buffer is full. A phone that drops off the network with its
// screen off stops reading and stops sending, but a room's traffic is a few
// hundred bytes a move, so its buffer can take a whole game's worth of frames
// without filling. And the read side had no deadline at all: conn.ReadJSON
// waited forever on a socket that would never deliver another byte. The ghost
// kept its seat in rm.clients, visible to every other player, until TCP gave
// up on its own - about fifteen minutes on Linux defaults - or enough room
// traffic backed up to trip writeWait as a side effect. Now the server pings
// every pingPeriod, and the read deadline is pushed pongWait into the future
// by every pong and before every read (handler.go - the second is what keeps
// a healthy peer from being dropped while its own handler goroutine is busy
// and cannot read the pong that has already arrived). A peer that answers
// nothing for pongWait fails ReadJSON with a timeout. That takes the read
// loop's ordinary exit: the
// disconnect defer removes the client and broadcasts PLAYER_LEFT exactly as
// it does for a clean close. Browsers answer pings on their own, with no
// frontend code at all.
//
// The numbers are gorilla's own, constant for constant and ratio for ratio
// (examples/chat/client.go in gorilla/websocket v1.5.3, pongWait and
// pingPeriod - in the module cache, not a vendor directory; this repo has
// none), the same precedent writeWait cites. Zach chose them on 2026-09-23 (board row
// 31), picking this recommendation over shorter values. The deadline is
// re-armed when a pong arrives, not when a ping leaves, so a silent peer is
// dropped pongWait after the last thing it sent was read: a phone that goes
// dark is seen as gone at most about sixty seconds later (plus however long
// the server was still handling its last message), and at least the six
// seconds between pingPeriod and pongWait. (Board row 31's prompt,
// 2026-09-23, put the worst case at about 114 seconds, one pingPeriod plus one
// pongWait; that arithmetic arms the deadline at the unanswered ping, which is
// not what this code - or gorilla's example - does.) pingPeriod is under
// pongWait so that a ping is always in flight before the deadline it has to
// renew runs out; the tenth of pongWait between them, six seconds, is the
// time a pong has to get back. It is not also the time the handler goroutine
// has to get back to reading it: the re-arm before every read takes that out
// of the budget.
//
// Too short fails the way writeWait's comment describes: silently, dropping a
// healthy player on a poor link. The cost of that false positive is the same
// one writeWait already accepts, and Zach accepted it again for this row on
// 2026-09-23: the browser reconnects a second after the close and re-sends
// JOIN, the room sees a left/joined pair of toasts, and no seat is lost. There
// is deliberately no grace window and no "reconnecting" state. To change
// either number, change it on board row 31 first;
// TestPongWaitAndPingPeriodAreTheValuesBoardRowThirtyOneChose in
// read_deadline_test.go pins both.
//
// Vars rather than consts for writeWait's reason: read_deadline_test.go
// shortens them. Nothing outside a test writes them.
var (
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
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
// WriteJSON below is the only write path in this package for messages;
// writePing, below it, is the one other write, and holds the same lock.
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
// message to a client in this package goes through here, and the keepalive
// ping goes through writePing, which does the same two things: reaching for
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

// pingLoop pings the client every period until stop is closed or a ping fails
// to write. HandleWebSocket runs one per connection, on its own goroutine, and
// the pong each ping asks for is what renews the read deadline there - see
// pongWait. period is a parameter rather than a read of pingPeriod in here so
// that the read happens on the handler's goroutine, before the go statement,
// where a test that shortens pingPeriod is already ordered against it.
//
// A failed ping ends the loop and does nothing else. It mirrors gorilla's own
// writePump (examples/chat/client.go, the ticker arm), which returns on the
// same error: a ping that cannot be written means the conn is dead or
// closing, and the read side is what notices and cleans up - the read
// deadline fires, or the close already made ReadJSON fail.
//
// stop is closed by HandleWebSocket's disconnect defer, after conn.Close(),
// and the defer then waits for this loop to return. Board row 31, build-level
// call: an explicit stop and a join, rather than letting the loop find the
// closed conn on its next tick. The loop would exit either way, but on its
// next tick, up to one pingPeriod - 54 seconds - after the close, and that
// last tick reads writeWait. The tests in this package shorten writeWait
// (broadcast_stall_test.go), and they are safe only because every goroutine
// a test starts is joined before it returns; a loop left to die on its own
// would read writeWait against the next test's write of it, unordered. The
// join costs the defer nothing measurable: the conn is closed first, so a
// ping in flight, or one queued on writeMu behind a Broadcast, fails at once.
// To reverse the call, drop stop and the join and return only on a failed
// write; nothing else depends on it.
//
// Nothing in here panics - gorilla's one writer panic, the concurrent write,
// is what writeMu rules out - and that matters more here than elsewhere,
// because this goroutine is not the handler's: Gin's Recovery (gin.Default,
// main.go) catches a panic on the handler's goroutine, not on this one, and a
// panic here would take down the one backend process.
func (c *Client) pingLoop(period time.Duration, stop <-chan struct{}) {
	ticker := time.NewTicker(period)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if err := c.writePing(); err != nil {
				return
			}
		}
	}
}

// writePing sends one ping frame, under writeMu and with writeWait armed,
// exactly as WriteJSON sends a message. A ping is a write like any other: it
// goes through gorilla's one writer (WriteMessage's server fast path reaches
// the same beginMessage and flushFrame that WriteJSON does), so without the
// lock it is one more goroutine that can hit "concurrent write to websocket
// connection" - this time from a goroutine nothing else in the package
// expects - and without the deadline a ping to a stalled peer would block
// the loop that exists to detect it.
func (c *Client) writePing() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	// Unchecked for WriteJSON's reason: SetWriteDeadline only records the time.
	c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
	return c.Conn.WriteMessage(websocket.PingMessage, nil)
}

type Message struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}
