package controllers

import (
	"testing"

	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// These reach exactly one thing - propertyPurchaseRejection - and that is the
// whole reason it is a pure function taking documents already read: config.DB
// is a nil *mongo.Database in a test binary, so every other line of
// PurchaseProperty panics on the nil dereference at config.DB.Client() instead
// of running. Nothing here proves a valid purchase moves the right money or
// that the two writes commit together; it proves an invalid one is refused
// before either write can happen. Same boundary transferRejection's tests sit
// on, and the same one HANDOFF 4 describes.
//
// The frozen-buyer case below is board row 19's assertion, moved here from
// websocket/websocketManager_test.go with its sentence unchanged when the rule
// moved into the transaction on 2026-09-22.

// forSale is a deed the Bank still holds. playerId is the zero ObjectID, which
// is what the driver decodes both an explicit null and an absent key into - the
// Go side of the `playerId: nil` filter GetAvailableProperties uses to decide
// what is purchasable at all.
func forSale(name string, price int) models.Property {
	return models.Property{ID: primitive.NewObjectID(), Name: name, Price: price}
}

// owned is the same deed after somebody has bought it.
func owned(name string, price int) models.Property {
	p := forSale(name, price)
	p.PlayerID = primitive.NewObjectID()
	return p
}

func TestPurchaseRejectionAcceptsAnAffordablePurchase(t *testing.T) {
	if err := propertyPurchaseRejection(active(500), forSale("Boardwalk", 400), 400); err != nil {
		t.Fatalf("expected no rejection, got %q", err)
	}
}

func TestPurchaseRejectionAcceptsSpendingTheWholeBalance(t *testing.T) {
	// The boundary, pinned deliberately and matching transferRejection: the
	// check is `balance < price`, so $400 of a $400 balance is allowed and
	// leaves the buyer on $0. Going to zero is legal; going below it is what
	// this row was about.
	if err := propertyPurchaseRejection(active(400), forSale("Boardwalk", 400), 400); err != nil {
		t.Fatalf("expected no rejection, got %q", err)
	}
}

func TestPurchaseRejectionRefusesAnUnaffordablePurchase(t *testing.T) {
	// Before 2026-09-22 there was no read on this path at all, so the $inc went
	// out unconditionally and drove the buyer as far negative as the price said.
	err := propertyPurchaseRejection(active(399), forSale("Boardwalk", 400), 400)

	wantRejection(t, err, "insufficient funds: buying Boardwalk for $400 is more than the $399 available")
}

func TestPurchaseRejectionRefusesANegativePrice(t *testing.T) {
	// handlePropertyPurchase refuses this first, so this is the backstop rather
	// than the guard that runs in production. It is here because
	// PurchaseProperty is exported and the floor belongs to the money write: the
	// balance $inc takes its sign from the code and not from the price, so a
	// negative price pays the buyer for taking the deed.
	err := propertyPurchaseRejection(active(500), forSale("Boardwalk", 400), -1000)

	wantRejection(t, err, "a property purchase has to be at least $1")
}

func TestPurchaseRejectionRefusesZero(t *testing.T) {
	err := propertyPurchaseRejection(active(500), forSale("Boardwalk", 400), 0)

	wantRejection(t, err, "a property purchase has to be at least $1")
}

func TestPurchaseRejectionRefusesARemovedBuyer(t *testing.T) {
	// Board row 16, decided 2026-09-17: a removed player may not move money.
	// This assertion is row 19's, moved here from the websocket package with the
	// same sentence; a frozen player is still on screen, because GetPlayersInRoom
	// deliberately does not filter on isActive (invariant 6).
	err := propertyPurchaseRejection(removed(500), forSale("Boardwalk", 400), 400)

	wantRejection(t, err, "a removed player cannot buy property")
}

func TestPurchaseRejectionRefusesADeedThatIsAlreadyOwned(t *testing.T) {
	// The fifth hole in this handler, closed with the other four because the
	// transaction reads the deed anyway. Without it, two players buying the same
	// property off two lists that were both current a second ago both succeed,
	// and the second $set moves the deed off the first buyer, who has already
	// paid for it and is never told.
	err := propertyPurchaseRejection(active(500), owned("Boardwalk", 400), 400)

	wantRejection(t, err, "Boardwalk has already been bought")
}

func TestPurchaseRejectionReadsOwnershipOffTheZeroValue(t *testing.T) {
	// What "still for sale" means in Go, pinned: the zero ObjectID and nothing
	// else. An unowned deed is stored as `playerId: null` or with no playerId at
	// all, and the driver decodes both into the zero value
	// (bson/bsoncodec/default_value_decoders.go has an explicit Null case for
	// ObjectID). A check written against nil, or against a pointer, would not
	// see either of them.
	var unowned models.Property
	if !unowned.PlayerID.IsZero() {
		t.Fatal("a decoded-null playerId is supposed to be the zero ObjectID")
	}
	if err := propertyPurchaseRejection(active(500), unowned, 400); err != nil {
		t.Fatalf("expected an unowned deed to be accepted, got %q", err)
	}
}

// --- ordering ---
//
// Mutation checks on the order of the rules rather than on any one of them. The
// order is the order a player would want to hear about a problem, and each test
// below fails if a later rule is hoisted above an earlier one.

func TestPurchaseRejectionReportsThePriceBeforeTheBuyer(t *testing.T) {
	err := propertyPurchaseRejection(removed(500), owned("Boardwalk", 400), -1000)

	wantRejection(t, err, "a property purchase has to be at least $1")
}

func TestPurchaseRejectionReportsTheRemovalBeforeTheDeed(t *testing.T) {
	err := propertyPurchaseRejection(removed(500), owned("Boardwalk", 400), 400)

	wantRejection(t, err, "a removed player cannot buy property")
}

func TestPurchaseRejectionReportsTheDeedBeforeTheBalance(t *testing.T) {
	// A player who cannot afford a property somebody else already owns is told
	// it is gone, not that they are broke. Swap the two and they are told to go
	// and earn $400 for a deed that was never going to be theirs.
	err := propertyPurchaseRejection(active(0), owned("Boardwalk", 400), 400)

	wantRejection(t, err, "Boardwalk has already been bought")
}
