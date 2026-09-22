package controllers

import (
	"context"
	"errors"
	"fmt"

	"github.com/zachmshort/emoney-backend/config"
	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type Transfer = models.Transfer

// transferRejection is every rule a SEND has to clear, written against values
// that have already been read rather than reading them itself. That shape is
// copied from bidRejection (websocket/websocketManager.go) and it is there for
// the same reason: config.DB is a nil *mongo.Database in a test binary, so a
// rule written inline inside the transaction below is a rule no test on this
// machine can reach. Pure, so every rule here has a test.
//
// The order is the order a player would want to hear about a problem: whether
// the amount itself is legal, whether they are just paying themselves,
// whether they may send at all, whether the person they are paying is still
// in the game, and only then whether they can afford it.
func transferRejection(from, to models.Player, amount int) error {
	if amount < 1 {
		// handleTransfer rejects this before PlayerTransfer is ever called, and
		// says why there. Kept here because the floor belongs to the money
		// write rather than to one of its callers: both $inc updates below take
		// their sign from the direction, so a negative amount runs the whole
		// transfer backwards and pays the sender out of the recipient.
		return errors.New("a transfer has to be at least $1")
	}
	if from.ID == to.ID {
		// A SEND to yourself nets to zero - the $inc below debits and credits
		// the same balance by the same amount - but every rule below it would
		// still pass: you are trivially active, real, and able to "afford" your
		// own money. Placed here, before either IsActive check, because a
		// self-transfer is knowable the instant both IDs are read and is true
		// regardless of whether either flag happens to be frozen; it would be
		// strange to tell a frozen player they can't pay themselves because
		// they're frozen; when the real reason is that it's themselves.
		return errors.New("you can't send money to yourself")
	}
	if !from.IsActive {
		// Board row 16, decided 2026-09-17: a removed player may not move
		// money. The kick force-closes their socket and JOIN refuses them
		// (websocket/handler.go), so reaching this needs a frame already in
		// flight - but a payload carries whatever player id it is given, and
		// this is the read that decides.
		return errors.New("a removed player cannot send money")
	}
	if !to.IsActive {
		// The same decision from the other end, and the half that loses real
		// money rather than just letting a ghost act. A removed player's estate
		// is auctioned and their balance leaves the game, so paying into it
		// takes that money out of play permanently. Worth guarding rather than
		// trusting the UI: GetPlayersInRoom deliberately does not filter on
		// isActive (invariant 6), so a frozen player is still on screen and
		// still tappable as a recipient.
		return errors.New("that player has been removed from the game")
	}
	if from.Balance < amount {
		return fmt.Errorf("insufficient funds: sending $%d is more than the $%d available", amount, from.Balance)
	}
	return nil
}

func PlayerTransfer(transfer Transfer) error {
	session, err := config.DB.Client().StartSession()
	if err != nil {
		return err
	}
	defer session.EndSession(context.Background())

	err = session.StartTransaction()
	if err != nil {
		return err
	}

	err = mongo.WithSession(context.Background(), session, func(sc mongo.SessionContext) error {
		playerColl := config.DB.Collection("Player")

		// Both players are read HERE, inside the transaction and on sc, and
		// then handed to transferRejection. Before 2026-09-17 there was no read
		// and no check at all on this path: the two $inc updates below went out
		// unconditionally, so a SEND drove the sender as far negative as it
		// liked and a negative amount reversed the whole thing.
		//
		// Inside the transaction because a balance checked outside it is a
		// balance that can have changed by the time the write lands - the same
		// reasoning that moved freeParking's read in (websocket/
		// websocketManager.go). This one does not use session.WithTransaction,
		// so the callback runs once and a concurrent write aborts it rather
		// than re-running it; the transfer then fails and the player is told.
		// That is a weaker guarantee than a retry and a deliberately smaller
		// change - the mechanism is not what this fix is about.
		//
		// Filtered on _id alone, with no isActive clause, so that a removed
		// player produces a document and a rule rather than a missing document
		// and a not-found error. Same choice, and the same wording, as the
		// winner read in handleCloseAuction: what decides whether they still
		// count is the IsActive check in transferRejection, not the absence of
		// a document.
		var fromPlayer, toPlayer models.Player
		if err := playerColl.FindOne(sc, bson.M{"_id": transfer.FromPlayerID}).Decode(&fromPlayer); err != nil {
			return fmt.Errorf("failed to get sender details: %w", err)
		}
		if err := playerColl.FindOne(sc, bson.M{"_id": transfer.ToPlayerID}).Decode(&toPlayer); err != nil {
			return fmt.Errorf("failed to get recipient details: %w", err)
		}
		if err := transferRejection(fromPlayer, toPlayer, transfer.Amount); err != nil {
			return err
		}

		fromUpdate := bson.M{"$inc": bson.M{"balance": -transfer.Amount}}
		toUpdate := bson.M{"$inc": bson.M{"balance": transfer.Amount}}

		if err := playerColl.FindOneAndUpdate(sc, bson.M{"_id": transfer.FromPlayerID}, fromUpdate).Err(); err != nil {
			return err
		}

		if err := playerColl.FindOneAndUpdate(sc, bson.M{"_id": transfer.ToPlayerID}, toUpdate).Err(); err != nil {
			return err
		}

		transferColl := config.DB.Collection("Transfer")
		_, err := transferColl.InsertOne(sc, transfer)
		return err
	})

	if err != nil {
		session.AbortTransaction(context.Background())
		return err
	}

	return session.CommitTransaction(context.Background())
}

func BankTransfer(transfer Transfer) error {
	return nil
}

func RequestTransfer(transfer Transfer) error { return nil }
