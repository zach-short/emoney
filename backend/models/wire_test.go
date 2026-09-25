package models

// These tests pin how the document structs in this package go over the two
// wires that carry them - BSON to Atlas, and JSON to the browser - at exactly
// the points where a driver upgrade can change the answer with every gate
// green. Each one was run against mongo-driver v1.17.10 first, on 2026-09-17,
// before the v1 -> v2 migration (board row 23), again on 2026-09-23 on main's
// tip (c5e8172) before the migration was rebuilt there, and again on 2026-09-25
// on main's tip (4234315) before it was rebuilt a third time; the migration
// then had to keep every one of them passing. They stay because the next
// driver bump has the same failure mode: a codec default that moves,
// compiles, and is wrong in production.
//
// Nothing here touches Mongo. bson.Marshal and bson.Unmarshal run the same
// codecs the driver runs on the wire, so this is the half of a round trip that
// a machine with no database can still prove; the other half is the deploy's
// health check and the runtime pass.

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

func rawOf(t *testing.T, v any) bson.Raw {
	t.Helper()
	b, err := bson.Marshal(v)
	if err != nil {
		t.Fatalf("bson.Marshal: %v", err)
	}
	return bson.Raw(b)
}

// GetAvailableProperties (controllers/propertyControllers.go) finds the Bank's
// deeds with {"playerId": nil}, which matches a missing key or an explicit
// null and nothing else. Every deed CreateRoom inserts has a zero PlayerID, so
// the Bank's whole inventory on a new room depends on omitempty dropping the
// key rather than writing ObjectId("000000000000000000000000") - which that
// filter would not match, and the Bank would show an empty list.
func TestPropertyWithNoOwnerCarriesNoPlayerIdKey(t *testing.T) {
	raw := rawOf(t, Property{ID: bson.NewObjectID(), RoomID: bson.NewObjectID(), Name: "Boardwalk"})

	if v, err := raw.LookupErr("playerId"); err == nil {
		t.Fatalf("expected no playerId key on an unowned deed, got %v", v)
	}
	if _, err := raw.LookupErr("_id"); err != nil {
		t.Fatalf("_id should be present: %v", err)
	}
}

// SELL (manager/propertyManager.go), the kick's BANK disposition and every
// auction close write {"playerId": null} rather than unsetting the key, and
// every later read decodes that null into the non-pointer PlayerID. It has to
// come back as the zero ObjectID and not as an error, or GetAvailableProperties
// and GetPlayersInRoom would fail on any room where a deed ever went back to
// the Bank.
func TestPropertyDecodesANullPlayerIdAsNoOwner(t *testing.T) {
	id := bson.NewObjectID()
	doc := rawOf(t, bson.M{"_id": id, "playerId": nil, "name": "Boardwalk"})

	var p Property
	if err := bson.Unmarshal(doc, &p); err != nil {
		t.Fatalf("decoding a null playerId: %v", err)
	}
	if !p.PlayerID.IsZero() {
		t.Fatalf("expected the zero ObjectID for a null playerId, got %s", p.PlayerID.Hex())
	}
	if p.ID != id {
		t.Fatalf("_id did not round-trip: got %s, want %s", p.ID.Hex(), id.Hex())
	}
}

// handleKickPlayer opens an auction with a write filtered on {"auction": nil},
// and the last close $unsets the key. A room that has never had an auction has
// to look the same on the wire as one whose auction ended: no key at all.
func TestRoomWithNoAuctionCarriesNoAuctionKey(t *testing.T) {
	raw := rawOf(t, Room{ID: bson.NewObjectID(), Code: "ABCDEF", Name: "wire"})

	if v, err := raw.LookupErr("auction"); err == nil {
		t.Fatalf("expected no auction key on a room with no auction, got %v", v)
	}
}

// The browser reads Room.auction straight out of GET /rooms/:code/players. An
// empty queue has to reach it as [] and no bidder as null with the key present
// (auctionModel.go says why), so both are pinned across the BSON round trip
// and the JSON encoding that follows it.
func TestAuctionRoundTripKeepsItsShape(t *testing.T) {
	in := Room{
		ID:   bson.NewObjectID(),
		Code: "ABCDEF",
		Auction: &Auction{
			KickedPlayerID: bson.NewObjectID(),
			PropertyID:     bson.NewObjectID(),
			Queue:          []bson.ObjectID{},
			HighBid:        0,
			HighBidderID:   nil,
		},
	}
	raw := rawOf(t, in)

	if v := raw.Lookup("auction", "highBidderId"); v.Type != bson.TypeNull {
		t.Fatalf("highBidderId should be stored as BSON null, got %v", v.Type)
	}
	if v := raw.Lookup("auction", "queue"); v.Type != bson.TypeArray {
		t.Fatalf("queue should be stored as a BSON array, got %v", v.Type)
	}

	var out Room
	if err := bson.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decoding the room: %v", err)
	}
	if out.Auction == nil {
		t.Fatal("the auction did not round-trip")
	}
	if out.Auction.Queue == nil || len(out.Auction.Queue) != 0 {
		t.Fatalf("expected an empty, non-nil queue, got %#v", out.Auction.Queue)
	}
	if out.Auction.HighBidderID != nil {
		t.Fatalf("expected no high bidder, got %s", out.Auction.HighBidderID.Hex())
	}

	js, err := json.Marshal(out.Auction)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	for _, want := range []string{`"queue":[]`, `"highBidderId":null`} {
		if !strings.Contains(string(js), want) {
			t.Fatalf("expected %s in the JSON the browser gets, got %s", want, js)
		}
	}
}

// Auction.LotCount (board row 27) landed after the pins above were written.
// frontend/components/room/auction-panel.tsx guards the "Lot N of M" row on
// auction.lotCount > 0, so a zero that reached the browser would hide the row
// for the whole auction, silently. The field has no omitempty on purpose: a
// zero LotCount still has to be written, and both a zero and a real count have
// to come back as a BSON integer and round-trip as the same number. If a codec
// default ever dropped the zero or stopped writing an integer, every gate
// would stay green and every auction would lose its lot counter.
func TestAuctionLotCountIsAlwaysWrittenAsAnInteger(t *testing.T) {
	for _, count := range []int{0, 4} {
		raw := rawOf(t, Auction{
			KickedPlayerID: bson.NewObjectID(),
			PropertyID:     bson.NewObjectID(),
			Queue:          []bson.ObjectID{},
			LotCount:       count,
		})

		v, err := raw.LookupErr("lotCount")
		if err != nil {
			t.Fatalf("lotCount %d: expected the key to be present, got %v", count, err)
		}
		if v.Type != bson.TypeInt32 && v.Type != bson.TypeInt64 {
			t.Fatalf("lotCount %d: should be stored as a BSON integer, got %v", count, v.Type)
		}

		var out Auction
		if err := bson.Unmarshal(raw, &out); err != nil {
			t.Fatalf("lotCount %d: decoding: %v", count, err)
		}
		if out.LotCount != count {
			t.Fatalf("lotCount did not round-trip: got %d, want %d", out.LotCount, count)
		}
	}
}

// frontend/types/schema.ts types every id as a string. Gin marshals these
// structs with encoding/json, so this is the ObjectID's own MarshalJSON, and
// the browser would break on any other shape.
func TestObjectIDsReachTheBrowserAsHexStrings(t *testing.T) {
	id := bson.NewObjectID()

	js, err := json.Marshal(Player{ID: id, RoomID: id, Name: "Alice"})
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if want := `"id":"` + id.Hex() + `"`; !strings.Contains(string(js), want) {
		t.Fatalf("expected %s in %s", want, js)
	}
}

// GetPlayersInRoom sorts the event history on timestamp descending. That is a
// time order only if the field is a BSON DateTime, and the value has to come
// back as the same instant, in UTC, at the millisecond precision BSON keeps.
func TestTimestampsAreStoredAsDateTime(t *testing.T) {
	stamp := time.Date(2026, 9, 17, 12, 34, 56, 789_000_000, time.UTC)
	raw := rawOf(t, EventHistory{ID: bson.NewObjectID(), RoomID: bson.NewObjectID(), TimeStamp: stamp, Event: "wire"})

	if v := raw.Lookup("timestamp"); v.Type != bson.TypeDateTime {
		t.Fatalf("timestamp should be stored as a BSON DateTime, got %v", v.Type)
	}

	var out EventHistory
	if err := bson.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	if !out.TimeStamp.Equal(stamp) {
		t.Fatalf("timestamp did not round-trip: got %v, want %v", out.TimeStamp, stamp)
	}
	if out.TimeStamp.Location() != time.UTC {
		t.Fatalf("expected UTC, got %v", out.TimeStamp.Location())
	}
}

// Offers (offerModel.go, board row 18) landed after the six pins above were
// written, and the migration that carried v2 onto main had to cover them too
// (2026-09-23). Every Offer read in websocket/offers.go and
// controllers/offerControllers.go filters on {"status": <OfferStatus>} - a
// string-kinded named type, matched against the string the document was
// stored with - and the inbox reads counterOf and the two properties arrays
// straight out of the JSON. If the named-string codec, the nil pointer's null
// or the empty slice's [] ever moved, every inbox would be empty and no offer
// could be answered, with every gate green.
func TestOfferKeepsItsShapeOnBothWires(t *testing.T) {
	deed := bson.NewObjectID()
	in := Offer{
		ID:           bson.NewObjectID(),
		RoomID:       bson.NewObjectID(),
		Status:       OfferPending,
		FromPlayerID: bson.NewObjectID(),
		ToPlayerID:   bson.NewObjectID(),
		Offer:        TradeSide{Properties: []bson.ObjectID{}, Amount: 50},
		Request:      TradeSide{Properties: []bson.ObjectID{deed}, Amount: 0},
		CreatedAt:    time.Date(2026, 9, 23, 1, 2, 3, 0, time.UTC),
		UpdatedAt:    time.Date(2026, 9, 23, 1, 2, 3, 0, time.UTC),
		CounterOf:    nil,
	}
	raw := rawOf(t, in)

	if v := raw.Lookup("status"); v.Type != bson.TypeString || v.StringValue() != string(OfferPending) {
		t.Fatalf("status should be stored as the BSON string %q, got %v %s", OfferPending, v.Type, v)
	}
	if v := raw.Lookup("counterOf"); v.Type != bson.TypeNull {
		t.Fatalf("counterOf should be present and BSON null on a first offer, got %v", v.Type)
	}
	if v := raw.Lookup("offer", "properties"); v.Type != bson.TypeArray {
		t.Fatalf("an empty properties side should be stored as a BSON array, got %v", v.Type)
	}
	if v := raw.Lookup("createdAt"); v.Type != bson.TypeDateTime {
		t.Fatalf("createdAt should be a BSON DateTime, got %v", v.Type)
	}

	var out Offer
	if err := bson.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decoding the offer: %v", err)
	}
	if out.Status != OfferPending {
		t.Fatalf("status did not round-trip: got %q", out.Status)
	}
	if out.CounterOf != nil {
		t.Fatalf("expected no counterOf, got %s", out.CounterOf.Hex())
	}
	if out.Offer.Properties == nil || len(out.Offer.Properties) != 0 {
		t.Fatalf("expected an empty, non-nil properties slice, got %#v", out.Offer.Properties)
	}
	if len(out.Request.Properties) != 1 || out.Request.Properties[0] != deed {
		t.Fatalf("request properties did not round-trip: %#v", out.Request.Properties)
	}

	js, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	for _, want := range []string{`"status":"PENDING"`, `"counterOf":null`, `"properties":[]`} {
		if !strings.Contains(string(js), want) {
			t.Fatalf("expected %s in the JSON the inbox gets, got %s", want, js)
		}
	}
}
