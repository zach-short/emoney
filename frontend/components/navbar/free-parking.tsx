import { FreeParkingPayload } from "@/types/payloads";
import { useState } from "react";
import { numeralFace } from "../ui/fonts";
import { formatMoney } from "@/lib/utils/money";
import { Player } from "@/types/schema";
import { DrawerClose } from "../ui/drawer";
import Toast from "../ui/toasts";
import { PiMoneyWavyThin, PiXThin } from "react-icons/pi";

interface FreeParkingDialogProps {
  onFreeParkingAction: (
    amount: string,
    freeParkingType: FreeParkingPayload["freeParkingType"],
    playerId: string,
  ) => void;
  player: Player;
  freeParking: number;
  onClick: () => void;
}

const FreeParkingDialog = ({
  onFreeParkingAction,
  player,
  freeParking,
  onClick,
}: FreeParkingDialogProps) => {
  const [amount, setAmount] = useState("");
  const [type, setType] = useState<"ADD" | "REMOVE">("ADD");

  const handleSubmit = () => {
    onFreeParkingAction(amount, type, player.id);
    setAmount("");
  };

  return (
    <div className="">
      <div className={`text-xl mb-4`}>
        Free Parking:{" "}
        <span className={numeralFace}>{formatMoney(freeParking)}</span>
      </div>

      <button
        className={`relative w-full h-12 rounded-full border border-neutral-400 shadow-raised font-semibold mb-4`}
        onClick={() => setType(type === "ADD" ? "REMOVE" : "ADD")}
      >
        <div
          className={`absolute w-1/2 h-full bg-white rounded-full shadow-raised transition-transform duration-300 ${
            type === "ADD" ? "translate-x-0" : "translate-x-full"
          }`}
        />
        <div className="relative flex h-full">
          <span
            className={`flex-1 flex items-center justify-center ${
              type === "ADD" ? "text-black" : "text-neutral-400"
            }`}
          >
            Add
          </span>
          <span
            className={`flex-1 flex items-center justify-center ${
              type === "REMOVE" ? "text-black" : "text-neutral-400"
            }`}
          >
            Collect
          </span>
        </div>
      </button>

      <input
        type="numeric"
        value={amount}
        onChange={(e) => setAmount(e.target.value)}
        placeholder="Enter amount"
        className="w-full p-4 rounded border bg-inherit text-xl"
      />

      <div className={`flex gap-4 mt-4 items-center font-semibold`}>
        <div className="flex-1 py-3 rounded text-center" onClick={onClick}>
          Cancel
        </div>
        <DrawerClose
          // D2 promotes this pair into the money-direction tokens. It used
          // to be green for ADD and BLUE for Collect, which encoded nothing:
          // from the player's balance, adding to the pot is money out and
          // collecting is money in. The glyph in the label carries the same
          // thing without the colour.
          className={`flex-1 py-3 text-center bg-white rounded-md font-semibold ${
            type === "ADD" ? "text-money-out" : "text-money-in"
          }`}
          onClick={() => {
            if (!amount || isNaN(Number(amount)) || Number(amount) <= 0) {
              Toast({
                message: "Please enter a valid amount",
                icon: <PiXThin className="text-red-700 text-xl" />,
              });
              return;
            } else if (type === "REMOVE" && Number(amount) > freeParking) {
              Toast({
                icon: <PiMoneyWavyThin className="text-red-700 text-xl" />,
                message: "Not enough money in free parking",
              });
              return;
            } else if (type === "ADD" && Number(amount) > player.balance) {
              Toast({
                icon: <PiMoneyWavyThin className="text-red-700 text-xl" />,
                message: "Insufficient funds",
                details: `You need ${formatMoney(Number(amount) - player.balance)}`,
              });
              return;
            } else {
              handleSubmit();
            }
          }}
        >
          {type === "ADD" ? "\u2212 Add to Pot" : "+ Collect"}
        </DrawerClose>
      </div>
    </div>
  );
};

export default FreeParkingDialog;
