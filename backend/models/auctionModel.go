package models

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Auction is one live auction of one kicked player's estate. It lives on the
// Room document rather than in RoomManager (design D18) for two reasons that
// in-memory state cannot give: a client that reconnects mid-auction learns the
// current high bid, because every client already refetches the whole room on
// every broadcast (GET /rooms/:code/players returns the Room document whole,
// controllers/playerControllers.go), so the auction reaches every screen with
// no new REST route; and the backend is one process deployed by hand with no
// drain, so an in-memory auction would vanish silently on every deploy.
//
// A Room carries at most one of these at a time. Room.Auction is a pointer and
// is absent on a room with no auction running; handleKickPlayer's write is
// conditional on that absence, which is what stops a second AUCTION kick from
// clobbering a running one.
//
// Exactly one deed is open at a time, in board order, and the banker advances
// (D16): PropertyID is the open lot and Queue holds the deeds still to come,
// already sorted by Property.PropertyIndex when the auction was created.
type Auction struct {
	// KickedPlayerID is whose estate this is. It is read to refuse a bid from
	// the removed player themselves - they have no socket after the kick's
	// force-close and no standing in the room, but the payload carries a
	// player id and nothing else here would catch it.
	KickedPlayerID primitive.ObjectID `bson:"kickedPlayerId" json:"kickedPlayerId"`

	// PropertyID is the open lot: the one deed bids are being taken on. A bid
	// naming any other property is refused, which is also what makes a bid
	// that arrives after its lot closed fail rather than land on the next one.
	PropertyID primitive.ObjectID `bson:"propertyId" json:"propertyId"`

	// Queue is the deeds still to come, in PropertyIndex order, not including
	// the open lot. Empty means this is the last lot and closing it ends the
	// auction.
	Queue []primitive.ObjectID `bson:"queue" json:"queue"`

	// HighBid is the current high bid on the open lot, in whole dollars. A lot
	// opens at 0 and the minimum raise is 1 (D15), so "beats the high bid" and
	// "is at least a dollar more" are the same comparison on integers.
	HighBid int `bson:"highBid" json:"highBid"`

	// HighBidderID is who placed HighBid, or nil when nobody has bid yet. A
	// pointer rather than a zero ObjectID: the zero value serialises as
	// "000000000000000000000000", which a client would have to know to treat
	// as "nobody". nil serialises as null, and no omitempty, so the key is
	// always present on the wire.
	HighBidderID *primitive.ObjectID `bson:"highBidderId" json:"highBidderId"`

	// LotCount is how many deeds this auction runs through in total: the open
	// lot plus everything in Queue at the moment handleKickPlayer opens it. It
	// does not shrink as lots close - PropertyID and Queue already do that job
	// - which is what lets a client derive the open lot's 1-based position as
	// LotCount - len(Queue) and show "lot 2 of 4" even after a reload. Before
	// this field the total was only ever stated in AUCTION_STARTED's transient
	// lotCount, which a reloading client never sees again (PASSOFF.md row 27).
	LotCount int `bson:"lotCount" json:"lotCount"`
}
