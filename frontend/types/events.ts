import { Offer, Property } from "./schema";

export interface PropertyUpdate {
  type: "PROPERTY_UPDATE";
  property: Property;
  playerId: string;
}

export interface BalanceUpdate {
  type: "BALANCE_UPDATE";
  playerId: string;
  newBalance: number;
}

// Broadcast to the whole room when the banker removes a player
// (`backend/websocket/websocketManager.go`, handleKickPlayer). `notification`
// is the prose every client toasts and `CreateEventHistory` stores; `playerId`
// is the removed player's.
//
// Nothing imports this file yet -- neither of the two shapes above is used
// either. It is the frontend's only written record of an inbound wire shape,
// and the thing to grep before renaming the Go event name, which is a bare
// string literal on that side (CLAUDE.md, "The websocket contract is typed on
// the frontend only").
export interface PlayerKicked {
  type: "PLAYER_KICKED";
  notification: string;
  playerId: string;
}

// The four trade events (`backend/websocket/offers.go`). The first three are
// the first messages in the app sent to ONE player rather than the room, through
// `RoomManager.SendTo`: an offer is not room news. Every tab that player has
// open gets them. The fourth is a broadcast, because a settled trade is.
//
// The room page refetches the inbox on every message it receives, whatever the
// type, so a dropped frame here never loses a trade - the offer is in Mongo and
// the next fetch finds it.

// To the player the offer was made to.
export interface OfferReceived {
  type: "OFFER_RECEIVED";
  notification: string;
  offer: Offer;
}

// To the player who made it - their own confirmation.
export interface OfferSent {
  type: "OFFER_SENT";
  notification: string;
  offer: Offer;
}

// To both players when an offer is declined or withdrawn. `notification` is
// written from each reader's side, so the two copies differ.
export interface OfferResolved {
  type: "OFFER_RESOLVED";
  notification: string;
  offerId: string;
  status: "DENIED";
}

// To the whole room when a trade settles. `notification` is the sentence the
// event history also stores, minus the note.
export interface OfferAccepted {
  type: "OFFER_ACCEPTED";
  notification: string;
  offerId: string;
  fromPlayerId: string;
  toPlayerId: string;
}

// The three the auction broadcasts (`backend/websocket/websocketManager.go`:
// handleKickPlayer's AUCTION arm, handlePlaceBid, handleCloseAuction). Every
// one of them carries `notification`, the prose the room reads; the rest of
// each payload is what the panel needs in order to react without waiting for a
// refetch.

// Raised by an AUCTION kick, once, after PLAYER_KICKED and only when the
// removed player actually held a deed. `lotCount` is how many deeds the whole
// auction will run through. It is corrected as of PASSOFF.md row 27:
// `models.Auction` now carries the same total, as `lotCount`, on the Room
// document itself (`frontend/types/schema.ts`), which is what a client that
// reloads mid-auction actually reads. The field here stays transient and
// panel-facing code still does not read it -- it is redundant with the
// broadcast's `notification` and `propertyId`, kept only because every other
// arm of this union states what changed.
export interface AuctionStarted {
  type: "AUCTION_STARTED";
  notification: string;
  propertyId: string;
  kickedPlayerId: string;
  lotCount: number;
}

// Raised by every accepted bid. This is the one broadcast the room handler
// must NOT toast and must NOT refetch on (PLAN.md section 3, "Toast per bid"):
// a lot can take twenty bids and each one reaches every client. The payload
// carries the whole of the new high bid, which is what makes the no-refetch
// path possible -- the panel applies it directly.
export interface BidPlaced {
  type: "BID_PLACED";
  notification: string;
  propertyId: string;
  bidderId: string;
  amount: number;
}

// Raised by the Banker's hammer. `winnerId` and `amount` are present only when
// the lot actually sold -- the three other outcomes (nobody bid, the winner
// could not cover it, the winner has since been removed) all send the deed to
// the Bank and omit both keys, so their absence is the wire's way of saying
// "no money moved". The notification says which of the three happened.
export interface AuctionLotClosed {
  type: "AUCTION_LOT_CLOSED";
  notification: string;
  propertyId: string;
  winnerId?: string;
  amount?: number;
}
