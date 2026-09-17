export type WebSocketPayload =
  | TransferPayload
  | JoinPayload
  | PurchasePropertyPayload
  | BankerTransactionPayload
  | ManagePropertiesPayload
  | FreeParkingPayload
  | KickPlayerPayload
  | PlaceBidPayload
  | CloseAuctionPayload;

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

// A raise on the open auction lot (`handlePlaceBid`).
//
// `amount` is a JSON number, not the string `TransferPayload` and
// `FreeParkingPayload` send: those are strings because a keypad produces them,
// and the server parses a bid with a whole-dollars check that refuses 120.9
// rather than truncating it -- a truncated bid is a bid nobody made, and it is
// the truncated figure that gets charged.
//
// `propertyId` names the lot being bid on and is required. It is what makes a
// bid that arrives after the hammer fail instead of landing on the next deed:
// a close advances the open lot, so the server's filter no longer matches.
export interface PlaceBidPayload {
  type: "PLACE_BID";
  roomId: string;
  propertyId: string;
  bidderId: string;
  amount: number;
}

// The Banker's hammer (`handleCloseAuction`). This is the only thing that ends
// a lot -- there is no timer anywhere in the backend (D14).
//
// Both `propertyId` and `kickedPlayerId` are required, and together they are
// the lot's identity. A property id alone identifies a deed, not an auction
// lot, and the same deed can be the open lot of two different auctions; the
// pair cannot repeat, because a player cannot be kicked twice. Sending the pair
// the panel is actually displaying is what makes a stale screen say so instead
// of hammering a deed the Banker was not looking at.
export interface CloseAuctionPayload {
  type: "CLOSE_AUCTION";
  roomId: string;
  // The Banker closing the lot. Named `playerId` on the wire, not `bankerId`:
  // the server reads it and then checks the role, and this is the one
  // server-side isBanker check in the product.
  playerId: string;
  propertyId: string;
  kickedPlayerId: string;
}
