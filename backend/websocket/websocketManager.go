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

func (rm *RoomManager) Broadcast(room string, message Message) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	if clients, ok := rm.clients[room]; ok {
		for client := range clients {
			err := client.Conn.WriteJSON(message)
			if err != nil {
				client.Conn.Close()
				delete(clients, client)
			}
		}
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

func (rm *RoomManager) CreateEventHistory(notification string, roomId primitive.ObjectID) error {
	var eventType []string

	switch {
	case strings.Contains(notification, "purchased"), strings.Contains(notification, "selling"):
		eventType = []string{"#10b981", "🏠"}
	case strings.Contains(notification, "Free Parking"):
		eventType = []string{"#f59e0b", "🅿️"}
	case strings.Contains(notification, "sent"):
		eventType = []string{"#3b82f6", "💸"}
	case strings.Contains(notification, "Banker"):
		eventType = []string{"#6366f1", "🏦"}
	case strings.Contains(notification, "mortgag"):
		eventType = []string{"#ef4444", "📄"}
	case strings.Contains(notification, "house") || strings.Contains(notification, "hotels"):
		eventType = []string{"#8b5cf6", "🏗️"}
	default:
		eventType = []string{"#6b7280", "ℹ️"}
	}
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
