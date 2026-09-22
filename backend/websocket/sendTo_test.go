package websocket

import (
	"bytes"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

// These cover RoomManager.SendTo, the targeted counterpart of Broadcast that
// an offer needs: a trade proposal goes to one player, not the room.
//
// They live beside broadcast_race_test.go and closeClient_test.go rather than
// in websocketManager_test.go for the reason those do: Client.Conn is a
// concrete *websocket.Conn with nothing to mock, so the conns here are real,
// upgraded by the httptest.Server in wsPairs. A peer's `got` channel is what
// proves a frame reached - or did not reach - an actual socket, which is the
// whole claim SendTo makes and the one a call into RoomManager could not
// prove on its own.
//
// `go test -race -count=3` is the gate for the last test here, for the same
// reason it is for Broadcast's: SendTo deletes dead clients from the room map,
// and a delete under RLock is fatal("concurrent map writes") the first time
// two of them overlap.

// seatRoom registers one client per id in room and returns them alongside the
// wsPairs they were upgraded from, so a test can both act on the *Client and
// read what its peer received.
func seatRoom(t *testing.T, rm *RoomManager, room string, ids ...string) ([]*Client, []wsPair) {
	t.Helper()

	pairs := wsPairs(t, len(ids))
	clients := make([]*Client, len(ids))
	for i, id := range ids {
		clients[i] = &Client{Conn: pairs[i].server, Room: room, PlayerID: id, PlayerName: id}
		rm.AddClient(clients[i])
	}
	return clients, pairs
}

func offerFrame(text string) Message {
	return Message{
		Type:    "OFFER_RECEIVED",
		Payload: map[string]interface{}{"notification": text},
	}
}

func wantReceived(t *testing.T, pair wsPair, who, text string) {
	t.Helper()
	select {
	case payload := <-pair.got:
		if !strings.Contains(string(payload), text) {
			t.Fatalf("%s got %q, want a message containing %q", who, payload, text)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s never received the message containing %q", who, text)
	}
}

func wantNothingReceived(t *testing.T, pair wsPair, who string) {
	t.Helper()
	select {
	case payload := <-pair.got:
		t.Fatalf("%s received %q and should have received nothing", who, payload)
	case <-time.After(200 * time.Millisecond):
		// Nothing arrived.
	}
}

func TestSendToDeliversOnlyToTheTarget(t *testing.T) {
	const room = "SENDTO-TARGET"
	rm := NewRoomManager()
	_, pairs := seatRoom(t, rm, room, "alice", "bob", "carol")

	reached := rm.SendTo(room, "bob", offerFrame("for bob only"))

	if reached != 1 {
		t.Fatalf("reached %d connections, want 1", reached)
	}
	wantReceived(t, pairs[1], "bob", "for bob only")
	// The claim the feature rests on: C sees nothing. And A, the sender in
	// the real flow, gets a different frame from a separate call - not this
	// one.
	wantNothingReceived(t, pairs[0], "alice")
	wantNothingReceived(t, pairs[2], "carol")
}

func TestSendToReachesEveryTabOnePlayerHasOpen(t *testing.T) {
	const room = "SENDTO-TABS"
	rm := NewRoomManager()
	// The same player seated twice: a phone and a laptop, or two tabs.
	_, pairs := seatRoom(t, rm, room, "bob", "bob", "alice")

	reached := rm.SendTo(room, "bob", offerFrame("both tabs"))

	if reached != 2 {
		t.Fatalf("reached %d connections, want 2 - every tab is the same person", reached)
	}
	wantReceived(t, pairs[0], "bob's first tab", "both tabs")
	wantReceived(t, pairs[1], "bob's second tab", "both tabs")
	wantNothingReceived(t, pairs[2], "alice")
}

func TestSendToReportsZeroForAPlayerWhoIsNotConnected(t *testing.T) {
	const room = "SENDTO-OFFLINE"
	rm := NewRoomManager()
	_, pairs := seatRoom(t, rm, room, "alice")

	// Not an error: the offer is in Mongo and the inbox fetches it on the
	// recipient's next load. The caller logs the count and moves on.
	if reached := rm.SendTo(room, "bob", offerFrame("nobody home")); reached != 0 {
		t.Fatalf("reached %d connections for a player who is not connected, want 0", reached)
	}
	wantNothingReceived(t, pairs[0], "alice")
}

func TestSendToReportsZeroForAnUnknownRoom(t *testing.T) {
	rm := NewRoomManager()
	_, _ = seatRoom(t, rm, "SENDTO-KNOWN", "bob")

	if reached := rm.SendTo("SENDTO-NO-SUCH-ROOM", "bob", offerFrame("wrong room")); reached != 0 {
		t.Fatalf("reached %d connections in a room that does not exist, want 0", reached)
	}
}

func TestSendToRefusesTheEmptyPlayerID(t *testing.T) {
	const room = "SENDTO-UNSEATED"
	rm := NewRoomManager()
	// Two conns that have upgraded but never sent JOIN: PlayerID "" is what
	// every connection carries until then. Matching on it would put a
	// private offer on every unseated socket in the room.
	_, pairs := seatRoom(t, rm, room, "", "")

	if reached := rm.SendTo(room, "", offerFrame("to nobody")); reached != 0 {
		t.Fatalf("reached %d unseated connections, want 0", reached)
	}
	wantNothingReceived(t, pairs[0], "the first unseated conn")
	wantNothingReceived(t, pairs[1], "the second unseated conn")
}

func TestSendToDoesNotReachTheSamePlayerInAnotherRoom(t *testing.T) {
	rm := NewRoomManager()
	_, here := seatRoom(t, rm, "SENDTO-HERE", "bob")
	_, there := seatRoom(t, rm, "SENDTO-THERE", "bob")

	reached := rm.SendTo("SENDTO-HERE", "bob", offerFrame("this room"))

	if reached != 1 {
		t.Fatalf("reached %d connections, want 1", reached)
	}
	wantReceived(t, here[0], "bob here", "this room")
	wantNothingReceived(t, there[0], "bob in the other room")
}

func TestSendToPrunesOnlyTheTabItFailedToWrite(t *testing.T) {
	const room = "SENDTO-PRUNE"
	rm := NewRoomManager()
	clients, pairs := seatRoom(t, rm, room, "bob", "bob", "alice")
	live, dead, bystander := clients[0], clients[1], clients[2]
	killConn(t, dead.Conn, 1)

	reached := rm.SendTo(room, "bob", offerFrame("one tab left"))

	if reached != 1 {
		t.Fatalf("reached %d connections, want 1 - the live tab", reached)
	}
	wantReceived(t, pairs[0], "bob's live tab", "one tab left")

	rm.mu.RLock()
	liveKept := rm.clients[room][live]
	deadKept := rm.clients[room][dead]
	bystanderKept := rm.clients[room][bystander]
	rm.mu.RUnlock()

	if !liveKept {
		t.Error("SendTo pruned the tab whose write succeeded")
	}
	if deadKept {
		t.Error("SendTo left the tab whose write failed in the room")
	}
	if !bystanderKept {
		t.Error("SendTo pruned a client it never wrote to")
	}
}

func TestSendToRefusesAnEmptyNotification(t *testing.T) {
	const room = "SENDTO-BLANK"
	rm := NewRoomManager()
	_, pairs := seatRoom(t, rm, room, "bob")

	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(nil) })

	reached := rm.SendTo(room, "bob", offerFrame(""))

	if reached != 0 {
		t.Fatalf("reached %d connections with an empty notification, want 0", reached)
	}
	wantNothingReceived(t, pairs[0], "bob")
	if !strings.Contains(buf.String(), "SendTo refused") {
		t.Fatalf("expected the refusal in the log, got %q", buf.String())
	}
}

func TestSendToIsSafeAgainstBroadcastsAndRoomChurn(t *testing.T) {
	const (
		room       = "SENDTO-RACE"
		clientN    = 8
		senderN    = 4
		churnN     = 4
		iterations = 500
	)

	pairs := wsPairs(t, clientN)
	rm := NewRoomManager()
	all := make([]*Client, clientN)
	for i, pair := range pairs {
		// Every conn is dead, for the reason broadcast_race_test.go gives: a
		// live conn cannot take concurrent writes, and this test is about the
		// room map, not the conn writer. Half the clients are "bob", so that
		// SendTo's prune path and Broadcast's prune path both stay hot on the
		// same map at the same time.
		killConn(t, pair.server, i)
		id := "alice"
		if i%2 == 0 {
			id = "bob"
		}
		all[i] = &Client{Conn: pair.server, Room: room, PlayerID: id, PlayerName: id}
		rm.AddClient(all[i])
	}

	var wg sync.WaitGroup

	for i := 0; i < senderN; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < iterations; n++ {
				rm.SendTo(room, "bob", offerFrame("race"))
				rm.Broadcast(room, Message{
					Type:    "TRANSFER",
					Payload: map[string]interface{}{"notification": "race"},
				})
			}
		}()
	}

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

	// Every conn is dead, so one quiet pass of each must leave the room
	// empty: SendTo prunes bob's, Broadcast prunes the rest. A SendTo that
	// collected its dead clients but never deleted them would pass the race
	// detector and fail here.
	rm.SendTo(room, "bob", offerFrame("settle"))
	rm.Broadcast(room, Message{Type: "TRANSFER", Payload: map[string]interface{}{"notification": "settle"}})

	rm.mu.RLock()
	remaining := len(rm.clients[room])
	rm.mu.RUnlock()
	if remaining != 0 {
		t.Fatalf("expected every dead client pruned from %q, %d left", room, remaining)
	}
}

func TestSendToAStalledTargetDoesNotHoldTheHubLock(t *testing.T) {
	// Board row 30. SendTo was written before row 11 and carried the shape row
	// 11 removed from Broadcast: every target's write ran under rm.mu.RLock().
	// Row 11's deadline reached SendTo for free - Client.WriteJSON is the one
	// write path - so the stall became bounded, but bounded is not absent.
	// While a private OFFER_RECEIVED to one frozen tab waits out writeWait,
	// the RLock it holds makes every Lock caller wait, and an RWMutex refuses
	// new readers while a writer is waiting, so every join, every leave and
	// every broadcast in every other room queues behind one player's phone.
	//
	// This is the assertion the deadline alone cannot pass: not that the wait
	// is short, that there is no wait. It is Broadcast's hub-lock test aimed
	// at SendTo, and it fails against the pre-row-30 body.
	shortWriteWait(t, 2*time.Second)
	const promptly = 500 * time.Millisecond

	rm := NewRoomManager()
	stalled := &Client{Conn: stalledConn(t), Room: "STUCK", PlayerID: "target", PlayerName: "Target"}
	rm.AddClient(stalled)

	pairs := wsPairs(t, 2)
	elsewhere := &Client{Conn: pairs[0].server, Room: "ELSEWHERE", PlayerID: "elsewhere", PlayerName: "Elsewhere"}
	joiner := &Client{Conn: pairs[1].server, Room: "ELSEWHERE", PlayerID: "joiner", PlayerName: "Joiner"}
	rm.AddClient(elsewhere)

	stuck := make(chan struct{})
	go func() {
		defer close(stuck)
		rm.SendTo("STUCK", "target", offerFrame("stuck"))
	}()
	waitUntilWriting(t, stalled)

	returnsWithin(t, promptly, "AddClient while a SendTo elsewhere is stalled", func() {
		rm.AddClient(joiner)
	})
	returnsWithin(t, promptly, "Broadcast to another room while a SendTo elsewhere is stalled", func() {
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
	// A second SendTo, to a different room, is the arm closest to what the
	// offers feature actually does: two trades proposed at the same instant in
	// two rooms, one of the recipients frozen.
	returnsWithin(t, promptly, "SendTo in another room while a SendTo elsewhere is stalled", func() {
		rm.SendTo("ELSEWHERE", "joiner", offerFrame("unblocked"))
	})
	returnsWithin(t, promptly, "RemoveClient while a SendTo elsewhere is stalled", func() {
		rm.RemoveClient(joiner)
	})

	// Let the deadline end the stalled send before writeWait is restored, so
	// nothing is left running when this test returns. One second of margin on
	// purpose: the macOS kernel reopens the stalled socket on its own after
	// about five (see stalledConn), and a wait that long would let a build
	// with no deadline at all pass this test.
	select {
	case <-stuck:
	case <-time.After(writeWait + time.Second):
		t.Fatal("the SendTo to the stalled target did not return when the deadline passed")
	}
}

func TestSendToIsSafeAgainstSeatingWhileItFansOut(t *testing.T) {
	// The cost of taking the fan-out out from under the read lock, and the
	// reason roomClientsSeatedAs filters inside the lock rather than leaving
	// the `client.PlayerID != playerID` test in SendTo's loop.
	//
	// Broadcast could move its whole loop out because it reads no Client field
	// any other goroutine writes; it only calls WriteJSON, which takes writeMu.
	// SendTo's loop reads PlayerID, and invariant 7 (HANDOFF.md) is that
	// PlayerID is written only through SeatClient, under rm.mu.Lock(). Ranging
	// over roomClients and comparing out in the open would be an
	// unsynchronized read of a string against every JOIN in the room - two
	// words, pointer and length, the runtime is entitled to let tear.
	//
	// The existing churn test here would not catch that: it moves clients in
	// and out of the map but never re-seats one, so nothing ever writes
	// PlayerID while a send is reading it. This does, and `go test -race` is
	// the whole gate - it is closeClient_test.go's seating race, which exists
	// for the same field, aimed at SendTo.
	const (
		room       = "SENDTO-SEAT-RACE"
		clientN    = 6
		senderN    = 3
		iterations = 300
	)

	rm := NewRoomManager()
	ids := make([]string, clientN)
	for i := range ids {
		ids[i] = "bob"
	}
	clients := kickRoom(t, rm, room, ids...)

	// Every conn is killed for killConn's own reason: gorilla allows one
	// concurrent writer per conn, so live conns under concurrent sends would
	// be their own race, drowning out the one under test. Dead conns also keep
	// SendTo's prune path hot against the seating.
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
				rm.SeatClient(client, "bob", "Bob")
			}
		}()
	}

	for i := 0; i < senderN; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for n := 0; n < iterations; n++ {
				rm.SendTo(room, "bob", offerFrame("seat-race"))
			}
		}()
	}

	// Churn keeps clients arriving and leaving under the snapshot, which is
	// what handler.go's disconnect defer does for real.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for n := 0; n < iterations; n++ {
			rm.RemoveClient(clients[0])
			rm.AddClient(clients[0])
		}
	}()

	wg.Wait()
}
