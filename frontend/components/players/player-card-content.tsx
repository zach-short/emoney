import { Offer, Player } from "@/types/schema";
import {
  Drawer,
  DrawerContent,
  DrawerTitle,
  DrawerTrigger,
} from "../ui/drawer";
import { DeedListPopover } from "../property/deed-popover";
import PlayerGlance from "./player-glance";
import { useState } from "react";
import SendReqToggle from "./pay-req-toggle-switch";
import PayRequestRent from "./pay.req.rent.component";
import { numeralFace } from "../ui/fonts";
import { formatMoney } from "@/lib/utils/money";
import PlayerTags from "./player-tags";
import ManageProperties from "./manage-properties";
import { BankerTransactionPayload, KickPlayerPayload, ManagePropertiesPayload, TransferType } from "@/types/payloads";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "../ui/dialog";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { CiCircleMinus, CiCirclePlus } from "react-icons/ci";
import RemovePlayer from "./remove-player";

// The Properties row is rendered twice -- once as the drawer trigger it has
// always been, once as F1's popover trigger at `lg` -- so its classes live in
// one constant and cannot drift apart. The display utility is deliberately NOT
// in here: each call site adds its own, so which one wins at which width is
// readable at the call site instead of resting on Tailwind's emission order.
const PROPERTIES_ROW =
  "items-center justify-between w-full mt-14 rounded-md px-1 transition-colors hover:bg-black/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black";

const PlayerDetails = ({
  player,
  currentPlayer,
  allPlayers,
  onTransfer,
  roomId,
  onManageProperties,
  onBankerTransaction,
  onKickPlayer,
  offers,
}: {
  player: Player;
  allPlayers: Player[];
  currentPlayer: Player;
  roomId: string;
  // F2's popover reads these; the card face itself does not (D6).
  offers: Offer[];
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
  onManageProperties?: (
    amount: number,
    managementType: ManagePropertiesPayload["managementType"],
    properties: { propertyId: string; count?: number }[],
    playerId: string
  ) => void;
  onBankerTransaction: (
    amount: string,
    playerId: string,
    transactionType: BankerTransactionPayload["transactionType"]
  ) => void;
  onKickPlayer: (
    targetPlayerId: string,
    disposition: KickPlayerPayload["disposition"],
    successorPlayerId?: string
  ) => void;
}) => {
  const [transferType, setTransferType] = useState<"SEND" | "REQUEST">("SEND");
  const handleTransfer = (
    amount: number,
    reason: string,
    transferDetails: {
      fromPlayerId: string;
      toPlayerId: string;
    }
  ) => {
    const payload = {
      amount: amount.toString(),
      transferType: transferType,
      fromPlayerId: transferDetails.fromPlayerId,
      toPlayerId: transferDetails.toPlayerId,
      reason,
      roomId,
    };

    onTransfer(payload.amount, payload.transferType as TransferType, {
      fromPlayerId: payload.fromPlayerId,
      toPlayerId: payload.toPlayerId,
      reason: payload.reason,
      roomId: payload.roomId,
    });
  };
  const getPropertiesToShow = () => {
    if (transferType !== "SEND") {
      return currentPlayer?.properties;
    }
    return player?.properties;
  };

  // A kicked player is marked, not deleted: `GetPlayersInRoom` still returns
  // them, deliberately, so that a FREEZE kick's deeds keep resolving to a name
  // (D4, D11). That makes this component the only thing in the app that can
  // tell the room they are gone -- without it a kick renders as nothing
  // happening at all. Compared against `false` rather than negated so that a
  // payload missing the field never renders a live player as removed.
  const isRemoved = player?.isActive === false;
  const isOwnCard = currentPlayer?.id === player?.id;

  const [dialogState, setDialogState] = useState<"add" | "remove" | null>(null);
  const [amount, setAmount] = useState("");

  const handleBankerAction = (isAdd: boolean) => {
    onBankerTransaction(
      amount,
      player.id,
      isAdd ? "BANKER_ADD" : "BANKER_REMOVE"
    );
    setAmount("");
    setDialogState(null);
  };

  return (
    <>
      <div
        className={`px-4 text-black flex flex-col items-evenly gap-y-2 justify-between text-2xl`}
      >
        <Dialog
          open={dialogState !== null}
          onOpenChange={(open) => !open && setDialogState(null)}
        >
          <DialogContent
            className={`sm:max-w-[425px] top-1/3`}
          >
            <DialogHeader>
              <DialogTitle>
                {dialogState === "add" ? "Add Money to" : "Remove Money from"}{" "}
                {player?.name}
              </DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="flex items-center gap-4">
                <Input
                  type="number"
                  placeholder="Enter amount"
                  value={amount}
                  onChange={(e) => setAmount(e.target.value)}
                  className="col-span-3"
                />
                <Button
                  onClick={() => handleBankerAction(dialogState === "add")}
                >
                  Confirm
                </Button>
              </div>
            </div>
          </DialogContent>
        </Dialog>
        <div
          className={`w-full absolute top-[6.5rem] right-1/2 transform translate-x-1/2`}
        >
          <div className={`flex items-center justify-center space-x-5 w-full`}>
            {currentPlayer?.isBanker && !isRemoved && (
              <CiCircleMinus
                onClick={() => setDialogState("remove")}
                className={`hover:cursor-pointer pb-1`}
              />
            )}
            {/* The headline number on the card, and the one Phase 5 animates.
                Through the shared formatter and in the numeral face so it does
                not reflow as it changes (DESIGN.md D3(a)).

                Below `lg` this stays exactly the plain <p> it has always been.
                At `lg` and above the same figure is also F2's trigger (D6): the
                card's headline reference number is what a glance anchors on, and
                it was not a control before, so nothing is displaced. Two
                renderings rather than one media-query hook, because the app
                prerenders every page (11/11 static) and a JS breakpoint read
                would disagree between the server and the first client frame.

                PHASE 5, READ THIS: at `lg` the figure Phase 5 animates is
                inside the <button> below, not the <p>. Both need the count-up. */}
            <p className={`${numeralFace} lg:hidden`}>
              {formatMoney(player?.balance)}
            </p>
            <PlayerGlance
              player={player}
              currentPlayer={currentPlayer}
              offers={offers}
            >
              <button
                type="button"
                className={`${numeralFace} hidden rounded-md px-1 transition-colors hover:bg-black/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black lg:block`}
              >
                {formatMoney(player?.balance)}
              </button>
            </PlayerGlance>{" "}
            {currentPlayer?.isBanker && !isRemoved && (
              <CiCirclePlus
                onClick={() => setDialogState("add")}
                className={`hover:cursor-pointer pb-1`}
              />
            )}
          </div>
        </div>
        {/* F1 at `lg`, on another player's card only (D6).

            Another player's Properties drawer is already a read-only deed
            browser -- settled 2026-09-16, and `room.client.tsx` omits
            `onManageProperties` for other players on purpose -- so at `lg` the
            same row opens the deeds as a popover instead: one interaction to the
            terms, with the room still behind them.

            Your OWN row keeps the drawer at every width, deliberately. It opens
            `ManageProperties`, which mortgages deeds and builds houses, and D6
            excludes consequential actions from popovers by name -- that is why
            F3 was parked. Reference gets a popover; action keeps its sheet. */}
        {!isOwnCard && (
          <DeedListPopover properties={player?.properties}>
            <button type="button" className={`${PROPERTIES_ROW} hidden lg:flex`}>
              <span>Properties</span>
              <span className={numeralFace}>
                {player?.properties?.length || 0}
              </span>
            </button>
          </DeedListPopover>
        )}
        <Drawer>
          <DrawerTrigger asChild>
            <button
              type="button"
              className={`${PROPERTIES_ROW} flex ${!isOwnCard ? "lg:hidden" : ""}`}
            >
              <span>{isOwnCard && "My"} Properties</span>
              <span className={numeralFace}>
                {player?.properties?.length || 0}
              </span>
            </button>
          </DrawerTrigger>
          <DrawerContent
            className={`h-[600px] px-3 mt-0 border-t border-x border-b-none`}
          >
            <DrawerTitle className={`text-black select-none`}>
              {player?.id}&apos; Properties
            </DrawerTitle>
            <ManageProperties
              onManageProperties={onManageProperties}
              player={player}
              currentPlayer={currentPlayer}
            />
          </DrawerContent>
        </Drawer>
        <PlayerTags
          player={player}
          allPlayers={allPlayers.filter((p) => p?.id !== player?.id)}
        />
        {currentPlayer?.isBanker && !isRemoved && (
          <RemovePlayer
            player={player}
            currentPlayer={currentPlayer}
            allPlayers={allPlayers}
            onKickPlayer={onKickPlayer}
          />
        )}
        {isRemoved && (
          <div
            className={`w-[calc(100%-4rem)] text-center border border-neutral-400 text-neutral-500 rounded-full absolute bottom-6 p-4 right-1/2 transform translate-x-1/2 font-semibold`}
          >
            No longer in the game
          </div>
        )}
        {currentPlayer?.id !== player?.id && !isRemoved && (
          <Drawer>
            <DrawerTrigger asChild>
              <button
                type="button"
                className={`shadow-raised w-[calc(100%-4rem)] text-center border rounded-full absolute bottom-6 p-4 right-1/2 transform translate-x-1/2 transition-colors hover:bg-black hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black font-semibold`}
              >
                Pay or Request
              </button>
            </DrawerTrigger>
            <DrawerContent className={`h-[90vh] px-2`}>
              <DrawerTitle className={`text-black`}>
                Choose Payment Type
              </DrawerTitle>
              <SendReqToggle onToggle={(newType) => setTransferType(newType)} />
              <PayRequestRent
                properties={getPropertiesToShow()}
                type={transferType}
                fromPlayer={transferType === "SEND" ? currentPlayer : player}
                toPlayer={transferType === "SEND" ? player : currentPlayer}
                onTransferRequest={(amount, reason, transferDetails) =>
                  handleTransfer(amount, reason, transferDetails)
                }
                roomId={roomId}
              />
            </DrawerContent>
          </Drawer>
        )}
      </div>
    </>
  );
};

export { PlayerDetails };
