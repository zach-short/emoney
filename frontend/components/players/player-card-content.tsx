import { Player } from "@/types/schema";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerTitle,
  DrawerTrigger,
} from "../ui/drawer";
import { useState } from "react";
import SendReqToggle from "./pay-req-toggle-switch";
import PayRequestRent from "./pay.req.rent.component";
import { josephinBold, josephinNormal } from "../ui/fonts";
import PlayerTags from "./player-tags";
import ManageProperties from "./manage-properties";
import { BankerTransactionPayload, KickPlayerPayload, ManagePropertiesPayload, TransferType } from "@/types/payloads";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "../ui/dialog";
import { Input } from "../ui/input";
import { Button } from "../ui/button";
import { CiCircleMinus, CiCirclePlus } from "react-icons/ci";
import RemovePlayer from "./remove-player";

const PlayerDetails = ({
  player,
  currentPlayer,
  allPlayers,
  onTransfer,
  roomId,
  onManageProperties,
  onBankerTransaction,
  onKickPlayer,
}: {
  player: Player;
  allPlayers: Player[];
  currentPlayer: Player;
  roomId: string;
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
        className={`px-4 text-black flex flex-col items-evenly gap-y-2 justify-between ${josephinNormal.className} text-2xl`}
      >
        <Dialog
          open={dialogState !== null}
          onOpenChange={(open) => !open && setDialogState(null)}
        >
          <DialogContent
            className={`sm:max-w-[425px] ${josephinBold.className} text-black top-1/3`}
          >
            <DialogHeader>
              <DialogTitle>
                {dialogState === "add" ? "Add Money to" : "Remove Money from"}{" "}
                {player?.name}
              </DialogTitle>
              <DialogDescription className={`sr-only`}>
                Enter an amount to{" "}
                {dialogState === "add" ? "add to" : "remove from"}{" "}
                {player?.name}&apos;s balance as the banker.
              </DialogDescription>
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
          className={`${josephinBold.className} w-full absolute top-[6.5rem] right-1/2 transform translate-x-1/2`}
        >
          <div className={`flex items-center justify-center space-x-5 w-full`}>
            {currentPlayer?.isBanker && !isRemoved && (
              <CiCircleMinus
                onClick={() => setDialogState("remove")}
                className={`hover:cursor-pointer pb-1`}
              />
            )}
            <p> ${player?.balance || 0}</p>{" "}
            {currentPlayer?.isBanker && !isRemoved && (
              <CiCirclePlus
                onClick={() => setDialogState("add")}
                className={`hover:cursor-pointer pb-1`}
              />
            )}
          </div>
        </div>
        <Drawer>
          <DrawerTrigger asChild>
            <button
              type="button"
              className={`flex items-center justify-between w-full mt-14 rounded-md px-1 transition-colors hover:bg-black/5 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black`}
            >
              <span>{currentPlayer?.id === player?.id && "My"} Properties</span>
              <span>{player?.properties?.length || 0}</span>
            </button>
          </DrawerTrigger>
          <DrawerContent
            className={`bg-black h-[600px] px-3 text-white  mt-0 border-t border-x border-b-none`}
          >
            <DrawerTitle className={`text-black select-none`}>
              {player?.id}&apos; Properties
            </DrawerTitle>
            <DrawerDescription className={`sr-only`}>
              {player?.name}&apos;s properties, with houses, mortgages, and
              sale controls for the banker.
            </DrawerDescription>
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
            className={`w-[calc(100%-4rem)] text-center border border-neutral-400 text-neutral-500 rounded-full absolute bottom-6 p-4 right-1/2 transform translate-x-1/2 ${josephinBold.className}`}
          >
            No longer in the game
          </div>
        )}
        {currentPlayer?.id !== player?.id && !isRemoved && (
          <Drawer>
            <DrawerTrigger asChild>
              <button
                type="button"
                className={`shadow-xl w-[calc(100%-4rem)] text-center border rounded-full absolute bottom-6 p-4 right-1/2 transform translate-x-1/2 transition-colors hover:bg-black hover:text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black ${josephinBold.className}`}
              >
                Pay or Request
              </button>
            </DrawerTrigger>
            <DrawerContent className={`h-[90vh] bg-black px-2`}>
              <DrawerTitle className={`text-black`}>
                Choose Payment Type
              </DrawerTitle>
              <DrawerDescription className={`sr-only`}>
                Send money to {player?.name} or request money from them.
              </DrawerDescription>
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
