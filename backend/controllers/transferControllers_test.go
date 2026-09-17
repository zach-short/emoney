package controllers

import (
	"testing"

	"github.com/zachmshort/emoney-backend/models"
)

// The first tests in this package. They reach exactly one thing -
// transferRejection - and that is the whole reason it is a pure function:
// config.DB is a nil *mongo.Database in a test binary, so every other line of
// PlayerTransfer panics on the nil dereference at config.DB.Client() instead of
// running. Nothing here proves a valid transfer moves the right money; it
// proves an invalid one is refused before it can. See HANDOFF 4 for the
// boundary this sits on.

func active(balance int) models.Player {
	return models.Player{IsActive: true, Balance: balance}
}

func removed(balance int) models.Player {
	return models.Player{IsActive: false, Balance: balance}
}

func TestTransferRejectionAcceptsAnAffordableSend(t *testing.T) {
	if err := transferRejection(active(500), active(0), 100); err != nil {
		t.Fatalf("expected no rejection, got %q", err)
	}
}

func TestTransferRejectionAcceptsSpendingTheWholeBalance(t *testing.T) {
	// The boundary, pinned deliberately: the check is `balance < amount`, so
	// $500 of a $500 balance is allowed and leaves the sender on $0. Going to
	// zero is legal; going below it is what board row 14 was about. Whether a
	// player may go below zero at all is a room rule Zach has still to build
	// (TRIAGE.md D7/D9) - when that lands, this is the line it changes.
	if err := transferRejection(active(500), active(0), 500); err != nil {
		t.Fatalf("expected no rejection, got %q", err)
	}
}

func TestTransferRejectionRefusesAnUnaffordableSend(t *testing.T) {
	err := transferRejection(active(499), active(0), 500)

	wantRejection(t, err, "insufficient funds: sending $500 is more than the $499 available")
}

func TestTransferRejectionRefusesANegativeAmount(t *testing.T) {
	// handleTransfer refuses this first, so this is the backstop rather than
	// the guard that runs in production. It is here because PlayerTransfer is
	// exported and the floor belongs to the money write: both $inc updates take
	// their sign from the direction, so a negative amount pays the sender out
	// of the recipient.
	err := transferRejection(active(500), active(0), -100)

	wantRejection(t, err, "a transfer has to be at least $1")
}

func TestTransferRejectionRefusesZero(t *testing.T) {
	err := transferRejection(active(500), active(0), 0)

	wantRejection(t, err, "a transfer has to be at least $1")
}

func TestTransferRejectionRefusesARemovedSender(t *testing.T) {
	// Board row 16, decided 2026-09-17: a removed player may not move money.
	err := transferRejection(removed(500), active(0), 100)

	wantRejection(t, err, "a removed player cannot send money")
}

func TestTransferRejectionRefusesARemovedRecipient(t *testing.T) {
	// The half that loses money rather than just letting a ghost act: a removed
	// player's estate is auctioned and their balance leaves the game, so paying
	// into it takes that money out of play. The room's player list does not
	// filter on isActive (invariant 6), so this is reachable from the UI and not
	// only from a hand-made frame.
	err := transferRejection(active(500), removed(0), 100)

	wantRejection(t, err, "that player has been removed from the game")
}

// --- ordering ---
//
// Two mutation checks on the order of the rules rather than on any one of them.
// The order is the order a player would want to hear about a problem, and each
// test below fails if a later rule is hoisted above an earlier one.

func TestTransferRejectionReportsTheAmountBeforeTheSender(t *testing.T) {
	err := transferRejection(removed(500), active(0), -100)

	wantRejection(t, err, "a transfer has to be at least $1")
}

func TestTransferRejectionReportsTheRemovalBeforeTheBalance(t *testing.T) {
	// A removed player with no money gets told they are removed. Swap the two
	// rules and they are told they are broke, which is both less useful and
	// less true.
	err := transferRejection(removed(0), active(0), 100)

	wantRejection(t, err, "a removed player cannot send money")
}

func wantRejection(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected rejection %q, got nil", want)
	}
	if err.Error() != want {
		t.Fatalf("expected rejection %q, got %q", want, err.Error())
	}
}
