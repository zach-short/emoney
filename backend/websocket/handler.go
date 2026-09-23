package websocket

import (
	"fmt"
	"log"
	"net/http"
	"time"

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

	// The read side's deadline, which it never had: without one, ReadJSON
	// below waits forever on a peer that went silent, and that peer keeps its
	// seat. The read loop arms it before every read, and every pong pushes it
	// out again. pingLoop (started below) is what asks for the pongs, so a
	// live browser - which answers pings by itself - never reaches it, and a
	// vanished phone does, pongWait after the last thing it sent. ReadJSON
	// then fails with a timeout and takes the same exit as any other error:
	// the disconnect defer. See pongWait in types.go for the numbers.
	//
	// gorilla calls the pong handler from inside ReadJSON, on this goroutine,
	// while it steps over control frames on its way to the next message.
	// SetReadDeadline's result is unchecked here and in the loop: it fails
	// only on a closed conn, and then ReadJSON fails too.
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	client := &Client{
		Conn:       conn,
		Room:       roomCode,
		PlayerID:   "",
		PlayerName: "",
	}

	Manager.AddClient(client)

	// One ping loop per connection, started only once the manager knows the
	// client, and stopped and joined by the defer below. Why a stop and a join
	// rather than letting it die on its next failed ping is on pingLoop in
	// types.go.
	stopPinging := make(chan struct{})
	pingerDone := make(chan struct{})
	period := pingPeriod
	go func() {
		defer close(pingerDone)
		client.pingLoop(period, stopPinging)
	}()

	defer func() {
		Manager.RemoveClient(client)
		conn.Close()
		// After the close, so a ping in flight fails at once rather than
		// waiting out writeWait, and before PLAYER_LEFT, so that anything
		// that sees PLAYER_LEFT knows this connection's pinger is gone too.
		close(stopPinging)
		<-pingerDone

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
		// Armed before every read, not only by a pong, because the pong
		// handler runs only while this goroutine is inside ReadJSON - and it
		// is not, for as long as it is handling the message before. That can
		// be long: a handler's Mongo calls, or a Broadcast waiting out
		// writeWait on another player's stalled conn. A pong that lands
		// meanwhile waits unread in the socket while the deadline the last
		// pong set runs out, and once a deadline has passed Go fails the next
		// read at once, before it looks at the socket (internal/poll,
		// FD.Read's prepareRead). A healthy player would be dropped for this
		// goroutine's own slowness. Re-arming here gives every read a full
		// pongWait from the moment this goroutine is ready to read again. A
		// ghost is unaffected: it sends no messages, so this goroutine sits in
		// one ReadJSON and only a pong could renew it.
		//
		// At the top of the loop rather than after the switch, because the
		// JOIN case leaves by continue on its error paths - after a Mongo
		// read, the slow case this exists for - and would skip a re-arm
		// placed at the bottom. Found by the independent audit of board row
		// 31, 2026-09-23, which measured healthy peers dropped with
		// i/o timeout when a handler outlasted the six seconds between
		// pingPeriod and pongWait. Pinned by
		// TestHandleWebSocketKeepsAPeerWhoseHandlerWasBusyPastTheDeadline
		// in read_deadline_test.go.
		conn.SetReadDeadline(time.Now().Add(pongWait))
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
		// The two trade cases (offers.go). Their successes are the first
		// messages in this package that go to one player rather than the
		// room - RoomManager.SendTo - and their ERROR replies go back on the
		// sender's own conn like every case above. For RESPOND_OFFER that
		// reply carries the sentence that makes a refused accept make sense:
		// "Alice doesn't own Baltic Avenue" is the whole of what a player
		// learns when two offers named the same deed and the other one
		// settled first.
		case "CREATE_OFFER":
			if err := Manager.handleCreateOffer(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		case "RESPOND_OFFER":
			if err := Manager.handleRespondOffer(client, message); err != nil {
				client.WriteJSON(Message{
					Type:    "ERROR",
					Payload: err.Error(),
				})
			}
		}
	}
}
