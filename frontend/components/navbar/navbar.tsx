"use client";
import { FreeParkingPayload } from "@/types/payloads";
import { EventHistory, Player, Property } from "@/types/schema";
import {
  Drawer,
  DrawerContent,
  DrawerTitle,
  DrawerTrigger,
} from "../ui/drawer";
import { AiOutlineMenu } from "react-icons/ai";
import { josephinBold, josephinNormal } from "../ui/fonts";
import SelectColorProperties from "../players/purchase-properties-bank";
import { useState } from "react";
import Link from "next/link";
import FreeParkingDialog from "./free-parking";
import { playerStore } from "@/lib/utils/playerHelpers";
import { IoCopyOutline } from "react-icons/io5";
import { toast } from "sonner";
import { formatTimeAgo } from "../ui/helper-funcs";
import ReturnToMenu from "../ui/return-to-menu";

// Menu rows were click-handled <div>s: no keyboard focus, no hover state, and
// nothing telling a mouse user they were targets at all.
const MENU_ROW =
  "flex w-full items-center justify-between rounded-md px-2 py-2 text-left transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white";

const Navbar = ({
  freeParking,
  player,
  eventHistory,
  availableProperties,
  onFreeParkingAction,
  roomCode,
  onPurchaseProperty,
}: {
  freeParking: number;
  roomCode: string;
  eventHistory: EventHistory[];
  player: Player;
  onPurchaseProperty: (
    propertyId: string,
    buyerId: string,
    price: number
  ) => void;
  onFreeParkingAction: (
    amount: string,
    freeParkingType: FreeParkingPayload["freeParkingType"],
    playerId: string,
  ) => void;
  availableProperties?: Property[];
}) => {
  const [showProperties, setShowProperties] = useState(false);
  const [showFreeParking, setShowFreeParking] = useState(false);
  const [showEvents, setShowEvents] = useState(false);

  return (
    <>
      <Drawer>
        <DrawerTrigger asChild>
          <button
            type="button"
            aria-label="Open menu"
            className={`border rounded-md p-2 absolute right-4 top-1/2 -translate-y-1/2 transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
          >
            <AiOutlineMenu size={25} />
          </button>
        </DrawerTrigger>
        <DrawerContent
          className={`${josephinNormal.className} h-[80vh] bg-black border-[1px] px-3 text-xl `}
        >
          <DrawerTitle className={`text-black`}>Menu</DrawerTitle>

          <ul className={`flex flex-col gap-1 h-[75vh] relative`}>
            {(showProperties || showFreeParking || showEvents) && (
              <ReturnToMenu
                onClick={() => {
                  setShowProperties(false);
                  setShowFreeParking(false);
                  setShowEvents(false);
                }}
              />
            )}
            {showProperties ? (
              <>
                <div
                  className={`${josephinBold.className} bg-black h-full  text-2xl overflow-y-auto`}
                >
                  <DrawerTitle className={`select-none text-black h-0`}>
                    Properties for Sale
                  </DrawerTitle>
                  <SelectColorProperties
                    properties={availableProperties}
                    onPurchase={onPurchaseProperty}
                    player={player}
                  />
                </div>
              </>
            ) : showFreeParking ? (
              <>
                <FreeParkingDialog
                  player={player}
                  onFreeParkingAction={onFreeParkingAction}
                  freeParking={freeParking}
                  onClick={() => setShowFreeParking(false)}
                />
              </>
            ) : showEvents ? (
              <>
                <div
                  className={`event-history-container overflow-y-auto pb-20`}
                >
                  {eventHistory.map((event: EventHistory, index: number) => (
                    <div className={`${josephinNormal.className}`} key={index}>
                      <div className="flex justify-between rounded-full items-center mb-2 py-1 sm:py-2">
                        <span className={`text-xs sm:text-sm`}>
                          <div className={`flex justify-start items-start`}>
                            <div className={`pr-[4px]`}>
                              {event?.eventType?.[1] || "🧍"}
                            </div>
                            <p
                              style={{
                                color: event?.eventType?.[0] || "white",
                              }}
                            >
                              {event.event}
                            </p>
                          </div>
                        </span>
                        <span
                          className={` text-gray-500 text-[.5rem] sm:text-xs text-end `}
                        >
                          {formatTimeAgo(new Date(event.timestamp))}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </>
            ) : (
              <>
                <li>
                  <button
                    type="button"
                    className={MENU_ROW}
                    onClick={() => setShowProperties(true)}
                  >
                    <span>Bank&apos;s Properties</span>
                    <span>{availableProperties?.length || 0}</span>
                  </button>
                </li>
                <li>
                  <button
                    type="button"
                    className={MENU_ROW}
                    onClick={() => setShowFreeParking(true)}
                  >
                    <span>Free Parking</span>
                    <span>${freeParking}</span>
                  </button>
                </li>
                <li>
                  <button
                    type="button"
                    className={MENU_ROW}
                    onClick={() => setShowEvents(true)}
                  >
                    <span>Event History</span>
                    <span>{eventHistory.length}</span>
                  </button>
                </li>
                <li>
                  <button
                  type="button"
                  className={MENU_ROW}
                  onClick={() => {
                    navigator.clipboard
                      .writeText(roomCode)
                      .then(() => {
                        toast.success("Room code copied to clipboard!", {
                          duration: 2000,
                          icon: "📋",
                          className: `${josephinBold.className}`,
                        });
                      })
                      .catch(() => {
                        toast.error("Failed to copy room code");
                      });
                  }}
                  >
                    <span>Room Code</span>
                    <span className={`flex items-center`}>
                      <IoCopyOutline className={`mr-1`} />
                      {roomCode}
                    </span>
                  </button>
                </li>
                <div className={`absolute bottom-5 w-full`}>
                  <div className={`flex flex-col items-start w-full `}>
                    <div className={` w-full text-lg text-red-300`}>
                      Danger Zone
                    </div>
                    <div
                      className={`flex flex-col items-start w-full space-y-5`}
                    >
                      <Link
                        href={`/`}
                        className={` w-full text-lg border border-red-300 p-2 rounded-sm`}
                      >
                        Leave Game
                      </Link>{" "}
                      <Link
                        href={`/`}
                        className={`  p-2 border rounded-sm w-full text-lg border-red-300`}
                        onClick={() => {
                          // playerStore keys by room CODE (`room_<code>_playerId`).
                          // Passing roomId cleared nothing, so "Delete My Player"
                          // left you rejoining as the same player.
                          playerStore.clearPlayerDataForRoom(roomCode);
                        }}
                      >
                        Delete My Player this Game
                      </Link>
                      <Link
                        href={`/`}
                        className={`w-full text-lg border rounded-sm p-2 border-red-300`}
                        onClick={() => playerStore.clearAllPlayerData()}
                      >
                        Delete My Players in All Games
                      </Link>
                    </div>
                  </div>
                </div>
              </>
            )}
          </ul>
        </DrawerContent>
      </Drawer>
    </>
  );
};

export default Navbar;
