package controllers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zachmshort/emoney-backend/config"
	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func MortgageProperty(c *gin.Context) {}
func AddProperty(c *gin.Context)      {}
func RemoveProperty(c *gin.Context)   {}

// propertyPurchaseRejection is every rule a purchase from the Bank has to
// clear, written against values that have already been read rather than
// reading them itself. Same shape as transferRejection (transferControllers.go)
// and there for the same reason: config.DB is a nil *mongo.Database in a test
// binary, so a rule written inline inside the transaction below is a rule no
// test on this machine can reach. Pure, so every rule here has a test.
//
// It lives in this package rather than in websocket/, where its frozen-buyer
// half used to sit as propertyPurchaseRejection, because the transaction that
// guards the money is here now and a rule the transaction cannot call is not a
// rule. controllers cannot import websocket - websocket imports controllers -
// so the rule moved to the write rather than the write to the rule. The
// frozen-buyer sentence is unchanged on purpose: it is board row 16's answer at
// this site and only its address changed.
//
// The order is the order a player would want to hear about a problem: whether
// this is a purchase at all, whether they may buy at all, whether the deed is
// still for sale, and only then whether they can afford it.
func propertyPurchaseRejection(buyer models.Player, property models.Property, price int) error {
	if price < 1 {
		// handlePropertyPurchase refuses this before PurchaseProperty is ever
		// called, and says why there. Kept here for the reason transferRejection
		// keeps its own copy of the transfer floor: the floor belongs to the
		// money write rather than to one of its callers, and PurchaseProperty is
		// exported. Without it, a negative price runs the balance $inc below
		// backwards - the buyer is PAID for taking the deed.
		return errors.New("a property purchase has to be at least $1")
	}
	if !buyer.IsActive {
		// Board row 16, decided 2026-09-17: a removed player may not move money.
		// The buyer is the actor here - they are spending their own balance on a
		// deed - so this is the same "a frozen player may not act" half of the
		// rule as freeParkingRejection and managePropertiesRejection, and not the
		// banker question bankTransactionRejection answers.
		return errors.New("a removed player cannot buy property")
	}
	if !property.PlayerID.IsZero() {
		// "Still for sale" means "the Bank still holds it", and the Bank holding
		// it is exactly what GetAvailableProperties below filters on: playerId
		// nil, which in Mongo matches both an explicit null and a missing key.
		// The Go side of that same test is IsZero(): the driver's ObjectID
		// decoder has an explicit Null case that leaves the value zero rather
		// than erroring (bson/bsoncodec/default_value_decoders.go, mongo-driver
		// 1.17.10), and an absent key leaves it zero too. So a deed that
		// GetAvailableProperties would list is one this accepts, and no other.
		//
		// This is a fifth hole rather than the price hole this row was opened
		// for, and it is closed here deliberately (Zach's call to confirm,
		// 2026-09-22) because the transaction is what makes it reachable to fix:
		// the deed is read inside the transaction anyway, so this is one branch
		// on a document already in hand, and the transaction is only worth
		// having if something checks the state it re-reads. Without it, two
		// players buying the same deed off two lists that were both current a
		// second ago BOTH succeed: the second $set silently moves the deed off
		// the first buyer, who has already paid for it and is never told.
		// GetAvailableProperties is refetched on the PURCHASE_PROPERTY broadcast
		// (frontend/app/room/[code]/page.tsx:156), so the window is the round
		// trip, not a hand-made frame.
		return fmt.Errorf("%s has already been bought", property.Name)
	}
	if buyer.Balance < price {
		// The boundary is deliberate and matches transferRejection: `balance <
		// price`, so spending the whole balance is allowed and leaves the buyer
		// on $0. Going to zero is legal; going below it is what this row was
		// about. Whether a player may go below zero at all is a room rule still
		// to be built (TRIAGE.md D7/D9) - when that lands, this is the line it
		// changes.
		return fmt.Errorf("insufficient funds: buying %s for $%d is more than the $%d available", property.Name, price, buyer.Balance)
	}
	return nil
}

// PurchaseProperty hands a deed from the Bank to a buyer and takes the price
// out of their balance, in one transaction, having decided the purchase from
// values read inside it.
//
// Until 2026-09-22 it was two bare UpdateOnes on context.Background() with no
// read, no session and no check of any kind - the counter-example the
// transaction comments in websocket/websocketManager.go all point at. Both
// halves of the hole it left are closed here: the price now has a floor and the
// buyer a balance (propertyPurchaseRejection above), and the two writes now
// commit or abort together, so a deed can no longer be handed over by a write
// that lands while the charge that pays for it fails.
//
// It returns the property and the buyer as they were read inside the committed
// transaction, because the caller needs their names for the notification and
// the property's room for the event-history row. Reading them again after the
// commit would be reading the state this purchase just changed; reading them
// before it, which is what GetPropertyAndBuyer used to do on
// context.Background(), is the untransacted read that made the floor unsafe in
// the first place.
//
// The callback's error is returned unwrapped, as PlayerTransfer does and unlike
// freeParking's "transaction failed: %w", because it is what the player is
// shown: "a removed player cannot buy property" is a sentence, and
// "transaction failed: a removed player cannot buy property" is a bug report.
func PurchaseProperty(propertyID, buyerID primitive.ObjectID, price int) (models.Property, models.Player, error) {
	var property models.Property
	var buyer models.Player

	session, err := config.DB.Client().StartSession()
	if err != nil {
		return models.Property{}, models.Player{}, fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(context.Background())

	_, err = session.WithTransaction(context.Background(), func(ctx mongo.SessionContext) (interface{}, error) {
		// Reset every captured variable at the top, because WithTransaction
		// re-runs this callback on a write conflict. Same discipline as
		// freeParking and handleCloseAuction, and for the same reason: a retry
		// that inherits the previous attempt's values is deciding from state that
		// was rolled back.
		property, buyer = models.Property{}, models.Player{}

		// Both documents are read HERE, inside the transaction and on ctx. The
		// balance and the deed's owner are the only things the rules below
		// consult, so a read outside the transaction means a retry re-checks
		// values the retry was triggered by someone else changing: balance $60,
		// a $60 purchase and a concurrent $60 debit race, the debit commits, this
		// callback loses the write conflict and is re-run, and the stale $60
		// still clears the floor. A floor is only a floor if the number it reads
		// comes from the same snapshot as the write it guards.
		//
		// Filtered on _id alone, with no isActive clause on the buyer, so that a
		// removed player produces a document and a rule rather than a missing
		// document and a not-found error. Same choice, and the same reasoning, as
		// the sender read in PlayerTransfer and the winner read in
		// handleCloseAuction.
		if err := config.DB.Collection("Property").FindOne(ctx, bson.M{"_id": propertyID}).Decode(&property); err != nil {
			return nil, fmt.Errorf("failed to find property: %w", err)
		}
		if err := config.DB.Collection("Player").FindOne(ctx, bson.M{"_id": buyerID}).Decode(&buyer); err != nil {
			return nil, fmt.Errorf("failed to find buyer: %w", err)
		}
		if err := propertyPurchaseRejection(buyer, property, price); err != nil {
			return nil, err
		}

		// The deed, pinned to the unowned state the read above saw. What stops a
		// concurrent purchase landing between them is not this filter - every
		// read here is on ctx, so the pin cannot fail to match on the attempt
		// that read it - it is that a transactional write to a document modified
		// since the snapshot raises a WriteConflict, which carries a transient
		// label, which makes WithTransaction abort and re-run this whole callback
		// against fresh state, where the rejection above now sees an owner. The
		// MatchedCount check is kept as defence against a future edit that moves
		// either read back outside the transaction, the way handleCloseAuction
		// keeps its own.
		result, err := config.DB.Collection("Property").UpdateOne(
			ctx,
			bson.M{"_id": propertyID, "playerId": nil},
			bson.M{"$set": bson.M{"playerId": buyerID}},
		)
		if err != nil {
			return nil, fmt.Errorf("failed to hand over the deed: %w", err)
		}
		if result.MatchedCount == 0 {
			return nil, fmt.Errorf("%s has already been bought", property.Name)
		}

		if _, err := config.DB.Collection("Player").UpdateOne(
			ctx,
			bson.M{"_id": buyerID},
			bson.M{"$inc": bson.M{"balance": -price}},
		); err != nil {
			return nil, fmt.Errorf("failed to charge the buyer: %w", err)
		}

		return nil, nil
	})
	if err != nil {
		return models.Property{}, models.Player{}, err
	}

	return property, buyer, nil
}

func AssignOwnerShipProperty(propertyID, buyerID primitive.ObjectID) error {
	_, err := config.DB.Collection("Property").UpdateOne(
		context.Background(),
		bson.M{"_id": propertyID},
		bson.M{"$set": bson.M{"playerId": buyerID}},
	)
	return err
}

func GetAvailableProperties(c *gin.Context) {
	roomCode := c.Param("code")

	roomCollection := config.DB.Collection("Room")
	var room models.Room
	err := roomCollection.FindOne(c, bson.M{"code": roomCode}).Decode(&room)
	if err != nil {
		c.JSON(http.StatusNoContent, gin.H{"error": "Room was not found with the following code"})
		return
	}

	propertyCollection := config.DB.Collection("Property")

	query := bson.M{
		"roomId":   room.ID,
		"playerId": nil,
	}

	var availableProperties []models.Property
	cursor, err := propertyCollection.Find(c, query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve available properties"})
		return
	}

	if err = cursor.All(c, &availableProperties); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode available properties"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"availableProperties": availableProperties,
		"roomId":              room.ID,
	})
}
