package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// This test drives HandleWebSocket itself, through a real gin router and a
// real upgrade - unlike the RoomManager-level tests next door in
// closeClient_test.go and broadcast_race_test.go, the guard under test here
// lives inline in handler.go's disconnect defer, not on a RoomManager method
// that could be called on its own. wsPairs (broadcast_race_test.go) supplies
// the observer conn the same way it does for every other test in this
// package: real, upgraded by an httptest.Server, with reads drained into a
// channel.
//
// It runs against the package-level Manager - the one global HandleWebSocket
// itself uses. read_deadline_test.go's tests do too (board row 31); a room
// code unique to each test is what keeps them isolated, not exclusive use of
// Manager.

// waitForRoomSize polls room's client count until it equals want, or fails
// the test at timeout. Used instead of a fixed sleep to synchronize with
// handler.go's own goroutines - both AddClient, right after a successful
// upgrade, and RemoveClient, inside the disconnect defer - without a race
// against either.
func waitForRoomSize(t *testing.T, room string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		Manager.mu.RLock()
		n := len(Manager.clients[room])
		Manager.mu.RUnlock()
		if n == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("room %q has %d clients, want %d", room, n, want)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestHandleWebSocketSuppressesPlayerLeftForAConnThatNeverJoined(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const room = "NOJOIN-GUARD"

	router := gin.New()
	router.GET("/ws/room/:code", HandleWebSocket)
	srv := httptest.NewServer(router)
	defer srv.Close()

	// The observer is seated in the room the ordinary way, through
	// Manager.AddClient, with its own real conn from wsPairs. Its `got`
	// channel is what proves whether PLAYER_LEFT reached the room - not a
	// call into RoomManager, which would only prove the broadcast was
	// skipped, not that nothing reached an actual peer.
	observerPairs := wsPairs(t, 1)
	observer := &Client{Conn: observerPairs[0].server, Room: room, PlayerID: "observer", PlayerName: "Observer"}
	Manager.AddClient(observer)
	t.Cleanup(func() { Manager.RemoveClient(observer) })
	waitForRoomSize(t, room, 1, 2*time.Second)

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/room/" + room
	dialHeader := http.Header{"Origin": {"http://localhost:3000"}}
	conn, _, err := websocket.DefaultDialer.Dial(url, dialHeader)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	// Wait for handler.go's own Manager.AddClient to land before closing, so
	// the close below is unambiguously "joined the room, never sent JOIN" -
	// not a close racing the upgrade itself.
	waitForRoomSize(t, room, 2, 2*time.Second)

	// Closed with no JOIN ever sent, so the server-side client's PlayerID is
	// still "" when handler.go's defer runs - the exact shape of a browser
	// that upgrades and leaves, and of a kicked player's reconnect once JOIN
	// is refused for them (the !player.IsActive check in handler.go).
	conn.Close()

	// Proof the disconnect defer ran to completion, RemoveClient included,
	// rather than a fixed sleep guessing how long that takes.
	waitForRoomSize(t, room, 1, 5*time.Second)

	// If the guard in handler.go's defer were missing, this is where
	// PLAYER_LEFT would land: Broadcast's own empty-notification guard
	// (emptyNotification, websocketManager.go) does not catch it, because
	// fmt.Sprintf("%s has left the game", "") is " has left the game" - a
	// non-empty string, just missing its subject. So a message arriving here
	// would prove this guard is gone, not that some other guard caught it.
	select {
	case payload := <-observerPairs[0].got:
		t.Fatalf("observer received a broadcast for a conn that never joined: %s", payload)
	case <-time.After(200 * time.Millisecond):
		// Nothing arrived - the guard suppressed PLAYER_LEFT.
	}
}

// TestHandleWebSocketRepliesErrorForAnUnknownMessageType pins the switch's
// default arm: a message.Type none of the other cases match used to be read
// and silently dropped, so a frontend deployed ahead of the backend - or any
// stale client sending a type this build does not know - got no answer at
// all. The dial here is the sender's own conn, which is exactly where
// handler.go's ERROR replies go (client.WriteJSON, not a room broadcast), so
// reading straight off it is proof enough without wsPairs' second observer.
func TestHandleWebSocketRepliesErrorForAnUnknownMessageType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const room = "UNKNOWN-TYPE"

	router := gin.New()
	router.GET("/ws/room/:code", HandleWebSocket)
	srv := httptest.NewServer(router)
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/room/" + room
	dialHeader := http.Header{"Origin": {"http://localhost:3000"}}
	conn, _, err := websocket.DefaultDialer.Dial(url, dialHeader)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	waitForRoomSize(t, room, 1, 2*time.Second)

	if err := conn.WriteJSON(Message{Type: "NOT_A_THING", Payload: map[string]interface{}{}}); err != nil {
		t.Fatalf("write: %v", err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	var reply Message
	if err := conn.ReadJSON(&reply); err != nil {
		t.Fatalf("read reply: %v", err)
	}

	if reply.Type != "ERROR" {
		t.Fatalf("got message type %q, want ERROR", reply.Type)
	}
	text, ok := reply.Payload.(string)
	const want = "unknown message type: NOT_A_THING"
	if !ok || !strings.Contains(text, want) {
		t.Fatalf("got payload %v, want it to contain %q", reply.Payload, want)
	}
}
