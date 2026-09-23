import { Player, Property } from "@/types/schema";
import { useEffect, useEffectEvent, useState } from "react";
import { MdArrowBackIos } from "react-icons/md";
import { numeralFace } from "../ui/fonts";
import { formatMoney } from "@/lib/utils/money";
import { calculateRent } from "../ui/helper-funcs";
import { toast } from "sonner";
import { DrawerClose } from "../ui/drawer";
import PropertyCard from "../property/cards/card";

interface PayRequestRentProps {
  properties?: Property[];
  type: "SEND" | "REQUEST";
  fromPlayer: Player;
  toPlayer: Player;
  onTransferRequest: (
    amount: number,
    reason: string,
    transferDetails: {
      fromPlayerId: string;
      toPlayerId: string;
    },
    roomId: string
  ) => void;
  roomId: string;
}
interface TransactionDetails {
  amount: number;
  reason: string;
  property: Property;
  propertyGroup: [string, Property[]];
}
const PayRequestRent = ({
  properties = [],
  type,
  fromPlayer,
  roomId,
  toPlayer,
  onTransferRequest,
}: PayRequestRentProps) => {
  const [currentView, setCurrentView] = useState<
    "colors" | "properties" | "confirmation"
  >("colors");
  const [roll, setRoll] = useState("12");
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);
  const [selectedProperty, setSelectedProperty] = useState<Pick<
    TransactionDetails,
    "property" | "propertyGroup"
  > | null>(null);

  // Utility rent is a function of the dice roll, which the user can still
  // change on the confirmation screen. Recomputing it during render keeps the
  // amount in step with the input instead of trailing it by one commit.
  const transactionDetails: TransactionDetails | null = selectedProperty
    ? {
        ...selectedProperty,
        ...calculateRent(
          selectedProperty.property,
          selectedProperty.propertyGroup,
          parseInt(roll)
        ),
      }
    : null;

  // Warning the user is a side effect, not state, so it stays in an effect --
  // reading the players through an effect event so only a new roll or a new
  // property re-triggers it, exactly as before.
  const warnOnInsufficientFunds = useEffectEvent(() => {
    if (!transactionDetails) return;

    const payingPlayer = type === "SEND" ? fromPlayer : toPlayer;
    if (payingPlayer.balance < transactionDetails.amount) {
      toast.error(
        `${payingPlayer.name} doesn't have sufficient funds (${transactionDetails.amount})`
      );
    }
  });

  useEffect(() => {
    if (selectedGroup === "utility") {
      warnOnInsufficientFunds();
    }
  }, [roll, selectedProperty?.property, selectedGroup]);

  if (!properties || properties?.length === 0) {
    return (
      <div className="flex items-start justify-center h-full">
        <p className={`text-sm font-semibold text-white`}>
          {type === "REQUEST"
            ? `You don't have any properties to request rent for`
            : `${toPlayer.name} doesn't have any properties to pay rent for`}
        </p>
      </div>
    );
  }

  const handlePropertySelect = (property: Property) => {
    const propertyOwner = type === "SEND" ? toPlayer : fromPlayer;
    const propertiesInGroup = properties.filter(
      (p) => p.group === property.group && p.playerId === propertyOwner.id
    );

    const { amount, reason } = calculateRent(
      property,
      [property.group, propertiesInGroup],
      parseInt(roll)
    );

    if (amount === 0) {
      toast.error(reason);
      return;
    }

    const payingPlayer = type === "SEND" ? fromPlayer : toPlayer;
    if (payingPlayer.balance < amount) {
      toast.error(
        `${payingPlayer.name} doesn't have sufficient funds (${amount})`
      );
      return;
    }

    const propertyGroup: [string, Property[]] = [
      property.group,
      propertiesInGroup,
    ];

    setSelectedProperty({
      property,
      propertyGroup,
    });
    setCurrentView("confirmation");
  };

  const handleConfirmTransaction = () => {
    if (!transactionDetails) return;

    const transferDetails = {
      fromPlayerId: type === "SEND" ? fromPlayer.id : toPlayer.id,
      toPlayerId: type === "SEND" ? toPlayer.id : fromPlayer.id,
    };

    const formattedReason = `${transactionDetails.reason}`;

    onTransferRequest(
      transactionDetails.amount,
      formattedReason,
      transferDetails,
      roomId
    );
  };

  const getUpdatedBalances = () => {
    if (!transactionDetails) return null;

    const payingPlayer = type === "SEND" ? fromPlayer : toPlayer;
    const receivingPlayer = type === "SEND" ? toPlayer : fromPlayer;

    return {
      payingBalance: payingPlayer.balance - transactionDetails.amount,
      receivingBalance: receivingPlayer.balance + transactionDetails.amount,
      payingPlayerName: payingPlayer.name,
      receivingPlayerName: receivingPlayer.name,
    };
  };

  const handleBack = () => {
    if (currentView === "confirmation") {
      setCurrentView("properties");
      setSelectedProperty(null);
    } else if (currentView === "properties") {
      setCurrentView("colors");
      setSelectedGroup(null);
    }
  };

  const groupedProperties = Object.entries(
    properties.reduce((acc, property) => {
      if (!acc[property.group]) {
        acc[property.group] = [];
      }
      acc[property.group].push(property);
      return acc;
    }, {} as Record<string, Property[]>)
  );

  const balances = getUpdatedBalances();

  const renderConfirmationView = () => {
    if (!transactionDetails || !balances) return null;

    return (
      <>
        <h1 className="flex items-center justify-between">
          <button onClick={handleBack} className={`flex`}>
            <MdArrowBackIos />
            <div>Back</div>
          </button>
          {selectedGroup === "utility" && (
            <div className={`relative`}>
              <input
                className={`w-24 pt-4 pl-1 border rounded text-black ${numeralFace}`}
                value={roll}
                onChange={(e) => {
                  const inputValue = e.target.value;
                  if (inputValue === "") {
                    setRoll(inputValue);
                    return;
                  }
                  const numericValue = Math.max(
                    1,
                    Math.min(parseInt(inputValue, 10) || 0, 12)
                  );
                  setRoll(numericValue.toString());
                }}
                onBlur={() => {
                  const numericValue = Math.max(
                    1,
                    Math.min(parseInt(roll, 10) || 1, 12)
                  );
                  setRoll(numericValue.toString());
                }}
                type="numeric"
                min={1}
                max={12}
              />
              <p className={`text-xs text-black absolute top-1 left-1`}>
                Dice Roll
              </p>
            </div>
          )}
        </h1>
        <div className="space-y-5">
          <div className="space-y-4">
            <h2 className="text-xl">Confirm Transaction</h2>
            <div className="bg-gray-800 rounded-lg p-4 space-y-4">
              <div className="flex justify-between items-center">
                <span>Amount:</span>
                <span className={`text-xl ${numeralFace}`}>
                  {formatMoney(transactionDetails?.amount)}
                </span>
              </div>
              <div className="text-sm opacity-80">
                {transactionDetails.reason}
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-lg">Updated Balances</h3>
            <div className="bg-gray-800 rounded-lg p-4 space-y-3">
              <div className="flex justify-between">
                <span>{balances.payingPlayerName}</span>
                <span className={`text-red-400 ${numeralFace}`}>
                  {formatMoney(balances?.payingBalance || fromPlayer.balance)}
                </span>
              </div>
              <div className="flex justify-between">
                <span>{balances?.receivingPlayerName}</span>
                <span className={`text-green-400 ${numeralFace}`}>
                  {formatMoney(balances?.receivingBalance || toPlayer.balance)}
                </span>
              </div>
            </div>
          </div>

          <DrawerClose asChild className="w-full">
            <button
              className="w-full bg-blue-600 hover:bg-blue-700 text-white py-3 rounded-lg mt-4"
              onClick={handleConfirmTransaction}
            >
              Confirm {type === "SEND" ? "Payment" : "Request"}
            </button>
          </DrawerClose>
        </div>
      </>
    );
  };

  return (
    <div className={`space-y-4 text-2xl ml-2 mt-2`}>
      {currentView === "confirmation" ? (
        renderConfirmationView()
      ) : currentView === "properties" ? (
        <>
          <h1 className="flex items-center justify-start" onClick={handleBack}>
            <MdArrowBackIos />
            <div>Back</div>
          </h1>
          <div className="overflow-x-auto flex gap-4">
            {groupedProperties
              .find(([group]) => group === selectedGroup)?.[1]
              .map((property) => (
                <PropertyCard
                  key={property.id}
                  className={`flex-shrink-0 cursor-pointer ${
                    property.isMortgaged ? "opacity-50" : ""
                  }`}
                  property={property}
                  onClick={() => {
                    handlePropertySelect(property);
                  }}
                />
              ))}
          </div>
        </>
      ) : (
        <>
          <div className="grid grid-cols-4 sm:grid-cols-10 gap-2">
            {groupedProperties.map(([group, props]) => (
              <button
                key={group}
                className="p-2 rounded border aspect-square w-full"
                style={{ backgroundColor: props[0].color }}
                onClick={() => {
                  setSelectedGroup(group);
                  setCurrentView("properties");
                }}
              />
            ))}
          </div>
        </>
      )}
    </div>
  );
};

export default PayRequestRent;
