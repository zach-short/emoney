"use client";
import { JSX, useId, useState } from "react";
import { josephinBold, numeralFace } from "@/components/ui/fonts";
import { formatMoney } from "@/lib/utils/money";
import { Offer, OfferNoID, Player } from "@/types/schema";
import Amount from "./amount";
import Properties from "./properties";
// import Immunity from "./immunity";

// The cap on the note, in UTF-16 units here and in runes on the Go side
// (`maxNoteLength` in `backend/websocket/offers.go`, pinned at 280 by a test
// there). The browser's count is never smaller than the server's, so a note
// this textarea accepts is never refused for length.
export const NOTE_MAX_LENGTH = 280;

// Every user-facing string in one place, so the register stays consistent
// and so the note copy -- the whole honesty of the feature -- is one edit.
// Warm, matching the kick and auction families.
const copy = {
  title: (isCounter: boolean) => (isCounter ? "Counter" : "Make an Offer"),
  offering: "I'm offering",
  asking: "I Would Like",
  noteHeading: "Add a note",
  // The note is a handshake the app writes down, not a rule it applies. This
  // app has no turns and no board position, so nothing typed here can be
  // enforced, and the line has to say so before the player relies on it.
  noteHelp:
    "For anything the app can't move for you - \"no rent on the browns for three turns\", say. E-Money writes it down for the record. Keeping to it is between the two of you.",
  notePlaceholder: "Optional",
  send: (isCounter: boolean) => (isCounter ? "Send counter" : "Send offer"),
  nothingYet: "Add cash or a property on either side to send.",
};

const newOffer = (
  fromPlayerId: string,
  roomId: string,
  toPlayerId: string,
  counterOf?: Offer
): OfferNoID => ({
  roomId,
  status: "PENDING",
  fromPlayerId,
  toPlayerId,
  // A counter starts from the original, swapped: what they asked of me is
  // now what I offer, what they offered is now what I ask. The note starts
  // blank rather than copied -- theirs was written from their side.
  offer: {
    properties: [...(counterOf?.request.properties ?? [])],
    amount: counterOf?.request.amount ?? 0,
    // immunity: [],
  },
  request: {
    properties: [...(counterOf?.offer.properties ?? [])],
    amount: counterOf?.offer.amount ?? 0,
    // immunity: [],
  },
  note: "",
  counterOf: counterOf?.id,
  createdAt: new Date(),
  updatedAt: new Date(),
});

// True when there is something tradeable on either side. The Go side refuses
// an offer with nothing on either side too (`offerRejection`); this is the
// same rule at the button, so the player is never shown a Send that fails.
export const hasSomethingToTrade = (offer: OfferNoID): boolean =>
  (offer.offer.properties?.length ?? 0) > 0 ||
  (offer.offer.amount ?? 0) > 0 ||
  (offer.request.properties?.length ?? 0) > 0 ||
  (offer.request.amount ?? 0) > 0;

const MakeOffer = ({
  player,
  currentPlayer,
  roomId,
  onCreateOffer,
  onSent,
  counterOf,
}: {
  player: Player;
  roomId: string;
  currentPlayer: Player;
  onCreateOffer: (offer: OfferNoID) => void;
  // Called after the offer is handed to the socket, so the drawer can close.
  // The send is fire-and-forget like every other action in this app: the
  // OFFER_SENT toast is the confirmation and an ERROR toast the rejection.
  onSent?: () => void;
  // Set when this form is answering an offer: the sides start swapped from it
  // and the id goes out as `counterOf`, which marks the original COUNTERED in
  // the same transaction that stores this one.
  counterOf?: Offer;
}) => {
  const [offer, setOffer] = useState<OfferNoID>(() =>
    newOffer(currentPlayer?.id, roomId, player?.id, counterOf)
  );
  const noteId = useId();

  const [view, setView] = useState<
    | "offer_amount"
    | "request_amount"
    | "offer_properties"
    | "request_properties"
    // | "offer_immunity"
    // | "request_immunity"
    | null
  >(null);

  const updateOffer = <K extends keyof OfferNoID>(
    key: K,
    value: Partial<OfferNoID[K]>
  ) => {
    setOffer((prev) => {
      const currentField = prev[key] || {};
      if (typeof currentField === "object" && !Array.isArray(currentField)) {
        return {
          ...prev,
          [key]: {
            ...currentField,
            ...value,
          },
        };
      }

      return {
        ...prev,
        [key]: value,
      };
    });
  };

  const isCounter = !!counterOf;
  const canSend = hasSomethingToTrade(offer);

  const send = () => {
    if (!canSend) return;
    onCreateOffer(offer);
    onSent?.();
  };

  const viewComponents: { [key: string]: JSX.Element } = {
    offer_amount: (
      <Amount
        offer={offer}
        updateOffer={updateOffer}
        type={"offer"}
        balance={currentPlayer.balance}
      />
    ),
    offer_properties: (
      <Properties
        offer={offer}
        updateOffer={updateOffer}
        properties={currentPlayer?.properties}
        type="offer"
      />
    ),
    // offer_immunity: (
    //   <Immunity
    //   // properties={player?.properties}
    //   // offer={offer}
    //   // updateOffer={updateOffer}
    //   />
    // ),
    request_amount: (
      <Amount
        offer={offer}
        updateOffer={updateOffer}
        type="request"
        name={player.name}
        balance={player.balance}
      />
    ),
    request_properties: (
      <Properties
        properties={player?.properties}
        offer={offer}
        updateOffer={updateOffer}
        type="request"
      />
    ),
    // request_immunity: (
    //   <Immunity
    //   // properties={player?.properties}
    //   // offer={offer}
    //   // updateOffer={updateOffer}
    //   />
    // ),
  };
  return (
    <>
      <section className={`w-full px-2 pb-8`}>
        {view ? (
          <>
            <button
              type="button"
              className={`border py-4 rounded w-full mt-5 text-2xl font-semibold`}
              onClick={() => setView(null)}
            >
              Back
            </button>
            {viewComponents[view]}
          </>
        ) : (
          <>
            <header>
              <p
                className={`${josephinBold.className} text-center text-2xl pt-5 px-2`}
              >
                {copy.title(isCounter)}
              </p>
              {isCounter && (
                <p className={`text-center text-sm text-neutral-400 px-2`}>
                  to {player?.name}
                </p>
              )}
            </header>
            <div
              className={`flex w-full items-center justify-between mt-2 px-3`}
            >
              <h1 className={`text-2xl`}>{copy.offering}</h1>
              <div className={`flex flex-col gap-y-3`}>
                <button
                  type="button"
                  className={` border
                  ${!offer?.offer?.amount ? "border-white" : `border-money-out`}
                     w-48 p-3 rounded-md`}
                  onClick={() => setView("offer_amount")}
                >
                  {!offer?.offer?.amount ? (
                    "Cash"
                  ) : (
                    <span className={numeralFace}>
                      {"\u2212"}
                      {formatMoney(offer?.offer?.amount)}
                    </span>
                  )}
                </button>
                <button
                  type="button"
                  onClick={() => setView("offer_properties")}
                  className={` border border-white p-3 rounded-md
                  ${
                    (offer?.offer?.properties?.length ?? 0) === 0
                      ? "border-white"
                      : `border-money-out`
                  }
                    p-3 rounded-md`}
                >
                  {(offer.offer.properties?.length ?? 0) > 0 && (
                    <span className={numeralFace}>
                      {"\u2212"}
                      {offer.offer.properties?.length}
                    </span>
                  )}{" "}
                  Properties
                </button>
                {/* <button
                  onClick={() => setView("offer_immunity")}
                  className={` border border-white p-3 rounded-md`}
                >
                  Immunity
                </button> */}
              </div>
            </div>
            <hr className={`my-4`} />
            <div
              className={`flex w-full items-center justify-between mt-2 px-3`}
            >
              <h2 className={`text-2xl`}>{copy.asking}</h2>
              <div className={`flex flex-col gap-y-3`}>
                <button
                  type="button"
                  onClick={() => setView("request_amount")}
                  className={`

                  ${
                    !offer?.request?.amount
                      ? "border-white"
                      : `border-money-in`
                  }

                    border p-3 rounded-md w-48`}
                >
                  {!offer?.request?.amount ? (
                    "Cash"
                  ) : (
                    <span className={numeralFace}>
                      +{formatMoney(offer.request.amount)}
                    </span>
                  )}
                </button>
                <button
                  type="button"
                  className={`
                  ${
                    (offer?.request?.properties?.length ?? 0) === 0
                      ? "border-white"
                      : `border-money-in`
                  }
                    border p-3 rounded-md`}
                  onClick={() => setView("request_properties")}
                >
                  {(offer.request.properties?.length ?? 0) > 0 && (
                    <span className={numeralFace}>
                      +{offer.request.properties?.length}
                    </span>
                  )}{" "}
                  Properties
                </button>
                {/* <button
                  onClick={() => setView("request_immunity")}
                  className={`
                  ${
                    offer?.request?.immunity.length === 0
                      ? "border-white"
                      : `border-money-in`
                  }
                    border border-white p-3 rounded-md`}
                >
                  Immunity
                </button> */}
              </div>
            </div>
            <hr className={`my-4`} />
            <div className={`px-3 flex flex-col gap-y-2`}>
              <label htmlFor={noteId} className={`text-2xl`}>
                {copy.noteHeading}
              </label>
              <p className={`text-sm text-neutral-400`}>{copy.noteHelp}</p>
              <textarea
                id={noteId}
                value={offer.note ?? ""}
                maxLength={NOTE_MAX_LENGTH}
                rows={3}
                placeholder={copy.notePlaceholder}
                // Set directly rather than through updateOffer, whose
                // object-spread branch would spread a string into indexed
                // characters when the current note is "".
                onChange={(e) =>
                  setOffer((prev) => ({ ...prev, note: e.target.value }))
                }
                className={`w-full rounded border border-white bg-black p-3 text-base text-white placeholder:text-neutral-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
              />
              <p className={`text-right text-xs text-neutral-500 ${numeralFace}`}>
                {(offer.note ?? "").length}/{NOTE_MAX_LENGTH}
              </p>
            </div>
            <div className={`px-3 mt-4 flex flex-col gap-y-2`}>
              <button
                type="button"
                onClick={send}
                disabled={!canSend}
                className={`font-semibold w-full rounded-full border border-white p-4 text-2xl transition-colors hover:bg-white hover:text-black disabled:cursor-not-allowed disabled:border-neutral-600 disabled:text-neutral-600 disabled:hover:bg-transparent focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
              >
                {copy.send(isCounter)}
              </button>
              {!canSend && (
                <p className={`text-center text-sm text-neutral-500`}>
                  {copy.nothingYet}
                </p>
              )}
            </div>
          </>
        )}
      </section>
    </>
  );
};

export default MakeOffer;
