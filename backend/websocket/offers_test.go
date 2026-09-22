package websocket

import (
	"strings"
	"testing"

	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These cover the two trade handlers in offers.go and the pure rules beside
// them. The same boundary as every other handler test in this package holds:
// config.DB is a nil *mongo.Database in a test binary, so a payload the
// validation accepts panics at config.DB.Client() rather than erroring, and
// that panic is the only available signal that a value was accepted. Nothing
// here proves that a valid accept moves the right money - that needs Mongo,
// and there is none on this machine. What is proven: bad input is refused
// before any read, and every rule that decides a settlement is pinned by a
// test that bites when it is weakened.

const (
	roomHex  = "507f1f77bcf86cd799439011"
	aliceHex = "507f1f77bcf86cd799439012"
	bobHex   = "507f1f77bcf86cd799439013"
	carolHex = "507f1f77bcf86cd799439014"
	deed1Hex = "507f1f77bcf86cd799439021"
	deed2Hex = "507f1f77bcf86cd799439022"
	deed3Hex = "507f1f77bcf86cd799439023"
	offerHex = "507f1f77bcf86cd799439031"
)

// validCreateOfferPayload is a CREATE_OFFER that gets as far as the first
// Mongo call: Alice offers one deed and $100 for one of Bob's deeds. Each test
// below changes exactly one thing.
func validCreateOfferPayload() map[string]any {
	return map[string]any{
		"roomId":       roomHex,
		"fromPlayerId": aliceHex,
		"toPlayerId":   bobHex,
		"offer": map[string]any{
			"properties": []any{deed1Hex},
			"amount":     float64(100),
		},
		"request": map[string]any{
			"properties": []any{deed2Hex},
			"amount":     float64(0),
		},
		"note": "",
	}
}

func createOfferOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handleCreateOffer(testClient(), Message{
		Type:    "CREATE_OFFER",
		Payload: payload,
	})
	return err, false
}

// wantRejected asserts a payload was refused before the first Mongo call:
// an error, and no panic.
func wantRejected(t *testing.T, err error, panicked bool, want string) {
	t.Helper()
	if panicked {
		t.Fatalf("expected the rejection %q before any database call, but the handler reached the database", want)
	}
	wantErrContains(t, err, want)
}

// wantAccepted asserts a payload cleared validation: the handler reached the
// nil config.DB and panicked there.
func wantAccepted(t *testing.T, err error, panicked bool) {
	t.Helper()
	if !panicked {
		t.Fatalf("expected the payload to be accepted and reach the database, got %v", err)
	}
}

// --- handleCreateOffer: payload shape ---

func TestCreateOfferRejectsNonObjectPayload(t *testing.T) {
	err, panicked := createOfferOutcome(t, "not an object")
	wantRejected(t, err, panicked, "invalid payload format")
}

func TestCreateOfferRejectsBadIDs(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		value       any
		want        string
	}{
		{"missing roomId", "roomId", nil, "expected string for roomId"},
		{"non-string roomId", "roomId", float64(1), "expected string for roomId"},
		{"malformed roomId", "roomId", "nope", "invalid room ID"},
		{"missing fromPlayerId", "fromPlayerId", nil, "expected string for fromPlayerId"},
		{"non-string fromPlayerId", "fromPlayerId", float64(1), "expected string for fromPlayerId"},
		{"malformed fromPlayerId", "fromPlayerId", "nope", "invalid fromPlayerId"},
		{"missing toPlayerId", "toPlayerId", nil, "expected string for toPlayerId"},
		{"non-string toPlayerId", "toPlayerId", true, "expected string for toPlayerId"},
		{"malformed toPlayerId", "toPlayerId", "nope", "invalid toPlayerId"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := validCreateOfferPayload()
			if tc.value == nil {
				delete(payload, tc.field)
			} else {
				payload[tc.field] = tc.value
			}
			err, panicked := createOfferOutcome(t, payload)
			wantRejected(t, err, panicked, tc.want)
		})
	}
}

func TestCreateOfferRejectsAnOfferToYourself(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["toPlayerId"] = aliceHex

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "you can't make an offer to yourself")
}

func TestCreateOfferRejectsANonObjectSide(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["offer"] = "everything"

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected an object for offer")
}

func TestCreateOfferRejectsNonArrayProperties(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["request"] = map[string]any{"properties": deed2Hex}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected an array for request.properties")
}

func TestCreateOfferRejectsANonStringPropertyID(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{"properties": []any{float64(7)}}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected strings in offer.properties")
}

func TestCreateOfferRejectsAMalformedPropertyID(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{"properties": []any{"baltic"}}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "invalid property ID in offer.properties")
}

func TestCreateOfferRejectsAPropertyListedTwice(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{"properties": []any{deed1Hex, deed1Hex}}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "a property is listed twice in offer.properties")
}

func TestCreateOfferRejectsAPropertyOnBothSides(t *testing.T) {
	// Offering Baltic and asking for Baltic back. The UI cannot compose this
	// - each side picks from a different player's deeds - so it only arrives
	// from a stale screen or a hand-built frame, and it has to be refused
	// rather than settled as a no-op with a history row.
	payload := validCreateOfferPayload()
	payload["request"] = map[string]any{"properties": []any{deed1Hex}}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "a property can't be on both sides of an offer")
}

func TestCreateOfferRejectsAStringAmount(t *testing.T) {
	// A number, like a bid and a purchase price - not the keypad string
	// freeParking takes.
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{"amount": "100"}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected number for offer.amount")
}

func TestCreateOfferRejectsAFractionalAmount(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["request"] = map[string]any{"amount": 120.9}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "not a whole number of dollars")
}

func TestCreateOfferRejectsANegativeAmount(t *testing.T) {
	// The cash write is one net $inc whose sign comes from which side is
	// larger, so a negative amount on one side is indistinguishable from a
	// positive one on the other - except that the player never agreed to it.
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{"amount": float64(-50)}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "offer.amount can't be negative")
}

func TestCreateOfferRejectsANonStringNote(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["note"] = float64(3)

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected string for note")
}

func TestCreateOfferRejectsANoteOverTheCap(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["note"] = strings.Repeat("a", maxNoteLength+1)

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "a note can be at most 280 characters")
}

func TestCreateOfferAcceptsANoteAtExactlyTheCap(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["note"] = strings.Repeat("a", maxNoteLength)

	err, panicked := createOfferOutcome(t, payload)
	wantAccepted(t, err, panicked)
}

func TestCreateOfferCountsTheNoteInRunesNotBytes(t *testing.T) {
	// 280 handshakes is 1120 bytes. The browser counts UTF-16 units, so this
	// is 560 there and refused before it is sent; the server's cap is the
	// looser of the two on purpose, so nothing the browser accepts is refused
	// here. A byte-counted cap would refuse a 71-character note in emoji.
	payload := validCreateOfferPayload()
	payload["note"] = strings.Repeat("🤝", maxNoteLength)

	err, panicked := createOfferOutcome(t, payload)
	wantAccepted(t, err, panicked)
}

func TestTheNoteCapIsTheNumberTheTextareaCarries(t *testing.T) {
	// frontend/components/players/make-offer/make-offer.tsx sets maxLength
	// on the note textarea to this same number. Nothing checks the two
	// agree; this pins the server's half so a change here is a deliberate
	// one that goes looking for the other.
	if maxNoteLength != 280 {
		t.Fatalf("maxNoteLength is %d, want 280 - change the textarea's maxLength with it", maxNoteLength)
	}
}

func TestCreateOfferRejectsAnEmptyOffer(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{}
	payload["request"] = map[string]any{}

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "an offer needs cash or a property on at least one side")
}

func TestCreateOfferRejectsANoteOnlyOffer(t *testing.T) {
	// A note with nothing tradeable beside it. Refused at the server, not
	// only by the disabled Send button, because the server is the contract.
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{}
	payload["request"] = map[string]any{}
	payload["note"] = "immunity on the browns for three turns"

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "an offer needs cash or a property on at least one side")
}

func TestCreateOfferAcceptsCashAlone(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{"amount": float64(100)}
	payload["request"] = map[string]any{}

	err, panicked := createOfferOutcome(t, payload)
	wantAccepted(t, err, panicked)
}

func TestCreateOfferAcceptsDeedsAlone(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{"properties": []any{deed1Hex}}
	payload["request"] = map[string]any{"properties": []any{deed2Hex}}

	err, panicked := createOfferOutcome(t, payload)
	wantAccepted(t, err, panicked)
}

func TestCreateOfferAcceptsAOneSidedRequest(t *testing.T) {
	// Asking for something and offering nothing is an offer the other player
	// may well accept - it is how "can I have Baltic?" is asked.
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{}

	err, panicked := createOfferOutcome(t, payload)
	wantAccepted(t, err, panicked)
}

func TestCreateOfferTreatsAbsentSidesAsEmpty(t *testing.T) {
	payload := validCreateOfferPayload()
	delete(payload, "offer")
	delete(payload, "note")

	err, panicked := createOfferOutcome(t, payload)
	wantAccepted(t, err, panicked)
}

func TestCreateOfferRejectsANonStringCounterOf(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["counterOf"] = float64(1)

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected string for counterOf")
}

func TestCreateOfferRejectsAMalformedCounterOf(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["counterOf"] = "the-last-one"

	err, panicked := createOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "invalid counterOf")
}

func TestCreateOfferTreatsEmptyAndNullCounterOfAsAbsent(t *testing.T) {
	for _, value := range []any{"", nil} {
		payload := validCreateOfferPayload()
		payload["counterOf"] = value

		err, panicked := createOfferOutcome(t, payload)
		wantAccepted(t, err, panicked)
	}
}

func TestCreateOfferAcceptsACounter(t *testing.T) {
	payload := validCreateOfferPayload()
	payload["counterOf"] = offerHex

	err, panicked := createOfferOutcome(t, payload)
	wantAccepted(t, err, panicked)
}

func TestCreateOfferRejectsBeforeReadingThePlayer(t *testing.T) {
	// The ordering guard. Every rejection above is only reachable because
	// offerRejection runs above the first Mongo call; if it ever slips below
	// the player read, this test panics instead of erroring.
	payload := validCreateOfferPayload()
	payload["offer"] = map[string]any{}
	payload["request"] = map[string]any{}

	err, panicked := createOfferOutcome(t, payload)
	if panicked {
		t.Fatal("the empty-offer rejection reached the database - offerRejection has slipped below the first read")
	}
	wantErrEqual(t, err, "an offer needs cash or a property on at least one side")
}

func TestCreateOfferAcceptsAValidPayload(t *testing.T) {
	err, panicked := createOfferOutcome(t, validCreateOfferPayload())
	wantAccepted(t, err, panicked)
}

func TestParseTradeSideNeverReturnsANilPropertyList(t *testing.T) {
	// A nil slice marshals as null and the browser's .length reads would
	// throw. make, not var, on every path that returns a side.
	for _, raw := range []any{nil, map[string]any{}, map[string]any{"amount": float64(5)}} {
		side, err := parseTradeSide(raw, "offer")
		if err != nil {
			t.Fatalf("parseTradeSide(%v): %v", raw, err)
		}
		if side.Properties == nil {
			t.Fatalf("parseTradeSide(%v) returned a nil property list", raw)
		}
	}
}

// --- handleRespondOffer: payload shape ---

func validRespondPayload() map[string]any {
	return map[string]any{
		"roomId":   roomHex,
		"offerId":  offerHex,
		"playerId": bobHex,
		"response": "ACCEPT",
	}
}

func respondOfferOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handleRespondOffer(testClient(), Message{
		Type:    "RESPOND_OFFER",
		Payload: payload,
	})
	return err, false
}

func TestRespondOfferRejectsNonObjectPayload(t *testing.T) {
	err, panicked := respondOfferOutcome(t, []any{"ACCEPT"})
	wantRejected(t, err, panicked, "invalid payload format")
}

func TestRespondOfferRejectsBadIDs(t *testing.T) {
	for _, tc := range []struct {
		name, field string
		value       any
		want        string
	}{
		{"missing roomId", "roomId", nil, "expected string for roomId"},
		{"non-string roomId", "roomId", float64(1), "expected string for roomId"},
		{"malformed roomId", "roomId", "nope", "invalid room ID"},
		{"missing offerId", "offerId", nil, "expected string for offerId"},
		{"non-string offerId", "offerId", float64(1), "expected string for offerId"},
		{"malformed offerId", "offerId", "nope", "invalid offer ID"},
		{"missing playerId", "playerId", nil, "expected string for playerId"},
		{"non-string playerId", "playerId", true, "expected string for playerId"},
		{"malformed playerId", "playerId", "nope", "invalid player ID"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			payload := validRespondPayload()
			if tc.value == nil {
				delete(payload, tc.field)
			} else {
				payload[tc.field] = tc.value
			}
			err, panicked := respondOfferOutcome(t, payload)
			wantRejected(t, err, panicked, tc.want)
		})
	}
}

func TestRespondOfferRejectsMissingResponse(t *testing.T) {
	payload := validRespondPayload()
	delete(payload, "response")

	err, panicked := respondOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected string for response")
}

func TestRespondOfferRejectsANonStringResponse(t *testing.T) {
	payload := validRespondPayload()
	payload["response"] = true

	err, panicked := respondOfferOutcome(t, payload)
	wantRejected(t, err, panicked, "expected string for response")
}

func TestRespondOfferRejectsAnUnrecognizedResponse(t *testing.T) {
	// COUNTER is the plausible one: a counter is a CREATE_OFFER carrying
	// counterOf, not a response, and a frame that sends it here must be told
	// so rather than treated as any of the three.
	for _, value := range []string{"COUNTER", "accept", "YES", ""} {
		payload := validRespondPayload()
		payload["response"] = value

		err, panicked := respondOfferOutcome(t, payload)
		wantRejected(t, err, panicked, "invalid response: "+value)
	}
}

func TestRespondOfferAcceptsTheThreeResponses(t *testing.T) {
	for _, value := range []string{"ACCEPT", "DENY", "WITHDRAW"} {
		payload := validRespondPayload()
		payload["response"] = value

		err, panicked := respondOfferOutcome(t, payload)
		wantAccepted(t, err, panicked)
	}
}

func TestRespondOfferRejectsBeforeReadingTheOffer(t *testing.T) {
	payload := validRespondPayload()
	payload["response"] = "MAYBE"

	err, panicked := respondOfferOutcome(t, payload)
	if panicked {
		t.Fatal("the response check reached the database - it has slipped below the first read")
	}
	wantErrEqual(t, err, "invalid response: MAYBE")
}

// --- acceptRejection ---

func pendingOffer() models.Offer {
	return models.Offer{
		ID:           mustObjectID(offerHex),
		RoomID:       mustObjectID(roomHex),
		Status:       models.OfferPending,
		FromPlayerID: mustObjectID(aliceHex),
		ToPlayerID:   mustObjectID(bobHex),
	}
}

func TestAcceptRejectionRefusesAnOfferThatIsNotPending(t *testing.T) {
	// The double-tap, from the retry's point of view: the first frame
	// settled it, the second re-reads and finds ACCEPTED. And every other
	// answered state, which may not move anything either.
	for _, status := range []models.OfferStatus{models.OfferAccepted, models.OfferDenied, models.OfferCountered} {
		offer := pendingOffer()
		offer.Status = status

		err := acceptRejection(offer, mustObjectID(bobHex))
		wantErrEqual(t, err, "that offer is no longer open")
	}
}

func TestAcceptRejectionRefusesAnyoneButTheRecipient(t *testing.T) {
	// The sender accepting their own offer on the recipient's behalf, and a
	// third player accepting it for them. Both hold the room code; neither
	// is the player the offer document names.
	for _, who := range []string{aliceHex, carolHex} {
		err := acceptRejection(pendingOffer(), mustObjectID(who))
		wantErrEqual(t, err, "only the player this offer was made to can accept it")
	}
}

func TestAcceptRejectionChecksTheStatusBeforeTheResponder(t *testing.T) {
	offer := pendingOffer()
	offer.Status = models.OfferDenied

	err := acceptRejection(offer, mustObjectID(carolHex))
	wantErrEqual(t, err, "that offer is no longer open")
}

func TestAcceptRejectionAcceptsTheRecipientOfAPendingOffer(t *testing.T) {
	if err := acceptRejection(pendingOffer(), mustObjectID(bobHex)); err != nil {
		t.Fatalf("expected the recipient to be allowed to accept, got %v", err)
	}
}

// --- tradePreconditions ---

func alice() models.Player {
	return models.Player{ID: mustObjectID(aliceHex), Name: "Alice", IsActive: true, Balance: 500}
}

func bob() models.Player {
	return models.Player{ID: mustObjectID(bobHex), Name: "Bob", IsActive: true, Balance: 500}
}

func deed(hex, name string, owner primitive.ObjectID) models.Property {
	return models.Property{ID: mustObjectID(hex), Name: name, PlayerID: owner}
}

func side(amount int, hexes ...string) models.TradeSide {
	ids := make([]primitive.ObjectID, 0, len(hexes))
	for _, hex := range hexes {
		ids = append(ids, mustObjectID(hex))
	}
	return models.TradeSide{Properties: ids, Amount: amount}
}

// tableDeeds is the room as the tests below see it: Alice holds Baltic and
// Mediterranean, Bob holds Oriental.
func tableDeeds() []models.Property {
	return []models.Property{
		deed(deed1Hex, "Baltic Avenue", mustObjectID(aliceHex)),
		deed(deed2Hex, "Oriental Avenue", mustObjectID(bobHex)),
		deed(deed3Hex, "Mediterranean Avenue", mustObjectID(aliceHex)),
	}
}

func TestTradePreconditionsAcceptsAValidTrade(t *testing.T) {
	err := tradePreconditions(alice(), bob(), side(100, deed1Hex), side(0, deed2Hex), tableDeeds())
	if err != nil {
		t.Fatalf("expected the trade to clear, got %v", err)
	}
}

func TestTradePreconditionsRefusesARemovedSender(t *testing.T) {
	// Board row 16, decided 2026-09-17: a removed player may not move money,
	// and a trade moves it in both directions.
	from := alice()
	from.IsActive = false

	err := tradePreconditions(from, bob(), side(100, deed1Hex), side(0, deed2Hex), tableDeeds())
	wantErrEqual(t, err, "Alice is no longer in the game")
}

func TestTradePreconditionsRefusesARemovedRecipient(t *testing.T) {
	to := bob()
	to.IsActive = false

	err := tradePreconditions(alice(), to, side(100, deed1Hex), side(0, deed2Hex), tableDeeds())
	wantErrEqual(t, err, "Bob is no longer in the game")
}

func TestTradePreconditionsRefusesADeedTheSenderDoesNotOwn(t *testing.T) {
	// Alice offering Oriental, which is Bob's. Also the second of two offers
	// naming the same deed, seen from the settlement's re-read: the first
	// accept moved it, and this is the sentence the second responder gets.
	err := tradePreconditions(alice(), bob(), side(0, deed2Hex), side(0), tableDeeds())
	wantErrEqual(t, err, "Alice doesn't own Oriental Avenue")
}

func TestTradePreconditionsRefusesADeedTheRecipientDoesNotOwn(t *testing.T) {
	err := tradePreconditions(alice(), bob(), side(0), side(0, deed1Hex), tableDeeds())
	wantErrEqual(t, err, "Bob doesn't own Baltic Avenue")
}

func TestTradePreconditionsRefusesADeedTheBankHolds(t *testing.T) {
	// A bank-held deed has no playerId at all, which decodes as the zero
	// ObjectID - never equal to a real player's.
	deeds := tableDeeds()
	deeds[0].PlayerID = primitive.NilObjectID

	err := tradePreconditions(alice(), bob(), side(0, deed1Hex), side(0), deeds)
	wantErrEqual(t, err, "Alice doesn't own Baltic Avenue")
}

func TestTradePreconditionsRefusesADeedNotInTheRoom(t *testing.T) {
	// Named on the offer, absent from the room-scoped read: another room's
	// property id, or a document that no longer exists.
	err := tradePreconditions(alice(), bob(), side(0, deed1Hex), side(0), []models.Property{})
	wantErrEqual(t, err, "a property in this offer is not in this room")
}

func TestTradePreconditionsRefusesASenderWhoIsShort(t *testing.T) {
	err := tradePreconditions(alice(), bob(), side(600), side(0), tableDeeds())
	wantErrEqual(t, err, "Alice is $100 short for this trade")
}

func TestTradePreconditionsRefusesARecipientWhoIsShort(t *testing.T) {
	err := tradePreconditions(alice(), bob(), side(0), side(750), tableDeeds())
	wantErrEqual(t, err, "Bob is $250 short for this trade")
}

func TestTradePreconditionsChecksTheNetNotTheGross(t *testing.T) {
	// $600 one way and $200 back moves $400. Alice holds $500. The gross
	// check would refuse her; the trade the table would shake on clears.
	err := tradePreconditions(alice(), bob(), side(600), side(200), tableDeeds())
	if err != nil {
		t.Fatalf("expected a net-affordable trade to clear, got %v", err)
	}
}

func TestTradePreconditionsAcceptsAnExactBalance(t *testing.T) {
	err := tradePreconditions(alice(), bob(), side(500), side(0), tableDeeds())
	if err != nil {
		t.Fatalf("expected a trade for exactly the balance to clear, got %v", err)
	}
}

func TestTradePreconditionsChecksActivityBeforeOwnershipBeforeCash(t *testing.T) {
	// A removed sender who also no longer holds the deed and cannot cover
	// the cash: the first thing wrong is the thing they are told.
	from := alice()
	from.IsActive = false
	deeds := tableDeeds()
	deeds[0].PlayerID = mustObjectID(bobHex)

	err := tradePreconditions(from, bob(), side(9000, deed1Hex), side(0), deeds)
	wantErrEqual(t, err, "Alice is no longer in the game")

	from.IsActive = true
	err = tradePreconditions(from, bob(), side(9000, deed1Hex), side(0), deeds)
	wantErrEqual(t, err, "Alice doesn't own Baltic Avenue")
}

func TestTradePreconditionsLeavesDevelopedDeedsTradeable(t *testing.T) {
	// Houses travel with the deed - see tradePreconditions on why. This pins
	// that no rule was quietly added; reversing it is a deliberate change
	// that comes here first.
	deeds := tableDeeds()
	deeds[0].DevelopmentLevel = 3
	deeds[0].IsMortgaged = true

	err := tradePreconditions(alice(), bob(), side(0, deed1Hex), side(0), deeds)
	if err != nil {
		t.Fatalf("expected a developed, mortgaged deed to be tradeable, got %v", err)
	}
}

// --- the copy ---

func TestSideDescription(t *testing.T) {
	deeds := deedsByID(tableDeeds())
	for _, tc := range []struct {
		name string
		side models.TradeSide
		want string
	}{
		{"empty", side(0), ""},
		{"cash only", side(200), "$200"},
		{"one deed", side(0, deed1Hex), "Baltic Avenue"},
		{"deed and cash", side(200, deed1Hex), "Baltic Avenue and $200"},
		{"two deeds", side(0, deed1Hex, deed3Hex), "Baltic Avenue and Mediterranean Avenue"},
		{"two deeds and cash", side(200, deed1Hex, deed3Hex), "Baltic Avenue, Mediterranean Avenue and $200"},
		{"unknown deed", side(0, "507f1f77bcf86cd799439099"), "a property"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := sideDescription(tc.side, deeds); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTradeNotificationArms(t *testing.T) {
	deeds := deedsByID(tableDeeds())
	for _, tc := range []struct {
		name      string
		give, get models.TradeSide
		want      string
	}{
		{"both sides", side(200, deed1Hex), side(0, deed2Hex), "Alice traded Baltic Avenue and $200 to Bob for Oriental Avenue."},
		{"gift from the sender", side(200, deed1Hex), side(0), "Alice traded Baltic Avenue and $200 to Bob, asking nothing back."},
		{"gift from the recipient", side(0), side(0, deed2Hex), "Bob traded Oriental Avenue to Alice, asking nothing back."},
		{"unreachable empty trade", side(0), side(0), "Alice traded with Bob."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tradeNotification("Alice", "Bob", tc.give, tc.get, deeds); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestTradeRowsTakeTheTradeIcon(t *testing.T) {
	// Every arm, with names and deeds chosen to trip the substring keys
	// further down eventTypeFor: "Warehouse" carries "house", "Vincent"
	// carries "sent", and "Park Place" is a real deed. The trade arm is first
	// so none of that matters.
	deeds := deedsByID([]models.Property{
		deed(deed1Hex, "Park Place", mustObjectID(aliceHex)),
		deed(deed2Hex, "Oriental Avenue", mustObjectID(bobHex)),
	})
	for _, tc := range []struct {
		name      string
		give, get models.TradeSide
	}{
		{"both sides", side(200, deed1Hex), side(0, deed2Hex)},
		{"gift from the sender", side(200, deed1Hex), side(0)},
		{"gift from the recipient", side(0), side(0, deed2Hex)},
		{"unreachable empty trade", side(0), side(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			text := tradeNotification("Warehouse", "Vincent", tc.give, tc.get, deeds)
			if !strings.Contains(text, " traded ") {
				t.Fatalf("%q does not carry the word the icon is keyed on", text)
			}
			got := eventTypeFor(tradeRecord(text, "houses for hotels, sent with love"))
			if got[1] != "🤝" {
				t.Fatalf("eventTypeFor(%q) = %v, want the trade icon", text, got)
			}
		})
	}
}

func TestTradeIconDoesNotStealExistingRows(t *testing.T) {
	// The key is " traded " with spaces, so a name like "Traded" at the
	// start of a sentence, or a purchase by a player called "Untraded", do
	// not fall into the trade arm.
	for text, want := range map[string]string{
		"Untraded purchased Boardwalk from the Bank": "🏠",
		"Alice just sent $50 to Bob for rent":        "💸",
		"Banker has added $100 to Alice's balance":   "🏦",
	} {
		if got := eventTypeFor(text); got[1] != want {
			t.Fatalf("eventTypeFor(%q) = %v, want %s", text, got, want)
		}
	}
}

func TestTradeRecordAppendsTheNoteInQuotes(t *testing.T) {
	got := tradeRecord("Alice traded Baltic Avenue to Bob for $50.", "immunity on the browns for three turns")
	want := "Alice traded Baltic Avenue to Bob for $50. Note: \"immunity on the browns for three turns\""
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTradeRecordWithoutANoteIsTheNotification(t *testing.T) {
	text := "Alice traded Baltic Avenue to Bob for $50."
	if got := tradeRecord(text, ""); got != text {
		t.Fatalf("got %q, want %q", got, text)
	}
}

func TestOfferReceivedAndSentCopy(t *testing.T) {
	for got, want := range map[string]string{
		offerReceivedNotification("Alice", false): "Alice sent you an offer.",
		offerReceivedNotification("Alice", true):  "Alice countered your offer.",
		offerSentNotification("Bob", false):       "Offer sent to Bob.",
		offerSentNotification("Bob", true):        "Counter sent to Bob.",
	} {
		if got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestOfferResolvedCopyIsWrittenFromEachReadersSide(t *testing.T) {
	forSender, forRecipient := offerResolvedNotifications("DENY", "Alice", "Bob")
	if forSender != "Bob declined your offer." || forRecipient != "You declined Alice's offer." {
		t.Fatalf("DENY: got %q / %q", forSender, forRecipient)
	}

	forSender, forRecipient = offerResolvedNotifications("WITHDRAW", "Alice", "Bob")
	if forSender != "Offer withdrawn." || forRecipient != "Alice withdrew their offer." {
		t.Fatalf("WITHDRAW: got %q / %q", forSender, forRecipient)
	}
}

func TestOfferCopySurvivesTheEmptyNotificationGuard(t *testing.T) {
	// SendTo and Broadcast both drop a payload whose notification is empty,
	// silently to the room. Every sentence an offer produces has to be
	// non-empty even with blank names, or an offer could vanish on the wire
	// while the database says it was made.
	deeds := deedsByID(nil)
	texts := []string{
		offerReceivedNotification("", false),
		offerReceivedNotification("", true),
		offerSentNotification("", false),
		offerSentNotification("", true),
		tradeNotification("", "", side(0), side(0), deeds),
		tradeRecord("", ""),
	}
	a, b := offerResolvedNotifications("DENY", "", "")
	c, d := offerResolvedNotifications("WITHDRAW", "", "")
	texts = append(texts, a, b, c, d)
	for i, text := range texts {
		if i == 5 {
			// tradeRecord("", "") is "" by construction; it is never sent with
			// an empty notification because tradeNotification is never empty.
			continue
		}
		if empty, hasField := emptyNotification(map[string]interface{}{"notification": text}); hasField && empty {
			t.Fatalf("text %d is empty and would be dropped by the guard", i)
		}
	}
}
