"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { toast } from "sonner";
import { ColorSelect } from "../players/color-select-drawer";
import Link from "../ui/cusotm-link";
import Button from "../ui/button-custom";
import { playerStore } from "@/lib/utils/playerHelpers";
import { InitialJoinRoomFormData, JoinRoomFormData } from "./utils";
import { BaseRoomInput } from "./room-code-input";
import { usePublicAction } from "@/hooks/use-public-fetch";
import { playerApi } from "@/lib/utils/api.service";

enum STEP {
  ROOM_LOOKUP,
  PLAYER_DETAILS,
}

const JoinRoomForm = () => {
  const [formData, setFormData] = useState<JoinRoomFormData>(
    InitialJoinRoomFormData,
  );
  const [step, setStep] = useState<STEP>(STEP.ROOM_LOOKUP);
  const router = useRouter();

  const updateFormData = (key: keyof JoinRoomFormData, value: string) => {
    setFormData((prev) => ({ ...prev, [key]: value }));
  };

  const { execute: joinRoomExecute, loading: joining } = usePublicAction(
    playerApi.join,
    {
      onSuccess(data: { roomCode: string; playerId: string }) {
        playerStore.setPlayerIdForRoom(data.roomCode, data.playerId);
        router.push(`/room/${data.roomCode}`);
      },
      onError(error) {
        toast.error("Failed to join room", {
          description: error?.error || "Please try again.",
          className: "font-semibold",
        });
      },
    },
  );

  const { execute: checkPlayerExecute, loading: checkingPlayer } =
    usePublicAction(playerApi.getDetails, {
      // GET /rooms/:code/players/:playerId answers { player, properties } --
      // there is no `isValid` flag, so the old check read undefined and this
      // branch never fired. Every returning player was sent to the
      // create-a-player form, where the backend then refused them with "Name
      // or color already taken": there was no way back into your own game.
      onSuccess(data: { player?: { id?: string } }) {
        if (data?.player?.id) {
          router.push(`/room/${formData.roomCode}`);
        } else {
          setStep(STEP.PLAYER_DETAILS);
        }
      },
      onError(error: { error: string }) {
        setStep(STEP.PLAYER_DETAILS);
        toast.error(error.error, {
          className: "font-semibold",
        });
      },
    });

  const checkExistingPlayer = async () => {
    const existingPlayerId = playerStore.getPlayerIdForRoom(formData.roomCode);
    if (existingPlayerId) {
      checkPlayerExecute(formData.roomCode, existingPlayerId);
    } else {
      setStep(STEP.PLAYER_DETAILS);
    }
  };

  const joinRoom = () => {
    if (!formData.playerName || !formData.playerColor) {
      toast.error("Please fill out all fields", {
        className: "font-semibold",
      });
      return;
    }
    joinRoomExecute(formData.roomCode, {
      playerName: formData.playerName,
      roomCode: formData.roomCode,
      playerColor: formData.playerColor,
    });
  };

  const RoomLookup = () => (
    <>
      <BaseRoomInput
        numeral
        placeholder="Room Code"
        value={formData.roomCode}
        onChange={(e) => updateFormData("roomCode", e.target.value)}
        disabled={checkingPlayer}
      />
      <Button
        onClick={() => {
          if (!formData.roomCode) {
            toast.error("Enter a room code", {
              className: "font-semibold",
            });
            return;
          }
          checkExistingPlayer();
        }}
        disabled={checkingPlayer}
        className="mt-4"
      >
        Next
      </Button>
    </>
  );

  const PlayerDetails = () => (
    <>
      <BaseRoomInput
        placeholder="Your Name"
        value={formData.playerName}
        onChange={(e) => updateFormData("playerName", e.target.value)}
        disabled={joining}
      />
      <ColorSelect
        onColorSelect={(color: string) => {
          updateFormData("playerColor", color);
        }}
      />
      <Button onClick={joinRoom} className="mt-4" disabled={joining}>
        Join Room
      </Button>
    </>
  );

  return (
    <div className="relative h-screen">
      <Link text="E-Money" href="/" className="top-2 left-2" />
      <div
        className={`flex flex-col items-center justify-center h-full`}
      >
        {step === STEP.ROOM_LOOKUP && RoomLookup()}
        {step === STEP.PLAYER_DETAILS && PlayerDetails()}
      </div>
    </div>
  );
};

export default JoinRoomForm;
