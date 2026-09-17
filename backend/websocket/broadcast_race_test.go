package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// These tests cover RoomManager's locking, which the money-handler tests in
// websocketManager_test.go do not touch. Broadcast used to delete from
// rm.clients[room] while holding only RLock(), so two broadcasts to one room
// could write the same Go map at once; the runtime answers that with
// fatal("concurrent map writes"), which is not a panic, so Gin's Recovery never
// sees it and the single backend process dies taking every room with it.
//
// Client.Conn is a concrete *websocket.Conn, so there is nothing to mock. The
// conns below are real, upgraded by an httptest.Server the same way handler.go
// upgrades a player's. `go test -race` is the gate these tests exist for.

// wsPair is one upgraded connection: the server side, which is what
// RoomManager writes to, and a channel of what the client side received.
type wsPair struct {
	server *websocket.Conn
	got    chan []byte
}

// wsPairs starts one httptest.Server that upgrades every request, dials it n
// times, and returns the n resulting pairs. A goroutine per client side drains
// reads into a buffered channel so a live conn's WriteJSON never blocks on a
// full socket buffer, and never blocks the drain either.
func wsPairs(t *testing.T, n int) []wsPair {
	t.Helper()

	var upgrader websocket.Upgrader // no CheckOrigin: the dialer sends no Origin
	served := make(chan *websocket.Conn, n)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		served <- conn
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	pairs := make([]wsPair, 0, n)
	for i := 0; i < n; i++ {
		peer, _, err := websocket.DefaultDialer.Dial(url, nil)
		if err != nil {
			t.Fatalf("dial %d: %v", i, err)
		}
		t.Cleanup(func() { peer.Close() })

		got := make(chan []byte, 16)
		go func() {
			for {
				_, payload, err := peer.ReadMessage()
				if err != nil {
					return
				}
				select {
				case got <- payload:
				default:
				}
			}
		}()

		var server *websocket.Conn
		select {
		case server = <-served:
		case <-time.After(5 * time.Second):
			t.Fatalf("timed out waiting for the upgrade of conn %d", i)
		}
		t.Cleanup(func() { server.Close() })

		pairs = append(pairs, wsPair{server: server, got: got})
	}
	return pairs
}

// killConn closes a server-side conn and latches its write error, so every
// later WriteJSON on it fails immediately and Broadcast takes the prune path
// every single iteration.
//
// The latching matters for more than determinism. gorilla/websocket allows one
// concurrent writer per conn and panics "concurrent write to websocket
// connection" otherwise, so a conn that several goroutines broadcast to must
// never reach the real write path - that would be its own data race, drowning
// out the map race under test. After the first failure the conn's cached write
// error short-circuits beginMessage before any shared writer state is touched,
// which is what makes the concurrent calls read-only inside gorilla.
func killConn(t *testing.T, conn *websocket.Conn, i int) {
	t.Helper()
	conn.Close()
	if err := conn.WriteJSON(Message{Type: "PRIME"}); err == nil {
		t.Fatalf("conn %d: WriteJSON on a closed conn succeeded, so Broadcast would never prune it", i)
	}
}

func TestBroadcastIsSafeAgainstConcurrentBroadcastsAndRoomChurn(t *testing.T) {
	const (
		room       = "RACE"
		clientN    = 8
		fanOutN    = 4
		churnN     = 4
		iterations = 500
	)

	pairs := wsPairs(t, clientN)
	rm := NewRoomManager()
	all := make([]*Client, clientN)
	for i, pair := range pairs {
		// Every conn is dead, not half of them, for the gorilla reason in
		// killConn: a live conn cannot take concurrent writes.
		killConn(t, pair.server, i)
		all[i] = &Client{Conn: pair.server, Room: room, PlayerID: "player", PlayerName: "Tester"}
		rm.AddClient(all[i])
	}

	var wg sync.WaitGroup

	for i := 0; i < fanOutN; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < iterations; n++ {
				rm.Broadcast(room, Message{
					Type:    "TRANSFER",
					Payload: map[string]interface{}{"notification": "race"},
				})
			}
		}()
	}

	// The churn goroutines are what keep the prune path hot: without something
	// putting the dead clients back, the first broadcast would delete them all
	// and every later one would find nothing to delete. Between them they cover
	// all eight clients, so they overlap the fan-out completely.
	for i := 0; i < churnN; i++ {
		first, second := all[i], all[i+churnN]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < iterations; n++ {
				rm.RemoveClient(first)
				rm.AddClient(first)
				rm.RemoveClient(second)
				rm.AddClient(second)
			}
		}()
	}

	wg.Wait()

	// Every conn is dead, so one quiet broadcast must leave the room empty.
	// A fix that collected the dead clients but never deleted them would pass
	// the race detector and fail here.
	rm.Broadcast(room, Message{Type: "TRANSFER", Payload: map[string]interface{}{"notification": "settle"}})

	rm.mu.RLock()
	remaining := len(rm.clients[room])
	rm.mu.RUnlock()
	if remaining != 0 {
		t.Fatalf("expected every dead client pruned from %q, %d left", room, remaining)
	}
}

func TestBroadcastPrunesOnlyTheClientsItFailedToWrite(t *testing.T) {
	const room = "PRUNE"

	pairs := wsPairs(t, 2)
	live, dead := pairs[0], pairs[1]
	killConn(t, dead.server, 1)

	rm := NewRoomManager()
	liveClient := &Client{Conn: live.server, Room: room, PlayerID: "live", PlayerName: "Live"}
	deadClient := &Client{Conn: dead.server, Room: room, PlayerID: "dead", PlayerName: "Dead"}
	rm.AddClient(liveClient)
	rm.AddClient(deadClient)

	rm.Broadcast(room, Message{
		Type:    "TRANSFER",
		Payload: map[string]interface{}{"notification": "delivered"},
	})

	select {
	case payload := <-live.got:
		if !strings.Contains(string(payload), "delivered") {
			t.Fatalf("live client got %q, want the broadcast notification", payload)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("live client never received the broadcast")
	}

	rm.mu.RLock()
	liveKept := rm.clients[room][liveClient]
	deadKept := rm.clients[room][deadClient]
	rm.mu.RUnlock()

	if !liveKept {
		t.Error("Broadcast pruned a client whose write succeeded")
	}
	if deadKept {
		t.Error("Broadcast left a client whose write failed in the room")
	}
}
