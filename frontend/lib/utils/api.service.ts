import API from "./api";
import { EventHistory, Offer, Player, Property, Room } from "@/types/schema";

const handleApiResponse = <T>(
  promise: Promise<{ data: T; status: number }>,
): Promise<ApiResponse<T>> => {
  return promise
    .then((response) => {
      return {
        success: true,
        data: response.data,
        status: response.status,
      };
    })
    .catch((error) => {
      if (error.message === "Network Error") {
        return {
          success: false,
          error: { message: "You're offline. Please check your connection." },
          status: 0,
          isOffline: true,
        };
      }

      return {
        success: false,
        error: error.response?.data || { message: error.message },
        status: error.response?.status || 500,
        isOffline: false,
      };
    });
};

const API_VERSION = "v1";

const ROOMS_BASE = `/${API_VERSION}/rooms`;
const ROOM = (code: string) => `${ROOMS_BASE}/${code}`;

const ROOM_PLAYERS = (code: string) => `${ROOM(code)}/players`;
const ROOM_PLAYER = (code: string, playerId: string) =>
  `${ROOM_PLAYERS(code)}/${playerId}`;

const PLAYER_PROPERTIES = (code: string, playerId: string) =>
  `${ROOM_PLAYER(code, playerId)}/properties`;
const PLAYER_PROPERTY = (code: string, playerId: string, propertyId: string) =>
  `${PLAYER_PROPERTIES(code, playerId)}/${propertyId}`;

export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  status?: number;
  error?: any;
}

async function apiRequest<T>(
  method: "get" | "post" | "put" | "delete",
  endpoint: string,
  body?: unknown,
  params?: Record<string, string>,
): Promise<ApiResponse<T>> {
  const url = endpoint.startsWith("/") ? endpoint : `/${endpoint}`;
  const config = params ? { params } : undefined;

  let promise: Promise<{ data: T; status: number }>;
  switch (method) {
    case "get":
      promise = API.get(url, config);
      break;
    case "post":
      promise = API.post(url, body ? body : undefined, config);
      break;
    case "put":
      promise = API.put(url, body ? body : undefined, config);
      break;
    case "delete":
      promise = API.delete(url, { data: body, ...config });
      break;
  }

  return handleApiResponse(promise);
}

// Request bodies, kept local to this file: they are what a form sends, not a
// domain shape from `types/schema.ts`. Matches `backend/controllers/roomControllers.go`'s
// `requestBody` structs.
interface CreateRoomRequest {
  playerName: string;
  roomName: string;
  roomCode: string;
  playerColor: string;
  startingCash?: number;
  houses?: number;
  hotels?: number;
}

interface JoinRoomRequest {
  roomCode: string;
  playerName: string;
  playerColor: string;
}

interface CreateRoomResponse {
  roomId: string;
  roomCode: string;
  playerId: string;
}

interface GetPlayersResponse {
  players: Player[];
  room: Room;
  eventHistory: EventHistory[];
  // Only present when the request carried a `playerId` query param -- see
  // `GetPlayersInRoom` (`backend/controllers/playerControllers.go:108-115`).
  // Nothing in the frontend passes that param today, so this is never sent
  // in practice; typed for completeness rather than as a live path.
  existingPlayer?: { id: string; name: string; color: string; isValid: true };
}

interface JoinRoomResponse {
  message: string;
  playerId: string;
  players: Player[];
  room: Room;
  roomCode: string;
}

export const roomApi = {
  create: (data: CreateRoomRequest) =>
    apiRequest<CreateRoomResponse>("post", ROOMS_BASE, data),
  getPlayers: (code: string) =>
    apiRequest<GetPlayersResponse>("get", ROOM_PLAYERS(code)),
  getProperties: (code: string) =>
    apiRequest<{ availableProperties: Property[]; roomId: string }>(
      "get",
      `${ROOM(code)}/properties`,
    ),
  checkExistingRoom: (code: string) =>
    apiRequest<{ exists: boolean }>("get", `${ROOM(code)}/exists`),
  // The inbox read: every PENDING offer this player made or was made to
  // (`controllers.GetPendingOffers`). Fetched on mount and again on every
  // websocket message, so a reload mid-offer and a dropped frame both find
  // the offer where it lives, in Mongo.
  getOffers: (code: string, playerId: string) =>
    apiRequest<{ offers: Offer[] }>(
      "get",
      `${ROOM(code)}/offers`,
      undefined,
      { playerId },
    ),
};

export const playerApi = {
  join: (code: string, data: JoinRoomRequest) =>
    apiRequest<JoinRoomResponse>("post", ROOM_PLAYERS(code), data),

  getDetails: (code: string, playerId: string) =>
    apiRequest<{ player: Player; properties: Property[] }>(
      "get",
      ROOM_PLAYER(code, playerId),
    ),

  // These three call an empty stub handler on the backend
  // (`backend/controllers/propertyControllers.go:15-17` -- `MortgageProperty`,
  // `AddProperty` and `RemoveProperty` all have an empty body, no response at
  // all) and nothing in the frontend calls them (grepped 2026-09-17). `unknown`
  // rather than a guessed shape, because there is no real contract yet.
  addProperty: (code: string, playerId: string, propertyId: string) =>
    apiRequest<unknown>("post", PLAYER_PROPERTY(code, playerId, propertyId)),

  removeProperty: (code: string, playerId: string, propertyId: string) =>
    apiRequest<unknown>("delete", PLAYER_PROPERTY(code, playerId, propertyId)),

  mortgageProperty: (code: string, playerId: string, propertyId: string) =>
    apiRequest<unknown>(
      "post",
      `${PLAYER_PROPERTY(code, playerId, propertyId)}/mortgage`,
    ),
};
