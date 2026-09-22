package websocket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zachmshort/emoney-backend/config"
	"github.com/zachmshort/emoney-backend/controllers"
	"github.com/zachmshort/emoney-backend/manager"
	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoomManager struct {
	clients map[string]map[*Client]bool
	mu      sync.RWMutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		clients: make(map[string]map[*Client]bool),
	}
}

func (rm *RoomManager) AddClient(client *Client) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.clients[client.Room] == nil {
		rm.clients[client.Room] = make(map[*Client]bool)
	}
	rm.clients[client.Room][client] = true
}

func (rm *RoomManager) RemoveClient(client *Client) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if _, ok := rm.clients[client.Room]; ok {
		delete(rm.clients[client.Room], client)
		if len(rm.clients[client.Room]) == 0 {
			delete(rm.clients, client.Room)
		}
	}
}

// SeatClient records which player a connection belongs to, under the same lock
// that guards the room map.
//
// PlayerID and PlayerName are written exactly once per connection, by that
// connection's own reader goroutine when its JOIN succeeds. Until
// CloseClientByPlayerID existed nothing else ever read them, and an
// unsynchronized assignment in handler.go was safe. It is not any more: a kick
// scans every client in a room for a PlayerID from the *kicking* player's
// goroutine, and an unsynchronized read of a string against a concurrent write
// to it is a data race - two words, pointer and length, that the runtime is
// entitled to let tear. rm.mu already orders "who is in this room"; this puts
// "who this connection is" under the same lock rather than giving Client a
// second mutex beside writeMu.
//
// The owning goroutine may still read its own client.PlayerID and PlayerName
// without the lock, because it is the only writer - which is what handler.go's
// disconnect defer does when it broadcasts PLAYER_LEFT.
func (rm *RoomManager) SeatClient(client *Client, playerID, playerName string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	client.PlayerID = playerID
	client.PlayerName = playerName
}

// roomClients returns the clients in room at this instant, copied out under
// the read lock so that the caller can do its network I/O without it. The
// lock is released by defer, so nothing that panics on the way out - nothing
// in here can, but Broadcast's fan-out used to sit inside this scope - leaves
// rm.mu read-locked for good, which would be the same process-wide stall a
// dead socket used to cause, with no TCP timeout to end it. A *Client copied
// here stays a valid pointer after its own goroutine removes it from the map;
// the worst case is one failed write to a closed conn.
func (rm *RoomManager) roomClients(room string) []*Client {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	clients := make([]*Client, 0, len(rm.clients[room]))
	for client := range rm.clients[room] {
		clients = append(clients, client)
	}
	return clients
}

// roomClientsSeatedAs returns the clients in room seated as playerID at this
// instant, copied out under the read lock so that the caller can do its
// network I/O without it. It is roomClients with the filter moved inside the
// lock, and the filter is inside on purpose rather than left in the caller's
// loop: invariant 7 (HANDOFF.md) is that Client.PlayerID is written only
// through SeatClient, which holds rm.mu.Lock(), and read from another
// player's goroutine only under rm.mu. A caller that ranged over roomClients
// and compared PlayerID itself would be doing that read with no lock at all -
// a data race on a string, two words the runtime is entitled to let tear,
// against every JOIN in the room. Broadcast has no such read, which is why it
// needs no sibling of this; SendTo's whole job is that read.
func (rm *RoomManager) roomClientsSeatedAs(room, playerID string) []*Client {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	var clients []*Client
	for client := range rm.clients[room] {
		if client.PlayerID == playerID {
			clients = append(clients, client)
		}
	}
	return clients
}

// Broadcast is the single fan-out point for every room message. It refuses to
// send a message whose "notification" field is present but empty (or present
// and not a string). Every websocket money handler in this package builds a
// human-readable notification string as its last step before calling
// Broadcast; board item 3 is what happens when a handler's own branching
// forgets to fill that string in - the FREE_PARKING message went out with
// notification "" for money that never moved, and every client in the room
// toasted blank text. That specific hole is closed in freeParking, but
// nothing stopped the next handler from reproducing it, because nothing
// checked the string ever got written. This is that check, at the one place
// every broadcast already passes through.
//
// A payload that never promises a notification at all - it has no
// "notification" key - is a different contract and is not this guard's
// concern; it is broadcast unchanged. Refusal is silent to the room (a
// dropped broadcast never reaches the frontend's ERROR toast - there is no
// path from here to there) and loud in the server log, on the theory that
// this case should not occur in practice - freeParking's own validation
// already prevents it - so the only audience for a rejection is whoever
// reads the backend log looking for why a handler's next-handler-copy of the
// same mistake produced silence instead of a bad toast.
func (rm *RoomManager) Broadcast(room string, message Message) {
	if empty, hasField := emptyNotification(message.Payload); hasField && empty {
		log.Printf("Broadcast refused for room %s: %s payload has an empty or non-string notification", room, message.Type)
		return
	}

	// The fan-out holds no lock. It used to run under rm.mu.RLock(), and a
	// write to one stalled peer - a tab whose socket buffer had filled, a
	// phone that had lost the network - blocked that goroutine with the read
	// lock held. The next RemoveClient then waited on rm.mu.Lock(), and
	// because an RWMutex refuses new readers while a writer is waiting, every
	// later Broadcast in every room, every join and every leave in the
	// process queued behind one dead socket, for as long as TCP took to give
	// up - about fifteen minutes on Linux defaults. Client.WriteJSON's
	// deadline now bounds any one write, but a bounded process-wide stall is
	// still a process-wide stall: the room is copied out under the read lock
	// in roomClients and written to after it is released, so the hub lock
	// never covers a network write, and a panic in the fan-out has no read
	// lock to leak.
	//
	// What the snapshot gives up is nothing that was promised. A client that
	// leaves mid-fan-out is written to once more; its conn is closed by then,
	// so the write fails, it lands in dead, and deleting a client the room has
	// already dropped is a no-op below. Two broadcasts from two goroutines
	// were never ordered against each other - both held RLock and interleaved
	// freely - and each still delivers in its own order to every client,
	// because this loop is sequential. What a stalled peer costs now is up to
	// writeWait of this goroutine's time, once, and of any other broadcaster
	// to the same room that queues on that client's writeMu meanwhile; the
	// first timeout closes the conn and latches its error, so nobody pays
	// twice.
	//
	// The dead clients are collected here and deleted below, under the write
	// lock, because the fan-out holds no lock, and RLock would not have been
	// enough anyway: two broadcasts to the same room deleting from one Go map
	// at once is answered by the runtime with fatal("concurrent map writes"),
	// not a panic, so Gin's Recovery does not catch it and the one backend
	// process dies.
	var dead []*Client
	for _, client := range rm.roomClients(room) {
		// Client.WriteJSON, not client.Conn.WriteJSON: nothing here orders
		// access to any one conn's writer. Two broadcasts to the same room
		// write to the same conns at once, and so does handler.go's ERROR
		// reply on a player's own conn. The per-client lock is what keeps one
		// goroutine at a time inside gorilla's writer, and the deadline it
		// arms is what keeps that goroutine from staying there - see the
		// Client doc comment in types.go.
		if err := client.WriteJSON(message); err != nil {
			client.Conn.Close()
			dead = append(dead, client)
		}
	}

	if len(dead) == 0 {
		return
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()
	// Re-checked: RemoveClient can empty the room and drop the key entirely
	// between the two locks.
	if clients, ok := rm.clients[room]; ok {
		for _, client := range dead {
			delete(clients, client)
		}
	}
}

// SendTo writes message to every live connection in room that is seated as
// playerID, and reports how many it reached. It is the targeted counterpart of
// Broadcast, and it exists because an offer is not room news: a trade proposal
// goes to the one player it was made to, and the sender gets their own
// confirmation, while everyone else in the room sees nothing until the trade
// settles.
//
// A player with two tabs open has two *Clients with the same PlayerID, and
// this sends to all of them. Decided here rather than left to fall out of the
// loop: the alternative - first match wins - would put the offer on whichever
// tab happened to be first in map iteration order, which is random in Go, so
// a player looking at the other tab would see nothing arrive and the toast
// would fire on a screen nobody was reading. Every tab is the same person and
// every tab refetches the inbox on its next message anyway; the cost of
// sending to all of them is one extra frame.
//
// The lock discipline is Broadcast's, exactly, and for the same reason: the
// targets are snapshotted under the read lock, written to holding no lock at
// all, and the dead among them deleted under the write lock - because two
// RLock holders deleting from one Go map is fatal("concurrent map writes"),
// which is not a panic, so Gin's Recovery does not catch it and the single
// backend process dies.
//
// The fan-out holds no lock because a write to a stalled peer - a phone that
// lost the network, a tab the browser froze - blocks until Client.WriteJSON's
// deadline fires, and doing that under rm.mu.RLock() made every Lock caller
// and, per RWMutex's writer preference, every later RLock caller in every room
// wait it out. Board row 11 took that shape out of Broadcast; this is the same
// removal here. Bounded by the deadline is not the same as absent: ten seconds
// of a frozen hub is still a frozen hub, and the peer that causes it is one
// player's phone.
//
// The PlayerID filter rides inside the snapshot rather than this loop, which
// is the one way SendTo is not simply Broadcast with an if. Invariant 7
// (HANDOFF.md) is that Client.PlayerID is written only through SeatClient
// under rm.mu.Lock(); comparing it out here, after the lock is released, would
// be an unsynchronized read against every JOIN in the room. See
// roomClientsSeatedAs. And the write goes through Client.WriteJSON, never
// client.Conn.WriteJSON, because a Broadcast on another goroutine can be
// inside this same conn's writer at this instant and gorilla permits exactly
// one.
//
// Same empty-notification guard as Broadcast: every message sent here carries
// a notification the recipient toasts, and an arm that forgot to set one
// should be dropped loudly in the log rather than toasted blank.
//
// Zero reached is not an error. The recipient may be offline; the offer is in
// Mongo and their inbox fetches it when they are back. Callers log the count.
func (rm *RoomManager) SendTo(room, playerID string, message Message) int {
	if playerID == "" {
		// Every connection carries PlayerID "" from the upgrade until its JOIN
		// succeeds, so matching on it would send a private message to every
		// unseated conn in the room. There is no player whose id is the empty
		// string. Same guard, same reason, as CloseClientByPlayerID.
		return 0
	}
	if empty, hasField := emptyNotification(message.Payload); hasField && empty {
		log.Printf("SendTo refused for room %s: %s payload has an empty or non-string notification", room, message.Type)
		return 0
	}

	var dead []*Client
	reached := 0

	for _, client := range rm.roomClientsSeatedAs(room, playerID) {
		if err := client.WriteJSON(message); err != nil {
			client.Conn.Close()
			dead = append(dead, client)
			continue
		}
		reached++
	}

	if len(dead) == 0 {
		return reached
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()
	if clients, ok := rm.clients[room]; ok {
		for _, client := range dead {
			delete(clients, client)
		}
	}
	return reached
}

// CloseClientByPlayerID force-closes every live connection in room that is
// seated as playerID, and reports how many it closed. It is what makes a kick
// actually remove someone: the kicked browser reconnects a second after any
// close and re-sends JOIN, which handler.go now refuses for an inactive player.
//
// It deliberately does not touch rm.clients. Each connection already has its
// own cleanup - handler.go's per-connection defer calls RemoveClient, which
// takes rm.mu.Lock(), and broadcasts PLAYER_LEFT once the closed conn makes
// that goroutine's blocked ReadJSON error and break. Removing the client here
// as well would either double-broadcast PLAYER_LEFT or take rm.mu a second
// time from a goroutine that is already inside it. Closing the conn is what
// makes that existing cleanup run; it is not a second copy of it.
//
// The conns are collected under the read lock and closed after it is released.
// Only the scan needs the lock, and keeping a syscall out from under the hub
// lock means a later edit inside that loop cannot reach back into rm.mu and
// deadlock against the RemoveClient it is about to provoke. A *Client
// collected here stays a valid pointer even if its own goroutine removes it
// from the map first; the worst case is Close on an already-closed conn, which
// returns an error nobody needs.
//
// Close, not WriteJSON: a close is not a write, so it takes no writeMu (see
// the Client doc comment in types.go) and there is no ordering between writeMu
// and rm.mu to get wrong here. gorilla's underlying net.Conn.Close is safe to
// call while the owning goroutine is blocked in Read, which is the whole
// mechanism this depends on.
func (rm *RoomManager) CloseClientByPlayerID(room, playerID string) int {
	if playerID == "" {
		// Every connection carries PlayerID "" from the upgrade until its JOIN
		// succeeds, so matching on it would close every unseated conn in the
		// room. There is no player whose id is the empty string.
		return 0
	}

	var targets []*Client

	rm.mu.RLock()
	for client := range rm.clients[room] {
		if client.PlayerID == playerID {
			targets = append(targets, client)
		}
	}
	rm.mu.RUnlock()

	for _, client := range targets {
		client.Conn.Close()
	}

	return len(targets)
}

// emptyNotification reports whether payload carries a "notification" field
// and, if so, whether that field is empty (hasField distinguishes "no such
// key" from "key present with a zero value"). It recognizes the two payload
// shapes the broadcast sites in this package actually build -
// map[string]interface{} (every websocketManager.go handler and
// PLAYER_JOINED) and map[string]string (PLAYER_LEFT) - and treats any other
// payload shape, including a bare string like the ones the ERROR path writes
// directly with WriteJSON rather than through Broadcast, as not having the
// field at all: nothing to refuse.
func emptyNotification(payload interface{}) (empty bool, hasField bool) {
	switch p := payload.(type) {
	case map[string]interface{}:
		raw, ok := p["notification"]
		if !ok {
			return false, false
		}
		text, isString := raw.(string)
		return !isString || text == "", true
	case map[string]string:
		text, ok := p["notification"]
		if !ok {
			return false, false
		}
		return text == "", true
	default:
		return false, false
	}
}

func (rm *RoomManager) handleTransfer(client *Client, message Message) error {

	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	amountStr, ok := payload["amount"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for amount")
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if amount < 1 {
		// strconv.Atoi parses "-100", and controllers.PlayerTransfer builds its
		// two $inc updates with the signs fixed by the direction rather than by
		// the amount - -transfer.Amount off the sender, +transfer.Amount onto
		// the recipient. So a SEND of -100 credits the sender $100 and debits
		// the recipient $100: a transfer that runs backwards, asked for by the
		// player it pays. Every route here is unauthenticated, so the only
		// credential that stands between a room and that frame is the room code.
		//
		// Unlike handleManageProperties, which reads a negative as SELL and
		// takes the absolute value (:774), nothing on this path gives the sign
		// a meaning: TRANSFER carries its direction in transferType. There is
		// no frame this rejects that used to do something legitimate.
		//
		// Rejected here rather than in the controller for the same reason the
		// free parking floor is: it is a fact about the payload, independent of
		// any state, so it costs no round trip and stays reachable from a test
		// (config.DB is nil in a test binary, so anything past the switch
		// panics instead of erroring). PlayerTransfer keeps its own copy of
		// this rule anyway - see transferRejection - because it is exported and
		// this handler is not the only caller it could ever have.
		//
		// $0 is refused with the negatives, same as free parking and a bid: it
		// moves nothing, but it still writes a Transfer row and toasts the whole
		// room about money that did not go anywhere.
		return errors.New("a transfer has to be at least $1")
	}

	roomIdStr, ok := payload["roomId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for roomId")
	}
	roomObjID, err := primitive.ObjectIDFromHex(roomIdStr)
	if err != nil {
		return fmt.Errorf("invalid room ID: %v", err)
	}

	reason, ok := payload["reason"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for reason")
	}

	transferType, ok := payload["transferType"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for transferType")
	}

	transfer := models.Transfer{
		ID:        primitive.NewObjectID(),
		RoomID:    roomObjID,
		Amount:    amount,
		Reason:    reason,
		Type:      transferType,
		TimeStamp: time.Now(),
		Status:    models.TransferPending,
	}

	var transferErr error
	switch transfer.Type {
	case "SEND":
		fromPlayerIdStr, ok := payload["fromPlayerId"].(string)
		if !ok {
			return errors.New("invalid payload: expected string for fromPlayerId")
		}
		fromID, err := primitive.ObjectIDFromHex(fromPlayerIdStr)
		if err != nil {
			return fmt.Errorf("invalid fromPlayerId: %w", err)
		}

		toPlayerIdStr, ok := payload["toPlayerId"].(string)
		if !ok {
			return errors.New("invalid payload: expected string for toPlayerId")
		}
		toID, err := primitive.ObjectIDFromHex(toPlayerIdStr)
		if err != nil {
			return fmt.Errorf("invalid toPlayerId: %w", err)
		}
		transfer.FromPlayerID = fromID
		transfer.ToPlayerID = toID

		transferErr = controllers.PlayerTransfer(transfer)
	case "REQUEST":
		transferErr = errors.New("request transfers not implemented yet")
	default:
		transferErr = fmt.Errorf("invalid transfer type: %s", transfer.Type)
	}

	if transferErr != nil {
		return transferErr
	}

	transfer.Status = models.TransferCompleted
	var fromPlayer, toPlayer *models.Player

	fromPlayer, err = controllers.GetPlayer(transfer.FromPlayerID)
	if err != nil {
		log.Printf("Failed to get from player details: %v", err)
		return err
	}

	toPlayer, err = controllers.GetPlayer(transfer.ToPlayerID)
	if err != nil {
		log.Printf("Failed to get to player details: %v", err)
		return err
	}
	notification := fmt.Sprintf("%s just sent $%s to %s for %s", fromPlayer.Name, strconv.Itoa(amount), toPlayer.Name, transfer.Reason)
	rm.Broadcast(client.Room, Message{
		Type: "TRANSFER",
		Payload: map[string]interface{}{
			"notification": notification,
		},
	})

	return nil
}

// freeParkingRejection is board row 16's decision at the Free Parking pot: a
// removed player may not move money. Both arms of this handler are the actor
// acting on their own balance - ADD pays into the pot out of their own money,
// REMOVE takes the pot into it - so the player named in the payload is the one
// moving the money and this is the "may not act" half of the rule. There is no
// second question here about who is being acted upon; the pot is not a player.
//
// Pure, and taking a player already read rather than reading one itself, for
// the reason transferRejection gives (controllers/transferControllers.go):
// config.DB is a nil *mongo.Database in a test binary, so a rule written inline
// inside the transaction below is a rule no test on this machine can reach.
//
// The amount floor is deliberately not here. It is a fact about the payload, so
// it sits above the first database call where it costs no round trip; this
// function only decides what the player document says.
func freeParkingRejection(player models.Player) error {
	if !player.IsActive {
		return errors.New("a removed player cannot use free parking")
	}
	return nil
}

func (rm *RoomManager) freeParking(client *Client, message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	amountStr, ok := payload["amount"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for amount")
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if amount < 1 {
		// strconv.Atoi happily parses "-50", and both arms below are built out
		// of $inc pairs whose signs are fixed by the arm, not by the amount -
		// so a negative amount runs the arm backwards while passing the arm's
		// own floor. ADD of -50 clears `player.Balance < amount` (1500 < -50 is
		// false) and then increments the balance by 50 and Free Parking by -50:
		// a contribution that pays the contributor out of the pot. REMOVE of
		// -1000 is worse, because it launders the balance floor entirely -
		// `room.FreeParking < amount` is false for any pot, and the player is
		// debited $1000 they do not have into a pot that gains it, with no
		// insufficient-funds check anywhere on that arm.
		//
		// Rejected here rather than inside the transaction for the same reason
		// the freeParkingType switch is hoisted: it is a fact about the
		// payload, independent of any state, so it costs no round trip and
		// stays reachable from a test. Same shape and same reasoning as
		// handlePlaceBid's `amount < 1`.
		//
		// $0 is refused with the negatives: it moves nothing, but it still
		// writes an event-history row and toasts the whole room about money
		// that did not go anywhere.
		return errors.New("a free parking amount has to be at least $1")
	}

	roomIdStr, ok := payload["roomId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for roomId")
	}
	roomObjID, err := primitive.ObjectIDFromHex(roomIdStr)
	if err != nil {
		return fmt.Errorf("invalid room ID: %v", err)
	}

	playerId, ok := payload["playerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for playerId")
	}
	playerObjID, err := primitive.ObjectIDFromHex(playerId)
	if err != nil {
		return fmt.Errorf("invalid player ID: %w", err)
	}

	actionType, ok := payload["freeParkingType"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for freeParkingType")
	}

	// Validated here, above controllers.GetPlayer, for the same two reasons
	// handleBankTransaction hoists its transactionType switch: an unrecognized
	// value costs no database round trip, and the rejection is reachable in a
	// test (config.DB is nil in a test binary, so anything past this point
	// panics instead of erroring).
	switch actionType {
	case "ADD", "REMOVE":
		// valid - the transaction below switches on these same two values
	default:
		return fmt.Errorf("invalid free parking type: %s", actionType)
	}

	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---

	var notification string

	session, err := config.DB.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
		// Reset every captured variable at the top, because WithTransaction
		// re-runs this callback on a write conflict. Same discipline as
		// handleCloseAuction, and for the same reason: a retry that inherits
		// the previous attempt's values is deciding from state that was rolled
		// back.
		notification = ""

		// The player is read HERE, inside the transaction and on ctx, rather
		// than through controllers.GetPlayer above it - which is where this
		// read used to be, on context.Background(), outside the session.
		//
		// The balance is the only thing the ADD arm's floor consults, so a read
		// outside the transaction means a retry re-checks a value the retry was
		// triggered by someone else changing. Concretely: balance $100, a
		// $100 ADD and a concurrent $100 debit race; the debit commits, this
		// callback loses the write conflict and is re-run, the stale $100 still
		// clears `balance < amount`, and the player is driven to -$100. The
		// floor is only a floor if the number it reads comes from the same
		// snapshot as the write it guards.
		//
		// Still filtered on _id alone - no roomId and no isActive - and that
		// is now a different decision than it was when row 13 moved this read
		// in. It used to mean "this read is unchanged, and whether a frozen
		// player is refused is raised on the board rather than answered here".
		// Row 16 answered it (no, they may not), and row 19 applies that
		// answer, so the question is settled - but the filter stays exactly as
		// it was, because the rule belongs in freeParkingRejection and not in
		// the query. An isActive clause here would turn a refusal into "failed
		// to get player details", which tells the player nothing and is the
		// distinction handleCloseAuction's winner read already draws.
		var player models.Player
		if err := config.DB.Collection("Player").FindOne(ctx, bson.M{"_id": playerObjID}).Decode(&player); err != nil {
			return nil, fmt.Errorf("failed to get player details: %w", err)
		}
		if err := freeParkingRejection(player); err != nil {
			return nil, err
		}

		switch actionType {
		case "ADD":
			if player.Balance < amount {
				return nil, fmt.Errorf("insufficient funds to contribute to free parking")
			}

			_, err = config.DB.Collection("Player").UpdateOne(
				ctx,
				bson.M{"_id": playerObjID},
				bson.M{"$inc": bson.M{"balance": -amount}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update player balance: %w", err)
			}

			_, err = config.DB.Collection("Room").UpdateOne(
				ctx,
				bson.M{"_id": roomObjID},
				bson.M{"$inc": bson.M{"freeParking": amount}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update free parking: %w", err)
			}

			notification = fmt.Sprintf("%s added $%d to Free Parking", player.Name, amount)
		case "REMOVE":
			var room models.Room
			err := config.DB.Collection("Room").FindOne(ctx, bson.M{"_id": roomObjID}).Decode(&room)
			if err != nil {
				return nil, fmt.Errorf("failed to get room details: %w", err)
			}

			if room.FreeParking < amount {
				return nil, fmt.Errorf("insufficient funds in free parking")
			}

			_, err = config.DB.Collection("Room").UpdateOne(
				ctx,
				bson.M{"_id": roomObjID},
				bson.M{"$inc": bson.M{"freeParking": -amount}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update free parking: %w", err)
			}

			_, err = config.DB.Collection("Player").UpdateOne(
				ctx,
				bson.M{"_id": playerObjID},
				bson.M{"$inc": bson.M{"balance": amount}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to update player balance: %w", err)
			}

			notification = fmt.Sprintf("%s collected $%d from Free Parking", player.Name, amount)
		default:
			// Unreachable: actionType was validated above. Kept so this switch
			// can never fall through to `return nil, nil` with no notification
			// set, which is what broadcast an empty-text FREE_PARKING toast to
			// the whole room while moving no money.
			return nil, fmt.Errorf("invalid free parking type: %s", actionType)
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	// After the transaction, not inside it. Inside, it was called with
	// context.Background() rather than the session's ctx, which meant it was
	// never in the transaction that appeared to contain it - so a retried
	// attempt inserted the row a second time, and an aborted attempt left a row
	// claiming money moved when none did. handleKickPlayer and
	// handleCloseAuction both call it out here; freeParking was the one that
	// did not.
	rm.CreateEventHistory(notification, roomObjID)

	rm.Broadcast(client.Room, Message{
		Type: "FREE_PARKING",
		Payload: map[string]interface{}{
			"notification": notification,
		},
	})
	log.Printf("Free parking update broadcast complete for room: %s", client.Room)

	return nil
}

// The purchase rules - the frozen buyer, the price floor, the deed still being
// for sale and the buyer being able to afford it - are
// controllers.propertyPurchaseRejection, and they used to be a
// propertyPurchaseRejection here that refused the frozen buyer and nothing
// else. They moved on 2026-09-22, when PurchaseProperty became a transaction
// that reads the buyer and the deed inside itself: a rule that decides from
// those values has to be callable from there, and this package cannot be
// imported by controllers. Their tests moved with them, to
// controllers/propertyControllers_test.go.
//
// What stayed here is the price floor below, which is a fact about the payload
// rather than about any document - the same split freeParkingRejection's
// comment describes, and the same placement handleTransfer's amount floor has.

func (rm *RoomManager) handlePropertyPurchase(client *Client, message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	priceFloat, ok := payload["price"].(float64)
	if !ok {
		return fmt.Errorf("invalid price format")
	}
	price := int(priceFloat)
	if price < 1 {
		// The price arrives as a bare JSON number and, until this line, nothing
		// anywhere looked at it: controllers.PurchaseProperty $inc'd the balance
		// by -price with no read and no check, so a price of -1000 CREDITED the
		// buyer $1000 and handed them the deed in the same frame. Every route
		// here is unauthenticated, so the only credential between a room and that
		// frame is the room code.
		//
		// Rejected here rather than only in the controller for the same reason
		// the transfer and free parking floors are: it is a fact about the
		// payload, independent of any state, so it costs no round trip and stays
		// reachable from a test (config.DB is nil in a test binary, so anything
		// past the first database call panics instead of erroring).
		// propertyPurchaseRejection keeps its own copy anyway, because the floor
		// belongs to the money write and PurchaseProperty is exported.
		//
		// int(priceFloat) truncates toward zero, so a fractional price under a
		// dollar arrives here as 0 and is refused with the negatives rather than
		// buying a deed for nothing. $0 is refused on its own account too: it
		// moves no money, but it still writes an event-history row and toasts the
		// whole room about a purchase that cost nothing.
		return errors.New("a property purchase has to be at least $1")
	}

	buyerIdStr, ok := payload["buyerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for buyerId")
	}
	buyerID, err := primitive.ObjectIDFromHex(buyerIdStr)
	if err != nil {
		log.Printf("Invalid buyerId error: %v", err)
		return fmt.Errorf("invalid buyerId: %w", err)
	}

	propertyIdStr, ok := payload["propertyId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for propertyId")
	}
	propertyID, err := primitive.ObjectIDFromHex(propertyIdStr)
	if err != nil {
		log.Printf("Invalid propertyId error: %v", err)
		return fmt.Errorf("invalid propertyId: %w", err)
	}
	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---
	//
	// One call, not a read and then a write: the read of the deed and the buyer
	// now happens inside PurchaseProperty's transaction, where the rules decide
	// from the same snapshot the writes land on. It hands back what it read so
	// the notification below names the buyer and the property without a second
	// round trip into the state this purchase just changed.
	property, buyer, err := controllers.PurchaseProperty(propertyID, buyerID, price)
	if err != nil {
		log.Printf("Property purchase failed: %v", err)
		return err
	}

	notification := fmt.Sprintf("%s purchased %s from the Bank", buyer.Name, property.Name)
	rm.CreateEventHistory(notification, property.RoomID)
	rm.Broadcast(client.Room, Message{
		Type: "PURCHASE_PROPERTY",
		Payload: map[string]interface{}{
			"notification": notification,
		},
	})

	return nil
}

// bankTransactionRejection is board row 16's decision at the banker's balance
// control, and this is the one site of the four where the frozen player is the
// TARGET rather than the actor: handleBankTransaction never reads who is
// acting, only who is being paid or charged. So the question it answers is not
// "may a frozen player act" but "may a banker act on a frozen player", and
// Zach's answer, 2026-09-17, is no.
//
// The reasoning, because it is a build-level call and a later reader should not
// have to re-derive it. There is no un-kick in this product: a removed player's
// estate has already been auctioned and their balance has already left the
// game, so the write changes a number nobody can ever spend while the broadcast
// tells the whole room "Banker has added $100 to X's balance" - a sentence
// about a consequence that does not exist. It also squares this handler with
// transferRejection, which has refused paying a frozen player since row 14
// (controllers/transferControllers.go): without this, a transfer into that
// document is refused and a banker add to it is not.
//
// To reverse it, delete this function and its call. The actor half of the rule
// at the other three sites stands on its own and does not depend on it.
func bankTransactionRejection(target models.Player) error {
	if !target.IsActive {
		// Deliberately the same sentence transferRejection's recipient arm
		// says, because it is the same situation reached by another route.
		return errors.New("that player has been removed from the game")
	}
	return nil
}

func (rm *RoomManager) handleBankTransaction(client *Client, message Message) error {

	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	amountStr, ok := payload["amount"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for amount")
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}
	if amount < 1 {
		// strconv.Atoi parses "-100", and the direction actually applied comes
		// from transactionType (BANKER_ADD/BANKER_REMOVE), not from amount's
		// sign - isAdd is decided separately below, and
		// controllers.UpdatePlayerBalanceByBanker builds its one $inc from
		// isAdd rather than from amount's sign. So a negative amount silently
		// reverses the real effect while the notification text below (built
		// from action/preposition, which only depend on isAdd) keeps
		// describing transactionType's direction - a banker who fat-fingers a
		// minus sign gets a notification that lies about which way the money
		// moved.
		//
		// Same floor, same reasoning and same placement as handleTransfer's
		// amount < 1 check above (:243-268): a fact about the payload,
		// independent of any state, so it costs no round trip and stays
		// reachable from a test (config.DB is nil in a test binary, so
		// anything past controllers.GetPlayer below panics instead of
		// erroring).
		//
		// $0 is refused with the negatives, same as transfer, free parking and
		// a bid: it moves nothing, but it still writes an event-history row
		// and toasts the whole room about money that did not go anywhere.
		//
		// Kept out of bankTransactionRejection deliberately, following
		// freeParkingRejection's reasoning rather than transferRejection's:
		// bankTransactionRejection only ever checks target.IsActive, which
		// needs the player read first, while this floor needs nothing but the
		// payload and belongs above that first database call.
		return errors.New("a bank transaction has to be at least $1")
	}

	toPlayerIdStr, ok := payload["toPlayerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for toPlayerId")
	}
	targetPlayerID, err := primitive.ObjectIDFromHex(toPlayerIdStr)
	if err != nil {
		return fmt.Errorf("invalid target player ID: %w", err)
	}

	roomIdStr, ok := payload["roomId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for roomId")
	}
	roomID, err := primitive.ObjectIDFromHex(roomIdStr)
	if err != nil {
		return fmt.Errorf("invalid target player ID: %w", err)
	}

	transactionType, ok := payload["transactionType"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for transactionType")
	}

	var isAdd bool
	switch transactionType {
	case "BANKER_ADD":
		isAdd = true
	case "BANKER_REMOVE":
		isAdd = false
	default:
		return fmt.Errorf("invalid transaction type: %s", transactionType)
	}

	targetPlayer, err := controllers.GetPlayer(targetPlayerID)
	if err != nil {
		return fmt.Errorf("failed to get target player details: %w", err)
	}
	if err := bankTransactionRejection(*targetPlayer); err != nil {
		return err
	}

	err = controllers.UpdatePlayerBalanceByBanker(roomID, targetPlayerID, amount, isAdd)
	if err != nil {
		return fmt.Errorf("failed to process bank transaction: %w", err)
	}

	var action, preposition string
	if isAdd {
		action = "added"
		preposition = "to"
	} else {
		action = "removed"
		preposition = "from"
	}

	notification := fmt.Sprintf("Banker has %s $%d %s %s's balance",
		action,
		amount,
		preposition,
		targetPlayer.Name,
	)

	rm.CreateEventHistory(notification, roomID)

	rm.Broadcast(client.Room, Message{
		Type: "BANKER_TRANSACTION",
		Payload: map[string]interface{}{
			"notification": notification,
		},
	})

	return nil
}

// managePropertiesRejection is board row 16's decision at property management.
// The player named in the payload is the actor - they are mortgaging,
// unmortgaging, developing or selling development on their own deeds - so this
// is the same "a frozen player may not act" half of the rule as
// freeParkingRejection and controllers.propertyPurchaseRejection (which lived
// in this file as propertyPurchaseRejection until 2026-09-22 - see the note
// above handlePropertyPurchase for where it went and why).
//
// Nothing here looks at the amount, and that is deliberate rather than an
// omission. handleManageProperties treats a negative amount as MEANINGFUL: it
// is what flips HOUSES to SELL below, so the amount < 1 floor the other money
// handlers carry would break this handler rather than fix it. The hazard is not
// uniform across these four sites and this one is the exception.
func managePropertiesRejection(player models.Player) error {
	if !player.IsActive {
		return errors.New("a removed player cannot manage properties")
	}
	return nil
}

func (rm *RoomManager) handleManageProperties(client *Client, message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	amountValue, ok := payload["amount"]
	if !ok {
		return fmt.Errorf("missing amount field")
	}

	var amount int
	switch v := amountValue.(type) {
	case float64:
		amount = int(v)
	case string:
		var err error
		amount, err = strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid amount value: %w", err)
		}
	default:
		return fmt.Errorf("unexpected type for amount: %T", v)
	}

	roomIdStr, ok := payload["roomId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for roomId")
	}
	roomObjID, err := primitive.ObjectIDFromHex(roomIdStr)
	if err != nil {
		return fmt.Errorf("invalid room ID: %w", err)
	}

	playerIdStr, ok := payload["playerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for playerId")
	}
	playerID, err := primitive.ObjectIDFromHex(playerIdStr)
	if err != nil {
		return fmt.Errorf("invalid player ID: %w", err)
	}

	properties, err := manager.ExtractPropertyDetails(payload["properties"])
	if err != nil {
		return fmt.Errorf("invalid properties: %w", err)
	}

	manageType, ok := payload["managementType"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for managementType")
	}

	// Validated here, above the first database call, for the two reasons
	// freeParking and handleBankTransaction hoist their own switches: an
	// unrecognized value costs no round trip, and the rejection stays reachable
	// from a test. This switch IS the work switch's old default arm, moved up.
	// It had to move when the player read went above that switch, because
	// otherwise an invalid managementType would panic on the nil config.DB
	// before it could ever be named.
	switch manageType {
	case "HOUSES", "MORTGAGE", "UNMORTGAGE", "SELL":
		// valid - the work switch below dispatches on these same four values
	default:
		return fmt.Errorf("invalid management type: %s", manageType)
	}

	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---

	// The player is read HERE, above every write, and that placement is the
	// point of the read rather than an accident of it. Until 2026-09-17 the
	// only read of this player was at the BOTTOM of the handler - after
	// manager.HandleHouseManagement had already rewritten the deeds and
	// manager.UpdatePlayerBalance had already moved the money - and it existed
	// only to put a name in the notification. A rule checked down there would
	// refuse nothing: it would report an error to a player whose houses were
	// already sold and whose balance was already credited for selling them.
	//
	// This read replaces that one. The block it replaced re-parsed the same
	// payload["playerId"] a second time under the name toPlayerId, and its
	// three error strings were already unreachable, because the parse at the
	// top of this handler had to succeed on that same field to get here.
	//
	// Filtered on _id alone, which is what controllers.GetPlayer does
	// (controllers/playerControllers.go:166), and left that way on purpose: the
	// rule belongs in managePropertiesRejection and not in the query, so that a
	// removed player gets a sentence about being removed rather than "failed to
	// get player details". Invariant 6 requires every read of Player to decide
	// about isActive; this one keeps the filter and answers it on the next line.
	player, err := controllers.GetPlayer(playerID)
	if err != nil {
		return fmt.Errorf("failed to get player details: %w", err)
	}
	if err := managePropertiesRejection(*player); err != nil {
		return err
	}

	switch manageType {
	case "HOUSES":
		err = manager.HandleHouseManagement(roomObjID, manageType, properties)
	case "MORTGAGE", "UNMORTGAGE", "SELL":
		err = manager.HandlePropertySaleMortgage(roomObjID, manageType, properties)
	default:
		// Unreachable: manageType was validated above. Kept so this switch
		// cannot fall through with err still nil and no work done, which would
		// then credit or debit the balance below for a change that never
		// happened.
		return fmt.Errorf("invalid management type: %s", manageType)
	}

	if err != nil {
		return err
	}

	err = manager.UpdatePlayerBalance(playerID, amount)
	if err != nil {
		return err
	}

	absAmount := amount
	if amount < 0 {
		if manageType == "HOUSES" {
			manageType = "SELL"
		}
		absAmount = -amount
	} else if manageType == "HOUSES" {
		manageType = "BUY"
	}

	var action, preposition, mType string

	switch manageType {
	case "MORTGAGE":
		action = "received"
		preposition = "for mortgaging"
		mType = "property"
	case "UNMORTGAGE":
		action = "paid"
		preposition = "to unmortgage"
		mType = "property"
	case "BUY":
		action = "spent"
		preposition = "to develop"
		mType = "properties"
	case "SELL":
		action = "received"
		preposition = "for selling development"
		mType = "on properties"
	default:
		return fmt.Errorf("unknown manageType: %s", manageType)
	}

	var totalCount int
	for _, property := range properties {
		totalCount += property.Count
	}

	notification := fmt.Sprintf("%s %s $%d %s %s",
		player.Name,
		action,
		absAmount,
		preposition,
		mType,
	)

	rm.CreateEventHistory(notification, roomObjID)
	rm.Broadcast(client.Room, Message{
		Type: "MANAGE_PROPERTIES",
		Payload: map[string]interface{}{
			"notification": notification,
		},
	})
	return nil
}

// handleKickPlayer removes a player from a live game. One banker action does
// four things: it disposes of the estate the way the banker chose, marks the
// player gone, hands the banker role on if the target held it, and force-closes
// the target's socket.
//
// The decisions this implements, so a later reader does not re-derive them from
// the code (docs/incomplete/kick-player/DESIGN.md):
//
//   - D10 - the cash write is *none*. The balance stays exactly as it is on the
//     document marked inactive. It is not credited to the room, and it does not
//     touch Room.freeParking, which already means something else. This is the
//     easiest decision in the feature to "improve" by accident.
//   - D11 - the player is marked isActive:false, never deleted. Freeze needs the
//     document so Property.playerId still resolves to a name, and every past
//     EventHistory row keeps meaning something.
//   - D12 - a banker may kick themselves. targetPlayerId equal to the caller is
//     a valid action on the same path, not an error.
//   - D13 - a deed going back to the bank is razed on the way.
//
// Everything above the first Mongo call is payload shape, and it is above it on
// purpose: config.DB is a nil *mongo.Database in a test binary, so a rejection
// that happens after the first read is not reachable from a test on this
// machine at all. TestKickRejectsBeforeReadingThePlayer is the guard on that
// ordering.
func (rm *RoomManager) handleKickPlayer(client *Client, message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	roomIdStr, ok := payload["roomId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for roomId")
	}
	roomObjID, err := primitive.ObjectIDFromHex(roomIdStr)
	if err != nil {
		return fmt.Errorf("invalid room ID: %w", err)
	}

	targetIdStr, ok := payload["targetPlayerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for targetPlayerId")
	}
	targetObjID, err := primitive.ObjectIDFromHex(targetIdStr)
	if err != nil {
		return fmt.Errorf("invalid target player ID: %w", err)
	}

	disposition, ok := payload["disposition"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for disposition")
	}
	// Validated here rather than at the write, for the same two reasons
	// handleBankTransaction hoists its transactionType switch: an unrecognized
	// value costs no database round trip, and the rejection is reachable in a
	// test. AUCTION was refused here until Phase 3 existed, because accepting
	// it would have marked a player gone and left their estate in limbo; it is
	// accepted now. frontend/types/payloads.ts gains the third arm in the same
	// commit as this line (BD-6), so the picker can never send a disposition
	// this switch rejects.
	switch disposition {
	case "BANK", "FREEZE", "AUCTION":
		// valid - the estate write below switches on these same three values
	default:
		return fmt.Errorf("invalid disposition: %s", disposition)
	}

	// successorPlayerId is optional on the wire because it is required only
	// when the target holds the banker role (D5), and whether they do is not
	// knowable without reading them. Absent, null and "" all mean "no successor
	// named"; anything else has to be a well-formed id, checked here so a
	// malformed one is refused before the first Mongo call rather than after.
	var successorObjID primitive.ObjectID
	hasSuccessor := false
	if raw, present := payload["successorPlayerId"]; present && raw != nil {
		successorIdStr, ok := raw.(string)
		if !ok {
			return errors.New("invalid payload: expected string for successorPlayerId")
		}
		if successorIdStr != "" {
			successorObjID, err = primitive.ObjectIDFromHex(successorIdStr)
			if err != nil {
				return fmt.Errorf("invalid successor player ID: %w", err)
			}
			if successorObjID == targetObjID {
				return errors.New("the successor cannot be the player being removed")
			}
			hasSuccessor = true
		}
	}

	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---

	playerColl := config.DB.Collection("Player")

	// Scoped to the room and to active players rather than going through
	// controllers.GetPlayer, which filters on _id alone: a kick must not reach
	// across rooms, and kicking an already-kicked player should say so rather
	// than silently re-run the estate write.
	var target models.Player
	err = playerColl.FindOne(context.Background(), bson.M{
		"_id":      targetObjID,
		"roomId":   roomObjID,
		"isActive": true,
	}).Decode(&target)
	if err != nil {
		return fmt.Errorf("failed to find the player to remove: %w", err)
	}

	var successor models.Player
	switch {
	case target.IsBanker && !hasSuccessor:
		// D5: the room may never be left bankerless. Every banker control in
		// the product is gated on the client's own isBanker, so a bankerless
		// room loses the balance controls from every screen, permanently.
		return errors.New("removing the banker requires naming a successor")
	case !target.IsBanker && hasSuccessor:
		// The client thinks this player is the banker and the database
		// disagrees - a stale view, most likely because the role moved since
		// the screen was drawn. Promoting anyway would leave two bankers, and
		// nothing in this app rejects that state. Zach's call, 2026-09-17:
		// refuse loudly rather than drop the promotion silently, because a
		// silently-skipped write here is indistinguishable from success.
		return fmt.Errorf("%s is not the banker, so there is no banker role to hand on", target.Name)
	case hasSuccessor:
		err = playerColl.FindOne(context.Background(), bson.M{
			"_id":      successorObjID,
			"roomId":   roomObjID,
			"isActive": true,
		}).Decode(&successor)
		if err != nil {
			return fmt.Errorf("failed to find the successor: %w", err)
		}
	}

	// The lots, read before the transaction rather than inside it, for two
	// reasons: the notification has to say whether there was anything to
	// auction at all, and the queue is then built once instead of on every
	// transaction retry.
	//
	// PropertyIndex order is board order (D16), and the sort is the ordering
	// rather than a convenience - Mongo's natural order would put Boardwalk
	// before Mediterranean Avenue about as often as not, and a player cannot
	// tell a shuffled auction from a correct one by looking at it.
	//
	// Reading outside the transaction is the shape freeParking already uses
	// for its own player read. Nothing can move a kicked player's deeds
	// between here and the write except an unauthorized MANAGE_PROPERTIES from
	// another player, which is a pre-existing hole - handleManageProperties
	// does not check who owns the deed - and not this feature's to close.
	var lots []models.Property
	if disposition == "AUCTION" {
		cursor, err := config.DB.Collection("Property").Find(
			context.Background(),
			bson.M{"roomId": roomObjID, "playerId": targetObjID},
			options.Find().SetSort(bson.D{{Key: "propertyIndex", Value: 1}}),
		)
		if err != nil {
			return fmt.Errorf("failed to read the properties to auction: %w", err)
		}
		if err := cursor.All(context.Background(), &lots); err != nil {
			return fmt.Errorf("failed to decode the properties to auction: %w", err)
		}
	}

	notification := kickNotification(target.Name, disposition, successor.Name, len(lots))

	// One transaction, following freeParking above. controllers.PurchaseProperty
	// was the counter-example this comment named until 2026-09-22 - two bare
	// writes with no session - and it is transacted itself now, for the reason
	// the next sentence gives in its own terms.
	// A kick that demotes the old banker and then fails to promote the new one
	// leaves the room bankerless; one that promotes and fails to demote leaves
	// two bankers. Neither half may land alone.
	session, err := config.DB.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
		if disposition == "BANK" {
			// manager.HandlePropertySaleMortgage's SELL case at
			// manager/propertyManager.go:53 writes {playerId: nil,
			// isMortgaged: false} and leaves developmentLevel alone. This
			// write is that one plus developmentLevel: 0, and the difference
			// is deliberate (D13), not drift: GetAvailableProperties filters
			// on playerId being nil, so without the raze a returned
			// hotel-bearing deed reappears in Bank's Properties at its face
			// price and the next buyer inherits the development for free.
			// SELL is left alone on purpose - fixing it would change
			// behaviour outside this feature.
			//
			// The mortgage clearing is not a choice made here: it is what
			// SELL does, and it is right, because the mortgage is a debt to
			// the bank and the bank now holds the deed (D3).
			_, err := config.DB.Collection("Property").UpdateMany(
				ctx,
				bson.M{"roomId": roomObjID, "playerId": targetObjID},
				bson.M{"$set": bson.M{
					"playerId":         nil,
					"isMortgaged":      false,
					"developmentLevel": 0,
				}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to return the properties to the bank: %w", err)
			}
		}
		if disposition == "AUCTION" && len(lots) > 0 {
			// The deeds stay on the kicked player for the length of the
			// auction, deliberately. Clearing playerId here would put every
			// one of them into GetAvailableProperties - which filters on
			// exactly playerId being nil (controllers/propertyControllers.go)
			// - so anyone in the room could buy Boardwalk from the Bank at its
			// face price while it was still a lot, undercutting the auction it
			// is meant to be sold at. Each lot's own close is what moves the
			// deed, to the winner or to the Bank.
			queue := make([]primitive.ObjectID, 0, len(lots)-1)
			for _, lot := range lots[1:] {
				queue = append(queue, lot.ID)
			}

			// Conditional on no auction already running.
			// bson.M{"auction": nil} matches a missing key as well as an
			// explicit null, so this is the "no auction" test both for a room
			// that has never had one and for a room whose last auction was
			// $unset by its final close. Without the clause a second AUCTION
			// kick would overwrite a live auction's lot, queue and high bid,
			// and the first auction's bidders would be bidding on someone
			// else's estate with nothing erroring anywhere.
			result, err := config.DB.Collection("Room").UpdateOne(
				ctx,
				bson.M{"_id": roomObjID, "auction": nil},
				bson.M{"$set": bson.M{"auction": models.Auction{
					KickedPlayerID: targetObjID,
					PropertyID:     lots[0].ID,
					Queue:          queue,
					HighBid:        0,
					HighBidderID:   nil,
				}}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to open the auction: %w", err)
			}
			if result.MatchedCount == 0 {
				return nil, errors.New("an auction is already running in this room")
			}
		}
		// FREEZE writes nothing to any property (D4). The deeds stay against
		// the player, houses intact, which is why the document has to survive.
		// AUCTION writes no property either - see above.

		// isBanker:false alongside isActive:false is a no-op for a player who
		// was not the banker, and is the demotion half of the succession for
		// one who was. Filtered on isActive:true so a second kick racing this
		// one matches nothing and aborts the transaction instead of running
		// the estate write twice.
		result, err := playerColl.UpdateOne(
			ctx,
			bson.M{"_id": targetObjID, "roomId": roomObjID, "isActive": true},
			bson.M{"$set": bson.M{"isActive": false, "isBanker": false}},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to remove the player: %w", err)
		}
		if result.MatchedCount == 0 {
			return nil, errors.New("that player has already been removed")
		}

		if hasSuccessor {
			result, err := playerColl.UpdateOne(
				ctx,
				bson.M{"_id": successorObjID, "roomId": roomObjID, "isActive": true},
				bson.M{"$set": bson.M{"isBanker": true}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to promote the successor: %w", err)
			}
			if result.MatchedCount == 0 {
				return nil, errors.New("the successor is no longer in this room")
			}
		}

		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	rm.CreateEventHistory(notification, roomObjID)

	// Broadcast before the close, not after. The close makes the target's own
	// goroutine broadcast PLAYER_LEFT on its way out; sending PLAYER_KICKED
	// first means every other client refetches against a room that is already
	// written, rather than racing the two messages. The kicked player is told
	// nothing either way (D6) - they are simply gone.
	rm.Broadcast(client.Room, Message{
		Type: "PLAYER_KICKED",
		Payload: map[string]interface{}{
			"notification": notification,
			"playerId":     targetIdStr,
		},
	})

	if disposition == "AUCTION" && len(lots) > 0 {
		// A second message rather than a longer kick sentence: the kick says
		// the estate is going to auction, this says which deed is open. They
		// are two events in the room's log because they are two things a
		// player needs to be able to read back separately.
		opened := lotOpenNotification(lots[0].Name)
		rm.CreateEventHistory(opened, roomObjID)
		rm.Broadcast(client.Room, Message{
			Type: "AUCTION_STARTED",
			Payload: map[string]interface{}{
				"notification":   opened,
				"propertyId":     lots[0].ID.Hex(),
				"kickedPlayerId": targetIdStr,
				"lotCount":       len(lots),
			},
		})
	}

	closed := rm.CloseClientByPlayerID(client.Room, targetIdStr)
	log.Printf("Kick complete in room %s: removed player %s, closed %d connection(s)", client.Room, targetIdStr, closed)

	return nil
}

// kickNotification builds the one string every client in the room toasts and
// the one CreateEventHistory stores.
//
// The register is the banker as subject, matching handleBankTransaction's
// "Banker has removed $100 from X's balance", which is the nearest existing
// line. Chosen by Zach 2026-09-17 over a passive form and a terser
// table-voice one.
//
// Every arm returns text, including the unreachable default: Broadcast refuses
// a payload whose notification is present and empty, so an arm that forgot to
// set one would be dropped silently to the room and logged only on the VM -
// a kick that worked and that nobody saw, which looks exactly like a kick that
// did nothing. The phrase "from the game" is load-bearing beyond the copy: it
// is what eventTypeFor keys the kick's icon on, and every arm here including
// the auction ones carries it.
//
// lotCount is how many deeds the target holds, and it is a parameter rather
// than something this function could work out, because AUCTION is the one
// disposition whose sentence depends on it: saying "their properties go up for
// auction" for a player who holds none is a promise no lot will ever keep.
func kickNotification(targetName, disposition, successorName string, lotCount int) string {
	var text string

	switch disposition {
	case "BANK":
		text = fmt.Sprintf("Banker removed %s from the game. Their properties returned to the Bank.", targetName)
	case "FREEZE":
		text = fmt.Sprintf("Banker removed %s from the game. Their properties stay where they are.", targetName)
	case "AUCTION":
		if lotCount == 0 {
			text = fmt.Sprintf("Banker removed %s from the game. They had no properties to auction.", targetName)
		} else {
			text = fmt.Sprintf("Banker removed %s from the game. Their properties go up for auction.", targetName)
		}
	default:
		// Unreachable: disposition is validated in handleKickPlayer before any
		// read. Kept so this switch can never fall through with text unset.
		text = fmt.Sprintf("Banker removed %s from the game.", targetName)
	}

	if successorName != "" {
		text += fmt.Sprintf(" %s is now the Banker.", successorName)
	}

	return text
}

// --- the live auction (D14-D18) ---
//
// Two inbound message types and one piece of state. handleKickPlayer opens an
// auction when its disposition is AUCTION; PLACE_BID raises the high bid on the
// open lot; CLOSE_AUCTION is the banker's hammer, and it is the only thing that
// ends a lot. There is no timer, no ticker and no deadline anywhere in this
// package, which is D14's whole point: a countdown racing a bid to determine
// one winner is exactly the silent failure this phase exists not to have.
//
// The state is a models.Auction on the Room document rather than a field on
// RoomManager (D18). Everything below therefore reasons about one Mongo
// document that several goroutines can reach at once, and the correctness
// argument is the same in both handlers: never decide from a value you read and
// then write as though it were still true.
//
//   - A bid raises the high bid with a single conditional UpdateOne whose
//     filter carries the comparison. Two bids that arrive together are ordered
//     by Mongo's single-document atomicity, not by which goroutine read first,
//     so exactly one of them matches and the other is told it was beaten.
//   - A close reads inside a transaction and then pins its write to exactly the
//     (lot, high bid, high bidder) triple it read. A bid that lands mid-close
//     makes the close fail and say so, rather than hammering on a bid the
//     banker never saw.
//
// The rules themselves live in pure functions - bidRejection, outcomeForClose,
// advanceAuction, closeAuthorized - because config.DB is a nil *mongo.Database
// in a test binary, so a rule written inside a handler below its first Mongo
// call is a rule no test in this repo can reach.

// wholeDollars converts a JSON number to whole dollars, refusing anything that
// is not one.
//
// handlePropertyPurchase takes its price the same way and simply truncates
// (int(priceFloat)). A bid does not, because a silently truncated 120.9 is a
// bid the bidder did not make, and it is the truncated figure that the
// settlement charges and that every other bidder has to beat.
func wholeDollars(amount float64) (int, error) {
	if math.IsNaN(amount) || math.IsInf(amount, 0) || amount != math.Trunc(amount) {
		return 0, fmt.Errorf("invalid amount: %v is not a whole number of dollars", amount)
	}
	if amount > math.MaxInt32 || amount < math.MinInt32 {
		return 0, fmt.Errorf("invalid amount: %v is out of range", amount)
	}
	return int(amount), nil
}

// bidRejection reports why amount is not a valid bid on auction by bidder, or
// nil if it is.
//
// Pure, and taking the read values rather than doing the reads itself, so that
// every rule here is one a test can reach. The order is the order a player
// would want to hear about a problem: what they are bidding on, whether they
// may bid at all, whether the number beats the field, and only then whether
// they can afford it.
func bidRejection(auction models.Auction, lotID, bidderID primitive.ObjectID, amount, bidderBalance int) error {
	if auction.PropertyID != lotID {
		// This is also the "the bid arrived after the hammer" case. A close
		// advances auction.propertyId to the next deed, so a bid still naming
		// the lot that just closed lands here instead of being applied to a
		// deed nobody bid that amount on.
		return errors.New("that lot is no longer open for bidding")
	}
	if bidderID == auction.KickedPlayerID {
		// They have no socket after the kick's force-close and no standing in
		// the room, but a payload carries whatever player id it likes and
		// nothing else on this path would catch it.
		return errors.New("a removed player cannot bid on their own estate")
	}
	if amount <= auction.HighBid {
		// D15: a lot opens at $0 and the minimum raise is $1, which on whole
		// dollars is exactly "has to beat the current high bid". An opening
		// bid is the same comparison against a high bid of 0.
		return fmt.Errorf("a bid has to beat the current high bid of $%d", auction.HighBid)
	}
	if bidderBalance < amount {
		// D17's first check. The second is at settlement, because a bidder can
		// pay rent between bidding and the hammer - checking once is the bug,
		// checking twice is the decision.
		return fmt.Errorf("insufficient funds: a $%d bid is more than the $%d available", amount, bidderBalance)
	}
	return nil
}

// closeOutcome is what a close does with the open lot.
type closeOutcome string

const (
	lotSold       closeOutcome = "SOLD"
	lotNoBid      closeOutcome = "NO_BID"
	lotWinnerGone closeOutcome = "WINNER_GONE"
	lotCannotPay  closeOutcome = "CANNOT_PAY"
)

// outcomeForClose decides what happens to the open lot, from values read inside
// the settlement transaction. This is where D17's second check lives.
//
// Three of the four outcomes put the deed in the Bank's hands, and that is a
// decision rather than a coincidence of the code. D16 already settled that a
// lot nobody bid on returns to the bank; a winner who cannot pay and a winner
// who has been removed from the room since bidding are the same situation
// reached by other routes. Zach's call, 2026-09-17: treat all three the same
// way, rather than opening the re-auction / next-highest-bidder / bank question
// that D17's argument-against exists to avoid. The sentence the room reads says
// which of the three happened, so the outcome is never silent.
//
// winnerFound is false when the high bidder is no longer a live player in this
// room, in which case winnerBalance is not read.
func outcomeForClose(auction models.Auction, winnerFound bool, winnerBalance int) closeOutcome {
	if auction.HighBidderID == nil || auction.HighBid <= 0 {
		return lotNoBid
	}
	if !winnerFound {
		return lotWinnerGone
	}
	if winnerBalance < auction.HighBid {
		return lotCannotPay
	}
	return lotSold
}

// winnerStatus turns the high bidder's read into the two facts the settlement
// needs: whether they still count as the winner, and whether the read failed in
// a way that has to abort the close rather than be taken as "they left".
//
// This exists because of the bug it prevents, which the Deep review of this
// phase found and which needed no concurrency at all to reach. The first
// version of this code was `if err == nil { winnerFound = winner.IsActive }`,
// so *every* error - a dropped connection to Mongo as easily as a missing
// document - became "the winner is gone": the deed went to the Bank, the real
// high bidder was neither charged nor given it, and the room was told they had
// left the game. A wrong winner, silently, from one flaky read.
//
// A Player document is never deleted (D11), and every highBidderId was written
// from a room-scoped active-player read, so ErrNoDocuments here genuinely does
// mean gone. Anything else is the read failing, and a failing read must not be
// allowed to decide who owns a property. Returned as an error it aborts the
// attempt; if it carries a transient label WithTransaction retries the whole
// callback, and if it does not, nothing settles and the banker is told.
func winnerStatus(readErr error, winner models.Player) (found bool, fatal error) {
	switch {
	case readErr == nil:
		return winner.IsActive, nil
	case errors.Is(readErr, mongo.ErrNoDocuments):
		return false, nil
	default:
		return false, fmt.Errorf("failed to read the winning bidder: %w", readErr)
	}
}

// advanceAuction is the auction after the open lot closes: the next deed in the
// queue, opened at $0 with no bidder, or nil when the queue is empty and the
// auction is over. It is the only place the queue advances.
//
// The remaining queue is copied rather than resliced. The slice this returns is
// marshalled straight into the Room document, and an alias into the caller's
// backing array is the kind of sharing that is correct until someone appends to
// one of the two.
func advanceAuction(auction models.Auction) *models.Auction {
	if len(auction.Queue) == 0 {
		return nil
	}

	rest := make([]primitive.ObjectID, len(auction.Queue)-1)
	copy(rest, auction.Queue[1:])

	return &models.Auction{
		KickedPlayerID: auction.KickedPlayerID,
		PropertyID:     auction.Queue[0],
		Queue:          rest,
		HighBid:        0,
		HighBidderID:   nil,
	}
}

// closeAuthorized reports whether closer may close an auction lot.
//
// This is the only server-side isBanker check in the product, and it is a
// deliberate exception rather than the start of a policy. PLAN.md section 4
// reserves general isBanker enforcement as not built, on purpose, and every
// other action in this app still trusts the client's own isBanker; the close is
// carved out because it is the single act that fixes a winner and moves the
// money (D14), so "whoever holds the room code can hammer someone else's
// auction" is a different proposition from "whoever holds the room code can
// move their own money".
func closeAuthorized(closer models.Player) error {
	if !closer.IsActive {
		// Unreachable today: the read that produces closer filters on
		// isActive: true. Kept so the function is total, and so the rule
		// survives a later change to that filter rather than quietly leaving
		// with it.
		return errors.New("a removed player cannot close a lot")
	}
	if !closer.IsBanker {
		return errors.New("only the Banker can close a lot")
	}
	return nil
}

// lotOpenNotification is the sentence a lot opening raises. The $0 is D15 and
// is worth saying out loud: it is what makes "nobody wants this deed" a lot
// that closes with no bid rather than an unmet reserve nobody can see.
func lotOpenNotification(lotName string) string {
	return fmt.Sprintf("%s is up for auction. Bidding starts at $0.", lotName)
}

// bidNotification is the sentence a bid raises. It is broadcast but never
// written to the event history: a lot can take twenty bids and the log is a
// record of what happened to the money, not of the bidding. What is live rather
// than historical lives on the Room document (D18).
func bidNotification(bidderName, lotName string, amount int) string {
	return fmt.Sprintf("%s bid $%d on %s.", bidderName, amount, lotName)
}

// lotClosedNotification is the one string a close broadcasts and the one
// CreateEventHistory stores for it.
//
// Every arm returns text, including the default that should not occur, for the
// reason kickNotification's does: Broadcast refuses a payload whose
// notification is present and empty, so an arm that forgot to set one would be
// dropped silently to the room and logged only on the VM - a lot that closed,
// and moved a deed and a balance, that nobody saw close.
//
// The trailing clause is always one of the two. A player needs to know whether
// to keep watching, and the alternative - saying nothing when the auction ends
// - makes the last lot indistinguishable from a lot whose next deed never
// opened.
func lotClosedNotification(outcome closeOutcome, lotName, winnerName string, price int, nextLotName string) string {
	var text string

	switch outcome {
	case lotSold:
		text = fmt.Sprintf("%s won %s for $%d.", winnerName, lotName, price)
	case lotNoBid:
		text = fmt.Sprintf("Nobody bid on %s. It goes back to the Bank.", lotName)
	case lotCannotPay:
		text = fmt.Sprintf("%s couldn't cover the $%d bid, so %s goes back to the Bank.", winnerName, price, lotName)
	case lotWinnerGone:
		text = fmt.Sprintf("%s is no longer in the game, so %s goes back to the Bank.", winnerName, lotName)
	default:
		text = fmt.Sprintf("%s is no longer up for auction.", lotName)
	}

	if nextLotName != "" {
		text += fmt.Sprintf(" %s is up next.", nextLotName)
	} else {
		text += " That's the last of them."
	}

	return text
}

// handlePlaceBid raises the high bid on the open lot.
//
// Every payload rejection is above the first Mongo call, and every rule that
// needs the auction's state is in bidRejection, which takes read values. What
// is left here is the reads and one conditional write.
func (rm *RoomManager) handlePlaceBid(client *Client, message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	roomIdStr, ok := payload["roomId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for roomId")
	}
	roomObjID, err := primitive.ObjectIDFromHex(roomIdStr)
	if err != nil {
		return fmt.Errorf("invalid room ID: %w", err)
	}

	propertyIdStr, ok := payload["propertyId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for propertyId")
	}
	propObjID, err := primitive.ObjectIDFromHex(propertyIdStr)
	if err != nil {
		return fmt.Errorf("invalid property ID: %w", err)
	}

	bidderIdStr, ok := payload["bidderId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for bidderId")
	}
	bidderObjID, err := primitive.ObjectIDFromHex(bidderIdStr)
	if err != nil {
		return fmt.Errorf("invalid bidder ID: %w", err)
	}

	// A JSON number, like handlePropertyPurchase's price, and unlike
	// freeParking's string amount - the string there is the shape of the keypad
	// that produces it, not a contract worth copying into a new handler.
	amountFloat, ok := payload["amount"].(float64)
	if !ok {
		return errors.New("invalid payload: expected number for amount")
	}
	amount, err := wholeDollars(amountFloat)
	if err != nil {
		return err
	}
	if amount < 1 {
		// A lot opens at $0 with no bidder, so the smallest bid that can exist
		// is $1 (D15). It is rejected here rather than in bidRejection because
		// it is a fact about the payload: it does not depend on any state.
		return errors.New("a bid has to be at least $1")
	}

	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---

	var room models.Room
	err = config.DB.Collection("Room").FindOne(context.Background(), bson.M{"_id": roomObjID}).Decode(&room)
	if err != nil {
		return fmt.Errorf("failed to find the room: %w", err)
	}
	if room.Auction == nil {
		return errors.New("there is no auction running in this room")
	}

	// Scoped to this room and to active players: D7 opens the auction to every
	// remaining player in the room, which is the banker included and a removed
	// player excluded. A bidder from another room would otherwise be able to
	// spend their own room's money here.
	var bidder models.Player
	err = config.DB.Collection("Player").FindOne(context.Background(), bson.M{
		"_id":      bidderObjID,
		"roomId":   roomObjID,
		"isActive": true,
	}).Decode(&bidder)
	if err != nil {
		return fmt.Errorf("failed to find the bidder: %w", err)
	}

	if err := bidRejection(*room.Auction, propObjID, bidderObjID, amount, bidder.Balance); err != nil {
		return err
	}

	var lot models.Property
	err = config.DB.Collection("Property").FindOne(context.Background(), bson.M{
		"_id":    propObjID,
		"roomId": roomObjID,
	}).Decode(&lot)
	if err != nil {
		return fmt.Errorf("failed to find the lot: %w", err)
	}

	// The raise, as one conditional update rather than a write that trusts the
	// read above it. "auction.highBid < amount" is the same comparison
	// bidRejection already made, restated as a filter, because between that
	// read and this write another bidder's frame can have raised the bid on
	// another goroutine: a plain $set would then overwrite a higher bid with a
	// lower one and hand the lot to the wrong player, with green gates and no
	// error anywhere. Mongo applies one document's update atomically, so of two
	// bids that arrive together exactly one matches.
	//
	// The propertyId clause is what stops a bid that arrives after its lot
	// closed. A close advances auction.propertyId, so the filter no longer
	// matches and the raise cannot land on the next deed.
	result, err := config.DB.Collection("Room").UpdateOne(
		context.Background(),
		bson.M{
			"_id":                roomObjID,
			"auction.propertyId": propObjID,
			"auction.highBid":    bson.M{"$lt": amount},
		},
		bson.M{"$set": bson.M{
			"auction.highBid":      amount,
			"auction.highBidderId": bidderObjID,
		}},
	)
	if err != nil {
		return fmt.Errorf("failed to place the bid: %w", err)
	}
	if result.MatchedCount == 0 {
		// Two different things land here and the bidder cannot tell them apart
		// from the state they can see, so the sentence says both: either
		// someone outbid them between the read above and this write, or the
		// banker closed the lot in that same window.
		return errors.New("that bid did not land - the lot was outbid or closed first")
	}

	notification := bidNotification(bidder.Name, lot.Name, amount)
	rm.Broadcast(client.Room, Message{
		Type: "BID_PLACED",
		Payload: map[string]interface{}{
			"notification": notification,
			"propertyId":   propertyIdStr,
			"bidderId":     bidderIdStr,
			"amount":       amount,
		},
	})

	return nil
}

// handleCloseAuction is the banker's hammer: it closes the open lot, settles
// it, and advances to the next deed or ends the auction (D14, D16).
//
// The settlement is one session.WithTransaction, following freeParking and
// handleKickPlayer. controllers.PurchaseProperty was the counter-example here
// until 2026-09-22 - two bare writes with no session and no floor - and it is
// transacted and floored itself now, for exactly the reason that follows. A
// close that hands over the deed and then fails to charge the winner is a free
// property; one that charges and fails to hand over is money for nothing.
// Neither half may land alone.
func (rm *RoomManager) handleCloseAuction(client *Client, message Message) error {
	payload, ok := message.Payload.(map[string]interface{})
	if !ok {
		return errors.New("invalid payload format")
	}

	roomIdStr, ok := payload["roomId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for roomId")
	}
	roomObjID, err := primitive.ObjectIDFromHex(roomIdStr)
	if err != nil {
		return fmt.Errorf("invalid room ID: %w", err)
	}

	closerIdStr, ok := payload["playerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for playerId")
	}
	closerObjID, err := primitive.ObjectIDFromHex(closerIdStr)
	if err != nil {
		return fmt.Errorf("invalid player ID: %w", err)
	}

	// The lot the banker is looking at, and it is required.
	//
	// The close could act on whatever lot happens to be open and take no
	// propertyId at all, and that is wrong in a way worth spelling out, because
	// it is subtle and it is the one this handler was written with first. Two
	// CLOSE_AUCTION frames - a double tap, or one frame retried by a flaky
	// reconnect - both read the same open lot, both try the pinned write, and
	// one of them loses the write conflict. WithTransaction then retries the
	// loser's callback, which re-reads and finds the NEXT lot open, pins on
	// that, and settles it. One banker, one intention, two deeds sold, and
	// nothing anywhere reports a problem.
	//
	// Naming the lot makes the close idempotent per lot: the retry finds its
	// named lot is no longer the open one and is refused. It is a precondition,
	// not a selector - the server still settles only the lot that is actually
	// open - which is also why a banker whose screen is stale is told so rather
	// than hammering a deed they were not looking at. handlePlaceBid already
	// requires the bidder to name the lot, for exactly this reason.
	lotIdStr, ok := payload["propertyId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for propertyId")
	}
	lotObjID, err := primitive.ObjectIDFromHex(lotIdStr)
	if err != nil {
		return fmt.Errorf("invalid property ID: %w", err)
	}

	// And whose estate, because a property id alone does not identify an
	// auction lot - it identifies a deed, and the same deed can be the open lot
	// of two different auctions.
	//
	// The trace, from this phase's Deep review: auction 1's last lot L closes
	// sold to W; W is then kicked with AUCTION; L is W's lowest-index deed, so
	// auction 2 opens on L at $0. A CLOSE frame naming L, queued on a device
	// that missed both the PLAYER_KICKED and AUCTION_STARTED broadcasts, then
	// passes a propertyId-only check and hammers auction 2's first lot to the
	// Bank with nobody able to bid. Narrow, and silent, which is the
	// combination this phase exists to not ship.
	//
	// (kickedPlayerId, propertyId) is a unique auction-lot identity, because a
	// player cannot be kicked twice - the kick's own mark-gone write is
	// filtered on isActive: true.
	kickedIdStr, ok := payload["kickedPlayerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for kickedPlayerId")
	}
	kickedObjID, err := primitive.ObjectIDFromHex(kickedIdStr)
	if err != nil {
		return fmt.Errorf("invalid kicked player ID: %w", err)
	}

	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---

	playerColl := config.DB.Collection("Player")
	propColl := config.DB.Collection("Property")
	roomColl := config.DB.Collection("Room")

	var closer models.Player
	err = playerColl.FindOne(context.Background(), bson.M{
		"_id":      closerObjID,
		"roomId":   roomObjID,
		"isActive": true,
	}).Decode(&closer)
	if err != nil {
		return fmt.Errorf("failed to find the player closing the lot: %w", err)
	}
	if err := closeAuthorized(closer); err != nil {
		return err
	}

	var notification string
	var closedLotID primitive.ObjectID
	var winnerID *primitive.ObjectID
	var price int

	session, err := config.DB.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
		// WithTransaction re-runs this callback on a write conflict, so every
		// value it decides from is read inside it and every captured variable
		// is reset at the top. Re-reading is not caution here: the point is
		// that a bid committed by another goroutine between attempts has to be
		// seen by the retry rather than settled around.
		notification, closedLotID, winnerID, price = "", primitive.NilObjectID, nil, 0

		var room models.Room
		if err := roomColl.FindOne(ctx, bson.M{"_id": roomObjID}).Decode(&room); err != nil {
			return nil, fmt.Errorf("failed to find the room: %w", err)
		}
		if room.Auction == nil {
			return nil, errors.New("there is no auction running in this room")
		}
		auction := *room.Auction
		if auction.PropertyID != lotObjID || auction.KickedPlayerID != kickedObjID {
			// Another close already advanced past this lot, or the banker's
			// screen is behind, or this frame belongs to an auction that has
			// since ended. All of them are "you are not looking at the lot that
			// is open", and none of them should settle anything.
			return nil, errors.New("that is not the lot that is open - the auction has moved on")
		}
		closedLotID = auction.PropertyID

		var lot models.Property
		if err := propColl.FindOne(ctx, bson.M{"_id": auction.PropertyID, "roomId": roomObjID}).Decode(&lot); err != nil {
			return nil, fmt.Errorf("failed to find the open lot: %w", err)
		}

		// Read without an isActive clause, so a high bidder who has been
		// removed since bidding still yields a name for the sentence; whether
		// they still count is the IsActive check below, not the absence of a
		// document. A player document is never deleted (D11), which is what
		// makes that distinction available at all.
		var winner models.Player
		winnerFound := false
		if auction.HighBidderID != nil {
			readErr := playerColl.FindOne(ctx, bson.M{
				"_id":    *auction.HighBidderID,
				"roomId": roomObjID,
			}).Decode(&winner)

			var fatal error
			winnerFound, fatal = winnerStatus(readErr, winner)
			if fatal != nil {
				return nil, fatal
			}
		}

		outcome := outcomeForClose(auction, winnerFound, winner.Balance)

		// The claim on the lot, pinned to exactly the state the reads above
		// saw.
		//
		// Be clear about what is actually protecting this, because the obvious
		// reading is wrong and the Deep review of this phase caught it. Every
		// read in this callback uses ctx, so they all come from one snapshot,
		// and the pin is built from that snapshot - which means the pin cannot
		// fail to match on the attempt that read it. What stops a bid landing
		// mid-close is not the pin: it is that a transactional write to a
		// document modified since the snapshot raises a WriteConflict, which
		// carries a transient label, which makes WithTransaction abort and
		// re-run this whole callback against fresh state.
		//
		// So a bid that commits before this close commits WINS - the retry
		// re-reads it and settles to that bidder at that price. That is the
		// right outcome, because the bid really did arrive before the hammer,
		// and it is the opposite of what an earlier version of this comment
		// claimed. The MatchedCount check below is kept as defence against a
		// future edit that reads outside the transaction, not because it fires
		// today.
		//
		// The lot precondition above is the load-bearing part, and it is
		// load-bearing precisely because of this retry: without it, a second
		// close frame whose first attempt lost the conflict would re-read, find
		// the NEXT lot open, and settle that one instead.
		next := advanceAuction(auction)
		pin := bson.M{
			"_id":                  roomObjID,
			"auction.propertyId":   auction.PropertyID,
			"auction.highBid":      auction.HighBid,
			"auction.highBidderId": nil,
		}
		if auction.HighBidderID != nil {
			pin["auction.highBidderId"] = *auction.HighBidderID
		}
		update := bson.M{"$unset": bson.M{"auction": ""}}
		if next != nil {
			update = bson.M{"$set": bson.M{"auction": *next}}
		}
		result, err := roomColl.UpdateOne(ctx, pin, update)
		if err != nil {
			return nil, fmt.Errorf("failed to close the lot: %w", err)
		}
		if result.MatchedCount == 0 {
			// Unreachable while every read above is on ctx, per the comment
			// there. If it ever fires, the auction changed underneath a read
			// that was not in the transaction, and settling on it would be
			// guessing.
			return nil, errors.New("the auction changed while that lot was closing - check the high bid and close it again")
		}

		// The deed, razed and unmortgaged on every arm including the sale.
		//
		// The unsold arms are D16 plus D13's raze - the same write
		// handleKickPlayer makes for the BANK disposition, and for the same
		// reason: manager.HandlePropertySaleMortgage's SELL case at
		// manager/propertyManager.go:53 sets only {playerId: nil, isMortgaged:
		// false}, so without the raze a returned hotel-bearing deed reappears
		// in GetAvailableProperties at its face price and the next buyer
		// inherits the development for free. SELL is left alone deliberately
		// (D13); the two paths disagree on purpose and this comment is the
		// record of it.
		//
		// The sold arm rases too, which D13 does not literally cover and which
		// is Zach's call, 2026-09-17. D13's own reasoning applies unchanged: if
		// a sold lot kept its houses, bidding $1 on an unwanted hotel deed
		// would be strictly better than letting it go unsold, because the
		// unsold path razes and the sold path would not. That asymmetry is the
		// exact shape of hole D13 exists to close. It is also the real rule -
		// buildings go back to the bank when a player is out of the game - so
		// what is auctioned is the deed, not the development on it.
		deed := bson.M{"playerId": nil, "isMortgaged": false, "developmentLevel": 0}
		if outcome == lotSold {
			deed["playerId"] = winner.ID
		}
		if _, err := propColl.UpdateOne(ctx,
			bson.M{"_id": auction.PropertyID, "roomId": roomObjID},
			bson.M{"$set": deed},
		); err != nil {
			return nil, fmt.Errorf("failed to hand over the deed: %w", err)
		}

		if outcome == lotSold {
			if _, err := playerColl.UpdateOne(ctx,
				bson.M{"_id": winner.ID},
				bson.M{"$inc": bson.M{"balance": -auction.HighBid}},
			); err != nil {
				return nil, fmt.Errorf("failed to charge the winning bid: %w", err)
			}
			winnerID = &winner.ID
			price = auction.HighBid
		}

		nextLotName := ""
		if next != nil {
			var nextLot models.Property
			if err := propColl.FindOne(ctx, bson.M{
				"_id":    next.PropertyID,
				"roomId": roomObjID,
			}).Decode(&nextLot); err != nil {
				return nil, fmt.Errorf("failed to find the next lot: %w", err)
			}
			nextLotName = nextLot.Name
		}

		notification = lotClosedNotification(outcome, lot.Name, winner.Name, auction.HighBid, nextLotName)
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	rm.CreateEventHistory(notification, roomObjID)

	closedPayload := map[string]interface{}{
		"notification": notification,
		"propertyId":   closedLotID.Hex(),
	}
	if winnerID != nil {
		closedPayload["winnerId"] = winnerID.Hex()
		closedPayload["amount"] = price
	}
	rm.Broadcast(client.Room, Message{
		Type:    "AUCTION_LOT_CLOSED",
		Payload: closedPayload,
	})

	return nil
}

// eventTypeFor picks the {colour, emoji} pair an event-history row is drawn
// with, by matching substrings of the notification prose. It is a separate
// function from CreateEventHistory only so that it can be tested:
// CreateEventHistory ends in an InsertOne, so calling it in a test binary
// panics on the nil config.DB before anything about the classification can be
// observed.
//
// Prose matching makes the arms order-dependent, and that is a real hazard
// rather than a stylistic one - see the kick arm's comment.
func eventTypeFor(notification string) []string {
	switch {
	// A settled trade, before every other arm. tradeNotification (offers.go)
	// names two players and up to 28 deeds, and a name like "Warehouse" or a
	// deed like "Park Place" would otherwise be matched by a substring key
	// further down ("house", "sent") and drawn as something it is not. "traded"
	// is the one word every arm of that copy carries; if the copy changes, this
	// key changes with it - TestTradeRowsTakeTheTradeIcon is the guard.
	case strings.Contains(notification, " traded "):
		return []string{"#0ea5e9", "🤝"}
	// Next, deliberately. This notification has "Banker" as its subject, so
	// left further down it would fall into the bank arm by substring and be
	// drawn identically to a balance change - which is the outcome
	// PLAN.md's icon dial exists to avoid. "removed" alone is not a safe key
	// either: handleBankTransaction writes "Banker has removed $100 from X's
	// balance". "from the game" is the phrase that distinguishes them. If the
	// kick copy in kickNotification changes, this key changes with it.
	case strings.Contains(notification, "from the game"):
		return []string{"#dc2626", "🚫"}
	case strings.Contains(notification, "purchased"), strings.Contains(notification, "selling"):
		return []string{"#10b981", "🏠"}
	case strings.Contains(notification, "Free Parking"):
		return []string{"#f59e0b", "🅿️"}
	case strings.Contains(notification, "sent"):
		return []string{"#3b82f6", "💸"}
	case strings.Contains(notification, "Banker"):
		return []string{"#6366f1", "🏦"}
	case strings.Contains(notification, "mortgag"):
		return []string{"#ef4444", "📄"}
	case strings.Contains(notification, "house") || strings.Contains(notification, "hotels"):
		return []string{"#8b5cf6", "🏗️"}
	default:
		return []string{"#6b7280", "ℹ️"}
	}
}

func (rm *RoomManager) CreateEventHistory(notification string, roomId primitive.ObjectID) error {
	eventType := eventTypeFor(notification)

	eventHistory := models.EventHistory{
		ID:        primitive.NewObjectID(),
		TimeStamp: time.Now(),
		Event:     notification,
		RoomID:    roomId,
		EventType: eventType,
	}

	_, err := config.DB.Collection("EventHistory").InsertOne(context.Background(), eventHistory)
	if err != nil {
		return fmt.Errorf("failed to insert event history: %w", err)
	}
	return nil
}
