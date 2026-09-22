package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OfferStatus is the four-state life of an offer. It mirrors the union in
// frontend/types/schema.ts exactly, and nothing checks the two agree (CLAUDE.md,
// "The websocket contract is typed on the frontend only") - change both.
type OfferStatus string

const (
	// OfferPending is an offer waiting on the player it was made to. It is the
	// only state the inbox shows and the only state a response can move.
	OfferPending OfferStatus = "PENDING"
	// OfferDenied is an offer the recipient declined - or one the sender
	// withdrew, which lands here too rather than in a fifth state, because the
	// four-state union is the fixed contract and the notification says which.
	OfferDenied OfferStatus = "DENIED"
	// OfferAccepted is a settled trade: the deeds and the cash have moved, in
	// one transaction, exactly once.
	OfferAccepted OfferStatus = "ACCEPTED"
	// OfferCountered is an offer answered with a new offer the other way. The
	// counter carries this offer's id in CounterOf; this one moves nothing.
	OfferCountered OfferStatus = "COUNTERED"
)

// TradeSide is one half of a trade: what one player hands over. An offer has
// two - Offer is what the sender gives, Request is what they want back - and
// either half may be empty, but not both (handleCreateOffer refuses that).
//
// Properties is never nil on a stored document: handleCreateOffer builds it
// with make, so it marshals as [] rather than null and the browser's
// `.length` reads do not have to guard. Amount is whole dollars and may be 0.
type TradeSide struct {
	Properties []primitive.ObjectID `bson:"properties" json:"properties"`
	Amount     int                  `bson:"amount" json:"amount"`
}

// Offer is one peer-to-peer trade proposal between two players in a room. It
// mirrors frontend/types/schema.ts's Offer field for field, and it is stored
// rather than held in process memory for the reason the auction is: a client
// that reloads mid-offer must still see it (the inbox fetches on mount), and a
// dropped websocket frame must not lose a trade.
//
// Note is the free-text half of the feature and it is deliberately free text
// rather than structure. This app has no concept of a turn and no board
// position, so a deal like "immunity on the browns for three turns" cannot be
// counted or enforced under any design; recording it as data the app displays
// and never acts on would be a worse lie than silence, because structure
// implies enforcement. The note is a handshake the app writes down, and the
// UI copy says so.
type Offer struct {
	ID           primitive.ObjectID `bson:"_id" json:"id"`
	RoomID       primitive.ObjectID `bson:"roomId" json:"roomId"`
	Status       OfferStatus        `bson:"status" json:"status"`
	FromPlayerID primitive.ObjectID `bson:"fromPlayerId" json:"fromPlayerId"`
	ToPlayerID   primitive.ObjectID `bson:"toPlayerId" json:"toPlayerId"`
	Offer        TradeSide          `bson:"offer" json:"offer"`
	Request      TradeSide          `bson:"request" json:"request"`
	Note         string             `bson:"note" json:"note"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
	// CounterOf is the offer this one answers, when it is a counter, else nil.
	// A pointer so it serialises as null rather than a zero ObjectID the
	// browser would have to recognise; no omitempty so the key is always on
	// the wire.
	CounterOf *primitive.ObjectID `bson:"counterOf" json:"counterOf"`
}
