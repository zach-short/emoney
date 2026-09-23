"use client";
import { Fragment, useState } from "react";
import { Offer, OfferNoID, Player, Property, Trade } from "@/types/schema";
import { RespondOfferPayload } from "@/types/payloads";
import { josephinBold } from "@/components/ui/fonts";
import { DeedPopover } from "@/components/property/deed-popover";
import MakeOffer from "./make-offer";

// Every user-facing string in one place. Warm register, matching the kick
// family. Two of these lines carry the feature's honesty and are worth
// reading twice: `noteCaption` is what stops a player reading a note as a
// rule the app is enforcing, and `acceptWarning` is what stops a stray tap
// moving deeds and cash that nothing in this app can move back.
const copy = {
  title: "Your offers",
  forYou: "Waiting on you",
  fromYou: "Waiting on them",
  empty: "Nothing on the table. Tap another player's name to make an offer.",
  theyGive: "They give",
  theyWant: "They want",
  youGive: "You give",
  youWant: "You want",
  nothing: "nothing",
  noteFrom: (name: string) => `Note from ${name}`,
  yourNote: "Your note",
  noteCaption:
    "A handshake between the two of you. E-Money keeps it on the record, but can't hold either of you to it.",
  accept: "Accept",
  counter: "Counter",
  decline: "Decline",
  withdraw: "Withdraw",
  acceptWarning:
    "The properties and cash move as soon as you confirm. There's no undo - the banker would have to put it back by hand.",
  confirmAccept: "Yes, accept",
  cancel: "Cancel",
  back: "Back",
  someone: "Someone",
};

// A deed named inside an offer. Below `lg` it is the plain text it always was;
// at `lg` it is F1's popover trigger (D6) -- and this is F1's strongest case,
// because an offer names deeds whose terms are otherwise unreachable from here.
// There is no path at all from an offer to a rent ladder today.
//
// Two renderings rather than one element that stops responding below `lg`, so
// that a phone never has a focusable control in its tab order that does
// nothing -- the mistake `property/cards/card-container.tsx` documents.
const DeedName = ({ deed }: { deed: Property }) => (
  <>
    <span className={`lg:hidden`}>{deed.name}</span>
    <DeedPopover property={deed}>
      <button
        type="button"
        className={`hidden underline decoration-dotted underline-offset-4 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white lg:inline`}
      >
        {deed.name}
      </button>
    </DeedPopover>
  </>
);

// One side of a trade as prose, from the same deed list every card already
// carries: "Baltic Avenue and $200", "$200", "Baltic Avenue", or "nothing".
// The same shape as `sideDescription` in `backend/websocket/offers.go`, which
// writes the settled record; a deed not found in the room reads as "a
// property" there too.
//
// Nodes rather than one joined string as of 2026-09-23, so each resolved deed
// can carry F1's trigger. The prose it builds is unchanged, including the
// `$${amount}` form: that mirrors `sideDescription` on the Go side, which
// writes the settled record, so it is deliberately NOT routed through
// `formatMoney()` here. Raised separately instead.
const describeSide = (side: Trade, deeds: Map<string, Property>) => {
  const parts: { key: string; node: React.ReactNode }[] = (
    side.properties ?? []
  ).map((id) => {
    const deed = deeds.get(id);
    return { key: id, node: deed ? <DeedName deed={deed} /> : "a property" };
  });
  if ((side.amount ?? 0) > 0)
    parts.push({ key: "amount", node: `$${side.amount}` });
  if (parts.length === 0) return copy.nothing;
  return (
    <>
      {parts.map((part, index) => (
        // Keyed on the deed id, never on position (PLAN.md section 5): every
        // refetch replaces the room object whole, so these arrays are rebuilt
        // with new identities on every socket message.
        <Fragment key={part.key}>
          {index > 0 && (index === parts.length - 1 ? " and " : ", ")}
          {part.node}
        </Fragment>
      ))}
    </>
  );
};

const ROW = `text-base`;
const ACTION =
  "rounded-full border px-4 py-2 text-base transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white";

const Note = ({ heading, note }: { heading: string; note: string }) => (
  <div className={`mt-2 border-l-2 border-neutral-500 pl-3`}>
    <p className={`font-semibold text-sm text-neutral-300`}>
      {heading}
    </p>
    <p className={`text-base whitespace-pre-wrap`}>
      &ldquo;{note}&rdquo;
    </p>
    <p className={`mt-1 text-xs text-neutral-500`}>
      {copy.noteCaption}
    </p>
  </div>
);

// The inbox behind your own name on your own card: every pending offer made
// to you, with accept / counter / decline, and every one you have made, with
// withdraw. It is a drawer on the card rather than a row in the menu because
// the card is where the other trade controls live, and because the badge on
// the name is the only thing that outlasts the four-second toast.
//
// A counter is the offer form again, pre-filled from the original and sent
// with its id -- one protocol, not a second one (`MakeOffer`, `counterOf`).
const OffersInbox = ({
  offers,
  currentPlayer,
  allPlayers,
  roomId,
  onCreateOffer,
  onRespondOffer,
  onClose,
}: {
  offers: Offer[];
  currentPlayer: Player;
  allPlayers: Player[];
  roomId: string;
  onCreateOffer: (offer: OfferNoID) => void;
  onRespondOffer: (
    offerId: string,
    response: RespondOfferPayload["response"]
  ) => void;
  onClose: () => void;
}) => {
  const [countering, setCountering] = useState<Offer | null>(null);
  const [confirmingId, setConfirmingId] = useState<string | null>(null);

  const nameOf = (id: string) =>
    allPlayers.find((p) => p?.id === id)?.name ?? copy.someone;
  const deeds = new Map<string, Property>(
    allPlayers.flatMap((p) => p?.properties ?? []).map((p) => [p.id, p])
  );

  const pending = offers.filter((o) => o.status === "PENDING");
  const forYou = pending.filter((o) => o.toPlayerId === currentPlayer?.id);
  const fromYou = pending.filter((o) => o.fromPlayerId === currentPlayer?.id);

  if (countering) {
    const sender = allPlayers.find((p) => p?.id === countering.fromPlayerId);
    // The sender left the room between the tap and this render: nothing to
    // counter to. Fall back to the list, where the offer will show or not.
    if (!sender) {
      setCountering(null);
      return null;
    }
    return (
      <section className={`w-full px-2`}>
        <button
          type="button"
          className={`border py-4 rounded w-full mt-5 text-2xl font-semibold`}
          onClick={() => setCountering(null)}
        >
          {copy.back}
        </button>
        <MakeOffer
          player={sender}
          currentPlayer={currentPlayer}
          roomId={roomId}
          counterOf={countering}
          onCreateOffer={onCreateOffer}
          onSent={onClose}
        />
      </section>
    );
  }

  return (
    <section className={`w-full px-4 pb-8 text-white`}>
      <header>
        <p
          className={`${josephinBold.className} text-center text-2xl pt-5 px-2`}
        >
          {copy.title}
        </p>
      </header>

      {forYou.length === 0 && fromYou.length === 0 && (
        <p
          className={`mt-8 text-center text-lg text-neutral-400`}
        >
          {copy.empty}
        </p>
      )}

      {forYou.length > 0 && (
        <>
          <h2 className={`${josephinBold.className} mt-6 text-xl`}>
            {copy.forYou}
          </h2>
          <ul className={`mt-2 flex flex-col gap-y-3`}>
            {forYou.map((o) => {
              const from = nameOf(o.fromPlayerId);
              const confirming = confirmingId === o.id;
              return (
                <li key={o.id} className={`rounded-md border p-3`}>
                  <p className={`font-semibold text-xl`}>
                    {from}
                    {o.counterOf ? " counters" : " offers"}
                  </p>
                  <p className={ROW}>
                    <span className={`text-neutral-400`}>{copy.theyGive}:</span>{" "}
                    {describeSide(o.offer, deeds)}
                  </p>
                  <p className={ROW}>
                    <span className={`text-neutral-400`}>{copy.theyWant}:</span>{" "}
                    {describeSide(o.request, deeds)}
                  </p>
                  {o.note && <Note heading={copy.noteFrom(from)} note={o.note} />}

                  {confirming ? (
                    <div className={`mt-3 flex flex-col gap-y-2`}>
                      <p className={`text-sm`}>
                        {copy.acceptWarning}
                      </p>
                      <div className={`flex flex-wrap gap-2`}>
                        <button
                          type="button"
                          className={`${ACTION} font-semibold border-white bg-white/[0.14] hover:bg-white/[0.22]`}
                          onClick={() => {
                            onRespondOffer(o.id, "ACCEPT");
                            setConfirmingId(null);
                            // Closed so the room, not this list, is what the
                            // player sees the trade land on. The refetch the
                            // OFFER_ACCEPTED broadcast triggers removes the
                            // offer from here either way.
                            onClose();
                          }}
                        >
                          {copy.confirmAccept}
                        </button>
                        <button
                          type="button"
                          className={`${ACTION} border-neutral-500 hover:bg-white/10`}
                          onClick={() => setConfirmingId(null)}
                        >
                          {copy.cancel}
                        </button>
                      </div>
                    </div>
                  ) : (
                    <div className={`mt-3 flex flex-wrap gap-2`}>
                      <button
                        type="button"
                        className={`${ACTION} font-semibold border-white hover:bg-white/[0.14]`}
                        onClick={() => setConfirmingId(o.id)}
                      >
                        {copy.accept}
                      </button>
                      <button
                        type="button"
                        className={`${ACTION} border-white hover:bg-white/10`}
                        onClick={() => setCountering(o)}
                      >
                        {copy.counter}
                      </button>
                      <button
                        type="button"
                        className={`${ACTION} border-neutral-500 hover:bg-white/10`}
                        onClick={() => onRespondOffer(o.id, "DENY")}
                      >
                        {copy.decline}
                      </button>
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        </>
      )}

      {fromYou.length > 0 && (
        <>
          <h2 className={`${josephinBold.className} mt-6 text-xl`}>
            {copy.fromYou}
          </h2>
          <ul className={`mt-2 flex flex-col gap-y-3`}>
            {fromYou.map((o) => (
              <li key={o.id} className={`rounded-md border p-3`}>
                <p className={`font-semibold text-xl`}>
                  To {nameOf(o.toPlayerId)}
                </p>
                <p className={ROW}>
                  <span className={`text-neutral-400`}>{copy.youGive}:</span>{" "}
                  {describeSide(o.offer, deeds)}
                </p>
                <p className={ROW}>
                  <span className={`text-neutral-400`}>{copy.youWant}:</span>{" "}
                  {describeSide(o.request, deeds)}
                </p>
                {o.note && <Note heading={copy.yourNote} note={o.note} />}
                <div className={`mt-3 flex flex-wrap gap-2`}>
                  <button
                    type="button"
                    className={`${ACTION} border-neutral-500 hover:bg-white/10`}
                    onClick={() => onRespondOffer(o.id, "WITHDRAW")}
                  >
                    {copy.withdraw}
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </>
      )}
    </section>
  );
};

export default OffersInbox;
