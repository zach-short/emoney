package websocket

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// These tests cover the read side of a dead peer, board row 31, the way
// broadcast_stall_test.go covers the write side, board row 11.
//
// writeWait catches a peer only once a write to it blocks, which takes a full
// kernel send buffer - dozens of room messages the peer never took. A phone
// that drops off the network with its screen off fills no buffer: it stops
// reading and stops sending, and a room's traffic is small. Before row 31 the
// read side had no deadline at all, so ReadJSON in HandleWebSocket waited on
// that socket forever and the ghost kept its seat. The fix is gorilla's
// keepalive - a read deadline of pongWait, renewed by every pong, and a
// pingLoop asking for one every pingPeriod - plus a re-arm before every read,
// so that a connection whose own goroutine was busy handling a message is not
// dropped for a pong it had not yet had the chance to read. See pongWait in
// types.go.
//
// Both halves are pinned here, through HandleWebSocket itself on a real gin
// router and real upgraded conns, because the mechanism is split across the
// upgrade (the deadline and the pong handler), a second goroutine (the ping
// loop) and the read loop's existing exit - no RoomManager method holds it
// whole. A peer that answers nothing is dropped, and PLAYER_LEFT reaches the
// room, pongWait after it went quiet and not before. A peer that answers
// pings survives many pongWaits in which it sends nothing else at all. And a
// peer whose handler goroutine was held up for longer than pongWait survives
// that too.
//
// The ghost is a gorilla client conn that is never read. gorilla answers a
// ping only from inside a read (its default ping handler runs in NextReader),
// so a conn nobody reads never pongs - the server sees exactly what it sees
// from a phone that has gone dark. The ping frames themselves are two bytes
// each and land in the kernel buffers without blocking, which is the other
// half of the premise: the write side never notices this peer.
//
// Like handler_test.go, these run against the package-level Manager that
// HandleWebSocket itself uses, isolated by a room code unique to each test.

// shortPongWait shortens pongWait and pingPeriod for one test and restores
// both afterwards: shortWriteWait's shape, and its safety argument - no test
// here calls t.Parallel, and every goroutine a test here starts is joined
// before it returns. Call it before starting the server, so that every
// handler goroutine the test provokes is started after the write.
//
// The test values do not keep production's 9:10 ratio, on purpose. That
// leaves 60ms for a pong to get back under -race on a loaded machine, which
// is a test of the machine rather than of the mechanism. The ratio itself is
// pinned by TestPongWaitAndPingPeriodAreTheValuesBoardRowThirtyOneChose.
func shortPongWait(t *testing.T, wait, period time.Duration) {
	t.Helper()
	wasWait, wasPeriod := pongWait, pingPeriod
	pongWait, pingPeriod = wait, period
	t.Cleanup(func() { pongWait, pingPeriod = wasWait, wasPeriod })
}

// handlerServer serves HandleWebSocket on a real gin router, exactly as
// handler_test.go does.
func handlerServer(t *testing.T) *httptest.Server {
	t.Helper()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/ws/room/:code", HandleWebSocket)
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)
	return srv
}

// seatObserver puts an ordinary client in room, through Manager.AddClient
// with a real conn from wsPairs, and returns what that conn receives. It is
// the room's point of view: PLAYER_LEFT arriving on it is the proof that a
// dropped peer was announced, not only removed from the map.
func seatObserver(t *testing.T, room string) (*Client, <-chan []byte) {
	t.Helper()
	pair := wsPairs(t, 1)[0]
	observer := &Client{Conn: pair.server, Room: room, PlayerID: "observer", PlayerName: "Observer"}
	Manager.AddClient(observer)
	t.Cleanup(func() { Manager.RemoveClient(observer) })
	waitForRoomSize(t, room, 1, 2*time.Second)
	return observer, pair.got
}

// dialAndSeat opens a real connection through HandleWebSocket and seats it
// as playerID, returning the client end of the conn and the server's *Client
// for it.
//
// Seated through Manager.SeatClient rather than a JOIN frame, because a real
// JOIN cannot succeed in a test binary: the JOIN case calls
// controllers.GetPlayer, which reads config.DB (controllers/playerControllers.go,
// GetPlayer), and config.DB is nil here, so the handler would panic before
// seating anyone. SeatClient is the exact state change a successful JOIN makes
// - handler.go calls it on the line after the IsActive check - and the
// disconnect defer's PLAYER_LEFT guard reads nothing but the PlayerID it
// writes.
//
// SeatClient writes PlayerID under Manager.mu; the handler's defer reads it
// after RemoveClient has taken that same lock, so the two are ordered as long
// as the seat lands before the read deadline fires. The pongWait these tests
// use leaves hundreds of milliseconds for a seat that takes a few.
func dialAndSeat(t *testing.T, srv *httptest.Server, room string, observer *Client, playerID, playerName string) (*websocket.Conn, *Client) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws/room/" + room
	conn, _, err := websocket.DefaultDialer.Dial(url, http.Header{"Origin": {"http://localhost:3000"}})
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	waitForRoomSize(t, room, 2, 2*time.Second)

	var server *Client
	Manager.mu.RLock()
	for c := range Manager.clients[room] {
		if c != observer {
			server = c
		}
	}
	Manager.mu.RUnlock()
	if server == nil {
		t.Fatal("HandleWebSocket's client never appeared in the room")
	}
	Manager.SeatClient(server, playerID, playerName)
	return conn, server
}

// waitForPlayerLeft reads the observer's messages until PLAYER_LEFT for
// playerID arrives, and returns when it did. Anything else arriving first
// fails the test: nothing in these tests broadcasts anything but PLAYER_LEFT.
func waitForPlayerLeft(t *testing.T, got <-chan []byte, playerID, playerName string, within time.Duration) time.Time {
	t.Helper()
	select {
	case raw := <-got:
		arrived := time.Now()
		var msg struct {
			Type    string            `json:"type"`
			Payload map[string]string `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("observer got %s, which is not a message: %v", raw, err)
		}
		if msg.Type != "PLAYER_LEFT" || msg.Payload["playerId"] != playerID {
			t.Fatalf("observer got %s, want PLAYER_LEFT for %s", raw, playerID)
		}
		if want := playerName + " has left the game"; msg.Payload["notification"] != want {
			t.Fatalf("PLAYER_LEFT notification is %q, want %q", msg.Payload["notification"], want)
		}
		return arrived
	case <-time.After(within):
		t.Fatalf("no PLAYER_LEFT for %s reached the room within %v", playerID, within)
		return time.Time{}
	}
}

func TestHandleWebSocketDropsAPeerThatStopsAnsweringPings(t *testing.T) {
	shortPongWait(t, 500*time.Millisecond, 100*time.Millisecond)
	const room = "GHOST"
	srv := handlerServer(t)
	observer, got := seatObserver(t, room)

	// Taken before the dial, and so before the read loop arms its first
	// deadline: the drop cannot legitimately come sooner than pongWait after
	// this.
	started := time.Now()

	// Never read and never written from here on. It is still a live TCP
	// peer - it acknowledges every ping at the transport level - which is
	// what makes it a ghost rather than a closed conn: nothing ends the
	// handler's ReadJSON except the deadline. The bound below is generous
	// against pongWait and tiny against anything else that could end it,
	// TCP's own give-up included.
	dialAndSeat(t, srv, room, observer, "ghost", "Ghost")

	arrived := waitForPlayerLeft(t, got, "ghost", "Ghost", pongWait+3*time.Second)

	// The other direction matters as much, as it does for writeWait: a
	// deadline that fires early drops a healthy player on a slow link. The
	// tolerance is for timer granularity, not slack in the value.
	if took := arrived.Sub(started); took < pongWait-pongWait/10 {
		t.Fatalf("the ghost was dropped %v after connecting, before the %v deadline", took, pongWait)
	}

	// PLAYER_LEFT is sent after RemoveClient in the defer, so this is only
	// confirming the seat is gone, not waiting for it.
	waitForRoomSize(t, room, 1, time.Second)
}

func TestHandleWebSocketKeepsAPeerThatAnswersPings(t *testing.T) {
	shortPongWait(t, 500*time.Millisecond, 100*time.Millisecond)
	const room = "PONGS"
	srv := handlerServer(t)
	observer, got := seatObserver(t, room)

	peer, _ := dialAndSeat(t, srv, room, observer, "healthy", "Healthy")

	// gorilla's default ping handler, plus a count. It is what a browser does
	// with no frontend code: answer every ping with a pong. The peer sends
	// nothing else, so the pongs are the only thing that can be keeping it
	// seated. Set before the read loop starts, which is the goroutine that
	// calls it.
	var pings atomic.Int32
	peer.SetPingHandler(func(data string) error {
		pings.Add(1)
		return peer.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
	})
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			if _, _, err := peer.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Four whole deadlines. With no pong handler the first would drop the
	// peer; with no ping loop there would be no pongs, and the same.
	hold := 4 * pongWait
	select {
	case raw := <-got:
		t.Fatalf("a peer answering every ping was dropped: the room got %s", raw)
	case <-time.After(hold):
	}
	waitForRoomSize(t, room, 2, time.Second)
	if n := pings.Load(); n < int32(hold/pongWait) {
		t.Fatalf("the peer answered %d pings in %v; at least one per %v deadline is what kept it seated", n, hold, pongWait)
	}

	// The ordinary close still takes the ordinary exit, and waiting for its
	// PLAYER_LEFT is also what joins this connection's handler and ping loop
	// before shortPongWait's cleanup restores the values they read: the
	// defer stops and joins the pinger before it broadcasts.
	peer.Close()
	select {
	case <-readerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the peer's read loop did not end after its conn was closed")
	}
	waitForPlayerLeft(t, got, "healthy", "Healthy", 2*time.Second)
	waitForRoomSize(t, room, 1, time.Second)
}

func TestHandleWebSocketKeepsAPeerWhoseHandlerWasBusyPastTheDeadline(t *testing.T) {
	// The independent audit of board row 31, 2026-09-23. The pong handler
	// runs only while the connection's own goroutine is inside ReadJSON.
	// While that goroutine is handling a message instead - a Mongo call, or a
	// Broadcast waiting out writeWait on another player's stalled conn - a
	// deadline renewed only by pongs keeps running, the pong that arrives
	// meanwhile sits unread, and when the goroutine comes back to read, Go
	// fails the read at once because the deadline has already passed. A
	// healthy player, answering every ping, dropped for the server's own
	// slowness. The re-arm at the top of the read loop is the fix; this pins
	// it by holding that goroutine busy for twice pongWait.
	//
	// The lever is the only one reachable without Mongo. A TRANSFER whose
	// payload is not an object is refused before any database call
	// (handleTransfer's first check), and the refusal's ERROR reply goes out
	// through client.WriteJSON on this player's own conn, which waits for
	// writeMu. The test holds writeMu, so the handler goroutine waits there,
	// exactly as it would inside a slow call. Holding writeMu also holds back
	// this conn's pings, the one way this differs from production, where the
	// pings keep going and a pong lands unread. It changes nothing about the
	// outcome on code without the fix: no pong can be read while the goroutine
	// is busy either way, and the deadline has passed by the time it reads.
	//
	// The lock is not taken until two pings have already been answered.
	// writeMu gates writePing too, so a lock taken immediately - before the
	// connection's first ping/pong round trip - starves every ping for as
	// long as it is held, and a connection that has never received a pong has
	// no deadline armed at all to expire: nothing would distinguish the fixed
	// code from the mutation this test exists to catch. Two round trips
	// first means the lock always finds a real, pong-armed deadline already
	// running, the way a live connection actually would.
	shortPongWait(t, 500*time.Millisecond, 100*time.Millisecond)
	const room = "BUSY"
	srv := handlerServer(t)
	observer, got := seatObserver(t, room)

	peer, server := dialAndSeat(t, srv, room, observer, "busy", "Busy")

	// gorilla's default ping handler, plus a count, so the test can wait for
	// two real round trips before it takes writeMu.
	var pings atomic.Int32
	peer.SetPingHandler(func(data string) error {
		pings.Add(1)
		return peer.WriteControl(websocket.PongMessage, []byte(data), time.Now().Add(time.Second))
	})

	// The client end answers pings (the handler above, which runs inside this
	// read loop) and hands every message it receives to replies, where the
	// ERROR reply will land.
	replies := make(chan []byte, 16)
	readerDone := make(chan struct{})
	go func() {
		defer close(readerDone)
		for {
			_, payload, err := peer.ReadMessage()
			if err != nil {
				return
			}
			select {
			case replies <- payload:
			default:
			}
		}
	}()

	waitUntil := time.Now().Add(2 * time.Second)
	for pings.Load() < 2 && time.Now().Before(waitUntil) {
		time.Sleep(5 * time.Millisecond)
	}
	if n := pings.Load(); n < 2 {
		t.Fatalf("only %d ping round trips completed before writeMu is taken; the lock below would test nothing", n)
	}

	// Released once, by the test below or, if it fails first, by cleanup -
	// which runs before the conns close, so no goroutine is left waiting on
	// the lock.
	var unlockOnce sync.Once
	unlock := func() { unlockOnce.Do(server.writeMu.Unlock) }
	server.writeMu.Lock()
	t.Cleanup(unlock)

	if err := peer.WriteJSON(Message{Type: "TRANSFER", Payload: "not an object"}); err != nil {
		t.Fatalf("sending the TRANSFER: %v", err)
	}

	busy := 2 * pongWait
	select {
	case raw := <-got:
		t.Fatalf("the room got %s while the handler was held; nothing can have ended the conn yet", raw)
	case raw := <-replies:
		t.Fatalf("the peer got %s while writeMu was held; the handler goroutine was not the one held", raw)
	case <-time.After(busy):
	}
	unlock()

	// The ERROR could only be written after the unlock, so its arrival is the
	// proof that the handler goroutine was inside that write, busy, for the
	// whole window - rather than idle in ReadJSON, where the deadline would
	// simply have been renewed by pongs.
	select {
	case raw := <-replies:
		var msg struct {
			Type    string `json:"type"`
			Payload string `json:"payload"`
		}
		if err := json.Unmarshal(raw, &msg); err != nil || msg.Type != "ERROR" || msg.Payload != "invalid payload format" {
			t.Fatalf("the peer got %s after the unlock, want the ERROR refusing the TRANSFER", raw)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the ERROR reply never arrived after writeMu was released")
	}

	// Without the re-arm, this is where the drop lands: the goroutine returns
	// to ReadJSON a whole pongWait past its deadline and fails at once.
	select {
	case raw := <-got:
		t.Fatalf("a peer whose own handler was busy for %v was dropped once the handler finished: the room got %s", busy, raw)
	case <-time.After(3 * pongWait):
	}
	waitForRoomSize(t, room, 2, time.Second)

	// Close normally, for the same join-before-restore reason as the test
	// above.
	peer.Close()
	select {
	case <-readerDone:
	case <-time.After(2 * time.Second):
		t.Fatal("the peer's read loop did not end after its conn was closed")
	}
	waitForPlayerLeft(t, got, "busy", "Busy", 2*time.Second)
	waitForRoomSize(t, room, 1, time.Second)
}

func TestPongWaitAndPingPeriodAreTheValuesBoardRowThirtyOneChose(t *testing.T) {
	// TestWriteWaitIsTheValueBoardRowElevenChose's reasoning, on the read
	// side. Too short is the silent direction - healthy players on poor links
	// dropped in production with every gate green - and pingPeriod at or over
	// pongWait is worse: every connection would be dropped between pings. This
	// turns either edit into a red test that points at the reasoning.
	if pongWait != 60*time.Second {
		t.Fatalf("pongWait is %v, not the 60s board row 31 chose; the reasoning is on pongWait's doc comment in types.go - change it there first", pongWait)
	}
	if pingPeriod != 54*time.Second {
		t.Fatalf("pingPeriod is %v, not the 54s (nine tenths of pongWait) board row 31 chose; the reasoning is on pongWait's doc comment in types.go - change it there first", pingPeriod)
	}
}
