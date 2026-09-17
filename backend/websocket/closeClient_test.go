package websocket

import (
	"sync"
	"testing"
)

// These cover CloseClientByPlayerID, the force-close a kick uses to actually
// remove someone, and SeatClient, which exists because that scan reads fields
// the connection's own goroutine writes.
//
// They live beside broadcast_race_test.go and client_write_race_test.go rather
// than in websocketManager_test.go for the reason those two do: Client.Conn is
// a concrete *websocket.Conn with nothing to mock, so the conns here are real,
// upgraded by the httptest.Server in wsPairs the same way handler.go upgrades a
// player's. The money-handler tests next door never touch a socket.
//
// A closed conn refuses every later write - that is what killConn asserts and
// what makes "was this conn closed?" observable at all without reading from the
// peer side, whose reads wsPairs already drains into a channel.

// kickRoom registers n clients in one room under the given player ids and
// returns them in order.
func kickRoom(t *testing.T, rm *RoomManager, room string, ids ...string) []*Client {
	t.Helper()

	pairs := wsPairs(t, len(ids))
	clients := make([]*Client, len(ids))
	for i, id := range ids {
		clients[i] = &Client{Conn: pairs[i].server, Room: room, PlayerID: id, PlayerName: id}
		rm.AddClient(clients[i])
	}
	return clients
}

func wantConnClosed(t *testing.T, client *Client, who string) {
	t.Helper()
	if err := client.WriteJSON(Message{Type: "PING"}); err == nil {
		t.Fatalf("%s's conn still accepts writes, so it was never closed", who)
	}
}

func wantConnOpen(t *testing.T, client *Client, who string) {
	t.Helper()
	if err := client.WriteJSON(Message{Type: "PING"}); err != nil {
		t.Fatalf("%s's conn was closed and should not have been: %v", who, err)
	}
}

func TestCloseClientByPlayerIDClosesOnlyTheTarget(t *testing.T) {
	const room = "KICK"

	rm := NewRoomManager()
	clients := kickRoom(t, rm, room, "alice", "bob", "carol")

	if closed := rm.CloseClientByPlayerID(room, "bob"); closed != 1 {
		t.Fatalf("CloseClientByPlayerID closed %d conns, want 1", closed)
	}

	wantConnClosed(t, clients[1], "bob")
	wantConnOpen(t, clients[0], "alice")
	wantConnOpen(t, clients[2], "carol")
}

func TestCloseClientByPlayerIDLeavesTheRoomMapToTheOwningGoroutine(t *testing.T) {
	// The close deliberately does not remove anything from rm.clients: the
	// target's own reader goroutine is about to wake out of ReadJSON, run
	// handler.go's defer and do the removal itself, along with the PLAYER_LEFT
	// broadcast. Doing it here as well would double-broadcast that message, and
	// would take rm.mu a second time from a goroutine already inside it.
	const room = "KICK"

	rm := NewRoomManager()
	kickRoom(t, rm, room, "alice", "bob", "carol")

	rm.CloseClientByPlayerID(room, "bob")

	rm.mu.RLock()
	remaining := len(rm.clients[room])
	rm.mu.RUnlock()

	if remaining != 3 {
		t.Fatalf("room %q has %d clients after the close, want all 3 left for RemoveClient", room, remaining)
	}
}

func TestCloseClientByPlayerIDIgnoresAPlayerWhoIsNotConnected(t *testing.T) {
	const room = "KICK"

	rm := NewRoomManager()
	clients := kickRoom(t, rm, room, "alice", "bob")

	if closed := rm.CloseClientByPlayerID(room, "nobody"); closed != 0 {
		t.Fatalf("CloseClientByPlayerID closed %d conns for an absent player, want 0", closed)
	}

	wantConnOpen(t, clients[0], "alice")
	wantConnOpen(t, clients[1], "bob")
}

func TestCloseClientByPlayerIDIgnoresAnUnknownRoom(t *testing.T) {
	rm := NewRoomManager()
	clients := kickRoom(t, rm, "KICK", "alice")

	if closed := rm.CloseClientByPlayerID("SOMEWHERE-ELSE", "alice"); closed != 0 {
		t.Fatalf("CloseClientByPlayerID closed %d conns in a room with no clients, want 0", closed)
	}

	wantConnOpen(t, clients[0], "alice")
}

func TestCloseClientByPlayerIDDoesNotCloseTheSamePlayerInAnotherRoom(t *testing.T) {
	// One person can hold a seat in two rooms at once; a kick from one of them
	// must not drop the other.
	rm := NewRoomManager()
	here := kickRoom(t, rm, "HERE", "alice")
	elsewhere := kickRoom(t, rm, "THERE", "alice")

	if closed := rm.CloseClientByPlayerID("HERE", "alice"); closed != 1 {
		t.Fatalf("CloseClientByPlayerID closed %d conns, want 1", closed)
	}

	wantConnClosed(t, here[0], "alice in HERE")
	wantConnOpen(t, elsewhere[0], "alice in THERE")
}

func TestCloseClientByPlayerIDRefusesTheEmptyPlayerID(t *testing.T) {
	// Every connection carries PlayerID "" from the upgrade until its JOIN
	// succeeds. Matching on it would close every unseated conn in the room -
	// including, in a real room, browsers that are one frame away from joining.
	const room = "KICK"

	rm := NewRoomManager()
	clients := kickRoom(t, rm, room, "", "", "alice")

	if closed := rm.CloseClientByPlayerID(room, ""); closed != 0 {
		t.Fatalf("CloseClientByPlayerID closed %d unseated conns, want 0", closed)
	}

	wantConnOpen(t, clients[0], "an unseated conn")
	wantConnOpen(t, clients[1], "a second unseated conn")
	wantConnOpen(t, clients[2], "alice")
}

func TestCloseClientByPlayerIDClosesEverySessionOnePlayerHasOpen(t *testing.T) {
	// A player with the room open on a phone and a laptop holds two conns under
	// one id. A kick that closed the first one it found would leave the other
	// live, and the kicked player would still be watching the game.
	const room = "KICK"

	rm := NewRoomManager()
	clients := kickRoom(t, rm, room, "bob", "bob", "alice")

	if closed := rm.CloseClientByPlayerID(room, "bob"); closed != 2 {
		t.Fatalf("CloseClientByPlayerID closed %d of bob's 2 conns, want 2", closed)
	}

	wantConnClosed(t, clients[0], "bob's first session")
	wantConnClosed(t, clients[1], "bob's second session")
	wantConnOpen(t, clients[2], "alice")
}

func TestCloseClientByPlayerIDIsSafeAgainstSeatingBroadcastsAndRoomChurn(t *testing.T) {
	// `go test -race` is the gate this test exists for, and it covers a race
	// this feature introduced rather than found: until the kick, PlayerID was
	// written once by the connection's own goroutine and never read by any
	// other, so handler.go could assign it with no lock. CloseClientByPlayerID
	// reads it from the *kicking* player's goroutine, which makes that
	// assignment a data race on a string - a pointer and a length the runtime
	// is entitled to let tear. SeatClient is the fix, and this is what proves
	// it: seating, scanning, broadcasting and room churn all at once.
	const (
		room       = "RACE"
		clientN    = 6
		closerN    = 3
		iterations = 300
	)

	rm := NewRoomManager()
	ids := make([]string, clientN)
	for i := range ids {
		ids[i] = "player"
	}
	clients := kickRoom(t, rm, room, ids...)

	// Every conn is killed for the reason killConn's own comment gives:
	// gorilla allows one concurrent writer per conn, so live conns under
	// concurrent broadcasts would be their own race, drowning out the one
	// under test.
	for i, client := range clients {
		killConn(t, client.Conn, i)
	}

	var wg sync.WaitGroup

	for i := range clients {
		client := clients[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < iterations; n++ {
				rm.SeatClient(client, "player", "Player")
			}
		}()
	}

	for i := 0; i < closerN; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < iterations; n++ {
				rm.CloseClientByPlayerID(room, "player")
			}
		}()
	}

	// The churn goroutine keeps clients arriving and leaving the map under the
	// scan, which is what handler.go's disconnect defer does for real the
	// moment a kicked conn closes.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := 0; n < iterations; n++ {
			rm.RemoveClient(clients[0])
			rm.AddClient(clients[0])
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := 0; n < iterations; n++ {
			rm.Broadcast(room, Message{
				Type:    "PLAYER_KICKED",
				Payload: map[string]interface{}{"notification": "race"},
			})
		}
	}()

	wg.Wait()
}
