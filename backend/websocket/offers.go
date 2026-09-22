package websocket

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/zachmshort/emoney-backend/config"
	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// --- peer-to-peer trades (TRIAGE.md D5) ---
//
// Two inbound message types and one stored document. CREATE_OFFER writes a
// models.Offer with status PENDING and sends it to exactly one player;
// RESPOND_OFFER answers it - ACCEPT settles the trade, DENY refuses it, and
// WITHDRAW is the sender taking it back. A counter is not a third response: it
// is a CREATE_OFFER the other way carrying counterOf, and the create marks the
// original COUNTERED in the same transaction as the insert. One protocol, one
// screen, no fork.
//
// Trades are peer-to-peer and the banker is not in the loop (D5). A trade is
// not a banker action and must not require isBanker (D2's enforcement is
// elsewhere and does not apply here). Nothing here is reversible by the app;
// the banker corrects a wrong trade by hand with the powers that exist.
//
// The correctness argument is handleCloseAuction's, restated for two players
// and up to 28 deeds: never decide from a value you read and then write as
// though it were still true. The settlement reads the offer, both players and
// every named deed INSIDE one session.WithTransaction on the session's ctx,
// checks every precondition against that one snapshot, and writes pinned to
// it. A concurrent write to any of those documents - a second accept, a
// transfer that drains a balance, another trade that moves the same deed -
// raises a WriteConflict, WithTransaction re-runs the callback against fresh
// state, and the re-read is what refuses. That is why two offers naming the
// same deed cannot both settle: the second accept's retry finds the deed no
// longer owned by the player who offered it and says so.
//
// The rules live in pure functions - offerRejection, acceptRejection,
// tradePreconditions - and the copy in pure functions beside them, because
// config.DB is a nil *mongo.Database in a test binary, so a rule written below
// a handler's first Mongo call is a rule no test in this repo can reach.

// maxNoteLength is the cap on an offer's note, in runes. 280 is deliberate: a
// note is a deal two players would say out loud at the table - "immunity on
// the browns for three turns", "I won't build on the reds until you pass GO"
// - and a cap that forces it to stay sayable is the point, because the app
// only writes the note down and never enforces it. The browser's textarea
// carries the same number as maxLength, counted in UTF-16 units there and in
// runes here, so anything the browser accepts the server accepts too.
const maxNoteLength = 280

// parseObjectIDs turns a JSON array of hex strings into ObjectIDs, refusing a
// non-array, a non-string element, a malformed id, and a repeat. An absent or
// null field is an empty list, which is the shape of a side with no deeds.
func parseObjectIDs(raw interface{}, label string) ([]primitive.ObjectID, error) {
	ids := make([]primitive.ObjectID, 0)
	if raw == nil {
		return ids, nil
	}
	items, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid payload: expected an array for %s", label)
	}
	seen := make(map[primitive.ObjectID]bool, len(items))
	for _, item := range items {
		hex, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("invalid payload: expected strings in %s", label)
		}
		id, err := primitive.ObjectIDFromHex(hex)
		if err != nil {
			return nil, fmt.Errorf("invalid property ID in %s: %w", label, err)
		}
		if seen[id] {
			return nil, fmt.Errorf("a property is listed twice in %s", label)
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids, nil
}

// parseTradeSide reads one half of an offer - {properties: [...], amount: n} -
// from the payload. Absent means empty: an offer of cash alone carries no
// properties key and an offer of deeds alone carries amount 0, and both are
// ordinary. Present-but-wrong-type is refused, because a payload field that
// is there and unreadable is the bug family the 19 unchecked assertions in
// websocketManager.go were (HANDOFF.md, Known facts).
//
// The amount is a JSON number, like a bid and a purchase price, and goes
// through wholeDollars for the reason a bid does: a silently truncated 120.9
// is a trade the player did not make.
func parseTradeSide(raw interface{}, label string) (models.TradeSide, error) {
	side := models.TradeSide{Properties: make([]primitive.ObjectID, 0)}
	if raw == nil {
		return side, nil
	}
	fields, ok := raw.(map[string]interface{})
	if !ok {
		return side, fmt.Errorf("invalid payload: expected an object for %s", label)
	}

	ids, err := parseObjectIDs(fields["properties"], label+".properties")
	if err != nil {
		return side, err
	}
	side.Properties = ids

	if rawAmount, present := fields["amount"]; present && rawAmount != nil {
		amountFloat, ok := rawAmount.(float64)
		if !ok {
			return side, fmt.Errorf("invalid payload: expected number for %s.amount", label)
		}
		amount, err := wholeDollars(amountFloat)
		if err != nil {
			return side, err
		}
		if amount < 0 {
			return side, fmt.Errorf("%s.amount can't be negative", label)
		}
		side.Amount = amount
	}
	return side, nil
}

// sideIsEmpty reports whether a trade side hands over nothing at all.
func sideIsEmpty(side models.TradeSide) bool {
	return len(side.Properties) == 0 && side.Amount == 0
}

// offerRejection is every rule an offer has to clear that needs no state -
// facts about the payload alone. Pure, and above the first Mongo call in
// handleCreateOffer, so every rule here has a test.
func offerRejection(from, to primitive.ObjectID, give, get models.TradeSide, note string) error {
	if from == to {
		return errors.New("you can't make an offer to yourself")
	}
	if utf8.RuneCountInString(note) > maxNoteLength {
		return fmt.Errorf("a note can be at most %d characters", maxNoteLength)
	}
	given := make(map[primitive.ObjectID]bool, len(give.Properties))
	for _, id := range give.Properties {
		given[id] = true
	}
	for _, id := range get.Properties {
		if given[id] {
			return errors.New("a property can't be on both sides of an offer")
		}
	}
	if sideIsEmpty(give) && sideIsEmpty(get) {
		// A note with nothing tradeable beside it is not a trade, and settling
		// it would write a history row about money that never moved. Decided
		// here rather than left to the UI's disabled button: the server is
		// the contract. Reverse by deleting this check and the test that
		// pins it, if a note-only handshake turns out to be wanted.
		return errors.New("an offer needs cash or a property on at least one side")
	}
	return nil
}

// acceptRejection reports why responder may not accept offer, from the offer
// as read inside the settlement transaction, or nil if they may.
//
// The responder's identity is taken from the payload and checked against the
// offer, not against the socket. That is the same trust every other handler
// in this package extends - a room code is the only credential (HANDOFF.md,
// Settled) - and it is the check that matters: the offer document says who it
// was made to, and only that player may move both players' money.
func acceptRejection(offer models.Offer, responder primitive.ObjectID) error {
	if offer.Status != models.OfferPending {
		// Already accepted, denied, withdrawn or countered - or this is the
		// retry of a double-tap whose first frame already settled it. All of
		// them are "there is nothing here to accept", and none of them may
		// move anything.
		return errors.New("that offer is no longer open")
	}
	if offer.ToPlayerID != responder {
		return errors.New("only the player this offer was made to can accept it")
	}
	return nil
}

// tradePreconditions is every rule a trade has to clear against the state of
// the room, written against values already read rather than reading them
// itself - the bidRejection shape, for the same reason. It runs twice per
// trade: once when the offer is made, so a stale screen is told immediately,
// and once inside the settlement transaction, which is the check that counts.
//
// deeds is every property named on either side, as read from the room; a
// named id missing from it is a deed that is not in this room at all.
//
// The order is the order a player would want to hear about a problem: whether
// the two of them are still in the game, whether each still holds what they
// are putting up, and only then whether each can cover the cash.
//
// Cash is checked on the net, not the gross. A trade of $500 for $200 the
// other way moves $300, and a player holding $400 can make it; refusing them
// because they cannot hand over $500 first would be refusing a trade the
// table would shake on. The floor is $0 - the same floor freeParking and a
// bid apply - until the negative-balance room rule exists (D7, D9), at which
// point this is the one place a trade consults it.
//
// A deed with houses on it trades with the houses. Monopoly's own rule says
// sell the buildings first; this app leaves that to the table, the way it
// leaves rent to the table, and the banker can raze by hand. Reverse by
// refusing DevelopmentLevel > 0 here.
func tradePreconditions(from, to models.Player, give, get models.TradeSide, deeds []models.Property) error {
	if !from.IsActive {
		return fmt.Errorf("%s is no longer in the game", from.Name)
	}
	if !to.IsActive {
		return fmt.Errorf("%s is no longer in the game", to.Name)
	}

	byID := make(map[primitive.ObjectID]models.Property, len(deeds))
	for _, deed := range deeds {
		byID[deed.ID] = deed
	}
	owns := func(owner models.Player, ids []primitive.ObjectID) error {
		for _, id := range ids {
			deed, found := byID[id]
			if !found {
				return errors.New("a property in this offer is not in this room")
			}
			if deed.PlayerID != owner.ID {
				return fmt.Errorf("%s doesn't own %s", owner.Name, deed.Name)
			}
		}
		return nil
	}
	if err := owns(from, give.Properties); err != nil {
		return err
	}
	if err := owns(to, get.Properties); err != nil {
		return err
	}

	if short := give.Amount - get.Amount - from.Balance; short > 0 {
		return fmt.Errorf("%s is $%d short for this trade", from.Name, short)
	}
	if short := get.Amount - give.Amount - to.Balance; short > 0 {
		return fmt.Errorf("%s is $%d short for this trade", to.Name, short)
	}
	return nil
}

// sideDescription is one half of a trade as prose: the deeds by name in the
// order they were offered, then the cash. "Baltic Avenue, Oriental Avenue and
// $200"; "$200"; "Baltic Avenue"; "" for an empty side. A deed whose name is
// not in deeds - which tradePreconditions has already ruled out on the paths
// that reach here - reads as "a property" rather than as a blank.
func sideDescription(side models.TradeSide, deeds map[primitive.ObjectID]models.Property) string {
	parts := make([]string, 0, len(side.Properties)+1)
	for _, id := range side.Properties {
		if deed, found := deeds[id]; found && deed.Name != "" {
			parts = append(parts, deed.Name)
		} else {
			parts = append(parts, "a property")
		}
	}
	if side.Amount > 0 {
		parts = append(parts, fmt.Sprintf("$%d", side.Amount))
	}
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + " and " + parts[len(parts)-1]
	}
}

// tradeNotification is the one sentence a settled trade broadcasts to the
// room and, with the note appended by tradeRecord, stores in the event
// history. Warm register, matching the kick family it sits beside in the log.
//
// Every arm carries the word "traded", including the two one-sided arms and
// the unreachable default, because eventTypeFor keys the trade icon on it;
// an arm without it would be drawn as whatever substring of a player's name
// or a deed's name matched further down. Every arm is also non-empty, for
// the reason kickNotification's are: Broadcast refuses an empty notification
// silently.
//
// A one-sided trade is a gift, and the sentence says so from the giver's
// side whichever side that is - "asking nothing back" rather than "for
// nothing", which reads as a complaint.
func tradeNotification(fromName, toName string, give, get models.TradeSide, deeds map[primitive.ObjectID]models.Property) string {
	gives := sideDescription(give, deeds)
	gets := sideDescription(get, deeds)

	switch {
	case gives != "" && gets != "":
		return fmt.Sprintf("%s traded %s to %s for %s.", fromName, gives, toName, gets)
	case gives != "":
		return fmt.Sprintf("%s traded %s to %s, asking nothing back.", fromName, gives, toName)
	case gets != "":
		return fmt.Sprintf("%s traded %s to %s, asking nothing back.", toName, gets, fromName)
	default:
		// Unreachable: offerRejection refuses an offer with nothing on either
		// side. Kept so the function is total and the icon key survives.
		return fmt.Sprintf("%s traded with %s.", fromName, toName)
	}
}

// tradeRecord is the event-history row for a settled trade: the broadcast
// sentence, plus the note when there is one. The note is on the record and
// not in the toast on purpose - a 280-character deal in a four-second toast at
// phone width is a wall, and the record is where a player goes to read back
// what was agreed. The quotation marks are what make the note read as the
// players' words rather than the app's.
func tradeRecord(notification, note string) string {
	if note == "" {
		return notification
	}
	return fmt.Sprintf("%s Note: \"%s\"", notification, note)
}

// The targeted notifications. Each is the sentence one player toasts and
// nobody else sees. Pure and total for the same reason as the rest.

func offerReceivedNotification(fromName string, isCounter bool) string {
	if isCounter {
		return fmt.Sprintf("%s countered your offer.", fromName)
	}
	return fmt.Sprintf("%s sent you an offer.", fromName)
}

func offerSentNotification(toName string, isCounter bool) string {
	if isCounter {
		return fmt.Sprintf("Counter sent to %s.", toName)
	}
	return fmt.Sprintf("Offer sent to %s.", toName)
}

// offerResolvedNotifications is the pair of sentences a DENY or a WITHDRAW
// produces: the first for the offer's sender, the second for the player it
// was made to. Both are written from the reader's side, which is why they
// differ - the same event is "Bob declined your offer" on one screen and
// "You declined Alice's offer" on the other.
func offerResolvedNotifications(response, fromName, toName string) (forSender, forRecipient string) {
	switch response {
	case "WITHDRAW":
		return "Offer withdrawn.", fmt.Sprintf("%s withdrew their offer.", fromName)
	default:
		return fmt.Sprintf("%s declined your offer.", toName), fmt.Sprintf("You declined %s's offer.", fromName)
	}
}

// readDeeds fetches every property named on either side of a trade, scoped to
// the room, in one query. On ctx, so inside a transaction it reads from the
// transaction's snapshot.
func readDeeds(ctx context.Context, roomObjID primitive.ObjectID, give, get models.TradeSide) ([]models.Property, error) {
	ids := make([]primitive.ObjectID, 0, len(give.Properties)+len(get.Properties))
	ids = append(ids, give.Properties...)
	ids = append(ids, get.Properties...)
	deeds := make([]models.Property, 0, len(ids))
	if len(ids) == 0 {
		return deeds, nil
	}
	cursor, err := config.DB.Collection("Property").Find(ctx, bson.M{
		"_id":    bson.M{"$in": ids},
		"roomId": roomObjID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to read the properties in the offer: %w", err)
	}
	if err := cursor.All(ctx, &deeds); err != nil {
		return nil, fmt.Errorf("failed to decode the properties in the offer: %w", err)
	}
	return deeds, nil
}

// deedsByID indexes a deed list for the copy functions.
func deedsByID(deeds []models.Property) map[primitive.ObjectID]models.Property {
	byID := make(map[primitive.ObjectID]models.Property, len(deeds))
	for _, deed := range deeds {
		byID[deed.ID] = deed
	}
	return byID
}

// handleCreateOffer stores a new offer and sends it to the one player it is
// for. With counterOf set it is also the counter: the offer it answers is
// marked COUNTERED in the same transaction as the insert, pinned to PENDING
// and to the two players being swapped, so a counter to an offer that has
// since been answered is refused rather than leaving two live offers between
// the same two people for the same deeds.
//
// Everything above the first Mongo call is payload shape and offerRejection,
// and it is above it on purpose: config.DB is nil in a test binary, so a
// rejection below the first read is unreachable from any test on this
// machine. TestCreateOfferRejectsBeforeReadingThePlayer is the guard on that
// ordering.
func (rm *RoomManager) handleCreateOffer(client *Client, message Message) error {
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

	fromIdStr, ok := payload["fromPlayerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for fromPlayerId")
	}
	fromObjID, err := primitive.ObjectIDFromHex(fromIdStr)
	if err != nil {
		return fmt.Errorf("invalid fromPlayerId: %w", err)
	}

	toIdStr, ok := payload["toPlayerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for toPlayerId")
	}
	toObjID, err := primitive.ObjectIDFromHex(toIdStr)
	if err != nil {
		return fmt.Errorf("invalid toPlayerId: %w", err)
	}

	give, err := parseTradeSide(payload["offer"], "offer")
	if err != nil {
		return err
	}
	get, err := parseTradeSide(payload["request"], "request")
	if err != nil {
		return err
	}

	note := ""
	if raw, present := payload["note"]; present && raw != nil {
		text, ok := raw.(string)
		if !ok {
			return errors.New("invalid payload: expected string for note")
		}
		note = strings.TrimSpace(text)
	}

	// counterOf is optional: absent, null and "" all mean a fresh offer. Any
	// other value has to be a well-formed id, checked here so a malformed one
	// is refused before the first Mongo call. Same shape as the kick's
	// optional successorPlayerId.
	var counterOf *primitive.ObjectID
	if raw, present := payload["counterOf"]; present && raw != nil {
		hex, ok := raw.(string)
		if !ok {
			return errors.New("invalid payload: expected string for counterOf")
		}
		if hex != "" {
			id, err := primitive.ObjectIDFromHex(hex)
			if err != nil {
				return fmt.Errorf("invalid counterOf: %w", err)
			}
			counterOf = &id
		}
	}

	if err := offerRejection(fromObjID, toObjID, give, get, note); err != nil {
		return err
	}

	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---

	now := time.Now()
	offer := models.Offer{
		ID:           primitive.NewObjectID(),
		RoomID:       roomObjID,
		Status:       models.OfferPending,
		FromPlayerID: fromObjID,
		ToPlayerID:   toObjID,
		Offer:        give,
		Request:      get,
		Note:         note,
		CreatedAt:    now,
		UpdatedAt:    now,
		CounterOf:    counterOf,
	}

	playerColl := config.DB.Collection("Player")
	offerColl := config.DB.Collection("Offer")

	var fromPlayer, toPlayer models.Player

	session, err := config.DB.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
		// Reset every captured variable at the top, because WithTransaction
		// re-runs this callback on a write conflict. Same discipline as
		// handleCloseAuction. The offer document itself is built once above
		// and re-inserted as-is on a retry: the first attempt's insert was
		// rolled back with the rest of it.
		fromPlayer, toPlayer = models.Player{}, models.Player{}

		// Scoped to the room, and read WITHOUT an isActive clause so that a
		// removed player produces a document and a rule - "X is no longer in
		// the game" - rather than a missing document and a not-found error.
		// tradePreconditions is what decides whether they still count. This is
		// the choice PlayerTransfer's reads make (controllers/
		// transferControllers.go) and board row 16's decision, 2026-09-17: a
		// removed player may not move money, in either direction.
		if err := playerColl.FindOne(ctx, bson.M{"_id": fromObjID, "roomId": roomObjID}).Decode(&fromPlayer); err != nil {
			return nil, fmt.Errorf("failed to find the player making the offer: %w", err)
		}
		if err := playerColl.FindOne(ctx, bson.M{"_id": toObjID, "roomId": roomObjID}).Decode(&toPlayer); err != nil {
			return nil, fmt.Errorf("failed to find the player the offer is for: %w", err)
		}

		deeds, err := readDeeds(ctx, roomObjID, give, get)
		if err != nil {
			return nil, err
		}
		// Checked at creation as well as at settlement, so a player composing
		// from a stale screen - a deed that changed hands since their card
		// was drawn - is told now, not when the other player taps Accept. The
		// settlement's own check is the one that counts; this one is for the
		// message.
		if err := tradePreconditions(fromPlayer, toPlayer, give, get, deeds); err != nil {
			return nil, err
		}

		if counterOf != nil {
			// Pinned to PENDING and to the two players swapped: the offer
			// being countered was made BY the player this counter goes TO,
			// and TO the player making it. A counter to someone else's offer,
			// or to one already answered, matches nothing and aborts before
			// the insert, so the two writes land together or not at all.
			result, err := offerColl.UpdateOne(ctx,
				bson.M{
					"_id":          *counterOf,
					"roomId":       roomObjID,
					"status":       models.OfferPending,
					"fromPlayerId": toObjID,
					"toPlayerId":   fromObjID,
				},
				bson.M{"$set": bson.M{"status": models.OfferCountered, "updatedAt": now}},
			)
			if err != nil {
				return nil, fmt.Errorf("failed to mark the original offer countered: %w", err)
			}
			if result.MatchedCount == 0 {
				return nil, errors.New("that offer is no longer open to counter")
			}
		}

		if _, err := offerColl.InsertOne(ctx, offer); err != nil {
			return nil, fmt.Errorf("failed to save the offer: %w", err)
		}
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	// No event-history row: an offer is not room news, and the log is a
	// record of what happened to the money, not of the negotiating.

	isCounter := counterOf != nil
	reached := rm.SendTo(client.Room, toIdStr, Message{
		Type: "OFFER_RECEIVED",
		Payload: map[string]interface{}{
			"notification": offerReceivedNotification(fromPlayer.Name, isCounter),
			"offer":        offer,
		},
	})
	rm.SendTo(client.Room, fromIdStr, Message{
		Type: "OFFER_SENT",
		Payload: map[string]interface{}{
			"notification": offerSentNotification(toPlayer.Name, isCounter),
			"offer":        offer,
		},
	})
	log.Printf("Offer %s created in room %s from %s to %s; reached %d of the recipient's connection(s)", offer.ID.Hex(), client.Room, fromIdStr, toIdStr, reached)

	return nil
}

// handleRespondOffer answers an offer. DENY and WITHDRAW are one conditional
// write each and move nothing; ACCEPT is the settlement, in one transaction,
// following handleCloseAuction and deliberately not controllers.
// PurchaseProperty. A trade that hands over the deeds and then fails to move
// the cash is a free property; one that moves the cash and fails on a deed is
// money for nothing. Neither half may land alone.
//
// Payload rejections are above the first Mongo call, as everywhere in this
// package; the rules that need the offer's state are in acceptRejection and
// tradePreconditions, which take read values.
func (rm *RoomManager) handleRespondOffer(client *Client, message Message) error {
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

	offerIdStr, ok := payload["offerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for offerId")
	}
	offerObjID, err := primitive.ObjectIDFromHex(offerIdStr)
	if err != nil {
		return fmt.Errorf("invalid offer ID: %w", err)
	}

	responderIdStr, ok := payload["playerId"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for playerId")
	}
	responderObjID, err := primitive.ObjectIDFromHex(responderIdStr)
	if err != nil {
		return fmt.Errorf("invalid player ID: %w", err)
	}

	response, ok := payload["response"].(string)
	if !ok {
		return errors.New("invalid payload: expected string for response")
	}
	// Validated here, above the first read, for the reasons handleKickPlayer
	// hoists its disposition switch: an unrecognized value costs no round
	// trip, and the rejection is reachable in a test.
	switch response {
	case "ACCEPT", "DENY", "WITHDRAW":
		// valid - the switch below acts on these same three values
	default:
		return fmt.Errorf("invalid response: %s", response)
	}

	// --- the first database call. Nothing below here is reachable from a test
	// on a machine with no Mongo; it panics on the nil config.DB instead. ---

	playerColl := config.DB.Collection("Player")
	propColl := config.DB.Collection("Property")
	offerColl := config.DB.Collection("Offer")
	now := time.Now()

	if response != "ACCEPT" {
		// One conditional write. The pin is the whole rule: PENDING, in this
		// room, and the responder is the recipient (DENY) or the sender
		// (WITHDRAW). A second tap, a stale screen, or someone answering an
		// offer that is not theirs all match nothing and are told the same
		// thing, because from where they stand it is the same thing.
		filter := bson.M{"_id": offerObjID, "roomId": roomObjID, "status": models.OfferPending}
		if response == "WITHDRAW" {
			filter["fromPlayerId"] = responderObjID
		} else {
			filter["toPlayerId"] = responderObjID
		}
		var offer models.Offer
		err := offerColl.FindOneAndUpdate(context.Background(), filter,
			bson.M{"$set": bson.M{"status": models.OfferDenied, "updatedAt": now}},
		).Decode(&offer)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.New("that offer is no longer open")
		}
		if err != nil {
			return fmt.Errorf("failed to answer the offer: %w", err)
		}

		// Names only, for the two sentences; no isActive clause, because a
		// declined offer moves nothing and a removed player's name is still
		// the right word for who made it.
		var fromPlayer, toPlayer models.Player
		if err := playerColl.FindOne(context.Background(), bson.M{"_id": offer.FromPlayerID}).Decode(&fromPlayer); err != nil {
			return fmt.Errorf("failed to find the player who made the offer: %w", err)
		}
		if err := playerColl.FindOne(context.Background(), bson.M{"_id": offer.ToPlayerID}).Decode(&toPlayer); err != nil {
			return fmt.Errorf("failed to find the player the offer was for: %w", err)
		}

		forSender, forRecipient := offerResolvedNotifications(response, fromPlayer.Name, toPlayer.Name)
		resolved := func(text string) Message {
			return Message{
				Type: "OFFER_RESOLVED",
				Payload: map[string]interface{}{
					"notification": text,
					"offerId":      offerIdStr,
					"status":       models.OfferDenied,
				},
			}
		}
		rm.SendTo(client.Room, offer.FromPlayerID.Hex(), resolved(forSender))
		rm.SendTo(client.Room, offer.ToPlayerID.Hex(), resolved(forRecipient))
		log.Printf("Offer %s in room %s: %s by %s", offerIdStr, client.Room, response, responderIdStr)
		return nil
	}

	var notification, record string
	var fromIdHex, toIdHex string

	session, err := config.DB.Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
		// WithTransaction re-runs this callback on a write conflict, so every
		// value it decides from is read inside it and every captured variable
		// is reset at the top. The re-read is the mechanism: a second accept
		// of this offer, a transfer that drained a balance, or another trade
		// that moved one of these deeds, committed between attempts, has to
		// be seen by the retry rather than settled around.
		notification, record, fromIdHex, toIdHex = "", "", "", ""

		var offer models.Offer
		if err := offerColl.FindOne(ctx, bson.M{"_id": offerObjID, "roomId": roomObjID}).Decode(&offer); err != nil {
			return nil, fmt.Errorf("failed to find that offer: %w", err)
		}
		if err := acceptRejection(offer, responderObjID); err != nil {
			return nil, err
		}

		// Same reads as the create, same reasons: room-scoped, no isActive
		// clause, and tradePreconditions decides.
		var fromPlayer, toPlayer models.Player
		if err := playerColl.FindOne(ctx, bson.M{"_id": offer.FromPlayerID, "roomId": roomObjID}).Decode(&fromPlayer); err != nil {
			return nil, fmt.Errorf("failed to find the player who made the offer: %w", err)
		}
		if err := playerColl.FindOne(ctx, bson.M{"_id": offer.ToPlayerID, "roomId": roomObjID}).Decode(&toPlayer); err != nil {
			return nil, fmt.Errorf("failed to find the player the offer was for: %w", err)
		}

		deeds, err := readDeeds(ctx, roomObjID, offer.Offer, offer.Request)
		if err != nil {
			return nil, err
		}
		// The check that counts. Both players still in the game, each still
		// holding every deed they are putting up, each able to cover the net
		// cash - all from this transaction's snapshot. This is where the
		// second of two offers naming the same deed is refused: the first
		// accept moved it, and this read sees the new owner.
		if err := tradePreconditions(fromPlayer, toPlayer, offer.Offer, offer.Request, deeds); err != nil {
			return nil, err
		}

		// The claim on the offer, pinned to PENDING. Be clear about what
		// protects this, because handleCloseAuction's Deep review caught the
		// obvious reading being wrong: every read above is on ctx, so this pin
		// cannot fail on the attempt that read it. What stops a double-tap
		// settling twice is that the second frame's write to this document
		// raises a WriteConflict, WithTransaction re-runs its callback, and
		// the re-read finds ACCEPTED and acceptRejection refuses. The
		// MatchedCount check is defence against a future edit that reads
		// outside the transaction, not something that fires today.
		result, err := offerColl.UpdateOne(ctx,
			bson.M{"_id": offer.ID, "status": models.OfferPending},
			bson.M{"$set": bson.M{"status": models.OfferAccepted, "updatedAt": now}},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to accept the offer: %w", err)
		}
		if result.MatchedCount == 0 {
			return nil, errors.New("that offer was answered while it was settling - check your offers and try again")
		}

		// The deeds, each write pinned to the owner the snapshot saw. Same
		// defence-in-depth as the offer pin, and the fix for the hole
		// HANDOFF 21 raised on the auction close: a deed write that does not
		// name the current owner would overwrite whoever bought it in the
		// gap. The mortgage flag and the development level travel with the
		// deed - see tradePreconditions on why houses are not razed here.
		handOver := func(ids []primitive.ObjectID, owner, newOwner models.Player) error {
			for _, id := range ids {
				result, err := propColl.UpdateOne(ctx,
					bson.M{"_id": id, "roomId": roomObjID, "playerId": owner.ID},
					bson.M{"$set": bson.M{"playerId": newOwner.ID}},
				)
				if err != nil {
					return fmt.Errorf("failed to hand over a property: %w", err)
				}
				if result.MatchedCount == 0 {
					return fmt.Errorf("a property changed hands while this trade was settling - check %s's card and try again", owner.Name)
				}
			}
			return nil
		}
		if err := handOver(offer.Offer.Properties, fromPlayer, toPlayer); err != nil {
			return nil, err
		}
		if err := handOver(offer.Request.Properties, toPlayer, fromPlayer); err != nil {
			return nil, err
		}

		// The cash, as one net movement rather than two gross ones, because
		// the net is what tradePreconditions cleared. Skipped entirely when
		// it is zero - a deed-for-deed swap - so a $inc of 0 does not pull
		// two documents into the write set for nothing.
		if net := offer.Request.Amount - offer.Offer.Amount; net != 0 {
			if _, err := playerColl.UpdateOne(ctx, bson.M{"_id": fromPlayer.ID}, bson.M{"$inc": bson.M{"balance": net}}); err != nil {
				return nil, fmt.Errorf("failed to move the cash: %w", err)
			}
			if _, err := playerColl.UpdateOne(ctx, bson.M{"_id": toPlayer.ID}, bson.M{"$inc": bson.M{"balance": -net}}); err != nil {
				return nil, fmt.Errorf("failed to move the cash: %w", err)
			}
		}

		notification = tradeNotification(fromPlayer.Name, toPlayer.Name, offer.Offer, offer.Request, deedsByID(deeds))
		record = tradeRecord(notification, offer.Note)
		fromIdHex, toIdHex = fromPlayer.ID.Hex(), toPlayer.ID.Hex()
		return nil, nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	// After the transaction, never inside it on context.Background() -
	// freeParking's bug (HANDOFF 25). The record carries the note; the toast
	// does not.
	rm.CreateEventHistory(record, roomObjID)

	// A completed trade is room news, unlike the offer that led to it.
	rm.Broadcast(client.Room, Message{
		Type: "OFFER_ACCEPTED",
		Payload: map[string]interface{}{
			"notification": notification,
			"offerId":      offerIdStr,
			"fromPlayerId": fromIdHex,
			"toPlayerId":   toIdHex,
		},
	})
	log.Printf("Offer %s settled in room %s", offerIdStr, client.Room)

	return nil
}
