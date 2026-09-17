package websocket

import (
	"errors"
	"math"
	"strings"
	"testing"

	"github.com/zachmshort/emoney-backend/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// --- the live auction (D14-D18) ---
//
// The same constraint shapes every test in this file that shapes the kick's:
// config.DB is a nil *mongo.Database in a test binary, so a payload that
// validation accepts panics at the first config.DB.Collection call rather than
// returning an error, and a rule written below that call is a rule no test here
// can reach. That is why the auction's rules live in bidRejection,
// outcomeForClose, advanceAuction and closeAuthorized: they take read values
// instead of doing the reads, so the decisions are testable even though the
// reads are not.
//
// What no test in this repo can assert, and it is worth writing down rather
// than leaving as an absence: that a valid bid raises the right high bid, that
// a close charges the right player the right amount, or that the conditional
// filter on the bid and the pin on the close do what they are for under two
// real concurrent frames. Those need Mongo. They rest on PLAN.md Phase 3's
// done-when device walk, on the Deep review of the close path, and on nothing
// else.

var (
	testLotID     = mustObjectID("507f1f77bcf86cd799439021")
	testNextLotID = mustObjectID("507f1f77bcf86cd799439022")
	testThirdLot  = mustObjectID("507f1f77bcf86cd799439023")
	testBidderID  = mustObjectID("507f1f77bcf86cd799439031")
	testKickedID  = mustObjectID("507f1f77bcf86cd799439032")
)

func mustObjectID(hex string) primitive.ObjectID {
	id, err := primitive.ObjectIDFromHex(hex)
	if err != nil {
		panic(err)
	}
	return id
}

// openAuction is an auction on testLotID with two more deeds queued and no bid
// yet - the state handleKickPlayer writes when it opens one. Each test below
// changes exactly the field it is about.
func openAuction() models.Auction {
	return models.Auction{
		KickedPlayerID: testKickedID,
		PropertyID:     testLotID,
		Queue:          []primitive.ObjectID{testNextLotID, testThirdLot},
		HighBid:        0,
		HighBidderID:   nil,
	}
}

// --- wholeDollars ---

func TestWholeDollarsAcceptsWholeNumbers(t *testing.T) {
	for _, in := range []float64{0, 1, 120, 1500, 12345} {
		got, err := wholeDollars(in)
		if err != nil {
			t.Fatalf("wholeDollars(%v) errored: %v", in, err)
		}
		if float64(got) != in {
			t.Fatalf("wholeDollars(%v) = %d", in, got)
		}
	}
}

func TestWholeDollarsRejectsAFraction(t *testing.T) {
	// handlePropertyPurchase truncates its price and this deliberately does
	// not: a truncated 120.9 is a bid the bidder did not make, and it is the
	// truncated figure that gets charged and that everyone else has to beat.
	_, err := wholeDollars(120.9)
	wantErrContains(t, err, "not a whole number of dollars")
}

func TestWholeDollarsRejectsNaNAndInfinity(t *testing.T) {
	for _, in := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := wholeDollars(in); err == nil {
			t.Fatalf("wholeDollars(%v) was accepted", in)
		}
	}
}

func TestWholeDollarsRejectsOutOfRange(t *testing.T) {
	// int(float64) on a value past the int range is implementation-defined
	// nonsense rather than an error, so the range check has to be explicit.
	for _, in := range []float64{math.MaxInt32 + 1, math.MinInt32 - 1, 1e18} {
		if _, err := wholeDollars(in); err == nil {
			t.Fatalf("wholeDollars(%v) was accepted", in)
		}
	}
}

// --- bidRejection ---

func TestBidAcceptsARaiseOverTheHighBid(t *testing.T) {
	auction := openAuction()
	auction.HighBid = 120
	auction.HighBidderID = &testKickedID // any prior bidder; not this one

	if err := bidRejection(auction, testLotID, testBidderID, 121, 1500); err != nil {
		t.Fatalf("expected $121 over $120 to be accepted, got %v", err)
	}
}

func TestBidAcceptsADollarOnAFreshLot(t *testing.T) {
	// D15: a lot opens at $0, so $1 is the smallest bid that can exist and it
	// must be accepted against a high bid of zero with no bidder.
	if err := bidRejection(openAuction(), testLotID, testBidderID, 1, 1500); err != nil {
		t.Fatalf("expected $1 on a $0 lot to be accepted, got %v", err)
	}
}

func TestBidRejectsADifferentLot(t *testing.T) {
	// The same rejection a bid gets when it arrives after the hammer: the close
	// advances auction.PropertyID, so a frame still naming the closed lot lands
	// here rather than being applied to the deed that is open now.
	err := bidRejection(openAuction(), testNextLotID, testBidderID, 50, 1500)
	wantErrEqual(t, err, "that lot is no longer open for bidding")
}

func TestBidRejectsTheKickedPlayer(t *testing.T) {
	err := bidRejection(openAuction(), testLotID, testKickedID, 50, 1500)
	wantErrEqual(t, err, "a removed player cannot bid on their own estate")
}

func TestBidRejectsAnAmountEqualToTheHighBid(t *testing.T) {
	// D15's minimum raise is $1, so equalling the high bid is not a bid. This
	// is done-when (b) of PLAN.md Phase 3's device walk, as a unit test.
	auction := openAuction()
	auction.HighBid = 120
	auction.HighBidderID = &testKickedID

	err := bidRejection(auction, testLotID, testBidderID, 120, 1500)
	wantErrEqual(t, err, "a bid has to beat the current high bid of $120")
}

func TestBidRejectsAnAmountBelowTheHighBid(t *testing.T) {
	auction := openAuction()
	auction.HighBid = 120
	auction.HighBidderID = &testKickedID

	err := bidRejection(auction, testLotID, testBidderID, 119, 1500)
	wantErrEqual(t, err, "a bid has to beat the current high bid of $120")
}

func TestBidRejectsZeroOnAFreshLot(t *testing.T) {
	// $0 does not beat a $0 opening. handlePlaceBid also refuses anything under
	// $1 above its first Mongo call, so this is the same rule enforced twice on
	// purpose - once as a fact about the payload and once against the state.
	err := bidRejection(openAuction(), testLotID, testBidderID, 0, 1500)
	wantErrEqual(t, err, "a bid has to beat the current high bid of $0")
}

func TestBidRejectsMoreThanTheBidderCanCover(t *testing.T) {
	// D17's first check. The second is at settlement, in outcomeForClose.
	err := bidRejection(openAuction(), testLotID, testBidderID, 1501, 1500)
	wantErrEqual(t, err, "insufficient funds: a $1501 bid is more than the $1500 available")
}

func TestBidAcceptsABidForExactlyTheBalance(t *testing.T) {
	// The check is "balance < amount", not "<=". A player may bid everything
	// they have; Monopoly has no rule that they must keep a dollar back.
	if err := bidRejection(openAuction(), testLotID, testBidderID, 1500, 1500); err != nil {
		t.Fatalf("expected a bid of the whole balance to be accepted, got %v", err)
	}
}

func TestBidChecksTheLotBeforeAnythingElse(t *testing.T) {
	// Order matters for the sentence the bidder reads. A frame that is both on
	// a closed lot and unaffordable should say the lot closed - that is the
	// thing that actually happened to them.
	err := bidRejection(openAuction(), testNextLotID, testKickedID, 99999, 1)
	wantErrEqual(t, err, "that lot is no longer open for bidding")
}

// --- outcomeForClose ---

func TestCloseOutcomeIsNoBidWhenNobodyBid(t *testing.T) {
	if got := outcomeForClose(openAuction(), false, 0); got != lotNoBid {
		t.Fatalf("got %v, want %v", got, lotNoBid)
	}
}

func TestCloseOutcomeIsNoBidWhenABidderIsSetAtZero(t *testing.T) {
	// Defence against a state no path writes today: a high bidder with a $0
	// high bid would otherwise settle as a sale for nothing.
	auction := openAuction()
	auction.HighBidderID = &testBidderID
	auction.HighBid = 0

	if got := outcomeForClose(auction, true, 1500); got != lotNoBid {
		t.Fatalf("got %v, want %v", got, lotNoBid)
	}
}

func TestCloseOutcomeIsWinnerGoneWhenTheBidderLeft(t *testing.T) {
	// Reachable: the banker can kick a bidder, with BANK or FREEZE, while an
	// auction is running. Zach's call 2026-09-17 - the deed goes to the Bank,
	// the same as a lot nobody bid on.
	auction := openAuction()
	auction.HighBidderID = &testBidderID
	auction.HighBid = 120

	if got := outcomeForClose(auction, false, 0); got != lotWinnerGone {
		t.Fatalf("got %v, want %v", got, lotWinnerGone)
	}
}

func TestCloseOutcomeIsCannotPayWhenTheWinnerIsShort(t *testing.T) {
	// D17's second check, and the whole reason it exists: the winner paid rent
	// between bidding and the hammer.
	auction := openAuction()
	auction.HighBidderID = &testBidderID
	auction.HighBid = 120

	if got := outcomeForClose(auction, true, 119); got != lotCannotPay {
		t.Fatalf("got %v, want %v", got, lotCannotPay)
	}
}

func TestCloseOutcomeIsSoldWhenTheWinnerCanExactlyAfford(t *testing.T) {
	auction := openAuction()
	auction.HighBidderID = &testBidderID
	auction.HighBid = 120

	if got := outcomeForClose(auction, true, 120); got != lotSold {
		t.Fatalf("got %v, want %v", got, lotSold)
	}
}

func TestCloseOutcomeIgnoresTheBalanceWhenTheWinnerIsGone(t *testing.T) {
	// A removed winner is refused before the balance is looked at, so a rich
	// one who left still does not take the deed.
	auction := openAuction()
	auction.HighBidderID = &testBidderID
	auction.HighBid = 120

	if got := outcomeForClose(auction, false, 100000); got != lotWinnerGone {
		t.Fatalf("got %v, want %v", got, lotWinnerGone)
	}
}

// --- winnerStatus ---

func TestWinnerStatusAcceptsAnActiveWinner(t *testing.T) {
	found, fatal := winnerStatus(nil, models.Player{Name: "Sam", IsActive: true})
	if fatal != nil {
		t.Fatalf("unexpected fatal: %v", fatal)
	}
	if !found {
		t.Fatal("an active winner was not counted")
	}
}

func TestWinnerStatusRejectsAWinnerWhoWasRemoved(t *testing.T) {
	// Reachable: the banker can kick a bidder, with BANK or FREEZE, while the
	// auction is running.
	found, fatal := winnerStatus(nil, models.Player{Name: "Sam", IsActive: false})
	if fatal != nil {
		t.Fatalf("unexpected fatal: %v", fatal)
	}
	if found {
		t.Fatal("a removed winner was counted")
	}
}

func TestWinnerStatusTreatsAMissingDocumentAsGone(t *testing.T) {
	found, fatal := winnerStatus(mongo.ErrNoDocuments, models.Player{})
	if fatal != nil {
		t.Fatalf("unexpected fatal: %v", fatal)
	}
	if found {
		t.Fatal("a missing winner was counted")
	}
}

func TestWinnerStatusRefusesToGuessOnAFailedRead(t *testing.T) {
	// The bug this pins, found by this phase's Deep review and needing no
	// concurrency to reach: reading every error as "the winner is gone" turns
	// one flaky read into a lot settled to the Bank with the real high bidder
	// neither charged nor given the deed, and the room told they left the game.
	// A failing read must abort the close, not decide who owns a property.
	found, fatal := winnerStatus(errors.New("connection reset by peer"), models.Player{})
	if fatal == nil {
		t.Fatal("a failed read was treated as a decision rather than an error")
	}
	if found {
		t.Fatal("a failed read counted a winner")
	}
	wantErrContains(t, fatal, "failed to read the winning bidder")
}

func TestWinnerStatusWrapsTheUnderlyingError(t *testing.T) {
	// Wrapped with %w so that a WriteConflict's transient label survives and
	// WithTransaction retries the callback instead of giving up.
	underlying := errors.New("write conflict")
	_, fatal := winnerStatus(underlying, models.Player{})
	if !errors.Is(fatal, underlying) {
		t.Fatalf("%v does not unwrap to the read error", fatal)
	}
}

// --- advanceAuction ---

func TestAdvanceAuctionEndsOnAnEmptyQueue(t *testing.T) {
	// PLAN.md Phase 3 scope 8's close-of-an-empty-queue, as a unit test: nil is
	// what makes handleCloseAuction $unset the auction rather than $set it.
	auction := openAuction()
	auction.Queue = nil

	if got := advanceAuction(auction); got != nil {
		t.Fatalf("got %+v, want nil", got)
	}
}

func TestAdvanceAuctionOpensTheNextDeedInOrder(t *testing.T) {
	got := advanceAuction(openAuction())
	if got == nil {
		t.Fatal("got nil, want the next lot")
	}
	if got.PropertyID != testNextLotID {
		t.Fatalf("opened %v, want %v", got.PropertyID, testNextLotID)
	}
	if len(got.Queue) != 1 || got.Queue[0] != testThirdLot {
		t.Fatalf("queue is %v, want [%v]", got.Queue, testThirdLot)
	}
	if got.KickedPlayerID != testKickedID {
		t.Fatalf("kicked player is %v, want %v", got.KickedPlayerID, testKickedID)
	}
}

func TestAdvanceAuctionResetsTheBidding(t *testing.T) {
	// The bug this pins: carrying the closed lot's high bid onto the next deed
	// would make the next lot open at a price nobody bid on it, and the first
	// bidder to beat it would win it for the previous deed's money.
	auction := openAuction()
	auction.HighBid = 400
	auction.HighBidderID = &testBidderID

	got := advanceAuction(auction)
	if got == nil {
		t.Fatal("got nil, want the next lot")
	}
	if got.HighBid != 0 {
		t.Fatalf("next lot opened at $%d, want $0", got.HighBid)
	}
	if got.HighBidderID != nil {
		t.Fatalf("next lot opened with bidder %v, want none", *got.HighBidderID)
	}
}

func TestAdvanceAuctionEndsAfterTheLastDeed(t *testing.T) {
	auction := openAuction()
	auction.Queue = []primitive.ObjectID{testNextLotID}

	next := advanceAuction(auction)
	if next == nil {
		t.Fatal("got nil, want the last lot")
	}
	if len(next.Queue) != 0 {
		t.Fatalf("queue is %v, want empty", next.Queue)
	}
	if after := advanceAuction(*next); after != nil {
		t.Fatalf("got %+v after the last lot, want nil", after)
	}
}

func TestAdvanceAuctionDoesNotAliasTheCallersQueue(t *testing.T) {
	// The returned slice is marshalled straight into the Room document. An
	// alias into the caller's backing array is correct right up until someone
	// appends to one of the two.
	auction := openAuction()
	next := advanceAuction(auction)
	if next == nil {
		t.Fatal("got nil, want the next lot")
	}

	next.Queue[0] = testLotID
	if auction.Queue[1] == testLotID {
		t.Fatal("advanceAuction returned a slice aliasing the caller's queue")
	}
}

// --- closeAuthorized ---

func TestCloseAuthorizedAcceptsTheBanker(t *testing.T) {
	banker := models.Player{Name: "Zach", IsActive: true, IsBanker: true}
	if err := closeAuthorized(banker); err != nil {
		t.Fatalf("expected the banker to be allowed to close, got %v", err)
	}
}

func TestCloseAuthorizedRejectsANonBanker(t *testing.T) {
	// PLAN.md Phase 3 scope 8's close-by-a-non-banker. This is the only
	// server-side isBanker check in the product and section 4 reserves general
	// enforcement as not built - the close is carved out because it is the one
	// act that fixes a winner and moves the money.
	player := models.Player{Name: "Sam", IsActive: true, IsBanker: false}
	err := closeAuthorized(player)
	wantErrEqual(t, err, "only the Banker can close a lot")
}

func TestCloseAuthorizedRejectsARemovedPlayer(t *testing.T) {
	// Unreachable through handleCloseAuction today, whose read filters on
	// isActive: true. The arm exists so the function is total and so the rule
	// survives a change to that filter rather than leaving with it.
	removed := models.Player{Name: "Rita", IsActive: false, IsBanker: true}
	err := closeAuthorized(removed)
	wantErrEqual(t, err, "a removed player cannot close a lot")
}

// --- the copy ---

func TestLotOpenNotification(t *testing.T) {
	got := lotOpenNotification("Boardwalk")
	want := "Boardwalk is up for auction. Bidding starts at $0."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestBidNotification(t *testing.T) {
	got := bidNotification("Sam", "Boardwalk", 120)
	want := "Sam bid $120 on Boardwalk."
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLotClosedNotificationArms(t *testing.T) {
	cases := []struct {
		name    string
		outcome closeOutcome
		next    string
		want    string
	}{
		{
			name:    "sold, with another deed to come",
			outcome: lotSold,
			next:    "Park Place",
			want:    "Sam won Boardwalk for $120. Park Place is up next.",
		},
		{
			name:    "sold, last lot",
			outcome: lotSold,
			want:    "Sam won Boardwalk for $120. That's the last of them.",
		},
		{
			name:    "nobody bid",
			outcome: lotNoBid,
			want:    "Nobody bid on Boardwalk. It goes back to the Bank. That's the last of them.",
		},
		{
			name:    "winner could not pay",
			outcome: lotCannotPay,
			want:    "Sam couldn't cover the $120 bid, so Boardwalk goes back to the Bank. That's the last of them.",
		},
		{
			name:    "winner left the game",
			outcome: lotWinnerGone,
			want:    "Sam is no longer in the game, so Boardwalk goes back to the Bank. That's the last of them.",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := lotClosedNotification(tc.outcome, "Boardwalk", "Sam", 120, tc.next)
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestLotClosedNotificationSurvivesBroadcastsEmptyGuard(t *testing.T) {
	// Broadcast drops a payload whose notification is present and empty,
	// silently to the room and loudly only in the VM's log. A close moves a
	// deed and a balance, so a close nobody saw is the worst version of this
	// failure in the app: every arm, including the one that should not occur,
	// has to produce text.
	outcomes := []closeOutcome{lotSold, lotNoBid, lotCannotPay, lotWinnerGone, closeOutcome("")}
	for _, outcome := range outcomes {
		for _, next := range []string{"", "Park Place"} {
			notification := lotClosedNotification(outcome, "Boardwalk", "Sam", 120, next)
			empty, hasField := emptyNotification(map[string]interface{}{
				"notification": notification,
				"propertyId":   testLotID.Hex(),
			})
			if !hasField || empty {
				t.Fatalf("outcome %q, next %q: Broadcast would drop %q", outcome, next, notification)
			}
		}
	}
}

func TestAuctionCopyAlwaysSaysWhatHappensNext(t *testing.T) {
	// One of the two trailing clauses, always. Saying nothing when the auction
	// ends would make the last lot indistinguishable from a lot whose next deed
	// never opened, and a player cannot tell those apart by waiting.
	outcomes := []closeOutcome{lotSold, lotNoBid, lotCannotPay, lotWinnerGone, closeOutcome("")}
	for _, outcome := range outcomes {
		last := lotClosedNotification(outcome, "Boardwalk", "Sam", 120, "")
		if !strings.HasSuffix(last, " That's the last of them.") {
			t.Fatalf("outcome %q on the last lot: %q", outcome, last)
		}
		more := lotClosedNotification(outcome, "Boardwalk", "Sam", 120, "Park Place")
		if !strings.HasSuffix(more, " Park Place is up next.") {
			t.Fatalf("outcome %q with more to come: %q", outcome, more)
		}
	}
}

// --- eventTypeFor over the auction's rows ---

func TestAuctionRowsTakeTheDefaultEventIcon(t *testing.T) {
	// This pins a choice rather than reporting an accident. eventTypeFor
	// classifies by substring and its arms are order-dependent, which is a
	// known hazard (see the kick arm's comment); PLAN.md Phase 3's scope does
	// not ask for an auction icon and section 3's icon dial was answered for
	// the kick alone, so no arm was added and the auction's rows take the
	// neutral info pair. If an auction arm is ever added, it goes below the
	// kick arm and this test is what will tell you it changed something.
	rows := []string{
		lotOpenNotification("Boardwalk"),
		lotClosedNotification(lotSold, "Boardwalk", "Sam", 120, "Park Place"),
		lotClosedNotification(lotNoBid, "Boardwalk", "Sam", 0, ""),
		lotClosedNotification(lotCannotPay, "Boardwalk", "Sam", 120, ""),
		lotClosedNotification(lotWinnerGone, "Boardwalk", "Sam", 120, ""),
	}

	for _, row := range rows {
		got := eventTypeFor(row)
		if got[0] != "#6b7280" || got[1] != "ℹ️" {
			t.Fatalf("%q was drawn as %v, want the default info pair", row, got)
		}
	}
}

func TestAuctionRowsAreNotDrawnAsBalanceChanges(t *testing.T) {
	// "It goes back to the Bank" is one letter away from the arm that matches
	// "Banker", which is the 🏦 balance-change icon. If the copy ever gains the
	// word "Banker" - by naming the banker who closed the lot, say - every
	// auction row silently becomes a balance change in the log, which is the
	// exact outcome section 3's icon dial exists to avoid.
	rows := []string{
		lotOpenNotification("Boardwalk"),
		lotClosedNotification(lotNoBid, "Boardwalk", "Sam", 0, ""),
		lotClosedNotification(lotCannotPay, "Boardwalk", "Sam", 120, ""),
	}

	for _, row := range rows {
		if got := eventTypeFor(row); got[1] == "\U0001f3e6" {
			t.Fatalf("%q is drawn as a balance change", row)
		}
	}
}

// --- handlePlaceBid, payload rejections ---
//
// bidOutcome is kickOutcome for the bid handler, and it exists for the same
// reason: a payload the validation accepts panics at the first
// config.DB.Collection call rather than returning an error, and that panic is
// the only signal available here that a payload was accepted rather than
// rejected. If a seam is ever put in front of the room read these tests stop
// panicking; change the accept tests to assert on the error at that point.

func bidOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handlePlaceBid(testClient(), Message{
		Type:    "PLACE_BID",
		Payload: payload,
	})
	return err, false
}

// validBidPayload is a PLACE_BID payload that gets as far as the first database
// call. Each test below changes exactly one field.
func validBidPayload() map[string]any {
	return map[string]any{
		"roomId":     "507f1f77bcf86cd799439011",
		"propertyId": "507f1f77bcf86cd799439021",
		"bidderId":   "507f1f77bcf86cd799439031",
		"amount":     float64(120),
	}
}

func TestBidRejectsNonObjectPayload(t *testing.T) {
	err, _ := bidOutcome(t, "not-an-object")
	wantErrEqual(t, err, "invalid payload format")
}

func TestBidRejectsMissingRoomID(t *testing.T) {
	payload := validBidPayload()
	delete(payload, "roomId")

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for roomId")
}

func TestBidRejectsNonStringRoomID(t *testing.T) {
	payload := validBidPayload()
	payload["roomId"] = 42

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for roomId")
}

func TestBidRejectsMalformedRoomID(t *testing.T) {
	payload := validBidPayload()
	payload["roomId"] = "not-an-object-id"

	err, _ := bidOutcome(t, payload)
	wantErrContains(t, err, "invalid room ID")
}

func TestBidRejectsMissingPropertyID(t *testing.T) {
	payload := validBidPayload()
	delete(payload, "propertyId")

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for propertyId")
}

func TestBidRejectsNonStringPropertyID(t *testing.T) {
	payload := validBidPayload()
	payload["propertyId"] = []string{"507f1f77bcf86cd799439021"}

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for propertyId")
}

func TestBidRejectsMalformedPropertyID(t *testing.T) {
	payload := validBidPayload()
	payload["propertyId"] = "507f1f77bcf86cd7994390"

	err, _ := bidOutcome(t, payload)
	wantErrContains(t, err, "invalid property ID")
}

func TestBidRejectsMissingBidderID(t *testing.T) {
	payload := validBidPayload()
	delete(payload, "bidderId")

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for bidderId")
}

func TestBidRejectsNonStringBidderID(t *testing.T) {
	payload := validBidPayload()
	payload["bidderId"] = nil

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for bidderId")
}

func TestBidRejectsMalformedBidderID(t *testing.T) {
	payload := validBidPayload()
	payload["bidderId"] = "zzzf1f77bcf86cd799439031"

	err, _ := bidOutcome(t, payload)
	wantErrContains(t, err, "invalid bidder ID")
}

func TestBidRejectsMissingAmount(t *testing.T) {
	payload := validBidPayload()
	delete(payload, "amount")

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected number for amount")
}

func TestBidRejectsAStringAmount(t *testing.T) {
	// The shape freeParking takes, and a plausible thing for a keypad-built
	// frontend payload to send. It is refused rather than parsed: this handler
	// takes a JSON number, like handlePropertyPurchase's price.
	payload := validBidPayload()
	payload["amount"] = "120"

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected number for amount")
}

func TestBidRejectsAFractionalAmount(t *testing.T) {
	payload := validBidPayload()
	payload["amount"] = 120.5

	err, _ := bidOutcome(t, payload)
	wantErrContains(t, err, "not a whole number of dollars")
}

func TestBidRejectsZero(t *testing.T) {
	// A lot opens at $0 with no bidder, so $0 is not a bid (D15). Refused as a
	// fact about the payload, above the first database call, which is what
	// makes it reachable from here at all.
	payload := validBidPayload()
	payload["amount"] = float64(0)

	err, panicked := bidOutcome(t, payload)
	if panicked {
		t.Fatal("expected $0 to be refused before the room read, got a panic")
	}
	wantErrEqual(t, err, "a bid has to be at least $1")
}

func TestBidRejectsANegativeAmount(t *testing.T) {
	// Without the floor this would be a bid that pays the bidder: the
	// settlement is an $inc of -amount on the winner's balance.
	payload := validBidPayload()
	payload["amount"] = float64(-50)

	err, _ := bidOutcome(t, payload)
	wantErrEqual(t, err, "a bid has to be at least $1")
}

func TestBidRejectsBeforeReadingTheRoom(t *testing.T) {
	// The point of validating the whole payload above the first database call:
	// a bad amount costs no round trip, and - the reason that matters on this
	// machine - a rejection below the read is not reachable from a test at all,
	// because config.DB is nil and the read panics. A panic here means the
	// validation has slipped below the read.
	payload := validBidPayload()
	payload["amount"] = float64(0)

	if _, panicked := bidOutcome(t, payload); panicked {
		t.Fatal("handlePlaceBid reached the database before validating the payload")
	}
}

func TestBidAcceptsAValidPayload(t *testing.T) {
	err, panicked := bidOutcome(t, validBidPayload())
	if !panicked {
		t.Fatalf("expected a valid bid to be accepted and reach the database, got %v", err)
	}
}

// --- handleCloseAuction, payload rejections ---

func closeAuctionOutcome(t *testing.T, payload any) (err error, panicked bool) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			panicked = true
		}
	}()
	err = NewRoomManager().handleCloseAuction(testClient(), Message{
		Type:    "CLOSE_AUCTION",
		Payload: payload,
	})
	return err, false
}

// validClosePayload is a CLOSE_AUCTION payload that gets as far as the first
// database call. Each test below changes exactly one field.
//
// propertyId is on it, and that is the interesting part of the contract: see
// handleCloseAuction's comment for the two-frames-settle-two-lots interleaving
// that naming the lot is what prevents.
func validClosePayload() map[string]any {
	return map[string]any{
		"roomId":         "507f1f77bcf86cd799439011",
		"playerId":       "507f1f77bcf86cd799439012",
		"propertyId":     "507f1f77bcf86cd799439021",
		"kickedPlayerId": "507f1f77bcf86cd799439032",
	}
}

func TestCloseRejectsNonObjectPayload(t *testing.T) {
	err, _ := closeAuctionOutcome(t, 42)
	wantErrEqual(t, err, "invalid payload format")
}

func TestCloseRejectsMissingRoomID(t *testing.T) {
	payload := validClosePayload()
	delete(payload, "roomId")

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for roomId")
}

func TestCloseRejectsNonStringRoomID(t *testing.T) {
	payload := validClosePayload()
	payload["roomId"] = true

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for roomId")
}

func TestCloseRejectsMalformedRoomID(t *testing.T) {
	payload := validClosePayload()
	payload["roomId"] = "room-one"

	err, _ := closeAuctionOutcome(t, payload)
	wantErrContains(t, err, "invalid room ID")
}

func TestCloseRejectsMissingPlayerID(t *testing.T) {
	payload := validClosePayload()
	delete(payload, "playerId")

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for playerId")
}

func TestCloseRejectsNonStringPlayerID(t *testing.T) {
	payload := validClosePayload()
	payload["playerId"] = 12

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for playerId")
}

func TestCloseRejectsMalformedPlayerID(t *testing.T) {
	payload := validClosePayload()
	payload["playerId"] = "me"

	err, _ := closeAuctionOutcome(t, payload)
	wantErrContains(t, err, "invalid player ID")
}

func TestCloseRejectsMissingPropertyID(t *testing.T) {
	// A close with no lot named would be a close of whatever is open, which is
	// the shape that lets a retried frame settle a deed the banker never looked
	// at. Required, not optional.
	payload := validClosePayload()
	delete(payload, "propertyId")

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for propertyId")
}

func TestCloseRejectsNonStringPropertyID(t *testing.T) {
	payload := validClosePayload()
	payload["propertyId"] = 0

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for propertyId")
}

func TestCloseRejectsMalformedPropertyID(t *testing.T) {
	payload := validClosePayload()
	payload["propertyId"] = "the-one-that-is-open"

	err, _ := closeAuctionOutcome(t, payload)
	wantErrContains(t, err, "invalid property ID")
}

func TestCloseRejectsMissingKickedPlayerID(t *testing.T) {
	// A property id alone does not identify an auction lot - the same deed can
	// be the open lot of two different auctions, so a stale close frame could
	// hammer a later one. See handleCloseAuction's comment for the trace.
	payload := validClosePayload()
	delete(payload, "kickedPlayerId")

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for kickedPlayerId")
}

func TestCloseRejectsNonStringKickedPlayerID(t *testing.T) {
	payload := validClosePayload()
	payload["kickedPlayerId"] = 1.5

	err, _ := closeAuctionOutcome(t, payload)
	wantErrEqual(t, err, "invalid payload: expected string for kickedPlayerId")
}

func TestCloseRejectsMalformedKickedPlayerID(t *testing.T) {
	payload := validClosePayload()
	payload["kickedPlayerId"] = "whoever-it-was"

	err, _ := closeAuctionOutcome(t, payload)
	wantErrContains(t, err, "invalid kicked player ID")
}

func TestCloseRejectsBeforeReadingThePlayer(t *testing.T) {
	payload := validClosePayload()
	payload["playerId"] = "me"

	if _, panicked := closeAuctionOutcome(t, payload); panicked {
		t.Fatal("handleCloseAuction reached the database before validating the payload")
	}
}

func TestCloseAcceptsAValidPayload(t *testing.T) {
	err, panicked := closeAuctionOutcome(t, validClosePayload())
	if !panicked {
		t.Fatalf("expected a valid close to be accepted and reach the database, got %v", err)
	}
}
