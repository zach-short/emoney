package websocket

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/zachmshort/emoney-backend/controllers"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		switch origin {
		case "http://localhost:3000",
			"https://emoney.club",
			"https://www.emoney.club":
			return true
		default:
			return false
		}
	},
}
var (
	Manager = NewRoomManager()
)

func HandleWebSocket(c *gin.Context) {
	roomCode := c.Param("code")

	if roomCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Room code required"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &Client{
		Conn:       conn,
		Room:       roomCode,
		PlayerID:   "",
		PlayerName: "",
	}

	Manager.AddClient(client)

	defer func() {
		Manager.RemoveClient(client)
		conn.Close()

		// Every conn carries PlayerID "" from the upgrade until its JOIN
		// succeeds (SeatClient's comment in websocketManager.go), and this
		// goroutine is the only writer to its own client.PlayerID, so
		// reading it unlocked here is the same safe case SeatClient itself
		// documents - not a race. The guard exists because an unseated conn
		// has no name to put in the sentence below: without it, PLAYER_LEFT
		// goes out with notification " has left the game", a leading space
		// and no subject. Broadcast's own empty-notification guard
		// (emptyNotification, websocketManager.go) does not catch this -
		// the string itself is not empty, only the name inside it is. A
		// kicked player's refused reconnect (the !player.IsActive check in
		// the JOIN case below) is what makes this the common path rather
		// than a rare one: their browser reconnects, JOIN is refused, and
		// the conn closes with PlayerID still "".
		if client.PlayerID != "" {
			Manager.Broadcast(roomCode, Message{
				Type: "PLAYER_LEFT",
				Payload: map[string]string{
					"playerId":     client.PlayerID,
					"notification": fmt.Sprintf("%s has left the game", client.PlayerName),
				},
			})
		}
	}()
	for {
		var message Message
		err := conn.ReadJSON(&message)
		if err != nil {
			break
		}

		switch message.Type {
		case "JOIN":
			if payload, ok := message.Payload.(map[string]interface{}); ok {
				if playerId, ok := payload["playerId"].(string); ok {
					playerObjID, err := primitive.ObjectIDFromHex(playerId)
					if err != nil {
						log.Printf("Error converting player ID: %v", err)
						continue
					}

					player, err := controllers.GetPlayer(playerObjID)
					if err != nil {
						log.Printf("Error fetching player: %v", err)
						continue
					}

					// A kicked player is marked isActive:false, not deleted, so
					// this read still succeeds for them. Refusing here is what
					// stops the kicked browser's one-second reconnect
					// (frontend/app/room/[code]/page.tsx:219-223) from re-seating
					// them: without it the kick is a no-op with green gates -
					// the room is written correctly and the player walks back in
					// a second later. The other way back in is the Join screen,
					// which goes through GetPlayerDetails and is already bolted
					// (controllers/playerControllers.go).
					//
					// Nothing is sent back and the socket is left open: the
					// kicked player is told nothing (design D6), and their own
					// room fetch failing is what their client reacts to.
					if !player.IsActive {
						log.Printf("JOIN refused for removed player %s in room %s", playerId, roomCode)
						continue
					}

					// Seated through the manager rather than assigned directly:
					// CloseClientByPlayerID scans these fields from another
					// player's goroutine. See SeatClient in websocketManager.go.
					Manager.SeatClient(client, playerId, player.Name)

					Manager.Broadcast(roomCode, Message{
						Type: "PLAYER_JOINED",
						Payload: map[string]interface{}{
							"playerId":     client.PlayerID,
							"playerName":   client.PlayerName,
							"notification": fmt.Sprintf("%s has joined the game", client.PlayerName),
						},
					})
				}
			}
		// Every ERROR reply below goes back on this player's own conn through
		// client.WriteJSON rather than conn.WriteJSON, because another player's
		// action can be fanning out to this same conn from their reader
		// goroutine at the same instant. Only the per-client lock orders the
		// two - see the Client doc comment in types.go.
		case "PURCHASE_PROPERTY":
			if err := Manager.handlePropertyPurchase(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		case "FREE_PARKING":
			if err := Manager.freeParking(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		case "BANKER_TRANSACTION":
			if err := Manager.handleBankTransaction(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		case "TRANSFER":
			if err := Manager.handleTransfer(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		case "MANAGE_PROPERTIES":
			if err := Manager.handleManageProperties(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		case "KICK_PLAYER":
			if err := Manager.handleKickPlayer(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		// The two auction cases. Both reply with ERROR on the sender's own
		// conn like every case above, and for PLACE_BID that reply is the
		// feature rather than a formality: a bid refused for insufficient
		// funds (D17) is the first rejection a player will meet that the rest
		// of this app would have allowed, and the browser's ERROR toast is the
		// only thing that tells them it happened.
		case "PLACE_BID":
			if err := Manager.handlePlaceBid(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		case "CLOSE_AUCTION":
			if err := Manager.handleCloseAuction(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		}
	}
}
