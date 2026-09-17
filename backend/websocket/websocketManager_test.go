package websocket

import (
	"bytes"
	"log"
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
