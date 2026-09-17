import { WebSocketPayload } from "@/types/payloads";
import { toast } from "sonner";

// ws - The WebSocket instance.
// type - The type of message being sent.
// payload - The payload to send.
const sendWebSocketMessage = (
  ws: WebSocket | null,
  type:
    | "TRANSFER"
    | "JOIN"
    | "PURCHASE_PROPERTY"
    | "BANKER_TRANSACTION"
    | "FREE_PARKING"
    | "MANAGE_PROPERTIES"
    | "KICK_PLAYER"
    | "PLACE_BID"
    | "CLOSE_AUCTION",
  payload: WebSocketPayload
) => {
  if (ws?.readyState === WebSocket.OPEN) {
    try {
      ws.send(JSON.stringify({ type, payload }));
    } catch (error) {
      console.error("Error sending message:", error);
      toast.error("Failed to send message");
    }
  } else {
    console.error("WebSocket not connected. State:", ws?.readyState);
    toast.error("Connection lost. Trying to reconnect...");
  }
};
export const sendMessage = (
  ws: WebSocket | null,
  type:
    | "TRANSFER"
    | "JOIN"
    | "PURCHASE_PROPERTY"
    | "BANKER_TRANSACTION"
    | "FREE_PARKING"
    | "MANAGE_PROPERTIES"
    | "KICK_PLAYER"
    | "PLACE_BID"
    | "CLOSE_AUCTION",
  payload: WebSocketPayload
) => {
  sendWebSocketMessage(ws, type, payload);
};
