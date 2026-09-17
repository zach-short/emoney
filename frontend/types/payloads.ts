export type WebSocketPayload =
  | TransferPayload
  | JoinPayload
  | PurchasePropertyPayload
  | BankerTransactionPayload
  | ManagePropertiesPayload
  | FreeParkingPayload
  | KickPlayerPayload;

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
