import { useEffect, useState } from "react";
import { Auction, Player, Property } from "@/types/schema";
import { josephinBold, josephinNormal } from "../ui/fonts";
import PropertyCard from "../property/cards/card";

// The inside of the bidding sheet: the open lot, what it is worth so far, what
// is still coming, the bid control, and -- for the Banker alone -- the hammer.
//
// Everything here is derived from `auction`, which is the Room document's
// auction with the live-bid overlay already applied by `auction-bar.tsx`. This
// component resolves nothing and fetches nothing; it renders what it is handed
// and raises two callbacks.

// The Banker's close moves a deed and a balance and cannot be undone from
// inside this app, so it is one tap plus a Confirm -- the shape every other
// banker write already uses (`player-card-content.tsx`'s add/remove dialog).
// The confirm is an inline swap rather than a Dialog because this panel is
// already inside a vaul Drawer and a second portal on top of it is a shape
// nothing else in this codebase does.
//
// The backstop below is only a backstop: what actually protects the confirm is
// that it is held against a high bid, so a bid landing while it is open closes
// it (see `confirmingClose`). The timer is for the other case -- a Banker who
// opened it and put the phone down with nobody bidding.
const CONFIRM_BACKSTOP_MS = 30000;

const Row = ({ label, value }: { label: string; value: string }) => (
  <div className={`flex w-full items-baseline justify-between gap-x-4`}>
    <span className={`${josephinNormal.className} text-sm text-neutral-400`}>
      {label}
    </span>
    <span className={`text-base text-white text-right`}>{value}</span>
  </div>
);

const AuctionPanel = ({
  auction,
  lot,
  upcoming,
  kickedPlayer,
  highBidder,
  currentPlayer,
  onPlaceBid,
  onCloseAuction,
}: {
  auction: Auction;
  lot?: Property;
  upcoming: Property[];
  kickedPlayer?: Player;
  highBidder?: Player;
  currentPlayer: Player;
  onPlaceBid: (propertyId: string, amount: number) => void;
  onCloseAuction: (propertyId: string, kickedPlayerId: string) => void;
}) => {
  // The smallest bid the server will take: a lot opens at $0 and the minimum
  // raise is $1 (D15), which on whole dollars is exactly "beats the high bid".
  const minimum = auction.highBid + 1;

  // The field follows the minimum until the player types in it, and starts
  // following again the moment the lot changes or they place a bid. Following
  // is what makes a fast auction usable -- the common action is "bid one more
  // than whoever just bid" -- and stopping on the first keystroke is what keeps
  // it from yanking a number out from under someone mid-type.
  const [typed, setTyped] = useState<string | null>(null);

  // The open confirm is stored as the high bid it was opened against, not as a
  // boolean, so that a bid landing while it is open closes it on its own -- no
  // effect, no cleanup, and no window in which Confirm means something other
  // than the sentence above it. The Banker taps again against the new number.
  //
  // Nothing here resets when the LOT advances, because nothing has to: the
  // caller keys this component on the lot's identity, so a new deed is a new
  // instance of it. A typed bid aimed at the deed that just sold, and a confirm
  // opened on it, both go with the old instance.
  const [confirmingAt, setConfirmingAt] = useState<number | null>(null);
  const confirmingClose = confirmingAt !== null && confirmingAt === auction.highBid;
  const setConfirmingClose = (open: boolean) =>
    setConfirmingAt(open ? auction.highBid : null);

  useEffect(() => {
    if (!confirmingClose) return;
    const timer = setTimeout(() => setConfirmingAt(null), CONFIRM_BACKSTOP_MS);
    return () => clearTimeout(timer);
  }, [confirmingClose]);

  const value = typed ?? String(minimum);
  const amount = Number(value);
  const amountIsWhole =
    value.trim() !== "" && Number.isInteger(amount) && amount > 0;

  const isRemoved = currentPlayer?.isActive === false;
  const isTheKickedPlayer = currentPlayer?.id === auction.kickedPlayerId;
  const mayBid = !isRemoved && !isTheKickedPlayer;

  // The same three rules `bidRejection` applies on the server, in the same
  // order, so a refusal reads the same whichever side catches it. The server is
  // still the authority -- these only save a round trip and an ERROR toast.
  const refusal = !amountIsWhole
    ? "Enter a whole number of dollars."
    : amount <= auction.highBid
      ? `A bid has to beat the current high bid of $${auction.highBid}.`
      : amount > (currentPlayer?.balance ?? 0)
        ? `That is more than your $${currentPlayer?.balance ?? 0}.`
        : null;

  const step = (by: number) => {
    const next = (amountIsWhole ? amount : minimum) + by;
    setTyped(String(Math.max(minimum, next)));
  };

  const placeBid = () => {
    if (refusal) return;
    onPlaceBid(auction.propertyId, amount);
    setTyped(null);
  };

  const closeLot = () => {
    onCloseAuction(auction.propertyId, auction.kickedPlayerId);
    setConfirmingClose(false);
  };

  // What the hammer will actually do, said out loud on the button. Three of the
  // four outcomes send the deed to the Bank, and the one that does not moves
  // real money, so the Banker should not have to infer which is about to
  // happen. The balance check is D17's first check restated: the server
  // re-checks it at settlement and sends the deed to the Bank if it fails, so
  // this warns rather than promises.
  const winnerCanPay =
    highBidder !== undefined &&
    highBidder.isActive !== false &&
    highBidder.balance >= auction.highBid;
  const closeLabel =
    auction.highBidderId === null || auction.highBid <= 0
      ? "No bids — return it to the Bank"
      : winnerCanPay
        ? `Sold to ${highBidder?.name ?? "the high bidder"} for $${auction.highBid}`
        : `Close — ${highBidder?.name ?? "the high bidder"} can't cover $${auction.highBid}, so it goes to the Bank`;

  return (
    <div
      className={`${josephinBold.className} flex flex-col gap-y-4 overflow-y-auto px-4 pb-8 text-white`}
    >
      <div className={`flex flex-col items-center gap-y-2`}>
        <p className={`${josephinNormal.className} text-sm text-neutral-400`}>
          {kickedPlayer?.name
            ? `${kickedPlayer.name}'s estate`
            : "A removed player's estate"}
        </p>
        {lot ? (
          <PropertyCard property={lot} />
        ) : (
          // The lot resolves out of the kicked player's own deeds, which the
          // room fetch carries. Missing means this screen is behind the
          // broadcast that opened the lot, not that anything is broken -- the
          // next refetch fills it in.
          <p className={`${josephinNormal.className} text-sm`}>
            Loading the open lot…
          </p>
        )}
      </div>

      {lot && (lot.developmentLevel > 0 || lot.isMortgaged) && (
        <p
          className={`${josephinNormal.className} rounded-md border border-neutral-700 p-3 text-xs text-neutral-300`}
        >
          {/* The close writes {isMortgaged: false, developmentLevel: 0} on
              every arm including the sale (D13), so the card above overstates
              what is being sold. Saying so is the difference between a bid and
              a guess. */}
          You are bidding on the deed alone —{" "}
          {lot.developmentLevel > 0 && "the buildings come off"}
          {lot.developmentLevel > 0 && lot.isMortgaged && " and "}
          {lot.isMortgaged && "the mortgage is cleared"} when the lot closes.
        </p>
      )}

      <div className={`flex flex-col gap-y-2`}>
        <Row
          label="High bid"
          value={
            auction.highBid > 0 && auction.highBidderId
              ? `$${auction.highBid} — ${highBidder?.name ?? "someone"}`
              : "No bids yet — opens at $0"
          }
        />
        <Row
          label="Still to come"
          value={
            upcoming.length === 0
              ? "Nothing — this is the last lot"
              : upcoming.map((p) => p.name).join(" · ")
          }
        />
      </div>

      {mayBid ? (
        <div className={`flex flex-col gap-y-2`}>
          <Row label="Your balance" value={`$${currentPlayer?.balance ?? 0}`} />
          <div className={`flex items-stretch gap-x-2`}>
            <button
              type="button"
              aria-label="Lower the bid by $1"
              onClick={() => step(-1)}
              className={`w-14 rounded-md border border-neutral-600 text-2xl transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
            >
              −
            </button>
            <input
              type="number"
              inputMode="numeric"
              min={minimum}
              step={1}
              value={value}
              aria-label="Your bid"
              onChange={(e) => setTyped(e.target.value)}
              className={`w-full rounded-md border border-neutral-600 bg-inherit p-4 text-center text-2xl`}
            />
            <button
              type="button"
              aria-label="Raise the bid by $1"
              onClick={() => step(1)}
              className={`w-14 rounded-md border border-neutral-600 text-2xl transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
            >
              +
            </button>
          </div>
          <button
            type="button"
            onClick={placeBid}
            disabled={refusal !== null}
            className={`w-full rounded-md bg-white p-4 text-lg text-black transition-colors disabled:cursor-not-allowed disabled:bg-neutral-700 disabled:text-neutral-400 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
          >
            {amountIsWhole ? `Bid $${amount}` : "Bid"}
          </button>
          {refusal && (
            <p
              className={`${josephinNormal.className} text-center text-xs text-neutral-400`}
            >
              {refusal}
            </p>
          )}
        </div>
      ) : (
        <p
          className={`${josephinNormal.className} rounded-md border border-neutral-700 p-3 text-center text-sm text-neutral-400`}
        >
          {isTheKickedPlayer
            ? "This is your estate. You can't bid on it."
            : "You're no longer in the game, so you can't bid."}
        </p>
      )}

      {currentPlayer?.isBanker && !isRemoved ? (
        confirmingClose ? (
          <div className={`flex flex-col gap-y-2`}>
            <p className={`${josephinNormal.className} text-center text-sm`}>
              {closeLabel}?
            </p>
            <div className={`flex gap-x-2`}>
              <button
                type="button"
                onClick={() => setConfirmingClose(false)}
                className={`flex-1 rounded-md border border-neutral-600 p-4 transition-colors hover:bg-white/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={closeLot}
                className={`flex-1 rounded-md bg-red-700 p-4 transition-colors hover:bg-red-600 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
              >
                Confirm
              </button>
            </div>
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setConfirmingClose(true)}
            className={`w-full rounded-md border border-red-400 p-4 text-base text-red-300 transition-colors hover:bg-red-400/10 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white`}
          >
            {closeLabel}
          </button>
        )
      ) : (
        // There is no timer anywhere in the backend (D14), so a lot that looks
        // stuck is a lot waiting on a person. Saying so is the difference
        // between "the Banker hasn't called it" and "this app is broken".
        <p
          className={`${josephinNormal.className} text-center text-xs text-neutral-500`}
        >
          The Banker closes each lot. Nothing closes on its own.
        </p>
      )}
    </div>
  );
};

export default AuctionPanel;
