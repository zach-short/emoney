"use client";
import { EventHistory, Player, Property, Room } from "@/types/schema";
import { use, useEffect, useEffectEvent, useRef, useState } from "react";
import RoomView from "@/components/room/room.client";
import { getWsUrl } from "@/lib/utils/wsHelpers";
import { playerStore } from "@/lib/utils/playerHelpers";
import { toast } from "sonner";
import { josephinBold } from "@/components/ui/fonts";
import { sendMessage } from "@/lib/utils/sendWsMessage";
import { ManagePropertiesPayload } from "@/types/payloads";
import { usePublicFetch } from "@/hooks/use-public-fetch";
import { roomApi } from "@/lib/utils/api.service";
import DataState from "@/components/containers/data-state";

interface WebSocketMessage {
  type: string;
  payload: {
    notification?: string;
  };
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

  // Straight projections of the fetched room payload -- no effect needed, and
  // this keeps the players, room and history from lagging a render behind the
  // data that produced them.
  const player: Player | null =
    playersData?.players?.find((p: Player) => p.id === storedPlayerId) || null;
  const otherPlayers: Player[] =
    playersData?.players?.filter((p: Player) => p.id !== storedPlayerId) || [];
  const room: Room | undefined = playersData?.room;
  const eventHistory: EventHistory[] = playersData?.eventHistory || [];

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
      toast.success(message.payload.notification, {
        duration: 4000,
        icon: getIconForType(message.type),
        position: "top-center",
        className: `${josephinBold.className} text-xs text-center`,
      });

      refetchPlayers();

      if (["PURCHASE_PROPERTY", "MANAGE_PROPERTIES"].includes(message.type)) {
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
    default:
      return "ℹ️";
  }
};
