"use client";

import { Offer, Player } from "@/types/schema";
import { Popover, PopoverContent, PopoverTrigger } from "../ui/popover";
import { numeralFace } from "../ui/fonts";
import { formatMoney } from "@/lib/utils/money";

// F2 (DESIGN.md D6): this player's four facts without opening the card's drawer
// stack. Gated to `lg` and above by its call site.
//
// D6 named four facts and the card face already shows two of them -- the
// balance and the property count are both on it (`player-card-content.tsx`), so
// this is the THIN version Zach chose in chat 2026-09-23 (PLAN.md BD-14): the
// two genuinely invisible facts, with the two visible ones restated so the
// panel reads as a whole rather than as a footnote.
//
// The two that were invisible:
//  - `isBanker` renders NOWHERE in the app. Grepped 2026-09-23: it appears only
//    as a gate on the banker's own controls (`player-card-content.tsx`) and in
//    `remove-player.tsx:115`'s successor logic. Nothing ever told the room who
//    the banker is.
//  - The pending-offer count, which the name bar shows only on your own card.
//
// What this deliberately does NOT claim: another player's TOTAL pending offers.
// `offers` holds only the current player's offers, both directions
// (`room.client.tsx`), so that number is not on the client at all. On another
// player's card this counts the offers pending BETWEEN the two of you, which is
// what the data supports and what the label says.
// At module scope, not inside the component: a component declared during render
// is a new type on every render, so React remounts it and it loses any state it
// had. `react-hooks/static-components` catches this and the gate is red on it.
const Row = ({ label, value }: { label: string; value: string }) => (
  <div className={`flex items-center justify-between gap-6`}>
    <span>{label}</span>
    <span className={numeralFace}>{value}</span>
  </div>
);

const PlayerGlance = ({
  player,
  currentPlayer,
  offers,
  children,
}: {
  player: Player;
  currentPlayer: Player;
  offers: Offer[];
  children: React.ReactNode;
}) => {
  const isSelf = currentPlayer?.id === player?.id;
  const pending = offers.filter((o) => o.status === "PENDING");
  // Your own card answers "what is waiting on me", which is the same set the
  // name bar badges. Another player's answers "what is open between us".
  const offerCount = isSelf
    ? pending.filter((o) => o.toPlayerId === currentPlayer?.id).length
    : pending.filter(
        (o) =>
          (o.fromPlayerId === currentPlayer?.id &&
            o.toPlayerId === player?.id) ||
          (o.fromPlayerId === player?.id && o.toPlayerId === currentPlayer?.id),
      ).length;

  return (
    <Popover>
      <PopoverTrigger asChild>{children}</PopoverTrigger>
      <PopoverContent className={`w-64 text-base`}>
        <div className={`flex flex-col gap-2`}>
          <div className={`flex items-center justify-between gap-6`}>
            <span className={`font-semibold`}>{player?.name}</span>
            {/* "Banker" -- terse, matching the room's one-word rows, chosen in
                chat 2026-09-23. Nothing in the room said this before. */}
            {player?.isBanker && (
              <span
                className={`rounded-full border border-border px-2 py-[2px] text-xs font-semibold`}
              >
                Banker
              </span>
            )}
          </div>
          <Row label="Balance" value={formatMoney(player?.balance)} />
          <Row
            label="Properties"
            value={String(player?.properties?.length || 0)}
          />
          <Row
            label={isSelf ? "Offers waiting on you" : "Offers between you"}
            value={String(offerCount)}
          />
        </div>
      </PopoverContent>
    </Popover>
  );
};

export default PlayerGlance;
