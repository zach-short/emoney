"use client";

import { useEffect, useState } from "react";
import { TINT_FADE_MS, TINT_HOLD_MS } from "./use-count-up";

// A deed changing hands (DESIGN.md D5, moment 2; PLAN.md Phase 6 item 1): the
// property count on every affected card moves, and the card a deed landed on
// acknowledges it.
//
// Derived from the card's own data, not from the socket message. PURCHASE_
// PROPERTY and OFFER_ACCEPTED both reach the card only as `page.tsx`'s player
// refetch, which replaces every player object whole; what actually says "a
// deed moved" is the set of property ids on this player gaining or losing one.
// So an auction lot or a kick that moves a deed is acknowledged the same way,
// which is the same moment in the player's terms.
//
// Keyed on property ids, never on the array or its index: a refetch that
// returns the same deeds -- or the same deeds with a house built or a mortgage
// taken -- is not a change. A different player in the same slot, or the first
// value, is not a change either: no roll, no acknowledgement.
//
// Reduced motion keeps the state: the acknowledgement is driven by timers,
// not by an animation, so it still holds for its full time. Only the roll and
// the fade are movement, and the consumer gates both with `motion-safe:`.
//
// The hold and fade are the balance tint's (PLAN.md section 3), deliberately:
// one "this card just changed" duration for the whole card, not a second one.

export type DeedRoll = "up" | "down" | null;
export type DeedAck = "hold" | "fade" | null;

type Holding = {
  identity: string | undefined;
  key: string; // the sorted ids, joined -- a primitive, so a new array is not a change
  n: number; // one per real change, so a stale timer can be recognised
  roll: DeedRoll;
  received: boolean; // at least one id is new to this player
};

const idsOf = (key: string) => (key === "" ? [] : key.split(","));

export function useDeedChange(
  properties: { id: string }[] | undefined,
  identity: string | undefined
): { n: number; roll: DeedRoll; ack: DeedAck } {
  // A player with no deeds may arrive with the field absent; that is zero
  // deeds, not "unknown", so a first purchase still counts as one received.
  const key = (properties ?? [])
    .map((p) => p.id)
    .sort()
    .join(",");

  const [holding, setHolding] = useState<Holding>({
    identity,
    key,
    n: 0,
    roll: null,
    received: false,
  });
  const [ackEnd, setAckEnd] = useState<{ n: number; phase: "fade" | "done" }>({
    n: -1,
    phase: "done",
  });

  // Recorded during render (React's "adjust state when a prop changes"
  // pattern), as `use-count-up.ts` does, so the change wins before anything
  // paints.
  if (holding.identity !== identity || holding.key !== key) {
    const samePlayer = holding.identity === identity;
    const before = idsOf(holding.key);
    const after = idsOf(key);
    setHolding({
      identity,
      key,
      n: holding.n + 1,
      roll: !samePlayer
        ? null
        : after.length > before.length
          ? "up"
          : after.length < before.length
            ? "down"
            : null,
      // A one-for-one trade leaves the count where it was but still lands a
      // deed on this card, so "received" is about ids, not the count.
      received: samePlayer && after.some((id) => !before.includes(id)),
    });
  }

  useEffect(() => {
    if (!holding.received) return;
    const { n } = holding;
    const fade = window.setTimeout(() => setAckEnd({ n, phase: "fade" }), TINT_HOLD_MS);
    const done = window.setTimeout(
      () => setAckEnd({ n, phase: "done" }),
      TINT_HOLD_MS + TINT_FADE_MS
    );
    return () => {
      window.clearTimeout(fade);
      window.clearTimeout(done);
    };
  }, [holding]);

  const phase = ackEnd.n === holding.n ? ackEnd.phase : "hold";
  const ack: DeedAck =
    holding.received && holding.identity === identity && phase !== "done" ? phase : null;

  return { n: holding.n, roll: holding.roll, ack };
}
