import { Player, Property } from "@/types/schema";
import { useState } from "react";
import { MdArrowBackIos } from "react-icons/md";
import { numeralFace } from "../ui/fonts";
import { formatMoney } from "@/lib/utils/money";
import { ManagePropertiesPayload } from "@/types/payloads";
import {
  getDisplayState,
  getDisplayStateManage,
} from "@/lib/utils/propertyHelpers";
import { CiCircleMinus, CiCirclePlus } from "react-icons/ci";
import PropertyCard from "../property/cards/card";
import Toast from "@/components/ui/toasts";
import { PiHouseSimpleThin, PiMoneyWavyThin } from "react-icons/pi";
import { doesPlayerOwnFullSet } from "@/components/ui/helper-funcs";

interface p {
  player: Player;
  currentPlayer: Player;
  onManageProperties?: (
    amount: number,
    managementType: ManagePropertiesPayload["managementType"],
    properties: { propertyId: string; count?: number }[],
    playerId: string
  ) => void;
}

const ManageProperties = ({ player, currentPlayer, onManageProperties }: p) => {
  const [currentView, setCurrentView] = useState<"colors" | "properties">(
    "colors"
  );
  const [houseBuildingMode, setHouseBuildingMode] = useState(false);
  const [currentHouses, setCurrentHouses] = useState(0);
  const [houseSync, setHouseSync] = useState<{
    group: string | null;
    properties: Property[] | undefined;
  }>({ group: null, properties: undefined });
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);

  // One gate for the whole drawer. This used to be asked two different ways --
  // `player.id === currentPlayer.id` for the mortgage controls and
  // `currentPlayer.id === property.playerId` for the tap target -- which can
  // disagree, and whose second branch only ever toggled a selection overlay
  // with nothing behind it. Another player's card is a read-only deed browser.
  const isOwnCard = currentPlayer?.id === player?.id;

  const groupedProperties = Object.entries(
    (player?.properties ?? []).reduce((acc, property) => {
      if (!acc[property.group]) {
        acc[property.group] = [];
      }
      acc[property.group].push(property);
      return acc;
    }, {} as Record<string, Property[]>)
  );

  // Re-seed the house counter from the server whenever a different group is
  // opened or the player's properties come back changed over the socket.
  // Done during render rather than in an effect so the counter can never be
  // painted against a set of properties it wasn't computed from.
  if (
    selectedGroup &&
    player.properties &&
    (houseSync.group !== selectedGroup ||
      houseSync.properties !== player.properties)
  ) {
    setHouseSync({ group: selectedGroup, properties: player.properties });
    setCurrentHouses(
      countHouses(
        groupedProperties.find(([group]) => group === selectedGroup)?.[1] || []
      )
    );
  }

  if (!player?.properties || player?.properties.length === 0) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className={`text-xl font-semibold text-white`}>
          No Properties Found
        </p>
      </div>
    );
  }

  const canManageHouses = (properties: Property[], property: Property) => {
    const groupedProperties: [string, Property[]][] = Object.entries(
      properties.reduce((acc, prop) => {
        if (!acc[prop.group]) acc[prop.group] = [];
        acc[prop.group].push(prop);
        return acc;
      }, {} as Record<string, Property[]>)
    );

    const allPropertiesInGroup = groupedProperties.find(
      ([group]) => group === property.group
    );
    if (!allPropertiesInGroup) {
      return "Could not find the other properties in this group.";
    }
    const group: [string, Property[]] = [
      allPropertiesInGroup[0],
      allPropertiesInGroup[1],
    ];
    const ownsFullSet = doesPlayerOwnFullSet(property, group);
    // if player doesnt have all the cards in set or if its railroad/utility cannot buy houses
    if (!ownsFullSet) {
      return "You do not own all properties in this group.";
    }

    if (
      allPropertiesInGroup[0] === "railroad" ||
      allPropertiesInGroup[0] === "utility"
    ) {
      return "You cannot build houses on railroads or utilities.";
    }

    if (properties.some((p) => p.isMortgaged)) {
      return "Some properties in this group are mortgaged.";
    }

    return true;
  };

  const canMortgageProperty = (properties: Property[]) => {
    if (properties.some((p) => p.developmentLevel > 0)) {
      return false;
    }
    return true;
  };

  const handleChangeHouses = (
    propertyCounts: { propertyId: string; count: number }[],
    totalCost: number
  ) => {
    console.log(propertyCounts);
    onManageProperties?.(totalCost, "HOUSES", propertyCounts, currentPlayer.id);
  };

  // const handleSellToBank = (property: Property) => {
  //   onManageProperties(
  //     property?.isMortgaged ? -property.price / 2 : -property.price,
  //     "SELL",
  //     [{ propertyId: property.id }],
  //     currentPlayer.id
  //   );
  // };

  const handleMortgage = (property: Property) => {
    onManageProperties?.(
      -property.price / 2,
      "MORTGAGE",
      [{ propertyId: property.id }],
      currentPlayer.id
    );
  };

  const handleUnmortgage = (property: Property) => {
    onManageProperties?.(
      property.price * 0.55,
      "UNMORTGAGE",
      [{ propertyId: property.id }],
      currentPlayer.id
    );
  };

  const renderManageHouseDialog = (properties: Property[]) => {
    const totalHousesAvailable = properties.length * 4;
    const totalHotelsAvailable = properties.length;
    const totalPropertiesAvailable =
      totalHotelsAvailable + totalHousesAvailable;

    // The counts were only ever rebuilt from the group's server-side
    // development levels plus the net change, never from their own previous
    // value, so they are a plain function of the counter.
    const initialHouses = countHouses(properties);
    const propertyCounts = distributeHouses(
      properties,
      currentHouses - initialHouses
    );

    const handleIncrement = () => {
      if (currentHouses < totalPropertiesAvailable) {
        setCurrentHouses((prev) => prev + 1);
      }
    };

    const handleDecrement = () => {
      if (currentHouses > 0) {
        setCurrentHouses((prev) => prev - 1);
      }
    };

    const totalCost =
      Math.abs(currentHouses - initialHouses) * (properties[0].houseCost ?? 0);
    const transactionType =
      currentHouses > initialHouses
        ? "ADD_HOUSES"
        : currentHouses < initialHouses
        ? "REMOVE_HOUSES"
        : "NO_CHANGE";

    const BUY = transactionType === "ADD_HOUSES";
    const NO_CHANGE = transactionType === "NO_CHANGE";
    const numHouses = Math.abs(currentHouses - initialHouses);
    return (
      <>
        {houseBuildingMode && (
          <>
            <div className={`text-center`}>
              Develop Property
            </div>
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <button onClick={handleDecrement}>
                  <CiCircleMinus className="h-12 w-12 text-white " />
                </button>
                {!NO_CHANGE ? (
                  <>
                    <span className={`font-semibold`}>
                      {transactionType === "ADD_HOUSES" ? "Buy " : "Sell "}
                      {getDisplayStateManage(numHouses, properties.length)}{" "}
                      <span className={numeralFace}>
                        ({transactionType === "ADD_HOUSES" ? "-" : "+"}
                        {formatMoney(BUY ? totalCost : totalCost / 2)})
                      </span>
                    </span>
                  </>
                ) : (
                  <div className={`font-semibold`}>No Change</div>
                )}
                <button onClick={handleIncrement}>
                  <CiCirclePlus className="h-12 w-12 rounded-full shadhow-md  text-white " />
                </button>
              </div>
              <div className="space-y-2">
                {propertyCounts.map((property) => (
                  <div
                    key={property.propertyId}
                    className="flex justify-between items-center"
                  >
                    <span className={`font-semibold`}>
                      {
                        properties.find((p) => p.id === property.propertyId)
                          ?.name
                      }
                    </span>
                    <span className={numeralFace}>
                      {getDisplayState(property.count)}
                    </span>
                  </div>
                ))}
              </div>
              <button
                className={`w-full p-2 font-semibold border border-border bg-white/[0.06] hover:bg-white/[0.12] transition-colors rounded shadow-raised disabled:opacity-50`}
                disabled={currentHouses === initialHouses}
                onClick={() => {
                  if (BUY && totalCost > player.balance) {
                    Toast({
                      message: `Insufficent funds
                      `,
                      details: `You need $${totalCost - player.balance}`,
                      icon: (
                        <PiMoneyWavyThin className="text-red-700 text-xl" />
                      ),
                    });
                  } else {
                    handleChangeHouses(
                      propertyCounts,
                      BUY ? totalCost : -totalCost / 2
                    );
                    setHouseBuildingMode(false);
                  }
                }}
              >
                Confirm
              </button>
              <button
                className={`w-full border-[1px] border-grey-400 p-2 rounded`}
                onClick={() => {
                  setHouseBuildingMode(false);
                }}
              >
                Cancel
              </button>
            </div>
          </>
        )}
      </>
    );
  };

  const renderPropertyManagement = (
    property: Property,
    groupProperties: Property[]
  ) => {
    const canMortgage = canMortgageProperty(groupProperties);
    return (
      <div className="relative group">
        <div className="relative">
          <PropertyCard
            property={property}
            className2={`pt-3`}
            onClick={
              isOwnCard
                ? () => {
                    const canManage = canManageHouses(groupProperties, property);
                    if (canManage !== true) {
                      Toast({
                        message: "You cannot build on this property",
                        details: canManage,
                        icon: (
                          <PiHouseSimpleThin className="text-red-700 text-xl" />
                        ),
                      });
                    } else {
                      setHouseBuildingMode(true);
                      setSelectedGroup(property.group);
                    }
                  }
                : undefined
            }
          />
        </div>

        {isOwnCard && (
          <>
            <div className="grid grid-cols-1 gap-2 py-2">
              {canMortgage && !property.isMortgaged && (
                <button
                  className="border border-border bg-white/[0.06] hover:bg-white/[0.12] transition-colors text-white p-2 rounded text-sm shadow-raised"
                  onClick={() => handleMortgage(property)}
                >
                  Mortgage (
                  <span className={numeralFace}>
                    {formatMoney(property.price / 2)}
                  </span>
                  )
                </button>
              )}
              {property.isMortgaged && (
                <button
                  className="border border-border bg-white/[0.06] hover:bg-white/[0.12] transition-colors text-white p-2 rounded text-sm shadow-raised"
                  onClick={() => handleUnmortgage(property)}
                >
                  Unmortgage (
                  <span className={numeralFace}>
                    {formatMoney(property.price * 0.55)}
                  </span>
                  )
                </button>
              )}
              {/* {property.developmentLevel === 0 && canMortgage && (
                <button
                  className="bg-gray-600 text-white p-2 rounded text-sm"
                  onClick={() => handleSellToBank(property)}
                >
                  Sell to Bank ($
                  {Math.floor(property?.isMortgaged ? 0 : property.price / 2)})
                </button>
              )} */}
            </div>
          </>
        )}
      </div>
    );
  };

  return (
    <div className={`space-y-4 !`}>
      {!isOwnCard && (
        <p className={`text-center text-sm text-gray-400`}>
          {player?.name}&apos;s deeds &mdash; view only
        </p>
      )}
      {currentView === "properties" ? (
        <>
          {houseBuildingMode ? (
            renderManageHouseDialog(
              groupedProperties.find(([group]) => group === selectedGroup)?.[1] ??
                []
            )
          ) : (
            <>
              <h1
                className="flex items-center justify-start text-white"
                onClick={() => {
                  setCurrentView("colors");
                  setSelectedGroup(null);
                }}
              >
                <MdArrowBackIos className={`text-white pb-1`} />
                <span className="ml-2 text-white -1">Back</span>
              </h1>
              <div className="overflow-x-auto flex gap-4">
                {groupedProperties
                  .find(([group]) => group === selectedGroup)?.[1]
                  .map((property) => (
                    <div key={property.id} className="flex-shrink-0">
                      {renderPropertyManagement(
                        property,
                        groupedProperties.find(
                          ([group]) => group === selectedGroup
                        )?.[1] ?? []
                      )}
                    </div>
                  ))}
              </div>
            </>
          )}
        </>
      ) : (
        <>
          <div className="grid grid-cols-4 sm:grid-cols-10 gap-2">
            {groupedProperties.map(([group, props]) => (
              <button
                key={group}
                className="p-2 rounded-sm border aspect-square w-full"
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

const countHouses = (properties: Property[]) =>
  properties.reduce((sum, p) => sum + p.developmentLevel, 0);

/**
 * Spreads `houseDifference` houses across a group as evenly as Monopoly's
 * even-build rule requires, starting from each property's current level.
 * Pure: the same group and difference always give the same distribution.
 */
const distributeHouses = (
  properties: Property[],
  houseDifference: number
): { propertyId: string; count: number }[] => {
  const distribution = properties.map((p) => ({
    propertyId: p.id,
    count: p.developmentLevel,
  }));

  if (houseDifference > 0) {
    for (let i = 0; i < houseDifference; i++) {
      const minHouses = Math.min(...distribution.map((p) => p.count));
      const candidateProps = distribution.filter((p) => p.count === minHouses);
      candidateProps[0].count++;
    }
  } else if (houseDifference < 0) {
    for (let i = 0; i < Math.abs(houseDifference); i++) {
      distribution.sort((a, b) => b.count - a.count);
      distribution[0].count--;
    }
    distribution.sort(
      (a, b) =>
        properties.findIndex((p) => p.id === a.propertyId) -
        properties.findIndex((p) => p.id === b.propertyId)
    );
  }

  return distribution;
};

export default ManageProperties;
