"use client";
import { Player } from "@/types/schema";
import { josephinBold } from "../ui/fonts";
import { PlayerDetails } from "./player-card-content";
import { BankerTransactionPayload, KickPlayerPayload, ManagePropertiesPayload, TransferType } from "@/types/payloads";
import {
  Drawer,
  DrawerContent,
  DrawerTitle,
  DrawerTrigger,
} from "../ui/drawer";
import MakeOffer from "./make-offer/make-offer";

const PlayerCard = ({
  player,
  currentPlayer,
  allPlayers,
  onTransfer,
  roomId,
  onBankerTransaction,
  onManageProperties,
  onKickPlayer,
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
}) => {
  const color = player?.color || "#fff";

  return (
    <>
      <div className="snap-center w-[360px] border bg-white border-black  aspect-[3/4] select-none relative">
        <div className={`p-3 w-full h-full border-black `}>
          <div className={`border border-black p-2 h-full`}>
            <Drawer>
              <DrawerTrigger asChild>
                <button
                  style={{ backgroundColor: color }}
                  className={`h-16 border-[1px] text-black ${josephinBold.className} text-center w-full border-black flex items-center justify-center text-3xl transition hover:brightness-95 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black`}
                >
                  {player?.name}
                </button>
              </DrawerTrigger>
              <DrawerContent
                className={`overflow-y-auto min-h-[90vh] bg-black`}
              >
                <DrawerTitle className={`hidden`}>Make an offer</DrawerTitle>
                <MakeOffer
                  player={player}
                  currentPlayer={currentPlayer}
                  roomId={roomId}
                />
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
