package websocket

import (
	"context"
	"errors"
	"fmt"
	"log"
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

	// The dead clients are collected here and deleted below, under the write
	// lock, because RLock does not exclude another RLock: two broadcasts to the
	// same room would otherwise delete from one Go map at once, and the runtime
	// answers that with fatal("concurrent map writes") - not a panic, so Gin's
	// Recovery does not catch it and the one backend process dies. The fan-out
	// keeps the read lock so broadcasts to different rooms still overlap while
	// each does its per-client network I/O.
	var dead []*Client

	rm.mu.RLock()
	if clients, ok := rm.clients[room]; ok {
		for client := range clients {
			// Client.WriteJSON, not client.Conn.WriteJSON: rm.mu orders access
			// to the room map, not to any one conn's writer, and two broadcasts
			// to the same room both hold RLock while writing to the same conns.
			// The per-client lock is what keeps one goroutine at a time inside
			// gorilla's writer - see the Client doc comment in types.go.
			err := client.WriteJSON(message)
			if err != nil {
				client.Conn.Close()
				dead = append(dead, client)
			}
		}
	}
	rm.mu.RUnlock()

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

	player, err := controllers.GetPlayer(playerObjID)
	if err != nil {
		return fmt.Errorf("failed to get player details: %w", err)
	}

	var notification string

	session, err := config.DB.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
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
			rm.CreateEventHistory(notification, roomObjID)
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
			rm.CreateEventHistory(notification, roomObjID)
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

	rm.Broadcast(client.Room, Message{
		Type: "FREE_PARKING",
		Payload: map[string]interface{}{
			"notification": notification,
		},
	})
	log.Printf("Free parking update broadcast complete for room: %s", client.Room)

	return nil
}

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
	property, buyer, err := controllers.GetPropertyAndBuyer(propertyID, buyerID)
	if err != nil {
		log.Printf("Failed to get property or buyer details: %v", err)
		return err
	}

	purchaseErr := controllers.PurchaseProperty(propertyID, buyerID, price)
	if purchaseErr != nil {
		log.Printf("Property update failed: %v", purchaseErr)
		return purchaseErr
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

	switch manageType {
	case "HOUSES":
		err = manager.HandleHouseManagement(roomObjID, manageType, properties)
	case "MORTGAGE", "UNMORTGAGE", "SELL":
		err = manager.HandlePropertySaleMortgage(roomObjID, manageType, properties)
	default:
		return fmt.Errorf("invalid management type: %s", manageType)
	}

	if err != nil {
		return err
	}

	err = manager.UpdatePlayerBalance(playerID, amount)
	if err != nil {
		return err
	}

	idValue, ok := payload["playerId"]
	if !ok || idValue == nil {
		return fmt.Errorf("toPlayerId is missing or nil")
	}

	idStr, ok := idValue.(string)
	if !ok {
		return fmt.Errorf("toPlayerId is not a string")
	}

	targetPlayerID, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		return fmt.Errorf("invalid target player ID: %w", err)
	}

	targetPlayer, err := controllers.GetPlayer(targetPlayerID)
	if err != nil {
		return fmt.Errorf("failed to get target player details: %w", err)
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
		targetPlayer.Name,
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
	// test. AUCTION is a real disposition in the design (D2) and is Phase 3;
	// until it exists it must be refused here rather than fall through to a
	// kick that disposes of nothing.
	switch disposition {
	case "BANK", "FREEZE":
		// valid - the transaction below switches on these same two values
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

	notification := kickNotification(target.Name, disposition, successor.Name)

	// One transaction, following freeParking above - the only other
	// multi-document write in this app - and deliberately not
	// controllers.PurchaseProperty, which is two bare writes with no session.
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
		// FREEZE writes nothing to any property (D4). The deeds stay against
		// the player, houses intact, which is why the document has to survive.

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
// is what eventTypeFor keys the kick's icon on.
func kickNotification(targetName, disposition, successorName string) string {
	var text string

	switch disposition {
	case "BANK":
		text = fmt.Sprintf("Banker removed %s from the game. Their properties returned to the Bank.", targetName)
	case "FREEZE":
		text = fmt.Sprintf("Banker removed %s from the game. Their properties stay where they are.", targetName)
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
	// First, deliberately. This notification has "Banker" as its subject, so
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
