"use client";

import PlayerCard from "@/components/players/player-card";
import { EventHistory, Player, Property, Room } from "@/types/schema";
import Navbar from "../navbar/navbar";
import { josephinBold } from "../ui/fonts";
import { BankerTransactionPayload, FreeParkingPayload, ManagePropertiesPayload, TransferType } from "@/types/payloads";

const RoomView = ({
  currentPlayer,
  otherPlayers,
  room,
  availableProperties,
  onTransfer,
  eventHistory,
  onPurchaseProperty,
  onFreeParkingAction,
  onBankerTransaction,
  onManageProperties,
}: {
  otherPlayers: Player[];
  eventHistory: EventHistory[];
  currentPlayer: Player;
  room: Room;
  onTransfer: (
    amount: string,
    transferType: TransferType,
    transferDetails: {
      fromPlayerId: string;
      toPlayerId: string;
      reason: string;
      roomId: string;
    },
  ) => void;
  loading: boolean;
  onFreeParkingAction: (
    amount: string,
    freeParkingType: FreeParkingPayload["freeParkingType"],
    playerId: string,
  ) => void;
  onBankerTransaction: (
    amount: string,
    targetPlayerId: string,
    transactionType: BankerTransactionPayload["transactionType"],
  ) => void;
  availableProperties: Property[];
  onPurchaseProperty: (
    propertyId: string,
    buyerId: string,
    price: number,
  ) => void;
  onManageProperties: (
    amount: number,
    managementType: ManagePropertiesPayload["managementType"],
    properties: { propertyId: string; count?: number }[],
    playerId: string,
  ) => void;
}) => {
  const allPlayers = [...otherPlayers, currentPlayer];
  return (
    <div className="min-h-screen w-full relative flex flex-col">
      {/* A real header box rather than a zero-height sticky wrapper holding two
          absolutely positioned children -- the old one was `bg-white` on a black
          page and only stayed invisible because nothing gave it height. */}
      <header className="sticky top-0 z-50 bg-black">
        <div className="relative flex h-16 items-center justify-center px-4">
          <div
            className={`${josephinBold.className} select-none text-white text-2xl`}
          >
            {room?.name || room?.code}
          </div>
          <Navbar
            freeParking={room?.freeParking || 0}
            player={currentPlayer}
            eventHistory={eventHistory}
            availableProperties={availableProperties}
            onPurchaseProperty={onPurchaseProperty}
            onFreeParkingAction={onFreeParkingAction}
            roomCode={room?.code}
          />
        </div>
      </header>
      <div className="flex-1 flex items-center justify-center py-6">
        {/* Below `lg` this stays the swipe strip the phone layout wants. At
            `lg` it wraps into a centred grid instead: a mouse has no swipe,
            and the strip was clipping the last player off the right edge with
            `hide-scrollbar` removing the only clue that they existed. */}
        <div
          className="w-full flex gap-4 px-4 overflow-x-auto snap-x snap-mandatory hide-scrollbar
            lg:mx-auto lg:max-w-[1160px] lg:flex-wrap lg:justify-center lg:gap-6 lg:overflow-x-visible lg:snap-none"
        >
          <div className="flex-none snap-center">
            <PlayerCard
              player={currentPlayer}
              currentPlayer={currentPlayer}
              onTransfer={onTransfer}
              roomId={room?.id}
              allPlayers={allPlayers}
              onBankerTransaction={onBankerTransaction}
              onManageProperties={onManageProperties}
            />
          </div>

          {/* `onManageProperties` is deliberately absent here: another player's
              Properties drawer is a read-only deed browser, decided 2026-09-16
              (HANDOFF, Settled). Passing it would not make the drawer work --
              the three call sites in `manage-properties.tsx` send
              `currentPlayer.id` as the player to charge, and neither
              `handleManageProperties` nor the property writes check who owns
              the deed, so it would mortgage their property into your balance. */}
          {otherPlayers?.map((oPlayer) => (
            <div key={oPlayer?.id} className="flex-none snap-center">
              <PlayerCard
                player={oPlayer}
                currentPlayer={currentPlayer}
                onTransfer={onTransfer}
                allPlayers={allPlayers}
                roomId={room?.id}
                onBankerTransaction={onBankerTransaction}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default RoomView;
