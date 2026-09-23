export type Room = {
  id: string;
  name: string;
  // The API serialises this as `code` (backend/models/roomModel.go). Calling it
  // `roomCode` here typechecked fine and was always undefined at runtime, which
  // is why the menu's Room Code row rendered blank and copied "undefined".
  code: string;
  bankerId: string;
  createdAt: Date;
  isActive: boolean;
  freeParking: number;
  // Present only while an auction is running: Go tags it
  // `json:"auction,omitempty"` on a pointer field
  // (backend/models/roomModel.go), so a room with no auction carries no key at
  // all rather than a null. Optional here for exactly that reason -- `room`
  // comes back whole from `GET /rooms/:code/players`, which is why the live
  // auction needs no REST route of its own (D18).
  auction?: Auction;
};

// One live auction of one kicked player's estate, mirroring
// `backend/models/auctionModel.go`. Read-only on this side: the client never
// writes it, it only renders what the last room fetch returned and raises
// PLACE_BID / CLOSE_AUCTION against it.
//
// A room carries at most one. Exactly one deed is open at a time, in board
// order, and the Banker advances it by hand (D14, D16).
export type Auction = {
  // Whose estate is being sold. The removed player may not bid on it, and it
  // is half of the (kickedPlayerId, propertyId) pair that identifies an
  // auction lot -- a deed alone does not, because the same deed can be the
  // open lot of two different auctions.
  kickedPlayerId: string;
  // The open lot: the one deed bids are being taken on.
  propertyId: string;
  // The deeds still to come, in board order, NOT including the open lot. Empty
  // means this is the last one. These deeds still belong to the kicked player
  // until each lot closes, so their names resolve out of that player's
  // `properties` -- nothing here needs a second fetch.
  queue: string[];
  // Whole dollars. A lot opens at 0 and the minimum raise is 1 (D15), so "beats
  // the high bid" and "is at least a dollar more" are the same comparison.
  highBid: number;
  // null until somebody bids. Go writes the key unconditionally (no
  // `omitempty`), so this is null rather than absent.
  highBidderId: string | null;
};

export type Player = {
  id: string;
  roomId: string;
  deviceId: string;
  name: string;
  color: string;
  balance: number;
  isActive: boolean;
  isBanker: boolean;
  properties?: Property[];
};

export type Property = {
  id: string;
  roomId: string;
  playerId: string;
  name: string;
  color: string;
  price: number;
  group: string;
  developmentLevel: number;
  images: string[];
  isMortgaged: boolean;
  rentPrices: number[];
  houseCost?: number;
  propertyIndex: number;
};

export type Transfer = {
  id: string;
  roomId: string;
  fromPlayerId: string | null; // null if from bank
  toPlayerId: string | null; // null if to bank
  amount: number;
  reason: string; // "transfer" | "rent" | "tax" | "chance" etc.
  timestamp: Date;
  status: string; // "pending" | "completed" | "rejected"
};

export type EventHistory = {
  id: string;
  roomId: string;
  event: string;
  timestamp: Date;
  eventType: string[];
};

// type Immunity {
//   propertyId?: string;
//   propertyGroup?: string;
//   count: number;
// }

// One half of a trade: what one player hands over. Mirrors `TradeSide` in
// `backend/models/offerModel.go` field for field; the Go side always sends both
// keys, `properties` as an array of property ids (never null) and `amount` in
// whole dollars.
export type Trade = {
  properties?: string[];
  amount?: number;
  // immunity?: Immunity[];
};

export type OfferStatus = "PENDING" | "DENIED" | "ACCEPTED" | "COUNTERED";

// Mirrors `Offer` in `backend/models/offerModel.go`. `note` is free text on
// purpose: this app has no turns and no board position, so a deal like
// "immunity on the browns for three turns" cannot be enforced under any
// design, and the note is a handshake the app writes down rather than a rule
// it applies. The commented-out `Immunity` above is the structured version
// that was started and stopped, and it stays stopped (HANDOFF 26).
export type Offer = {
  id: string;
  roomId: string;
  status: OfferStatus;
  fromPlayerId: string;
  toPlayerId: string;
  offer: Trade;
  request: Trade;
  createdAt: Date;
  updatedAt: Date;
  note?: string;
  // The offer this one answers when it is a counter. The Go side sends null
  // for a fresh offer; the form leaves it undefined and the payload drops it.
  counterOf?: string | null;
};

export type OfferNoID = Omit<Offer, "id">;
