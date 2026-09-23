"use client";
import {
  EventHistory,
  Offer,
  OfferNoID,
  Player,
  Property,
  Room,
} from "@/types/schema";
import { use, useEffect, useEffectEvent, useRef, useState } from "react";
import RoomView from "@/components/room/room.client";
import { getWsUrl } from "@/lib/utils/wsHelpers";
import { playerStore } from "@/lib/utils/playerHelpers";
import { toast } from "sonner";
import { sendMessage } from "@/lib/utils/sendWsMessage";
import {
  KickPlayerPayload,
  ManagePropertiesPayload,
  RespondOfferPayload,
} from "@/types/payloads";
import { usePublicFetch } from "@/hooks/use-public-fetch";
import { roomApi } from "@/lib/utils/api.service";
import DataState from "@/components/containers/data-state";

interface WebSocketMessage {
  type: string;
  // Go sends `Message{Type string, Payload interface{}}` and builds each payload at the
  // send site, so the shape is per-type and nothing checks the two sides agree. Every
  // broadcast type sends an object carrying `notification`; `ERROR` alone sends the bare
  // error string (`backend/websocket/handler.go:116-121`).
  payload: { notification?: string } | string;
}

const RoomPage = ({ params }: { params: Promise<{ code: string }> }) => {
  const { code } = use(params);
  const [initialLoadComplete, setInitialLoadComplete] = useState(false);

  const ws = useRef<WebSocket | null>(null);
  const storedPlayerId = playerStore.getPlayerIdForRoom(code);

  const {
    data: playersData,
    error: playersError,
    loading: playersLoading,
    refetch: refetchPlayers,
  } = usePublicFetch<{
    players: Player[];
    room: Room;
    eventHistory: EventHistory[];
  }>(roomApi.getPlayers, {
    resourceParams: [code, storedPlayerId],
    dependencies: [code, storedPlayerId],
    enabled: !!code && !!storedPlayerId,
  });

  const {
    data: propertiesData,
    error: propertiesError,
    loading: propertiesLoading,
    refetch: refetchProperties,
  } = usePublicFetch<{ availableProperties: Property[]; roomId: string }>(
    roomApi.getProperties,
    {
      resourceParams: [code],
      dependencies: [code],
      enabled: !!code,
    },
  );

  // The inbox: every PENDING offer this player made or was made to. Its own
  // fetch rather than a field on the players payload, because that payload
  // goes to every client in the room and an offer is not room news. Fetched
  // on mount -- an offer lives in Mongo so a reload mid-offer finds it --
  // and refetched on every websocket message below, so a dropped
  // OFFER_RECEIVED frame is caught by the next frame of any kind.
  const { data: offersData, refetch: refetchOffers } = usePublicFetch<{
    offers: Offer[];
  }>(roomApi.getOffers, {
    resourceParams: [code, storedPlayerId],
    dependencies: [code, storedPlayerId],
    enabled: !!code && !!storedPlayerId,
  });

  // Straight projections of the fetched room payload -- no effect needed, and
  // this keeps the players, room and history from lagging a render behind the
  // data that produced them.
  const player: Player | null =
    playersData?.players?.find((p: Player) => p.id === storedPlayerId) || null;
  const otherPlayers: Player[] =
    playersData?.players?.filter((p: Player) => p.id !== storedPlayerId) || [];
  const room: Room | undefined = playersData?.room;
  const eventHistory: EventHistory[] = playersData?.eventHistory || [];
  const offers: Offer[] = offersData?.offers || [];

  const handleCreateOffer = (offer: OfferNoID) => {
    if (!player?.id || !room?.id) return;

    // Whole dollars, floored: the percent buttons in `make-offer/amount.tsx`
    // floor too, but the Go side refuses a fraction outright rather than
    // truncating (`parseTradeSide`), so the floor here is the last word.
    // `counterOf` left undefined is dropped by JSON.stringify, which the Go
    // side reads as a fresh offer.
    sendMessage(ws.current, "CREATE_OFFER", {
      type: "CREATE_OFFER",
      roomId: room.id,
      fromPlayerId: offer.fromPlayerId,
      toPlayerId: offer.toPlayerId,
      offer: {
        properties: offer.offer.properties ?? [],
        amount: Math.floor(offer.offer.amount ?? 0),
      },
      request: {
        properties: offer.request.properties ?? [],
        amount: Math.floor(offer.request.amount ?? 0),
      },
      note: offer.note ?? "",
      counterOf: offer.counterOf ?? undefined,
    });
  };

  const handleRespondOffer = (
    offerId: string,
    response: RespondOfferPayload["response"],
  ) => {
    if (!player?.id || !room?.id) return;

    sendMessage(ws.current, "RESPOND_OFFER", {
      type: "RESPOND_OFFER",
      roomId: room.id,
      offerId,
      playerId: player.id,
      response,
    });
  };

  const handleBankerTransaction = (
    amount: string,
    targetPlayerId: string,
    transactionType: "BANKER_ADD" | "BANKER_REMOVE",
  ) => {
    if (!player?.id || !room?.id) return;

    sendMessage(ws.current, "BANKER_TRANSACTION", {
      type: "BANKER_TRANSACTION",
      amount,
      fromPlayerId: player.id,
      toPlayerId: targetPlayerId,
      transactionType,
      roomId: room.id,
    });
  };

  const handleKickPlayer = (
    targetPlayerId: string,
    disposition: KickPlayerPayload["disposition"],
    successorPlayerId?: string,
  ) => {
    if (!player?.id || !room?.id) return;

    // `successorPlayerId` left undefined is dropped by JSON.stringify, which is
    // the shape the Go handler reads as "no successor named" -- it is required
    // there only for a banker target (D5) and refused for any other.
    sendMessage(ws.current, "KICK_PLAYER", {
      type: "KICK_PLAYER",
      roomId: room.id,
      targetPlayerId,
      disposition,
      successorPlayerId,
    });
  };

  const handlePurchaseProperty = (
    propertyId: string,
    buyerId: string,
    price: number,
  ) => {
    sendMessage(ws.current, "PURCHASE_PROPERTY", {
      type: "PURCHASE_PROPERTY",
      propertyId,
      buyerId,
      price,
      roomId: code,
    });
  };

  // An effect event so the socket always calls the current refetchers without
  // the connection effect having to re-run (and reconnect) on every render.
  const handleWebSocketNotification = useEffectEvent(
    (message: WebSocketMessage) => {
      const text =
        typeof message.payload === "string"
          ? message.payload
          : message.payload.notification;

      if (message.type === "ERROR") {
        // The server rejects before it writes, so nothing moved and there is nothing to
        // refetch. Without this branch the rejection rendered as an empty green success.
        toast.error(text || "Something went wrong", {
          duration: 4000,
          position: "top-center",
          className: `font-semibold text-sm text-center`,
        });
        return;
      }

      toast.success(text, {
        duration: 4000,
        icon: getIconForType(message.type),
        position: "top-center",
        className: `font-semibold text-sm text-center`,
      });

      // Every frame refetches the inbox, not only the offer ones: that is
      // what makes a dropped OFFER_RECEIVED recoverable, and the reconnect's
      // own JOIN broadcast covers a socket that was down when it was sent.
      refetchOffers();

      // The three private offer frames move no money and change no card, so
      // the room read is skipped for them; everything else refetches it.
      if (
        !["OFFER_RECEIVED", "OFFER_SENT", "OFFER_RESOLVED"].includes(
          message.type,
        )
      ) {
        refetchPlayers();
      }

      // PLAYER_KICKED belongs here because a `BANK` disposition clears
      // `playerId` on the target's deeds, and `GetAvailableProperties` is the
      // read that filters on exactly that -- without the refetch the freed
      // deeds do not appear in Bank's Properties until some later property
      // event happens to fire. A `FREEZE` kick writes no property, so this is
      // one wasted fetch on that arm; the message does not say which arm ran.
      if (
        ["PURCHASE_PROPERTY", "MANAGE_PROPERTIES", "PLAYER_KICKED"].includes(
          message.type,
        )
      ) {
        refetchProperties();
      }
    },
  );

  const handleFreeParkingAction = (
    amount: string,
    freeParkingType: "ADD" | "REMOVE",
    playerId: string,
  ) => {
    if (!room?.id) return;

    sendMessage(ws.current, "FREE_PARKING", {
      type: "FREE_PARKING",
      freeParkingType,
      amount,
      playerId,
      roomId: room.id,
    });
  };

  const handleManageProperties = (
    amount: number,
    managementType: ManagePropertiesPayload["managementType"],
    properties: { propertyId: string; count?: number }[],
    playerId: string,
  ) => {
    if (!room?.id) return;

    sendMessage(ws.current, "MANAGE_PROPERTIES", {
      managementType,
      playerId,
      properties,
      roomId: room.id,
      amount,
    });
  };

  const handleTransfer = (
    amount: string,
    transferType:
      | "SEND"
      | "REQUEST"
      | "ADD"
      | "SUBTRACT"
      | "BANKER_ADD"
      | "BANKER_REMOVE",
    transferDetails: {
      fromPlayerId?: string;
      toPlayerId?: string;
      reason: string;
      roomId: string;
    },
  ) => {
    sendMessage(ws.current, "TRANSFER", {
      type: "TRANSFER",
      amount,
      transferType: transferType,
      fromPlayerId: transferDetails.fromPlayerId,
      toPlayerId: transferDetails.toPlayerId,
      reason: transferDetails.reason,
      roomId: transferDetails.roomId,
    });
  };

  useEffect(() => {
    if (!storedPlayerId) {
      toast.error("No player found for this room");
      return;
    }

    let disposed = false;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;

    const connect = () => {
      if (disposed || ws.current?.readyState === WebSocket.OPEN) return;

      const socket = new WebSocket(getWsUrl(code));
      ws.current = socket;

      socket.onopen = () => {
        sendMessage(socket, "JOIN", { playerId: storedPlayerId });
      };

      socket.onerror = (error) => {
        console.error("WebSocket error:", error);
      };

      socket.onclose = () => {
        // Don't resurrect the socket the cleanup below just closed.
        if (disposed) return;
        reconnectTimer = setTimeout(connect, 1000);
      };

      socket.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data);
          handleWebSocketNotification(message);
        } catch (error) {
          console.error("Error processing WebSocket message:", error);
        }
      };
    };

    connect();

    return () => {
      disposed = true;
      clearTimeout(reconnectTimer);
      ws.current?.close();
      ws.current = null;
    };
  }, [code, storedPlayerId]);

  const isLoading = playersLoading || propertiesLoading;
  const error = playersError || propertiesError;

  const combinedData =
    playersData && propertiesData
      ? {
        players: playersData,
        properties: propertiesData,
      }
      : null;

  // A one-way latch: once the room has rendered, later refetches must not drop
  // it back to the full-page loader. Adjusting it during render (rather than
  // in an effect) keeps the first paint from flashing the loader after the
  // data has already arrived.
  if (!initialLoadComplete && playersData && propertiesData) {
    setInitialLoadComplete(true);
  }

  return (
    <DataState
      data={combinedData}
      loading={!initialLoadComplete}
      error={error}
      refetch={() => {
        refetchPlayers();
        refetchProperties();
      }}
    >
      {() =>
        room &&
        player && (
          <RoomView
            room={room}
            loading={isLoading}
            currentPlayer={player}
            otherPlayers={otherPlayers}
            availableProperties={propertiesData?.availableProperties || []}
            eventHistory={eventHistory}
            onTransfer={handleTransfer}
            onPurchaseProperty={handlePurchaseProperty}
            onFreeParkingAction={handleFreeParkingAction}
            onBankerTransaction={handleBankerTransaction}
            onManageProperties={handleManageProperties}
            onKickPlayer={handleKickPlayer}
            offers={offers}
            onCreateOffer={handleCreateOffer}
            onRespondOffer={handleRespondOffer}
          />
        )
      }
    </DataState>
  );
};

export default RoomPage;

const getIconForType = (type: string) => {
  switch (type) {
    case "PLAYER_JOINED":
      return "🧍";
    case "PLAYER_LEFT":
      return "🧍";
    case "BANKER_TRANSACTION":
      return "🏦";
    case "PROPERTY_CHANGE":
      return "🧾";
    case "TRANSFER":
      return "💵";
    case "MANAGE_PROPERTIES":
      return "🏠";
    // Matches the pair `eventTypeFor` stores on the history row for the same
    // notification (`backend/websocket/websocketManager.go`), so the toast and
    // the event log show the player the same symbol.
    case "PLAYER_KICKED":
      return "🚫";
    // The four trade frames share the pair `eventTypeFor` stores on a settled
    // trade's history row, so the toast and the log agree.
    case "OFFER_RECEIVED":
    case "OFFER_SENT":
    case "OFFER_RESOLVED":
    case "OFFER_ACCEPTED":
      return "🤝";
    default:
      return "ℹ️";
  }
};
