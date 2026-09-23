import { numeralFace } from "@/components/ui/fonts";
import { formatMoney } from "@/lib/utils/money";
import { OfferNoID } from "@/types/schema";
import { useState } from "react";
import { FaDollarSign } from "react-icons/fa";

const Amount = ({
  offer,
  updateOffer,
  type = "offer",
  balance = 0,
  name,
}: {
  offer: OfferNoID;
  updateOffer: (
    key: keyof OfferNoID,
    value: Partial<OfferNoID[keyof OfferNoID]>
  ) => void;
  type?: "offer" | "request";
  balance?: number;
  name?: string;
}) => {
  const currentAmount =
    type === "offer" ? offer?.offer?.amount : offer?.request?.amount;

  const displayValue = currentAmount
    ? Math.floor(currentAmount).toString()
    : "";

  const handleChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const sanitizedValue = e.target.value.replace(/[^\d]/g, "");

    const newAmount = sanitizedValue === "" ? 0 : parseInt(sanitizedValue, 10);

    if (type === "offer") {
      if (newAmount <= balance) {
        updateOffer("offer", { amount: newAmount });
      } else {
        updateOffer("offer", { amount: balance });
      }
    } else {
      updateOffer("request", { amount: newAmount });
    }
  };
  // set the initial percent state by dividing the balance based on either offer.offer.amount or .request.amount depending on the type
  const [percent, setPercent] = useState(() => {
    if (!balance || balance === 0) return 0;

    if (type === "offer") {
      const offerAmount = offer?.offer?.amount || 0;
      return (offerAmount / balance) * 100;
    }

    const requestAmount = offer?.request?.amount || 0;
    return (requestAmount / balance) * 100;
  });

  return (
    <div className="flex flex-col gap-2">
      <p className={`text-xl mt-4 font-semibold`}>
        {type === "offer" ? "Your" : `${name}'s`} balance:{" "}
        {/* Was `balance.toLocaleString()`, which rendered `1,500` against the
            player card's `$1875` for the same number. Both now go through the
            one formatter (DESIGN.md D3(a)). */}
        <span className={numeralFace}>{formatMoney(balance)}</span>
      </p>
      <div className={`flex w-full items-center gap-x-3`}>
        {[10, 25, 50, 75, 100].map((num: number, index: number) => (
          <button
            key={index}
            className={`
            border w-24 p-2 rounded-full ${numeralFace}
            ${
              percent === num
                ? type === "offer"
                  ? "border-red-700"
                  : "border-green-700"
                : "border-white"
            }
          `}
            onClick={() => {
              setPercent(num);
              // Floored: 10% of $1,555 is $155.50, and the Go side refuses a
              // fraction rather than truncating it (`parseTradeSide`), so an
              // unfloored percent would make the Send button send nothing.
              const share = Math.floor((balance * num) / 100);
              if (type === "offer") {
                updateOffer("offer", { amount: share });
              } else {
                updateOffer("request", { amount: share });
              }
            }}
          >
            {num}%
          </button>
        ))}
      </div>
      <div className={`relative`}>
        <FaDollarSign
          className={`absolute left-1 top-1/2 transform -translate-y-1/2 text-4xl ${
            type === "offer" ? "text-red-700" : "text-green-700"
          }`}
        />
        <input
          type="text"
          value={displayValue}
          onChange={handleChange}
          placeholder={`${type === "offer" ? "Offer" : "Request"} Amount`}
          onWheel={(e) => e.currentTarget.blur()}
          onKeyDown={(e) => {
            if (e.key === "." || e.key.toLowerCase() === "e") {
              e.preventDefault();
            }
          }}
          className={`${numeralFace} bg-black text-white border rounded py-6 w-full pl-10 text-sm`}
        />
      </div>
    </div>
  );
};

export default Amount;
