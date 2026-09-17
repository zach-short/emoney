package websocket

import (
	"strings"
	"testing"
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

// --- freeParking ---

// freeParkingOutcome is bankTransactionOutcome for the free parking handler,
// and it exists for the same reason: config.DB is a nil *mongo.Database in a
// test binary, so a freeParkingType the switch accepts panics inside
// controllers.GetPlayer (controllers/playerControllers.go:154). That panic is
// the only signal available here that a value was accepted rather than
// rejected. If a seam is ever put in front of GetPlayer these tests stop
// panicking; change the two accept tests to assert on the error at that point.
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
