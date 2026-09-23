"use client";
import { useState } from "react";
import { Offer, OfferNoID, Player } from "@/types/schema";
import { josephinBold, numeralFace } from "../ui/fonts";
import { PlayerDetails } from "./player-card-content";
import {
  BankerTransactionPayload,
  KickPlayerPayload,
  ManagePropertiesPayload,
  RespondOfferPayload,
  TransferType,
} from "@/types/payloads";
import {
  Drawer,
  DrawerContent,
  DrawerTitle,
  DrawerTrigger,
} from "../ui/drawer";
import MakeOffer from "./make-offer/make-offer";
import OffersInbox from "./make-offer/offers-inbox";

const PlayerCard = ({
  player,
  currentPlayer,
  allPlayers,
  onTransfer,
  roomId,
  onBankerTransaction,
  onManageProperties,
  onKickPlayer,
  offers,
  onCreateOffer,
  onRespondOffer,
}: {
  player: Player;
  currentPlayer: Player;
  allPlayers: Player[];
  onTransfer: (
    amount: string,
    transferType: TransferType,
    transferDetails: {
      fromPlayerId: string;
      toPlayerId: string;
      reason: string;
      roomId: string;
    }
  ) => void;
  roomId: string;
  onBankerTransaction: (
    amount: string,
    playerId: string,
    transactionType: BankerTransactionPayload["transactionType"]
  ) => void;
  onManageProperties?: (
    amount: number,
    managementType: ManagePropertiesPayload["managementType"],
    properties: { propertyId: string; count?: number }[],
    playerId: string
  ) => void;
  onKickPlayer: (
    targetPlayerId: string,
    disposition: KickPlayerPayload["disposition"],
    successorPlayerId?: string
  ) => void;
  offers: Offer[];
  onCreateOffer: (offer: OfferNoID) => void;
  onRespondOffer: (
    offerId: string,
    response: RespondOfferPayload["response"]
  ) => void;
}) => {
  const color = player?.color || "#fff";

  // The name bar opens one of two drawers. On another player's card it is
  // the offer form, as it always was -- except that the form now has a Send
  // button. On your own card it is your inbox: making an offer to yourself is
  // nothing, and this is where the offers made to you have to be findable
  // from, because a toast is gone in four seconds. Controlled so the form
  // can close the drawer once it sends.
  const [open, setOpen] = useState(false);
  const isSelf = currentPlayer?.id === player?.id;
  // Compared against `false` rather than negated, as `player-card-content.tsx`
  // does, so a payload missing the field never disables a live player's bar.
  const isRemoved = player?.isActive === false;
  const waitingOnMe = offers.filter(
    (o) => o.toPlayerId === currentPlayer?.id && o.status === "PENDING"
  ).length;

  return (
    <>
      <div className="snap-center w-[360px] border bg-white border-black  aspect-[3/4] select-none relative">
        <div className={`p-3 w-full h-full border-black `}>
          <div className={`border border-black p-2 h-full`}>
            <Drawer open={open} onOpenChange={setOpen}>
              <DrawerTrigger asChild>
                <button
                  type="button"
                  style={{ backgroundColor: color }}
                  // A removed player cannot be traded with -- the Go side
                  // refuses it -- so their bar does not open the form.
                  disabled={isRemoved && !isSelf}
                  className={`h-16 border-[1px] text-black ${josephinBold.className} text-center w-full border-black flex items-center justify-center gap-x-3 text-3xl transition hover:brightness-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black disabled:cursor-default disabled:hover:brightness-100`}
                >
                  <span>{player?.name}</span>
                  {isSelf && waitingOnMe > 0 && (
                    <span
                      className={`rounded-full bg-black px-3 py-1 text-base text-white`}
                    >
                      <span className={numeralFace}>{waitingOnMe}</span>{" "}
                      {waitingOnMe === 1 ? "offer" : "offers"}
                    </span>
                  )}
                </button>
              </DrawerTrigger>
              <DrawerContent
                className={`overflow-y-auto min-h-[90vh]`}
              >
                <DrawerTitle className={`hidden`}>
                  {isSelf ? "Your offers" : "Make an offer"}
                </DrawerTitle>
                {isSelf ? (
                  <OffersInbox
                    offers={offers}
                    currentPlayer={currentPlayer}
                    allPlayers={allPlayers}
                    roomId={roomId}
                    onCreateOffer={onCreateOffer}
                    onRespondOffer={onRespondOffer}
                    onClose={() => setOpen(false)}
                  />
                ) : (
                  <MakeOffer
                    player={player}
                    currentPlayer={currentPlayer}
                    roomId={roomId}
                    onCreateOffer={onCreateOffer}
                    onSent={() => setOpen(false)}
                  />
                )}
              </DrawerContent>
            </Drawer>

            <PlayerDetails
              player={player}
              currentPlayer={currentPlayer}
              onTransfer={onTransfer}
              onManageProperties={onManageProperties}
              onBankerTransaction={onBankerTransaction}
              onKickPlayer={onKickPlayer}
              allPlayers={allPlayers}
              roomId={roomId}
            />
          </div>
        </div>
      </div>
    </>
  );
};

export default PlayerCard;
