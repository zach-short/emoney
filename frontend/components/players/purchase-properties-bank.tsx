import { Player, Property } from "@/types/schema";
import { useState } from "react";
import { MdArrowBackIos } from "react-icons/md";
import { numeralFace } from "../ui/fonts";
import { formatMoney } from "@/lib/utils/money";
import { toast } from "sonner";
import { DrawerClose } from "../ui/drawer";
import PropertyCard from "../property/cards/card";

interface SelectColorPropertiesProps {
  properties?: Property[];
  player?: Player;
  onPurchase?: (propertyId: string, buyerId: string, price: number) => void;
}

const SelectColorProperties = ({
  properties = [],
  player,
  onPurchase,
}: SelectColorPropertiesProps) => {
  const [currentView, setCurrentView] = useState<
    "colors" | "properties" | "confirmation"
  >("colors");
  const [selectedGroup, setSelectedGroup] = useState<string | null>(null);
  const [selectedProperty, setSelectedProperty] = useState<Property | null>(
    null
  );

  if (!properties || properties.length === 0 || !player) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className={`text-xl font-semibold text-white`}>
          No Properties Found
        </p>
      </div>
    );
  }

  const handlePropertySelect = (property: Property) => {
    // `canPurchase` defaulted to false and the only call site (navbar) never
    // passed it, so this guard never ran -- you could buy your way negative.
    if (player.balance < property.price) {
      toast.error(
        `Insufficient funds to purchase ${property.name} ($${property.price})`
      );
      return;
    }
    setSelectedProperty(property);
    setCurrentView("confirmation");
  };

  const handleConfirmPurchase = () => {
    if (!selectedProperty) return;

    onPurchase?.(selectedProperty.id, player.id, selectedProperty.price);
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

  const renderConfirmationView = () => {
    if (!selectedProperty) return null;

    const newBalance = player.balance - selectedProperty.price;

    return (
      <>
        <button
          type="button"
          aria-label="Back"
          className="flex items-center justify-start mb-6 rounded-md p-2 -ml-2 transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
          onClick={handleBack}
        >
          <MdArrowBackIos />
        </button>
        <div className="space-y-2">
          <div className="space-y-2">
            <h2 className="text-xl">Confirm Purchase</h2>
            <div className="bg-gray-800 rounded-lg p-4 space-y-2">
              <div className="flex justify-between items-center">
                <span>Price:</span>
                <span className={`text-xl ${numeralFace}`}>
                  {formatMoney(selectedProperty.price)}
                </span>
              </div>
              <div className="text-sm opacity-80">{selectedProperty.name}</div>
            </div>
          </div>

          <div className="space-y-2">
            <h3 className="text-lg">Balance After Purchase</h3>
            <div className="bg-gray-800 rounded-lg p-4">
              <div className="flex justify-between">
                <span>{player.name}</span>
                <span className={`text-yellow-400 ${numeralFace}`}>
                  {formatMoney(newBalance)}
                </span>
              </div>
            </div>
          </div>

          <DrawerClose asChild className="w-full">
            <button
              className="w-full bg-green-600 hover:bg-green-700 text-white py-3 rounded-lg mt-4"
              onClick={handleConfirmPurchase}
            >
              Confirm Purchase
            </button>
          </DrawerClose>
        </div>
      </>
    );
  };

  return (
    <div className={`space-y-2 text-2xl`}>
      {currentView === "confirmation" ? (
        renderConfirmationView()
      ) : currentView === "properties" ? (
        <>
          <button
            type="button"
            aria-label="Back"
            className="flex items-center justify-start rounded-md p-2 -ml-2 transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
            onClick={handleBack}
          >
            <MdArrowBackIos />
          </button>
          <div className="overflow-x-auto flex gap-4">
            {groupedProperties
              .find(([group]) => group === selectedGroup)?.[1]
              .map((property: Property, index: number) => (
                <div className="relative" key={index}>
                  <PropertyCard
                    property={property}
                    className="flex-shrink-0 cursor-pointer"
                    onClick={() => handlePropertySelect(property)}
                  />
                  <div className="absolute bottom-0 left-0 right-0 bg-black bg-opacity-50 p-2 text-center">
                    <span className={`text-lg ${numeralFace}`}>
                      {formatMoney(property?.price)}
                    </span>
                  </div>
                </div>
              ))}
          </div>
        </>
      ) : (
        <>
          <div className="grid grid-cols-4 sm:grid-cols-10 gap-2">
            {groupedProperties.map(([group, props]) => (
              <button
                key={group}
                className="p-2 rounded border aspect-square w-full transition hover:scale-105 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
                style={{ backgroundColor: props[0].color }}
                aria-label={`${group} (${props.length} for sale)`}
                title={`${group} (${props.length} for sale)`}
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

export default SelectColorProperties;
