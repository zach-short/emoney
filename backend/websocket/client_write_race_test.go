package websocket

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests cover the other half of the concurrency problem in this package.
// broadcast_race_test.go covers rm.clients, the room map, and says in its own
// comments that it deliberately keeps every conn dead so no two goroutines ever
// reach gorilla's real write path - a live conn taking concurrent writes would
// have been a second race drowning out the map race it was written for. This
// file is that second race, on purpose.
//
// A *websocket.Conn permits exactly one writer. gorilla's flushFrame panics
// "concurrent write to websocket connection" when a second goroutine arrives,
// and it panics with the same string again on the way out if a concurrent call
// clobbered isWriting in between; c.writer, c.writeBuf and c.isWriting have no
// lock of their own. The production shape that hits this: every player has
// their own reader goroutine in handler.go, a money action fans out through
// Broadcast to every conn in the room, and an ERROR reply goes straight back
// on the acting player's conn - so two players acting at once puts two
// goroutines in one conn's writer. Client.writeMu is the fix; these tests are
// the proof, and they need LIVE conns to be that proof. killConn is therefore
// exactly what must not be used here: a dead conn short-circuits on its cached
// write error before touching any writer state, which is the whole reason the
// sibling test could get away with it.
//
// `go test -race` is one gate here. The recovered panic is the other, and it
// fires without -race, because gorilla's own check is not a data-race detector.

// writeStorm is one goroutine's worth of concurrent writing: it runs body
// iterations times and reports the first panic it recovers, so a
// "concurrent write to websocket connection" surfaces as a test failure
// instead of taking the test binary down with it.
func writeStorm(wg *sync.WaitGroup, panics chan<- interface{}, iterations int, body func()) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				select {
				case panics <- r:
				default:
				}
			}
		}()
		for n := 0; n < iterations; n++ {
			body()
		}
	}()
}

func TestConcurrentWritesToOneClientNeverRaceTheConnWriter(t *testing.T) {
	const (
		room       = "WRITERS"
		clientN    = 4
		fanOutN    = 6
		iterations = 200
	)

	pairs := wsPairs(t, clientN)
	rm := NewRoomManager()
	clients := make([]*Client, clientN)
	for i, pair := range pairs {
		// Live, not killed: a dead conn proves nothing about the writer state
		// this test exists to protect. wsPairs drains each client side into a
		// buffered channel, so these writes never block on a full socket.
		clients[i] = &Client{Conn: pair.server, Room: room, PlayerID: "player", PlayerName: "Tester"}
		rm.AddClient(clients[i])
	}

	var wg sync.WaitGroup
	panics := make(chan interface{}, fanOutN+clientN)

	// The fan-out goroutines: several players' money actions landing at once,
	// each writing to all four conns.
	for i := 0; i < fanOutN; i++ {
		writeStorm(&wg, panics, iterations, func() {
			rm.Broadcast(room, Message{
				Type:    "TRANSFER",
				Payload: map[string]interface{}{"notification": "concurrent"},
			})
		})
	}

	// The reply goroutines: handler.go's ERROR path, writing back on one
	// player's own conn while every fan-out above is writing to that same conn.
	for i := range clients {
		client := clients[i]
		writeStorm(&wg, panics, iterations, func() {
			client.WriteJSON(Message{Type: "ERROR", Payload: "invalid payload format"})
		})
	}

	wg.Wait()
	close(panics)

	if r, ok := <-panics; ok {
		t.Fatalf("a write panicked: %v", r)
	}

	// Nothing should have been pruned: every conn is live and every write
	// succeeded. A conn whose frames got interleaved by a concurrent writer
	// fails its next write, so a shortfall here is the same bug reported from
	// the other side.
	rm.mu.RLock()
	remaining := len(rm.clients[room])
	rm.mu.RUnlock()
	if remaining != clientN {
		t.Errorf("room %q holds %d clients, want %d - a write failed", room, remaining, clientN)
	}
}

func TestClientWriteJSONDeliversThroughTheLock(t *testing.T) {
	const room = "DELIVERY"

	pairs := wsPairs(t, 1)
	rm := NewRoomManager()
	client := &Client{Conn: pairs[0].server, Room: room, PlayerID: "solo", PlayerName: "Solo"}
	rm.AddClient(client)

	// Both write paths, in the order a player would see them: the fan-out from
	// somebody's action, then an ERROR reply on this player's own conn.
	rm.Broadcast(room, Message{
		Type:    "TRANSFER",
		Payload: map[string]interface{}{"notification": "delivered by broadcast"},
	})
	if err := client.WriteJSON(Message{Type: "ERROR", Payload: "delivered by reply"}); err != nil {
		t.Fatalf("WriteJSON on a live conn: %v", err)
	}

	for _, want := range []string{"delivered by broadcast", "delivered by reply"} {
		select {
		case payload := <-pairs[0].got:
			if !strings.Contains(string(payload), want) {
				t.Fatalf("client got %q, want a message containing %q", payload, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("client never received the message containing %q", want)
		}
	}
}
