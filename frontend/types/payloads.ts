export type WebSocketPayload =
  | TransferPayload
  | JoinPayload
  | PurchasePropertyPayload
  | BankerTransactionPayload
  | ManagePropertiesPayload
  | FreeParkingPayload
  | KickPlayerPayload
  | CreateOfferPayload
  | RespondOfferPayload;

export type TransferType =
  | "SEND"
  | "REQUEST"
  | "ADD"
  | "SUBTRACT"
  | "BANKER_ADD"
  | "BANKER_REMOVE";

export interface TransferPayload {
  type: "TRANSFER";
  amount: string;
  transferType: TransferType;
  fromPlayerId?: string;
  toPlayerId?: string;
  reason: string;
  roomId: string;
}

export type ManagePropertiesPayload =
  | {
      managementType: "HOUSES";
      playerId: string;
      properties: { propertyId: string; count: number }[];
      roomId: string;
      amount: number;
    }
  | {
      managementType: "MORTGAGE" | "UNMORTGAGE" | "SELL";
      playerId: string;
      properties: { propertyId: string }[];
      roomId: string;
      amount: number;
    };

export interface BankerTransactionPayload {
  type: "BANKER_TRANSACTION";
  amount: string;
  fromPlayerId: string;
  toPlayerId: string;
  transactionType: "BANKER_ADD" | "BANKER_REMOVE";
  roomId: string;
}

export interface FreeParkingPayload {
  type: "FREE_PARKING";
  freeParkingType: "ADD" | "REMOVE";
  amount: string;
  playerId: string;
  roomId: string;
}

interface JoinPayload {
  playerId: string;
}

export interface KickPlayerPayload {
  type: "KICK_PLAYER";
  roomId: string;
  targetPlayerId: string;
  // Three arms, as of Phase 3. This union and the Go handler's guard
  // (`backend/websocket/websocketManager.go`, handleKickPlayer) are changed in
  // the same commit, always: they are the two halves of one contract and
  // nothing checks they agree, so a member here that the guard refuses
  // typechecks straight into a runtime rejection (PLAN.md BD-6). That is why
  // "AUCTION" was absent until the auction existed rather than present and
  // disabled.
  disposition: "BANK" | "FREEZE" | "AUCTION";
  // Required by the server only when the target holds the banker role (D5), and
  // refused when they do not. Omitted, null and "" all read as "no successor
  // named" there, so leaving it undefined is the correct non-banker shape.
  successorPlayerId?: string;
}

export interface PurchasePropertyPayload {
  type: "PURCHASE_PROPERTY";
  propertyId: string;
  buyerId: string;
  price: number;
  roomId: string;
}

// A trade proposal, read by `handleCreateOffer` in
// `backend/websocket/offers.go`. Amounts are JSON numbers in whole dollars, like
// a bid; the Go side refuses a fraction rather than truncating it. Either side
// may be empty but not both, and `note` is capped at 280 characters there and in
// the form's `maxLength` - the two numbers are kept equal by hand.
//
// A counter is this same message with `counterOf` set to the id of the offer it
// answers: the Go side marks that one COUNTERED in the same transaction as it
// stores this one. There is no COUNTER response - see `RespondOfferPayload`.
export interface CreateOfferPayload {
  type: "CREATE_OFFER";
  roomId: string;
  fromPlayerId: string;
  toPlayerId: string;
  offer: { properties: string[]; amount: number };
  request: { properties: string[]; amount: number };
  note: string;
  counterOf?: string;
}

// The answer to a pending offer, read by `handleRespondOffer`. ACCEPT and DENY
// are the recipient's; WITHDRAW is the sender taking their own offer back, and
// the Go side refuses each from the wrong player. ACCEPT settles the trade in
// one transaction and is the only response that moves anything.
export interface RespondOfferPayload {
  type: "RESPOND_OFFER";
  roomId: string;
  offerId: string;
  playerId: string;
  response: "ACCEPT" | "DENY" | "WITHDRAW";
}
