import { Property } from "./schema";

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
