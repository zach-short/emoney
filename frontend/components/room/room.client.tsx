"use client";

import PlayerCard from "@/components/players/player-card";
import {
  EventHistory,
  Offer,
  OfferNoID,
  Player,
  Property,
  Room,
} from "@/types/schema";
import Navbar from "../navbar/navbar";
import { josephinBold, numeralFace } from "../ui/fonts";
import {
  BankerTransactionPayload,
  FreeParkingPayload,
  KickPlayerPayload,
  ManagePropertiesPayload,
  RespondOfferPayload,
  TransferType,
} from "@/types/payloads";
import AuctionBar, { LiveBid } from "./auction-bar";

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
  onKickPlayer,
  offers,
  onCreateOffer,
  onRespondOffer,
  liveBid,
  onPlaceBid,
  onCloseAuction,
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
  onKickPlayer: (
    targetPlayerId: string,
    disposition: KickPlayerPayload["disposition"],
    successorPlayerId?: string,
  ) => void;
  // The current player's pending offers, both directions, and the two sends.
  // They go to every card: your own card's name opens the inbox, another
  // player's opens the form to make them an offer.
  offers: Offer[];
  onCreateOffer: (offer: OfferNoID) => void;
  onRespondOffer: (
    offerId: string,
    response: RespondOfferPayload["response"],
  ) => void;
  liveBid: LiveBid | null;
  onPlaceBid: (propertyId: string, amount: number) => void;
  onCloseAuction: (propertyId: string, kickedPlayerId: string) => void;
}) => {
  const allPlayers = [...otherPlayers, currentPlayer];
  return (
    <div className="min-h-screen w-full relative flex flex-col">
      {/* A real header box rather than a zero-height sticky wrapper holding two
          absolutely positioned children -- the old one was `bg-white` on a black
          page and only stayed invisible because nothing gave it height. */}
      {/* Translucent with an 8px blur (D4; PLAN.md section 3 dials), so the
          card strip visibly passes under the header instead of vanishing at a
          hard edge. The header is thin -- more blur than this reads as smear
          rather than depth, which is why its dial is lower than the scrim's.
          The border is the bottom edge the opaque version never needed. */}
      <header
        className="sticky top-0 z-50 border-b border-white/10 bg-black/60 backdrop-blur-[8px]"
      >
        <div className="relative flex h-16 items-center justify-center px-4">
          {/* A name is a word and keeps the display face; a code is a string of
              characters that has to be read out loud and typed by someone else,
              so it gets the numeral face (DESIGN.md D3(a)). */}
          <div
            className={`${
              room?.name ? josephinBold.className : numeralFace
            } select-none text-white text-2xl`}
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
        {/* Inside the sticky header, below the title row, so a live auction
            travels with it and pushes the cards down instead of covering the
            bottom of one. Renders nothing when no auction is running. */}
        <AuctionBar
          room={room}
          allPlayers={allPlayers}
          availableProperties={availableProperties}
          currentPlayer={currentPlayer}
          liveBid={liveBid}
          onPlaceBid={onPlaceBid}
          onCloseAuction={onCloseAuction}
        />
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
              onKickPlayer={onKickPlayer}
              offers={offers}
              onCreateOffer={onCreateOffer}
              onRespondOffer={onRespondOffer}
            />
          </div>

          {/* `onManageProperties` is deliberately absent here: another player's
              Properties drawer is a read-only deed browser, decided 2026-09-16
              (HANDOFF, Settled). Passing it would not make the drawer work --
              the three call sites in `manage-properties.tsx` send
              `currentPlayer.id` as the player to charge, and neither
              `handleManageProperties` nor the property writes check who owns
              the deed, so it would mortgage their property into your balance.

              `onKickPlayer` is passed to both cards, deliberately: a banker
              removing themselves is in scope through the same flow (D12), so
              it is not an "other players only" control the way the prop above
              is. */}
          {otherPlayers?.map((oPlayer) => (
            <div key={oPlayer?.id} className="flex-none snap-center">
              <PlayerCard
                player={oPlayer}
                currentPlayer={currentPlayer}
                onTransfer={onTransfer}
                allPlayers={allPlayers}
                roomId={room?.id}
                onBankerTransaction={onBankerTransaction}
                onKickPlayer={onKickPlayer}
                offers={offers}
                onCreateOffer={onCreateOffer}
                onRespondOffer={onRespondOffer}
              />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

export default RoomView;
