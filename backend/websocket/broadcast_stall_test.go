package websocket

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// These tests cover the third concurrency problem in this package, after the
// room-map race (broadcast_race_test.go) and the one-writer-per-conn rule
// (client_write_race_test.go): a write that never returns.
//
// gorilla writes with no deadline unless one is set, and nothing in this
// package set one. A write to a peer whose socket buffers are full - a phone
// that lost the network, a tab the browser froze - blocks until TCP gives up,
// about fifteen minutes on Linux defaults. Broadcast used to do that write
// while holding rm.mu.RLock(), so the next RemoveClient waited on
// rm.mu.Lock(), and an RWMutex refuses new readers while a writer is waiting:
// every later Broadcast in every room, and every join and leave in the
// process, queued behind one dead socket. Not a panic and not a fatal - a
// stall, and there is exactly one backend process.
//
// The fix is in two places and the tests pin the two halves separately.
// Client.WriteJSON arms writeWait before every write, so no write outlives
// it. Broadcast copies the room out under the read lock and fans out after
// releasing it, so no network write happens under rm.mu at all and a panic in
// the fan-out has no read lock to leak.
//
// The stalled peer is real, not simulated. stalledConn upgrades a socket the
// way wsPairs does, but its peer never reads, and the server side is written
// to until the kernel stops taking bytes; the next websocket write on it then
// blocks the way a write to a dead phone does. wsPairs is not reused because
// its drain goroutine is the one thing a stalled peer must not have.
//
// Every assertion that something returns is made with a timeout, because the
// failure under test is a call that never returns, and a test that hangs
// until go test's own ten-minute limit does not say which call hung.

// stalledConn returns a server-side conn whose peer will never read another
// byte and whose socket buffers are already full, so the next write on it
// blocks until a deadline fires or the conn is closed. t.Cleanup closes it,
// which is also what frees a write a failing test left blocked.
func stalledConn(t *testing.T) *websocket.Conn {
	t.Helper()

	var upgrader websocket.Upgrader // no CheckOrigin: the dialer sends no Origin
	served := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		served <- conn
	}))
	t.Cleanup(srv.Close)

	peer, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { peer.Close() })

	var server *websocket.Conn
	select {
	case server = <-served:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the upgrade")
	}
	t.Cleanup(func() { server.Close() })

	// Pin both socket buffers small so the fill stays quick. The send-side
	// pin holds (netstat showed Send-Q at exactly 16384 while a write was
	// blocked, 2026-09-17); the receive side autotuned up to ~526 KiB
	// regardless, which only means the fill writes ~540 KiB rather than 32.
	//
	// What the pin does not buy, on macOS: the stall is not permanent. About
	// five seconds after the window closes, a persist probe finds the peer's
	// grown receive buffer has room and the blocked write completes with no
	// error (measured 4.7s, same date). Every deadline these tests arm is
	// well under that, and each test asserts its deadline itself fired -
	// the timing and the timeout error - so the reopening cannot pass a test
	// for it. On Linux, where the backend runs, a peer that keeps answering
	// zero-window probes holds the write for as long as it likes: there is no
	// TCP timeout for a peer that is alive and not reading, only for one that
	// has stopped answering at all.
	if tcp, ok := server.UnderlyingConn().(*net.TCPConn); ok {
		tcp.SetWriteBuffer(16 << 10)
	}
	if tcp, ok := peer.UnderlyingConn().(*net.TCPConn); ok {
		tcp.SetReadBuffer(16 << 10)
	}

	// Fill the pipe. The peer never reads, so its receive buffer fills and
	// then this side's send buffer does; once a write makes no progress for
	// a while the socket is full. These are raw bytes on the underlying
	// net.Conn rather than websocket frames: the peer will never parse them,
	// and going around gorilla leaves its writer state untouched, so the conn
	// returned here carries no latched error and the next websocket write on
	// it is a real one that really blocks. A timed-out raw write is not fatal
	// to a net.Conn the way a timed-out frame is fatal to a websocket.Conn,
	// which is why the deadline set here can simply be cleared afterwards.
	//
	// Two passes. The bulk goes in 64 KiB chunks, but a large write refuses
	// to start once the free space is below the socket's low-water mark,
	// which leaves up to a couple of kilobytes free - enough for the small
	// frame the test is about to send to slip through and never block. Single
	// bytes take that space too, and only stop when there is none.
	raw := server.UnderlyingConn()
	filled := false
	for _, chunk := range [][]byte{make([]byte, 64<<10), {0}} {
		const attempts = 1 << 16 // 4 GiB of chunks, or 64 KiB of single bytes: far beyond any socket buffer
		filled = false
		for i := 0; i < attempts; i++ {
			raw.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
			if _, err := raw.Write(chunk); err != nil {
				filled = true
				break
			}
		}
		if !filled {
			t.Fatalf("wrote %d chunks of %d bytes to a peer that never reads and the socket never filled", attempts, len(chunk))
		}
	}
	raw.SetWriteDeadline(time.Time{})
	return server
}

// shortWriteWait shortens the package deadline for one test and restores it
// afterwards. Safe because no test in this package calls t.Parallel, and every
// goroutine a test here starts is joined before the test returns.
func shortWriteWait(t *testing.T, d time.Duration) {
	t.Helper()
	was := writeWait
	writeWait = d
	t.Cleanup(func() { writeWait = was })
}

// returnsWithin runs fn on its own goroutine and fails the test if it has not
// returned after limit. fn must not call into t itself.
func returnsWithin(t *testing.T, limit time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(limit):
		t.Fatalf("%s did not return within %v", what, limit)
	}
}

// waitUntilWriting blocks until some goroutine holds client's writeMu - that
// is, is inside Client.WriteJSON - so a test can be sure the write it wants to
// observe has actually started before asserting anything about what that
// write does or does not hold. TryLock is the probe: while it succeeds nobody
// is writing, and the lock is handed straight back.
func waitUntilWriting(t *testing.T, client *Client) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for client.writeMu.TryLock() {
		client.writeMu.Unlock()
		if time.Now().After(deadline) {
			t.Fatal("no goroutine ever started writing to the client")
		}
		time.Sleep(time.Millisecond)
	}
}

// isTimeout reports whether err is a write that hit its deadline. gorilla
// wraps a timed-out write in its own net.Error (conn.go, hideTempErr), which
// keeps Timeout() but does not unwrap to os.ErrDeadlineExceeded, so errors.Is
// would be the wrong test.
func isTimeout(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

func TestWriteJSONGivesUpOnAStalledPeerWhenTheDeadlinePasses(t *testing.T) {
	shortWriteWait(t, 500*time.Millisecond)
	client := &Client{Conn: stalledConn(t), Room: "STALL", PlayerID: "stalled", PlayerName: "Stalled"}

	var err error
	started := time.Now()
	returnsWithin(t, writeWait+3*time.Second, "WriteJSON to a stalled peer", func() {
		err = client.WriteJSON(Message{
			Type:    "TRANSFER",
			Payload: map[string]interface{}{"notification": "stuck"},
		})
	})
	took := time.Since(started)

	if !isTimeout(err) {
		t.Fatalf("WriteJSON returned %v after %v, want a timeout", err, took)
	}
	// The other direction matters as much: a deadline that fires early drops
	// a slow but healthy peer. The tolerance is for timer granularity, not
	// for slack in the value.
	if took < writeWait-writeWait/10 {
		t.Fatalf("WriteJSON gave up after %v, before the %v deadline", took, writeWait)
	}

	// Fatal to the conn, not only to the message (gorilla's SetWriteDeadline
	// doc): the next write fails at once without waiting a second deadline.
	// Broadcast relies on this to treat a timed-out client as dead, and it is
	// what stops a goroutine queued behind the stalled one on writeMu from
	// paying another writeWait for the same peer.
	started = time.Now()
	if err := client.WriteJSON(Message{Type: "PING"}); err == nil {
		t.Fatal("a write after a timeout succeeded; gorilla should have latched the error")
	}
	if since := time.Since(started); since > writeWait/2 {
		t.Fatalf("a write after a timeout took %v; it should fail at once, not wait another deadline", since)
	}
}

func TestBroadcastPrunesAStalledPeerAndDeliversToTheRest(t *testing.T) {
	shortWriteWait(t, 500*time.Millisecond)
	const room = "STALL"

	rm := NewRoomManager()
	stalled := &Client{Conn: stalledConn(t), Room: room, PlayerID: "stalled", PlayerName: "Stalled"}
	live := wsPairs(t, 1)[0]
	liveClient := &Client{Conn: live.server, Room: room, PlayerID: "live", PlayerName: "Live"}
	rm.AddClient(stalled)
	rm.AddClient(liveClient)

	returnsWithin(t, writeWait+3*time.Second, "Broadcast to a room with a stalled peer", func() {
		rm.Broadcast(room, Message{
			Type:    "TRANSFER",
			Payload: map[string]interface{}{"notification": "delivered"},
		})
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
	stalledKept := rm.clients[room][stalled]
	liveKept := rm.clients[room][liveClient]
	rm.mu.RUnlock()
	if stalledKept {
		t.Error("Broadcast left the stalled client in the room")
	}
	if !liveKept {
		t.Error("Broadcast pruned the live client")
	}
	wantConnClosed(t, stalled, "the stalled peer")
}

func TestBroadcastToAStalledPeerDoesNotHoldTheHubLock(t *testing.T) {
	// The process-wide half. While one goroutine is stuck writing to a
	// stalled peer, the hub must still serve every other caller: a Lock
	// taker - any join or leave, in any room - and a Broadcast to another
	// room. Before the fix the first waited on the stuck goroutine's RLock,
	// and the second then waited on the first, because an RWMutex refuses new
	// readers while a writer is waiting. The deadline alone does not pass
	// this test: it bounds the wait, and this asserts there is none.
	shortWriteWait(t, 2*time.Second)
	const promptly = 500 * time.Millisecond

	rm := NewRoomManager()
	stalled := &Client{Conn: stalledConn(t), Room: "STUCK", PlayerID: "stalled", PlayerName: "Stalled"}
	rm.AddClient(stalled)

	pairs := wsPairs(t, 2)
	elsewhere := &Client{Conn: pairs[0].server, Room: "ELSEWHERE", PlayerID: "elsewhere", PlayerName: "Elsewhere"}
	joiner := &Client{Conn: pairs[1].server, Room: "ELSEWHERE", PlayerID: "joiner", PlayerName: "Joiner"}
	rm.AddClient(elsewhere)

	stuck := make(chan struct{})
	go func() {
		defer close(stuck)
		rm.Broadcast("STUCK", Message{
			Type:    "TRANSFER",
			Payload: map[string]interface{}{"notification": "stuck"},
		})
	}()
	waitUntilWriting(t, stalled)

	returnsWithin(t, promptly, "AddClient while a broadcast elsewhere is stalled", func() {
		rm.AddClient(joiner)
	})
	returnsWithin(t, promptly, "Broadcast to another room while a broadcast elsewhere is stalled", func() {
		rm.Broadcast("ELSEWHERE", Message{
			Type:    "TRANSFER",
			Payload: map[string]interface{}{"notification": "unaffected"},
		})
	})
	for i, pair := range pairs {
		select {
		case payload := <-pair.got:
			if !strings.Contains(string(payload), "unaffected") {
				t.Fatalf("client %d in the other room got %q, want the broadcast", i, payload)
			}
		case <-time.After(promptly):
			t.Fatalf("client %d in the other room never received the broadcast", i)
		}
	}
	returnsWithin(t, promptly, "RemoveClient while a broadcast elsewhere is stalled", func() {
		rm.RemoveClient(joiner)
	})

	// Let the deadline end the stalled broadcast before writeWait is
	// restored, so nothing is left running when this test returns. The
	// margin is one second on purpose: the macOS kernel reopens the stalled
	// socket on its own after about five (see stalledConn), and a wait that
	// long would let a build with no deadline at all pass this test.
	select {
	case <-stuck:
	case <-time.After(writeWait + time.Second):
		t.Fatal("the broadcast to the stalled peer did not return when the deadline passed")
	}
}

func TestAPanicInsideTheFanOutDoesNotLeakTheHubLock(t *testing.T) {
	// The second half of board row 11. Broadcast's RUnlock was not deferred,
	// so any panic inside the fan-out left rm.mu read-locked for good: Gin's
	// Recovery keeps the process up, every later Lock waits behind the leaked
	// read lock, every later RLock waits behind that - the same stall a dead
	// socket caused, with no TCP timeout to end it. The only panic that can
	// reach the fan-out today is gorilla's concurrent-write one, which
	// Client.writeMu makes unreachable, so a Client with no Conn stands in:
	// WriteJSON on it dereferences nil, which is a panic from inside the loop
	// like any other.
	const room = "PANIC"
	rm := NewRoomManager()
	rm.AddClient(&Client{Conn: nil, Room: room, PlayerID: "broken", PlayerName: "Broken"})

	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		rm.Broadcast(room, Message{
			Type:    "TRANSFER",
			Payload: map[string]interface{}{"notification": "boom"},
		})
	}()
	if recovered == nil {
		t.Fatal("Broadcast to a client with no conn did not panic, so nothing was injected into the fan-out")
	}

	pair := wsPairs(t, 1)[0]
	after := &Client{Conn: pair.server, Room: "AFTER", PlayerID: "after", PlayerName: "After"}
	returnsWithin(t, 500*time.Millisecond, "AddClient after a panic inside Broadcast", func() {
		rm.AddClient(after)
	})
	returnsWithin(t, 500*time.Millisecond, "Broadcast after a panic inside Broadcast", func() {
		rm.Broadcast("AFTER", Message{
			Type:    "TRANSFER",
			Payload: map[string]interface{}{"notification": "still running"},
		})
	})
	select {
	case payload := <-pair.got:
		if !strings.Contains(string(payload), "still running") {
			t.Fatalf("client got %q, want the broadcast", payload)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("the client never received the broadcast after the panic")
	}
}

func TestWriteWaitIsTheValueBoardRowElevenChose(t *testing.T) {
	// Not a tautology. writeWait fails silently in exactly one direction: a
	// value shortened to make these tests faster, or to drop dead peers
	// sooner, drops healthy players on poor links in production with every
	// other gate green. This turns that edit into a red test that points at
	// the reasoning, so the number is changed there first.
	if writeWait != 10*time.Second {
		t.Fatalf("writeWait is %v, not the 10s board row 11 chose; the reasoning is on writeWait's doc comment in types.go and in the ledger step the row's DONE cell names - change it there first", writeWait)
	}
}
