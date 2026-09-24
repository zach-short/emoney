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
  DrawerDescription,
  DrawerTitle,
  DrawerTrigger,
} from "../ui/drawer";
import MakeOffer from "./make-offer/make-offer";
import OffersInbox from "./make-offer/offers-inbox";
import { DRAWER_MIN_HEIGHT_TALL } from "../ui/drawer-sizes";

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

  // An offer arriving (DESIGN.md D5, moment 3; PLAN.md Phase 6 item 2). The
  // badge is the one persistent signal of it -- the toast is gone in four
  // seconds -- so it arrives rather than appearing between frames.
  //
  // `arrivals` rises only when the count does, and keys the badge, so it
  // replays its entrance for each new offer and for nothing else: a refetch
  // that repeats the count, or an offer resolved, keeps the element as it is.
  // Recorded during render, as `hooks/use-count-up.ts` does.
  const [badge, setBadge] = useState({ count: waitingOnMe, arrivals: 0 });
  if (badge.count !== waitingOnMe) {
    setBadge({
      count: waitingOnMe,
      arrivals: waitingOnMe > badge.count ? badge.arrivals + 1 : badge.arrivals,
    });
  }

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
                  className={`relative h-16 border-[1px] text-black ${josephinBold.className} text-center w-full border-black flex items-center justify-center gap-x-3 text-3xl transition hover:brightness-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black disabled:cursor-default disabled:hover:brightness-100`}
                >
                  <span>{player?.name}</span>
                  {/* Out of flow, on the bar's top-right corner, so its arrival
                      moves nothing: inline, it pushed the name sideways the
                      moment it appeared, and names have no length cap, so
                      holding room for it inline would squeeze every name on
                      your own card for good (PLAN.md BD-19). It sits in the
                      padding band the card already has. Opacity and scale only;
                      reduced motion keeps the badge and drops the entrance. */}
                  {isSelf && waitingOnMe > 0 && (
                    <span
                      key={badge.arrivals}
                      className={`absolute -right-2.5 -top-3 rounded-full bg-black px-3 py-1 text-base text-white shadow-raised ring-2 ring-white motion-safe:animate-in motion-safe:fade-in-0 motion-safe:zoom-in-50 motion-safe:duration-standard motion-safe:ease-out`}
                    >
                      <span className={numeralFace}>{waitingOnMe}</span>{" "}
                      {waitingOnMe === 1 ? "offer" : "offers"}
                    </span>
                  )}
                </button>
              </DrawerTrigger>
              <DrawerContent
                className={`overflow-y-auto ${DRAWER_MIN_HEIGHT_TALL}`}
              >
                <DrawerTitle className={`sr-only`}>
                  {isSelf ? "Your offers" : "Make an offer"}
                </DrawerTitle>
                <DrawerDescription className={`sr-only`}>
                  {isSelf
                    ? "Review trade offers sent to you, and respond to each."
                    : `Propose a trade with ${player?.name}: properties, cash, or both.`}
                </DrawerDescription>
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
              // F2's glance popover counts the offers pending between these two
              // players (D6); the card face itself does not read them.
              offers={offers}
            />
          </div>
        </div>
      </div>
    </>
  );
};

export default PlayerCard;
