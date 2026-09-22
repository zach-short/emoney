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
