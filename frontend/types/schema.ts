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
