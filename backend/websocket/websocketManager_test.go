package websocket

import (
	"bytes"
	"log"
	"runtime/debug"
	"strings"
	"testing"

	"github.com/zachmshort/emoney-backend/models"
)

// These tests cover the rejection branches of the websocket money handlers -
// the paths that return an error before any call into controllers/ or manager/,
// which is as far as a test can reach without a live Mongo. config.DB is a nil
// *mongo.Database in a test binary, so anything that reaches it panics rather
// than erroring; see HANDOFF 4 for which branches that rules out.

func testClient() *Client {
	return &Client{Room: "ABCD", PlayerID: "507f1f77bcf86cd799439012", PlayerName: "Tester"}
}

// validManagePayload is a MANAGE_PROPERTIES payload that gets as far as the
// managementType switch. Each test below changes exactly one field.
func validManagePayload() map[string]any {
	return map[string]any{
		"amount":         "100",
		"roomId":         "507f1f77bcf86cd799439011",
		"playerId":       "507f1f77bcf86cd799439012",
		"properties":     []any{},
		"managementType": "HOUSES",
	}
}

// validTransferPayload is a TRANSFER payload that gets as far as the
// transferType switch. Each test below changes exactly one field.
func validTransferPayload() map[string]any {
	return map[string]any{
		"amount":       "100",
		"roomId":       "507f1f77bcf86cd799439011",
		"reason":       "rent",
		"transferType": "SEND",
	}
}

// validBankPayload is a BANKER_TRANSACTION payload that gets as far as the
// transactionType switch. Each test below changes exactly one field.
func validBankPayload() map[string]any {
	return map[string]any{
		"amount":          "100",
		"roomId":          "507f1f77bcf86cd799439011",
		"toPlayerId":      "507f1f77bcf86cd799439012",
		"transactionType": "BANKER_ADD",
	}
}

// validFreeParkingPayload is a FREE_PARKING payload that gets as far as the
// freeParkingType switch. Each test below changes exactly one field.
func validFreeParkingPayload() map[string]any {
	return map[string]any{
		"amount":          "100",
		"roomId":          "507f1f77bcf86cd799439011",
		"playerId":        "507f1f77bcf86cd799439012",
		"freeParkingType": "ADD",
	}
}

// validPurchasePayload is a PURCHASE_PROPERTY payload that gets as far as the
// buyerId parse. Each test below changes exactly one field.
func validPurchasePayload() map[string]any {
	return map[string]any{
		"price":      float64(60),
		"buyerId":    "507f1f77bcf86cd799439012",
		"propertyId": "507f1f77bcf86cd799439013",
	}
}

func manageErr(t *testing.T, payload any) error {
	t.Helper()
	return NewRoomManager().handleManageProperties(testClient(), Message{
		Type:    "MANAGE_PROPERTIES",
		Payload: payload,
	})
}

func transferErr(t *testing.T, payload any) error {
	t.Helper()
	return NewRoomManager().handleTransfer(testClient(), Message{
		Type:    "TRANSFER",
		Payload: payload,
	})
}

func purchaseErr(t *testing.T, payload any) error {
	t.Helper()
	return NewRoomManager().handlePropertyPurchase(testClient(), Message{
		Type:    "PURCHASE_PROPERTY",
		Payload: payload,
	})
}

func wantErrEqual(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q, got nil", want)
	}
	if err.Error() != want {
		t.Fatalf("expected error %q, got %q", want, err.Error())
	}
}

func wantErrContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected an error containing %q, got nil", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected an error containing %q, got %q", want, err.Error())
	}
}

// --- handleManageProperties ---

func TestManagePropertiesRejectsUnrecognizedManagementType(t *testing.T) {
	payload := validManagePayload()
	payload["managementType"] = "HOUSE" // the plausible typo for "HOUSES"

	err := manageErr(t, payload)

	wantErrEqual(t, err, "invalid management type: HOUSE")
}

func TestManagePropertiesRejectsUntrimmedManagementType(t *testing.T) {
	// manager.HandlePropertySaleMortgage trims before its own switch
	// (manager/propertyManager.go:37), but the caller's switch does not, so a
	// padded value never reaches the trim.
	payload := validManagePayload()
	payload["managementType"] = " MORTGAGE"

	err := manageErr(t, payload)

	wantErrEqual(t, err, "invalid management type:  MORTGAGE")
}

func TestManagePropertiesRejectsNonObjectPayload(t *testing.T) {
	err := manageErr(t, "not-an-object")

	wantErrEqual(t, err, "invalid payload format")
}

func TestManagePropertiesRejectsMissingAmount(t *testing.T) {
	payload := validManagePayload()
	delete(payload, "amount")

	err := manageErr(t, payload)

	wantErrEqual(t, err, "missing amount field")
}

func TestManagePropertiesRejectsUnparseableStringAmount(t *testing.T) {
	payload := validManagePayload()
	payload["amount"] = "one hundred"

	err := manageErr(t, payload)

	wantErrContains(t, err, "invalid amount value")
}

func TestManagePropertiesRejectsAmountOfUnsupportedType(t *testing.T) {
	payload := validManagePayload()
	payload["amount"] = true

	err := manageErr(t, payload)

	wantErrEqual(t, err, "unexpected type for amount: bool")
}

func TestManagePropertiesAcceptsFloatAmountFromJSON(t *testing.T) {
	// encoding/json decodes every number into float64, so the float64 arm is
	// the one real clients take. Reaching past it means reaching Mongo, so the
	// assertion is only that the amount itself was not what was rejected.
	payload := validManagePayload()
	payload["amount"] = float64(100)
	payload["managementType"] = "HOUSE"

	err := manageErr(t, payload)

	wantErrEqual(t, err, "invalid management type: HOUSE")
}

func TestManagePropertiesRejectsMalformedRoomID(t *testing.T) {
	payload := validManagePayload()
	payload["roomId"] = "not-an-object-id"

	err := manageErr(t, payload)

	wantErrContains(t, err, "invalid room ID")
}

func TestManagePropertiesRejectsMalformedPlayerID(t *testing.T) {
	payload := validManagePayload()
	payload["playerId"] = "not-an-object-id"

	err := manageErr(t, payload)

	wantErrContains(t, err, "invalid player ID")
}

func TestManagePropertiesRejectsNonArrayProperties(t *testing.T) {
	payload := validManagePayload()
	payload["properties"] = "not-an-array"

	err := manageErr(t, payload)

	wantErrContains(t, err, "invalid properties")
}

// --- handleTransfer ---

func TestTransferRejectsUnrecognizedTransferType(t *testing.T) {
	payload := validTransferPayload()
	payload["transferType"] = "SENT" // the plausible typo for "SEND"

	err := transferErr(t, payload)

	wantErrEqual(t, err, "invalid transfer type: SENT")
}

func TestTransferRejectsRequestTypeAsUnimplemented(t *testing.T) {
	// The REQUEST arm short-circuits here, which is why the
	// controllers.RequestTransfer stub (controllers/transferControllers.go:57)
	// is never reached from the websocket path.
	payload := validTransferPayload()
	payload["transferType"] = "REQUEST"

	err := transferErr(t, payload)

	wantErrEqual(t, err, "request transfers not implemented yet")
}

func TestTransferRejectsNonObjectPayload(t *testing.T) {
	err := transferErr(t, "not-an-object")

	wantErrEqual(t, err, "invalid payload format")
}

func TestTransferRejectsUnparseableAmount(t *testing.T) {
	payload := validTransferPayload()
	payload["amount"] = "one hundred"
	payload["transferType"] = "SENT"

	err := transferErr(t, payload)

	wantErrContains(t, err, "invalid amount")
}

func TestTransferRejectsMalformedRoomID(t *testing.T) {
	payload := validTransferPayload()
	payload["roomId"] = "not-an-object-id"
	payload["transferType"] = "SENT"

	err := transferErr(t, payload)

	wantErrContains(t, err, "invalid room ID")
}

func TestTransferRejectsMalformedFromPlayerID(t *testing.T) {
	payload := validTransferPayload()
	payload["fromPlayerId"] = "not-an-object-id"
	payload["toPlayerId"] = "507f1f77bcf86cd799439012"

	err := transferErr(t, payload)

	wantErrContains(t, err, "invalid fromPlayerId")
}

func TestTransferRejectsMalformedToPlayerID(t *testing.T) {
	payload := validTransferPayload()
	payload["fromPlayerId"] = "507f1f77bcf86cd799439012"
	payload["toPlayerId"] = "not-an-object-id"

	err := transferErr(t, payload)

	wantErrContains(t, err, "invalid toPlayerId")
}

// transferOutcome is freeParkingOutcome for the transfer handler, and it exists
// for the same reason: config.DB is a nil *mongo.Database in a test binary, so
// a SEND the amount floor accepts panics inside controllers.PlayerTransfer
// rather than erroring. That panic is the only signal available here that a
// payload was accepted rather than rejected, which is what makes the "not
// panicked" half of the assertions below meaningful - without it a test cannot
// tell a rejection from a value that sailed through into Mongo.
func transferOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handleTransfer(testClient(), Message{
		Type:    "TRANSFER",
		Payload: payload,
	})
	return err, false
}

// --- the transfer amount floor (board row 14) ---
//
// strconv.Atoi parses "-100" and PlayerTransfer's two $inc updates take their
// signs from the direction rather than from the amount, so before 2026-09-17 a
// negative SEND ran the transfer backwards: the sender was credited and the
// recipient debited, on an unauthenticated route, by one frame. These three
// pin the floor and where it sits.

func TestTransferRejectsNegativeSend(t *testing.T) {
	// The exploit as row 14 states it. Both player ids are valid here on
	// purpose: without the floor this payload reaches
	// controllers.PlayerTransfer and panics on the nil config.DB, so the
	// "not panicked" check is what proves the floor rejected it rather than
	// something further down.
	payload := validTransferPayload()
	payload["amount"] = "-100"
	payload["fromPlayerId"] = "507f1f77bcf86cd799439012"
	payload["toPlayerId"] = "507f1f77bcf86cd799439013"

	err, panicked := transferOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a transfer has to be at least $1")
}

func TestTransferRejectsZeroAmount(t *testing.T) {
	// $0 moves no money, so it is a noise bug rather than a money one: it
	// writes a Transfer row and toasts the whole room about a payment that did
	// not happen. Refused with the negatives, same as free parking and a bid.
	payload := validTransferPayload()
	payload["amount"] = "0"
	payload["fromPlayerId"] = "507f1f77bcf86cd799439012"
	payload["toPlayerId"] = "507f1f77bcf86cd799439013"

	err, panicked := transferOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a transfer has to be at least $1")
}

func TestTransferChecksTheAmountBeforeTheTransferType(t *testing.T) {
	// A mutation check on placement, not on behaviour. The floor sits directly
	// under the Atoi, above the transferType switch - so a payload that is
	// wrong in both ways reports the amount. Move the floor below the switch
	// and this test reports "invalid transfer type: SENT" instead; move it into
	// PlayerTransfer and the two tests above panic on the nil config.DB.
	payload := validTransferPayload()
	payload["amount"] = "-100"
	payload["transferType"] = "SENT"

	err, _ := transferOutcome(t, payload)

	wantErrEqual(t, err, "a transfer has to be at least $1")
}

// --- handleBankTransaction ---

// bankTransactionOutcome runs handleBankTransaction and reports either the
// error it returned or the fact that it panicked. config.DB is a nil
// *mongo.Database in a test binary, so any payload accepted by the
// transactionType switch panics inside controllers.GetPlayer
// (controllers/playerControllers.go:154) - that panic is the only signal
// available here that a value was accepted rather than rejected. If a seam is
// ever put in front of GetPlayer these tests stop panicking; change the two
// accept tests to assert on the error at that point.
func bankTransactionOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handleBankTransaction(testClient(), Message{
		Type:    "BANKER_TRANSACTION",
		Payload: payload,
	})
	return err, false
}

func TestBankTransactionRejectsUnrecognizedTransactionType(t *testing.T) {
	// The bug board item 1 exists for: before it, anything that was not
	// exactly "BANKER_ADD" was treated as a remove, so this typo took $100
	// off the player instead of erroring.
	payload := validBankPayload()
	payload["transactionType"] = "BANKER_REMOVED"

	err, panicked := bankTransactionOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid transaction type: BANKER_REMOVED")
}

func TestBankTransactionRejectsEmptyTransactionType(t *testing.T) {
	payload := validBankPayload()
	payload["transactionType"] = ""

	err, panicked := bankTransactionOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid transaction type: ")
}

func TestBankTransactionRejectsBeforeReadingThePlayer(t *testing.T) {
	// The point of hoisting the switch above controllers.GetPlayer: a bad
	// transactionType costs no database round trip. A panic here means the
	// switch has slipped back below the read.
	payload := validBankPayload()
	payload["transactionType"] = "NOPE"

	_, panicked := bankTransactionOutcome(t, payload)

	if panicked {
		t.Fatal("handleBankTransaction reached the database before validating transactionType")
	}
}

func TestBankTransactionAcceptsBankerAdd(t *testing.T) {
	payload := validBankPayload()
	payload["transactionType"] = "BANKER_ADD"

	err, panicked := bankTransactionOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected BANKER_ADD to be accepted and reach the database, got %v", err)
	}
}

func TestBankTransactionAcceptsBankerRemove(t *testing.T) {
	payload := validBankPayload()
	payload["transactionType"] = "BANKER_REMOVE"

	err, panicked := bankTransactionOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected BANKER_REMOVE to be accepted and reach the database, got %v", err)
	}
}

func TestBankTransactionRejectsNonObjectPayload(t *testing.T) {
	err, _ := bankTransactionOutcome(t, "not-an-object")

	wantErrEqual(t, err, "invalid payload format")
}

func TestBankTransactionRejectsUnparseableAmount(t *testing.T) {
	payload := validBankPayload()
	payload["amount"] = "one hundred"

	err, _ := bankTransactionOutcome(t, payload)

	wantErrContains(t, err, "invalid amount")
}

func TestBankTransactionRejectsMalformedTargetPlayerID(t *testing.T) {
	payload := validBankPayload()
	payload["toPlayerId"] = "not-an-object-id"

	err, _ := bankTransactionOutcome(t, payload)

	wantErrContains(t, err, "invalid target player ID")
}

func TestBankTransactionReportsMalformedRoomIDAsATargetPlayerError(t *testing.T) {
	// Documents a mislabel, not a blessing: websocketManager.go:406 reuses the
	// "invalid target player ID" message for the roomId branch. Raised in
	// HANDOFF 4; change the message and this test together.
	payload := validBankPayload()
	payload["roomId"] = "not-an-object-id"

	err, _ := bankTransactionOutcome(t, payload)

	wantErrContains(t, err, "invalid target player ID")
}

// --- the bank transaction amount floor (board row 15) ---
//
// strconv.Atoi parses "-100", and the direction actually applied comes from
// transactionType (BANKER_ADD/BANKER_REMOVE) rather than amount's sign - so
// before this floor existed, a negative amount silently reversed the real
// effect while the notification text kept describing transactionType's
// direction. A banker who fat-fingered a minus sign got a notification that
// lied about which way the money moved. Same shape as the transfer floor
// above (board row 14).

func TestBankTransactionRejectsNegativeAmount(t *testing.T) {
	// Both player-relevant ids are valid here on purpose: without the floor
	// this payload reaches controllers.GetPlayer and panics on the nil
	// config.DB, so the "not panicked" check is what proves the floor
	// rejected it rather than something further down.
	payload := validBankPayload()
	payload["amount"] = "-100"

	err, panicked := bankTransactionOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a bank transaction has to be at least $1")
}

func TestBankTransactionRejectsZeroAmount(t *testing.T) {
	// $0 moves no money, so it is a noise bug rather than a money one: it
	// still writes an event-history row and toasts the whole room about a
	// transaction that did not happen. Refused with the negatives, same as
	// transfer and free parking.
	payload := validBankPayload()
	payload["amount"] = "0"

	err, panicked := bankTransactionOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a bank transaction has to be at least $1")
}

func TestBankTransactionChecksTheAmountBeforeTheTransactionType(t *testing.T) {
	// A mutation check on placement, not on behaviour. The floor sits
	// directly under the Atoi, above every other field parse including the
	// transactionType switch - so a payload that is wrong in both ways
	// reports the amount. Move the floor below the switch and this test
	// reports "invalid transaction type: BANKER_ADDED" instead; move it into
	// bankTransactionRejection or past controllers.GetPlayer and this test
	// panics on the nil config.DB.
	payload := validBankPayload()
	payload["amount"] = "-100"
	payload["transactionType"] = "BANKER_ADDED"

	err, _ := bankTransactionOutcome(t, payload)

	wantErrEqual(t, err, "a bank transaction has to be at least $1")
}

// --- freeParking ---

// freeParkingOutcome is bankTransactionOutcome for the free parking handler,
// and it exists for the same reason: config.DB is a nil *mongo.Database in a
// test binary, so a freeParkingType the switch accepts panics rather than
// erroring. That panic is the only signal available here that a value was
// accepted rather than rejected.
//
// The panic site moved when row 13 was fixed. It used to be inside
// controllers.GetPlayer, which freeParking called above the session; the player
// read now happens inside the transaction, so the first nil-config.DB
// dereference on this path is config.DB.Client() at the StartSession call. The
// signal is unchanged and so are the assertions - but if a seam is ever put in
// front of either, these tests stop panicking, and the accept tests below
// should assert on the error at that point instead.
func freeParkingOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().freeParking(testClient(), Message{
		Type:    "FREE_PARKING",
		Payload: payload,
	})
	return err, false
}

func TestFreeParkingRejectsUnrecognizedType(t *testing.T) {
	// The bug this handler had: with no default arm, "COLLECT" ran the whole
	// Mongo transaction doing nothing, returned nil, and broadcast a
	// FREE_PARKING message whose notification was the empty string - every
	// client in the room toasting blank text for money that never moved.
	payload := validFreeParkingPayload()
	payload["freeParkingType"] = "COLLECT" // the plausible synonym for "REMOVE"

	err, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid free parking type: COLLECT")
}

func TestFreeParkingRejectsEmptyType(t *testing.T) {
	payload := validFreeParkingPayload()
	payload["freeParkingType"] = ""

	err, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid free parking type: ")
}

func TestFreeParkingRejectsLowercaseType(t *testing.T) {
	// The switch is case-sensitive and the frontend union is upper case
	// (frontend/types/payloads.ts). "add" is the shape a hand-rolled client
	// would most plausibly send.
	payload := validFreeParkingPayload()
	payload["freeParkingType"] = "add"

	err, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid free parking type: add")
}

func TestFreeParkingRejectsBeforeReadingThePlayer(t *testing.T) {
	// The point of hoisting the validation above controllers.GetPlayer: a bad
	// freeParkingType costs no database round trip. A panic here means the
	// validation has slipped back below the read.
	payload := validFreeParkingPayload()
	payload["freeParkingType"] = "NOPE"

	_, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("freeParking reached the database before validating freeParkingType")
	}
}

func TestFreeParkingAcceptsAdd(t *testing.T) {
	payload := validFreeParkingPayload()
	payload["freeParkingType"] = "ADD"

	err, panicked := freeParkingOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected ADD to be accepted and reach the database, got %v", err)
	}
}

func TestFreeParkingAcceptsRemove(t *testing.T) {
	payload := validFreeParkingPayload()
	payload["freeParkingType"] = "REMOVE"

	err, panicked := freeParkingOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected REMOVE to be accepted and reach the database, got %v", err)
	}
}

func TestFreeParkingRejectsNonObjectPayload(t *testing.T) {
	err, _ := freeParkingOutcome(t, "not-an-object")

	wantErrEqual(t, err, "invalid payload format")
}

func TestFreeParkingRejectsUnparseableAmount(t *testing.T) {
	payload := validFreeParkingPayload()
	payload["amount"] = "one hundred"

	err, _ := freeParkingOutcome(t, payload)

	wantErrContains(t, err, "invalid amount")
}

func TestFreeParkingRejectsMalformedPlayerID(t *testing.T) {
	payload := validFreeParkingPayload()
	payload["playerId"] = "not-an-object-id"

	err, _ := freeParkingOutcome(t, payload)

	wantErrContains(t, err, "invalid player ID")
}

// --- freeParking's amount floor ---
//
// strconv.Atoi parses a leading minus, and both arms of the transaction are
// built from $inc pairs whose signs belong to the arm rather than to the
// amount. So before this floor existed, a negative amount ran the arm in
// reverse and sailed past the arm's own insufficient-funds check, because that
// check compares against the same negative number. These four are the whole
// reason row 13's third bug needs no concurrency to reproduce: it is one frame.

func TestFreeParkingRejectsNegativeAdd(t *testing.T) {
	// The exploit as row 13 states it: ADD of -50 clears `player.Balance <
	// amount` for any balance, then increments the balance by +50 and Free
	// Parking by -50. A contribution that pays the contributor.
	payload := validFreeParkingPayload()
	payload["amount"] = "-50"
	payload["freeParkingType"] = "ADD"

	err, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a free parking amount has to be at least $1")
}

func TestFreeParkingRejectsNegativeRemove(t *testing.T) {
	// The worse half, and the one the row's write-up does not spell out: the
	// REMOVE arm has no balance check at all, so -1000 debits a player $1000
	// they do not have and credits the pot with it. Money out of nothing.
	payload := validFreeParkingPayload()
	payload["amount"] = "-1000"
	payload["freeParkingType"] = "REMOVE"

	err, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a free parking amount has to be at least $1")
}

func TestFreeParkingRejectsZeroAmount(t *testing.T) {
	// $0 moves no money, so it is not a money bug - it is a noise bug. It
	// writes an event-history row and toasts every client in the room about a
	// transfer that did not happen, which is the same class of thing board
	// item 3's empty notification was.
	payload := validFreeParkingPayload()
	payload["amount"] = "0"

	err, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a free parking amount has to be at least $1")
}

func TestFreeParkingChecksTheAmountBeforeTheFreeParkingType(t *testing.T) {
	// A mutation check on placement, not on behaviour. The floor sits above the
	// freeParkingType switch, which sits above the session - so a payload that
	// is wrong in both ways reports the amount, and a floor moved down into the
	// transaction would report the type instead. Moving it below the session
	// start would panic on the nil config.DB and fail the two tests above.
	payload := validFreeParkingPayload()
	payload["amount"] = "-50"
	payload["freeParkingType"] = "COLLECT"

	err, _ := freeParkingOutcome(t, payload)

	wantErrEqual(t, err, "a free parking amount has to be at least $1")
}

// --- handlePropertyPurchase's price floor (board row 24) ---
//
// The price arrives as a bare JSON number and, until 2026-09-22, nothing
// anywhere looked at it: controllers.PurchaseProperty $inc'd the balance by
// -price with no read, no session and no check, so a negative price credited
// the buyer AND handed them the deed in one unauthenticated frame. The floor
// below is the payload half of the fix; the state half - a frozen buyer, a deed
// already sold, a balance that cannot cover the price - is
// controllers.propertyPurchaseRejection, tested in
// controllers/propertyControllers_test.go because it needs values only the
// transaction can read.

// purchaseOutcome is freeParkingOutcome for the purchase handler, and it exists
// for the same reason: a price the floor accepts reaches
// controllers.PurchaseProperty and panics on the nil config.DB at
// config.DB.Client() rather than erroring. That panic is the only signal
// available here that a payload was accepted rather than rejected, which is
// what makes the "not panicked" half of the assertions below mean anything.
func purchaseOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handlePropertyPurchase(testClient(), Message{
		Type:    "PURCHASE_PROPERTY",
		Payload: payload,
	})
	return err, false
}

func TestPropertyPurchaseRejectsANegativePrice(t *testing.T) {
	// The exploit as row 24 states it: -1000 pays the buyer $1000 out of the
	// Bank and gives them the deed. Both ids are valid here on purpose, so
	// without the floor this payload reaches Mongo and panics - the "not
	// panicked" check is what proves the floor turned it away rather than
	// something further down.
	payload := validPurchasePayload()
	payload["price"] = float64(-1000)

	err, panicked := purchaseOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a property purchase has to be at least $1")
}

func TestPropertyPurchaseRejectsAZeroPrice(t *testing.T) {
	// $0 moves no money, so it is a noise bug rather than a money one - it
	// writes an event-history row and toasts the whole room about a purchase
	// that cost nothing. Refused with the negatives, same as a transfer, a free
	// parking amount and a bid.
	payload := validPurchasePayload()
	payload["price"] = float64(0)

	err, panicked := purchaseOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a property purchase has to be at least $1")
}

func TestPropertyPurchaseRejectsAFractionalPriceUnderADollar(t *testing.T) {
	// int(priceFloat) truncates toward zero, so 0.99 arrives at the floor as 0.
	// Pinned because the truncation is invisible at the call site: a floor
	// written as `priceFloat < 1` and a floor written after the conversion agree
	// here, but one written as `price != 0` would let this through and buy a
	// deed for nothing.
	payload := validPurchasePayload()
	payload["price"] = float64(0.99)

	err, panicked := purchaseOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the database, got a panic")
	}
	wantErrEqual(t, err, "a property purchase has to be at least $1")
}

func TestPropertyPurchaseChecksThePriceBeforeTheIDs(t *testing.T) {
	// A mutation check on placement, not on behaviour. The floor sits directly
	// under the price conversion, above both id parses - so a payload that is
	// wrong in both ways reports the price. Move the floor below the parses and
	// this reports the buyer id instead; move it out of the handler entirely and
	// the three tests above panic on the nil config.DB.
	payload := validPurchasePayload()
	payload["price"] = float64(-1000)
	payload["buyerId"] = "not-an-object-id"

	err, _ := purchaseOutcome(t, payload)

	wantErrEqual(t, err, "a property purchase has to be at least $1")
}

func TestPropertyPurchaseAcceptsAPriceOfADollar(t *testing.T) {
	// The bottom of the accepted range, and the proof the floor is a floor
	// rather than a wall: $1 clears it and reaches the database, where there is
	// no Mongo to serve it. Whether the buyer can afford that $1, and whether
	// the deed is still for sale, are decided inside the transaction - see
	// controllers/propertyControllers_test.go.
	payload := validPurchasePayload()
	payload["price"] = float64(1)

	err, panicked := purchaseOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected a $1 price to be accepted and reach the database, got %v", err)
	}
}

func TestPropertyPurchaseAcceptsAFacePrice(t *testing.T) {
	// The ordinary frame the frontend sends: the property's own price, off the
	// available-properties list (frontend/components/players/
	// purchase-properties-bank.tsx:54). It must still reach the database.
	err, panicked := purchaseOutcome(t, validPurchasePayload())

	if !panicked {
		t.Fatalf("expected a valid purchase to reach the database, got %v", err)
	}
}

// --- wrong-typed payload fields ---
//
// Every payload field these handlers read as a string used to be a single-value
// type assertion, so a field that was absent or of another JSON type panicked
// rather than erroring. gin.Default() installs Recovery so the process survived,
// but the deferred cleanup in handler.go:54-64 ran first: the socket closed and
// the player was bounced out of the room (HANDOFF 3). The cases below are one
// per handler, and each asserts the rejection happens before any Mongo access.

func TestBankTransactionRejectsNonStringTransactionType(t *testing.T) {
	payload := validBankPayload()
	payload["transactionType"] = float64(1)

	err, panicked := bankTransactionOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection, got a panic")
	}
	wantErrEqual(t, err, "invalid payload: expected string for transactionType")
}

func TestFreeParkingRejectsMissingFreeParkingType(t *testing.T) {
	// An absent field asserts to the zero value of interface{}, which is not a
	// string - the same branch a wrong-typed field takes.
	payload := validFreeParkingPayload()
	delete(payload, "freeParkingType")

	err, panicked := freeParkingOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection, got a panic")
	}
	wantErrEqual(t, err, "invalid payload: expected string for freeParkingType")
}

func TestManagePropertiesRejectsNonStringManagementType(t *testing.T) {
	payload := validManagePayload()
	payload["managementType"] = []any{"HOUSES"}

	err := manageErr(t, payload)

	wantErrEqual(t, err, "invalid payload: expected string for managementType")
}

func TestTransferRejectsNonStringReason(t *testing.T) {
	payload := validTransferPayload()
	payload["reason"] = float64(42)

	err := transferErr(t, payload)

	wantErrEqual(t, err, "invalid payload: expected string for reason")
}

func TestPropertyPurchaseRejectsNonStringBuyerID(t *testing.T) {
	payload := validPurchasePayload()
	payload["buyerId"] = true

	err := purchaseErr(t, payload)

	wantErrEqual(t, err, "invalid payload: expected string for buyerId")
}

// --- Broadcast: refuses a notification nobody wrote (board item 8) ---
//
// Broadcast is pure enough to test directly: with no clients registered for
// the room, there is nothing to deliver to, so the only thing left to
// observe is whether the guard's refusal fired. It logs on refusal
// (log.Printf) and does nothing observable on success, so these tests
// capture log output as the signal - the same shape of trick
// bankTransactionOutcome/freeParkingOutcome use above, where a panic is the
// only available signal that a value was accepted.

// broadcastLogOutput runs Broadcast against a room with no registered
// clients and returns whatever it wrote to the log during the call.
func broadcastLogOutput(t *testing.T, message Message) string {
	t.Helper()
	var buf bytes.Buffer
	prevOutput := log.Writer()
	prevFlags := log.Flags()
	log.SetOutput(&buf)
	log.SetFlags(0)
	defer func() {
		log.SetOutput(prevOutput)
		log.SetFlags(prevFlags)
	}()

	NewRoomManager().Broadcast("ROOM1", message)

	return buf.String()
}

func TestBroadcastRefusesEmptyNotification(t *testing.T) {
	output := broadcastLogOutput(t, Message{
		Type:    "FREE_PARKING",
		Payload: map[string]interface{}{"notification": ""},
	})

	if !strings.Contains(output, "Broadcast refused") {
		t.Fatalf("expected a refusal to be logged, got %q", output)
	}
}

func TestBroadcastSendsNonEmptyNotification(t *testing.T) {
	output := broadcastLogOutput(t, Message{
		Type:    "FREE_PARKING",
		Payload: map[string]interface{}{"notification": "Zach added $100 to Free Parking"},
	})

	if output != "" {
		t.Fatalf("expected no refusal to be logged, got %q", output)
	}
}

func TestBroadcastSendsPayloadWithNoNotificationKey(t *testing.T) {
	// A payload that never promises a notification at all - a future message
	// type that isn't a human-readable toast, say - is not this guard's
	// concern; it must pass through unrefused.
	output := broadcastLogOutput(t, Message{
		Type:    "SOME_OTHER_EVENT",
		Payload: map[string]interface{}{"playerId": "507f1f77bcf86cd799439012"},
	})

	if output != "" {
		t.Fatalf("expected a payload with no notification key to pass through unrefused, got %q", output)
	}
}

func TestBroadcastRefusesEmptyNotificationInStringMap(t *testing.T) {
	// handler.go's PLAYER_LEFT builds its payload as map[string]string, not
	// map[string]interface{} like every other broadcast site - the guard has
	// to recognize this shape too, or it silently never applies to that site.
	output := broadcastLogOutput(t, Message{
		Type: "PLAYER_LEFT",
		Payload: map[string]string{
			"playerId":     "507f1f77bcf86cd799439012",
			"notification": "",
		},
	})

	if !strings.Contains(output, "Broadcast refused") {
		t.Fatalf("expected a refusal to be logged, got %q", output)
	}
}

func TestBroadcastSendsNonEmptyNotificationInStringMap(t *testing.T) {
	output := broadcastLogOutput(t, Message{
		Type: "PLAYER_LEFT",
		Payload: map[string]string{
			"playerId":     "507f1f77bcf86cd799439012",
			"notification": "Zach has left the game",
		},
	})

	if output != "" {
		t.Fatalf("expected no refusal to be logged, got %q", output)
	}
}

func TestBroadcastRefusesNonStringNotification(t *testing.T) {
	// A "notification" key that isn't a string can't be shown as a toast
	// either - the guard treats it the same as empty rather than trusting it.
	output := broadcastLogOutput(t, Message{
		Type:    "FREE_PARKING",
		Payload: map[string]interface{}{"notification": 42},
	})

	if !strings.Contains(output, "Broadcast refused") {
		t.Fatalf("expected a non-string notification to be refused, got %q", output)
	}
}

func TestBroadcastDoesNotPanicOnNonMapPayload(t *testing.T) {
	// The ERROR path writes a bare string payload directly with WriteJSON,
	// never through Broadcast - but nothing stops a future caller from
	// routing one here, and the guard's type switch must not panic on it.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Broadcast panicked on a non-map payload: %v", r)
		}
	}()

	output := broadcastLogOutput(t, Message{Type: "ERROR", Payload: "not-a-map"})

	if output != "" {
		t.Fatalf("expected a non-map payload to pass through unrefused, got %q", output)
	}
}

func TestBroadcastDoesNotPanicOnNilPayload(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Broadcast panicked on a nil payload: %v", r)
		}
	}()

	broadcastLogOutput(t, Message{Type: "ERROR", Payload: nil})
}

// --- emptyNotification ---

func TestEmptyNotification(t *testing.T) {
	cases := []struct {
		name         string
		payload      interface{}
		wantEmpty    bool
		wantHasField bool
	}{
		{
			name:         "map[string]interface{} with empty notification",
			payload:      map[string]interface{}{"notification": ""},
			wantEmpty:    true,
			wantHasField: true,
		},
		{
			name:         "map[string]interface{} with non-empty notification",
			payload:      map[string]interface{}{"notification": "Zach sent $50 to Alex for rent"},
			wantEmpty:    false,
			wantHasField: true,
		},
		{
			name:         "map[string]interface{} with no notification key",
			payload:      map[string]interface{}{"playerId": "507f1f77bcf86cd799439012"},
			wantEmpty:    false,
			wantHasField: false,
		},
		{
			name:         "map[string]interface{} with non-string notification",
			payload:      map[string]interface{}{"notification": 42},
			wantEmpty:    true,
			wantHasField: true,
		},
		{
			name:         "map[string]string with empty notification",
			payload:      map[string]string{"notification": ""},
			wantEmpty:    true,
			wantHasField: true,
		},
		{
			name:         "map[string]string with non-empty notification",
			payload:      map[string]string{"notification": "Zach has left the game"},
			wantEmpty:    false,
			wantHasField: true,
		},
		{
			name:         "map[string]string with no notification key",
			payload:      map[string]string{"playerId": "507f1f77bcf86cd799439012"},
			wantEmpty:    false,
			wantHasField: false,
		},
		{
			name:         "non-map payload",
			payload:      "not-a-map",
			wantEmpty:    false,
			wantHasField: false,
		},
		{
			name:         "nil payload",
			payload:      nil,
			wantEmpty:    false,
			wantHasField: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			empty, hasField := emptyNotification(tc.payload)
			if empty != tc.wantEmpty || hasField != tc.wantHasField {
				t.Fatalf("emptyNotification(%#v) = (%v, %v), want (%v, %v)", tc.payload, empty, hasField, tc.wantEmpty, tc.wantHasField)
			}
		})
	}
}

// --- handleKickPlayer ---
//
// A kick is the first action in this app that deliberately closes someone
// else's socket, and the first whose failure mode is "green gates, nobody
// removed". The rejections below are all of the ones reachable without Mongo -
// everything that is a fact about the payload rather than about the room - and
// TestKickRejectsBeforeReadingThePlayer is the guard that keeps them that way.
// The document-level rules (the target must be in this room and still active,
// a banker target needs a successor, the successor must be a live player in the
// same room) cannot be reached here at all: they need a read, and a read is a
// panic on this machine. They rest on the device walk in PLAN.md's done-when.

// validKickPayload is a KICK_PLAYER payload that gets as far as the first
// database call. Each test below changes exactly one field. The target is
// deliberately not testClient()'s own PlayerID, so that a self-kick (D12) is
// something a test has to opt into rather than something every test does.
func validKickPayload() map[string]any {
	return map[string]any{
		"roomId":         "507f1f77bcf86cd799439011",
		"targetPlayerId": "507f1f77bcf86cd799439013",
		"disposition":    "BANK",
	}
}

// kickOutcome is bankTransactionOutcome for the kick handler, and it exists for
// the same reason: config.DB is a nil *mongo.Database in a test binary, so a
// payload the validation accepts panics at the first config.DB.Collection call
// rather than returning an error. That panic is the only signal available here
// that a payload was accepted rather than rejected. If a seam is ever put in
// front of the target read these tests stop panicking; change the accept tests
// to assert on the error at that point.
func kickOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handleKickPlayer(testClient(), Message{
		Type:    "KICK_PLAYER",
		Payload: payload,
	})
	return err, false
}

func TestKickRejectsNonObjectPayload(t *testing.T) {
	err, _ := kickOutcome(t, "not-an-object")
	wantErrEqual(t, err, "invalid payload format")
}

func TestKickRejectsMissingRoomID(t *testing.T) {
	payload := validKickPayload()
	delete(payload, "roomId")

	err, _ := kickOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for roomId")
}

func TestKickRejectsMalformedRoomID(t *testing.T) {
	payload := validKickPayload()
	payload["roomId"] = "not-an-object-id"

	err, _ := kickOutcome(t, payload)
	wantErrContains(t, err, "invalid room ID")
}

func TestKickRejectsMissingTargetPlayerID(t *testing.T) {
	payload := validKickPayload()
	delete(payload, "targetPlayerId")

	err, _ := kickOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for targetPlayerId")
}

func TestKickRejectsNonStringTargetPlayerID(t *testing.T) {
	payload := validKickPayload()
	payload["targetPlayerId"] = 12345

	err, _ := kickOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for targetPlayerId")
}

func TestKickRejectsMalformedTargetPlayerID(t *testing.T) {
	payload := validKickPayload()
	payload["targetPlayerId"] = "507f1f77bcf86cd79943901"

	err, _ := kickOutcome(t, payload)
	wantErrContains(t, err, "invalid target player ID")
}

func TestKickRejectsMissingDisposition(t *testing.T) {
	payload := validKickPayload()
	delete(payload, "disposition")

	err, _ := kickOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for disposition")
}

func TestKickRejectsNonStringDisposition(t *testing.T) {
	payload := validKickPayload()
	payload["disposition"] = true

	err, _ := kickOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for disposition")
}

func TestKickRejectsUnrecognizedDisposition(t *testing.T) {
	// The shape of the bug board item 3 existed for, one handler along: a
	// switch with no default arm runs the whole transaction doing nothing and
	// broadcasts an empty notification. Here the equivalent would be worse - a
	// player marked gone with an estate nobody disposed of.
	payload := validKickPayload()
	payload["disposition"] = "BANKK"

	err, panicked := kickOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid disposition: BANKK")
}

func TestKickRejectsEmptyDisposition(t *testing.T) {
	payload := validKickPayload()
	payload["disposition"] = ""

	err, panicked := kickOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid disposition: ")
}

func TestKickRejectsLowercaseDisposition(t *testing.T) {
	payload := validKickPayload()
	payload["disposition"] = "bank"

	err, panicked := kickOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid disposition: bank")
}

func TestKickAcceptsAuctionDisposition(t *testing.T) {
	// This test is the inverse of the one it replaces. Until Phase 3 existed,
	// AUCTION was refused here - accepting it would have marked a player gone
	// and left their estate in limbo, which is worse than refusing the action.
	// The auction exists now, so the guard has to let it through to the read.
	payload := validKickPayload()
	payload["disposition"] = "AUCTION"

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected AUCTION to be accepted and reach the database, got %v", err)
	}
}

func TestKickRejectsNonStringSuccessorPlayerID(t *testing.T) {
	payload := validKickPayload()
	payload["successorPlayerId"] = 42

	err, _ := kickOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for successorPlayerId")
}

func TestKickRejectsMalformedSuccessorPlayerID(t *testing.T) {
	payload := validKickPayload()
	payload["successorPlayerId"] = "nope"

	err, _ := kickOutcome(t, payload)
	wantErrContains(t, err, "invalid successor player ID")
}

func TestKickRejectsSuccessorEqualToTarget(t *testing.T) {
	// Promoting the player being removed would leave the room bankerless with
	// every write reporting success.
	payload := validKickPayload()
	payload["successorPlayerId"] = payload["targetPlayerId"]

	err, panicked := kickOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "the successor cannot be the player being removed")
}

func TestKickRejectsBeforeReadingThePlayer(t *testing.T) {
	// The point of validating the whole payload above the first database call:
	// a bad disposition costs no round trip, and - the reason that matters on
	// this machine - a rejection below the read is not reachable from a test at
	// all, because config.DB is nil and the read panics. A panic here means the
	// validation has slipped below the read.
	payload := validKickPayload()
	payload["disposition"] = "NOPE"

	_, panicked := kickOutcome(t, payload)

	if panicked {
		t.Fatal("handleKickPlayer reached the database before validating the payload")
	}
}

func TestKickAcceptsBankDisposition(t *testing.T) {
	payload := validKickPayload()
	payload["disposition"] = "BANK"

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected BANK to be accepted and reach the database, got %v", err)
	}
}

func TestKickAcceptsFreezeDisposition(t *testing.T) {
	payload := validKickPayload()
	payload["disposition"] = "FREEZE"

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected FREEZE to be accepted and reach the database, got %v", err)
	}
}

func TestKickAcceptsANamedSuccessor(t *testing.T) {
	payload := validKickPayload()
	payload["successorPlayerId"] = "507f1f77bcf86cd799439014"

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected a well-formed successor to be accepted and reach the database, got %v", err)
	}
}

func TestKickAcceptsAnAbsentSuccessor(t *testing.T) {
	// successorPlayerId is required only when the target holds the banker role,
	// which is not knowable without reading them. Its absence must not be a
	// payload-level rejection.
	payload := validKickPayload()

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected a kick with no successor to reach the database, got %v", err)
	}
}

func TestKickTreatsEmptySuccessorAsAbsent(t *testing.T) {
	// A picker with nothing chosen sends "" rather than omitting the key.
	payload := validKickPayload()
	payload["successorPlayerId"] = ""

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected an empty successor to read as absent and reach the database, got %v", err)
	}
}

func TestKickTreatsNullSuccessorAsAbsent(t *testing.T) {
	// JSON null decodes to a nil interface, not to a missing key.
	payload := validKickPayload()
	payload["successorPlayerId"] = nil

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected a null successor to read as absent and reach the database, got %v", err)
	}
}

func TestKickAcceptsTheCallerAsTheTarget(t *testing.T) {
	// A banker kicking themselves is a valid action on the same path, not an
	// error: it is the only exit a banker has, and without it the only way out
	// is deleting the whole room.
	payload := validKickPayload()
	payload["targetPlayerId"] = testClient().PlayerID

	err, panicked := kickOutcome(t, payload)

	if !panicked {
		t.Fatalf("expected a self-kick to be accepted and reach the database, got %v", err)
	}
}

// --- kickNotification ---

func TestKickNotificationBankArm(t *testing.T) {
	got := kickNotification("Claude", "BANK", "", 0)
	want := "Banker removed Claude from the game. Their properties returned to the Bank."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestKickNotificationFreezeArm(t *testing.T) {
	got := kickNotification("Claude", "FREEZE", "", 0)
	want := "Banker removed Claude from the game. Their properties stay where they are."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestKickNotificationAuctionArm(t *testing.T) {
	got := kickNotification("Claude", "AUCTION", "", 2)
	want := "Banker removed Claude from the game. Their properties go up for auction."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestKickNotificationAuctionArmWithNothingToAuction(t *testing.T) {
	// A player holding no deeds kicked with AUCTION: no auction is opened, so
	// the sentence must not promise one. Saying "their properties go up for
	// auction" here would be the only line in the room that no lot ever
	// follows, and there is no later message to correct it.
	got := kickNotification("Claude", "AUCTION", "", 0)
	want := "Banker removed Claude from the game. They had no properties to auction."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestKickNotificationAppendsTheSuccession(t *testing.T) {
	got := kickNotification("Claude", "BANK", "Zach", 0)
	want := "Banker removed Claude from the game. Their properties returned to the Bank. Zach is now the Banker."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestKickNotificationSurvivesBroadcastsEmptyGuard(t *testing.T) {
	// Broadcast drops a payload whose notification is present and empty,
	// silently to the room and loudly only in the VM's log. A kick that worked
	// and that nobody saw is indistinguishable from a kick that did nothing, so
	// every arm - including the unreachable default - has to produce text.
	for _, disposition := range []string{"BANK", "FREEZE", "AUCTION", ""} {
		for _, successor := range []string{"", "Zach"} {
			for _, lotCount := range []int{0, 1, 4} {
				notification := kickNotification("Claude", disposition, successor, lotCount)
				empty, hasField := emptyNotification(map[string]interface{}{
					"notification": notification,
					"playerId":     "507f1f77bcf86cd799439013",
				})
				if !hasField || empty {
					t.Fatalf("disposition %q, successor %q, lots %d: Broadcast would drop %q", disposition, successor, lotCount, notification)
				}
			}
		}
	}
}

// --- eventTypeFor ---

func TestKickEventIconIsNotTheBankIcon(t *testing.T) {
	// The kick's notification has "Banker" as its subject, so an arm added
	// anywhere below the bank arm would never be reached and the row would be
	// drawn exactly like a balance change. This is the test that catches a
	// reorder of that switch.
	got := eventTypeFor(kickNotification("Claude", "BANK", "", 0))
	want := []string{"#dc2626", "\U0001f6ab"}

	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestKickEventIconHoldsForEveryArmOfTheCopy(t *testing.T) {
	// AUCTION is in this list with both lot counts: its sentence is the one
	// that does not end in "returned to the Bank", and "from the game" is what
	// has to keep carrying the icon.
	for _, disposition := range []string{"BANK", "FREEZE", "AUCTION"} {
		for _, successor := range []string{"", "Zach"} {
			for _, lotCount := range []int{0, 3} {
				notification := kickNotification("Claude", disposition, successor, lotCount)
				if got := eventTypeFor(notification); got[1] != "\U0001f6ab" {
					t.Fatalf("disposition %q, successor %q, lots %d: got %v for %q", disposition, successor, lotCount, got, notification)
				}
			}
		}
	}
}

func TestEventTypeForLeavesTheExistingArmsAlone(t *testing.T) {
	// The kick arm is matched first, so every one of these is also a check that
	// it did not start swallowing rows that are not kicks.
	cases := []struct {
		name         string
		notification string
		want         []string
	}{
		{"banker balance change", "Banker has removed $100 from Claude's balance", []string{"#6366f1", "\U0001f3e6"}},
		{"purchase", "Claude purchased Boardwalk from the Bank", []string{"#10b981", "\U0001f3e0"}},
		{"free parking", "Claude added $200 to Free Parking", []string{"#f59e0b", "\U0001f17f️"}},
		{"transfer", "Claude just sent $50 to Zach for rent", []string{"#3b82f6", "\U0001f4b8"}},
		{"mortgage", "Claude received $110 for mortgaging property", []string{"#ef4444", "\U0001f4c4"}},
		{"unrecognized", "something nobody wrote an arm for", []string{"#6b7280", "ℹ️"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := eventTypeFor(tc.notification)
			if len(got) != len(tc.want) || got[0] != tc.want[0] || got[1] != tc.want[1] {
				t.Fatalf("eventTypeFor(%q) = %v, want %v", tc.notification, got, tc.want)
			}
		})
	}
}

// --- board row 19: a frozen player may not move money, at every site ---
//
// Board row 16, decided by Zach 2026-09-17: a removed (isActive:false) player
// may not move money, and the rule is answered once rather than per site. Row
// 14 could only honour that inside controllers.PlayerTransfer, which is the
// handler it owned. These are the other four money handlers.
//
// Each rule is a pure function taking a player document already read, for the
// reason the whole file is built around: config.DB is a nil *mongo.Database in
// a test binary, so every one of the four reads panics rather than returning,
// and a rule written inline at any of them would be a rule no test on this
// machine can reach. Pulling them out is what makes these assertions exist.
//
// Three of the four refuse the ACTOR - the player spending, contributing or
// managing is the one named in the payload. bankTransactionRejection is the
// fourth and it is different: the banker is acting and the frozen player is
// what they are acting ON. That is a separate decision, taken separately, and
// its own test below says so.

func activePlayer() models.Player {
	return models.Player{Name: "Zach", Balance: 1500, IsActive: true}
}

func removedPlayer() models.Player {
	return models.Player{Name: "Claude", Balance: 1500, IsActive: false}
}

func wantNoRejection(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected an active player to be accepted, got %q", err.Error())
	}
}

// --- freeParkingRejection (the actor) ---

func TestFreeParkingRefusesARemovedPlayer(t *testing.T) {
	// Reachable from a real client, not only from a hand-made frame: a frozen
	// player is still on screen, because GetPlayersInRoom deliberately does
	// not filter on isActive (invariant 6). Free Parking's REMOVE arm is the
	// one that loses real money - it takes the pot into a balance that has
	// already left the game.
	wantErrEqual(t, freeParkingRejection(removedPlayer()), "a removed player cannot use free parking")
}

func TestFreeParkingAcceptsAnActivePlayer(t *testing.T) {
	wantNoRejection(t, freeParkingRejection(activePlayer()))
}

func TestFreeParkingRefusesOnTheFlagAndNotTheBalance(t *testing.T) {
	// The rule is about standing, not about money: a removed player with a
	// full balance is still refused, and an active player with nothing is not
	// refused HERE - the ADD arm's own floor is what turns them away, inside
	// the transaction, where it can see the amount.
	wantErrEqual(t, freeParkingRejection(models.Player{Name: "Claude", Balance: 99999}), "a removed player cannot use free parking")
	wantNoRejection(t, freeParkingRejection(models.Player{Name: "Zach", Balance: 0, IsActive: true}))
}

// --- managePropertiesRejection (the actor) ---

func TestManagePropertiesRefusesARemovedPlayer(t *testing.T) {
	wantErrEqual(t, managePropertiesRejection(removedPlayer()), "a removed player cannot manage properties")
}

func TestManagePropertiesAcceptsAnActivePlayer(t *testing.T) {
	wantNoRejection(t, managePropertiesRejection(activePlayer()))
}

// --- propertyPurchaseRejection (the actor) ---
//
// Moved, not dropped. On 2026-09-22 the purchase rule grew a price floor, a
// balance check and a still-for-sale check, which need the deed and the buyer
// as the transaction re-reads them - so the rule moved into
// controllers.PurchaseProperty's transaction, and this package cannot be
// imported from there. The frozen-buyer assertion that stood here is
// TestPurchaseRejectionRefusesARemovedBuyer in
// controllers/propertyControllers_test.go, with its sentence unchanged. What is
// still tested here is the price floor, at the top of this file with the rest
// of the payload-shape checks, because that one stayed in the handler.

// --- bankTransactionRejection (the target, which is the different one) ---

func TestBankTransactionRefusesARemovedTarget(t *testing.T) {
	// This is the half of row 19 that is NOT "a frozen player may not act" -
	// the actor here is the banker and the banker is fine. What is refused is
	// moving money onto a document that has already left the game, where the
	// write means nothing and the broadcast says it meant something.
	wantErrEqual(t, bankTransactionRejection(removedPlayer()), "that player has been removed from the game")
}

func TestBankTransactionAcceptsAnActiveTarget(t *testing.T) {
	wantNoRejection(t, bankTransactionRejection(activePlayer()))
}

func TestBankTransactionSaysTheSameThingAsTransferDoesAboutARemovedRecipient(t *testing.T) {
	// The wording is a deliberate echo of transferRejection's recipient arm
	// (controllers/transferControllers.go), because it is the same situation
	// reached by a different route, and a player should not have to learn two
	// sentences for it. This pins that on purpose: if one is reworded and the
	// other is not, this is what says so.
	wantErrEqual(t, bankTransactionRejection(removedPlayer()), "that player has been removed from the game")
}

// --- the hoist that row 19's manage-properties rule needed ---
//
// The player read in handleManageProperties used to sit at the BOTTOM of the
// handler, after the deeds had been rewritten and the balance moved, and it
// existed only to put a name in the notification. A refusal there refuses
// nothing, so row 19 moved the read above both writes. That in turn forced the
// managementType check up above the read: the check used to live in the work
// switch's own default arm, which is now below the first nil-config.DB
// dereference and would panic before it could name the bad value.

// manageOutcome is freeParkingOutcome for the manage-properties handler, and it
// exists for the same reason: a payload that clears every payload-shape check
// now reaches controllers.GetPlayer and panics on the nil config.DB rather than
// erroring. The "not panicked" half is what proves a rejection fired above the
// read rather than something further down.
func manageOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handleManageProperties(testClient(), Message{
		Type:    "MANAGE_PROPERTIES",
		Payload: payload,
	})
	return err, false
}

func TestManagePropertiesChecksTheManagementTypeBeforeReadingThePlayer(t *testing.T) {
	// Everything else in this payload is valid, so without the hoisted switch
	// the handler reaches the player read and panics instead of naming the bad
	// value. That panic is what this test is really guarding against: it is
	// what the three managementType tests at the top of this file would start
	// doing if the check ever slipped back down into the work switch.
	payload := validManagePayload()
	payload["managementType"] = "HOUSE"

	err, panicked := manageOutcome(t, payload)

	if panicked {
		t.Fatal("expected a rejection before the player read, got a panic")
	}
	wantErrEqual(t, err, "invalid management type: HOUSE")
}

// managePanicStack runs the handler on a payload that clears every shape check
// and returns the stack of the panic it takes on the nil config.DB, or "" if it
// did not panic.
//
// This is a step past what the other outcome helpers do, and it is here because
// nothing weaker can see the thing row 19 actually changed in this handler. The
// deed writes and the player read BOTH panic on the nil config.DB, so "it
// panicked" cannot tell which of them the handler reached first - and which it
// reaches first is the whole point of moving the read up. The stack can tell
// them apart. Verified by mutation: with the read put back where it used to
// be, below the writes, every other test in this file stays green and only
// this one goes red.
//
// The cost is that this test knows the name of a function in another package.
// If controllers.GetPlayer is renamed, or a seam is put in front of it, this
// goes red and the fix is to name whatever now stands in for the read - not to
// delete the assertion.
func managePanicStack(t *testing.T, payload any) (stack string) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			stack = string(debug.Stack())
		}
	}()
	NewRoomManager().handleManageProperties(testClient(), Message{
		Type:    "MANAGE_PROPERTIES",
		Payload: payload,
	})
	return ""
}

func TestManagePropertiesReadsThePlayerBeforeTouchingTheDeeds(t *testing.T) {
	// Until 2026-09-17 this handler rewrote the deeds and moved the balance
	// and only then read the player, to get a name for the notification. Row
	// 19 put a rule on that read, and a rule that runs after the money has
	// moved is not a rule. This is what holds the read above the writes.
	stack := managePanicStack(t, validManagePayload())

	if stack == "" {
		t.Fatal("expected a valid payload to reach Mongo and panic on the nil config.DB")
	}
	if !strings.Contains(stack, "controllers.GetPlayer") {
		t.Fatalf("expected the handler to panic at the player read; it got further first:\n%s", stack)
	}
	if strings.Contains(stack, "HandleHouseManagement") {
		t.Fatalf("the deeds were rewritten before the player was read:\n%s", stack)
	}
}
