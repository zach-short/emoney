"use client";

import { animate, useReducedMotion } from "motion/react";
import { useEffect, useState } from "react";

// A balance that counts to the server's number (DESIGN.md D5, PLAN.md Phase 5).
//
// A count shows numbers the server never sent. The three ratified rules are what
// keep that honest, and nothing else in the repo checks them, so each is
// enforced here and marked where it is:
//   RULE 1  Duration 450 ms by default, never longer than 600 ms.
//   RULE 2  Always count TO the authoritative value, FROM the previous
//           authoritative value. Never chain deltas, never ease through a
//           computed intermediate target.
//   RULE 3  A new value arriving mid-count SNAPS to that value. It does not
//           re-ease from wherever the interpolation had got to.
// Reduced motion = state kept, movement removed: no count (the number changes
// at once) but the tint still shows for its full hold.
//
// Also enforced: a different player in the same slot, or the first value, snaps
// with no tint and no count from 0. A refetch that repeats the balance is not a
// change: the inputs are primitives, so a new `player` object does nothing.

export const COUNT_UP_MS = 450;
export const COUNT_UP_CAP_MS = 600;
export const TINT_HOLD_MS = 900;
export const TINT_FADE_MS = 300;

export type MoneyDirection = "in" | "out";
export type BalanceTint = { direction: MoneyDirection; phase: "hold" | "fade" };

type Change = {
  identity: string | undefined;
  from: number | null; // the previous authoritative value; null = nothing to count from
  to: number | null; // the current authoritative value
  n: number; // one per real change, so a stale frame or timer can be recognised
  counts: boolean; // whether this change may animate at all
};

const isMove = (c: Change) => c.from !== null && c.to !== null && c.from !== c.to;

export function useCountUp(
  value: number | null | undefined,
  identity: string | undefined
): { display: number | null; tint: BalanceTint | null } {
  const to = value ?? null;
  // `null` on the server and `true` under reduced motion. Only an explicit
  // `false` allows a count.
  const reduceMotion = useReducedMotion();

  const [change, setChange] = useState<Change>({
    identity,
    from: null,
    to,
    n: 0,
    counts: false,
  });
  const [frame, setFrame] = useState<{ n: number; v: number } | null>(null);
  const [finished, setFinished] = useState(-1); // `n` of the last count that ended
  const [tintEnd, setTintEnd] = useState<{ n: number; phase: "fade" | "done" }>({
    n: -1,
    phase: "done",
  });

  // A new server value, or a different player, recorded during render (React's
  // "adjust state when a prop changes" pattern), so it wins before anything
  // paints.
  if (change.identity !== identity || change.to !== to) {
    const samePlayer = change.identity === identity;
    const midFlight = change.counts && finished !== change.n;
    const next = {
      identity,
      // RULE 2: `from` is the last value the SERVER sent, never an
      // interpolated frame.
      from: samePlayer ? change.to : null,
      to,
      n: change.n + 1,
      counts: false,
    };
    // RULE 3: a value landing mid-count does not start a new count; it snaps.
    // Reduced motion: `useReducedMotion` reads the setting once at mount in
    // motion 13.4.3, so the live media query is checked too. This branch only
    // runs on a prop change, which is always on the client.
    next.counts =
      isMove(next) &&
      !midFlight &&
      reduceMotion === false &&
      !window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    setChange(next);
  }

  useEffect(() => {
    // No other early return: a counting change ALWAYS starts, and always ends
    // by completion or the cap, because render shows `from` until it does.
    if (!change.counts) return;
    const { n } = change;
    let live = true;
    // A holder rather than a `const`, so `end` is safe even if `animate` were
    // ever to complete synchronously, before its return value is assigned.
    const handle: { controls?: ReturnType<typeof animate> } = {};
    const halt = () => {
      live = false;
      handle.controls?.stop();
    };
    // Only a completed or capped count is `finished`. Cleanup halts without
    // marking it, so Strict Mode's mount/unmount/mount in dev re-runs the
    // count instead of hiding it.
    const end = () => {
      if (!live) return;
      halt();
      setFinished(n);
    };
    handle.controls = animate(change.from as number, change.to as number, {
      // RULE 1: the duration is clamped to the cap...
      duration: Math.min(COUNT_UP_MS, COUNT_UP_CAP_MS) / 1000,
      // An ease with no overshoot, so every frame lies between two
      // authoritative values.
      ease: "easeOut",
      onUpdate: (v) => live && setFrame({ n, v }),
      onComplete: end,
    });
    // RULE 1: ...and a timer ends it at the cap even if frames stall (a
    // hidden tab pauses requestAnimationFrame, but not this).
    const cap = window.setTimeout(end, COUNT_UP_CAP_MS);
    return () => {
      window.clearTimeout(cap);
      halt();
    };
  }, [change]);

  useEffect(() => {
    if (!isMove(change)) return;
    const { n } = change;
    const fade = window.setTimeout(() => setTintEnd({ n, phase: "fade" }), TINT_HOLD_MS);
    const done = window.setTimeout(
      () => setTintEnd({ n, phase: "done" }),
      TINT_HOLD_MS + TINT_FADE_MS
    );
    return () => {
      window.clearTimeout(fade);
      window.clearTimeout(done);
    };
  }, [change]);

  // RULE 2 and RULE 3, at render: a frame shows only if it belongs to the
  // current change, for the current player, counting to the current value.
  // Before its first frame lands it shows `from`, the previous server value,
  // so the count does not flash the end value and drop back. Anything else
  // shows the authoritative value itself, so after any sequence of updates
  // the screen settles on exactly `value` -- at most COUNT_UP_CAP_MS after the
  // last change.
  const counting =
    change.counts &&
    finished !== change.n &&
    change.identity === identity &&
    change.to === to;
  const display = !counting
    ? to
    : frame !== null && frame.n === change.n
      ? Math.round(frame.v)
      : change.from;

  // Reduced motion keeps the tint: it is driven by timers, not by `counts`.
  const tintPhase: BalanceTint["phase"] | "done" = tintEnd.n === change.n ? tintEnd.phase : "hold";
  const tint =
    isMove(change) && change.identity === identity && tintPhase !== "done"
      ? {
          direction: ((change.to as number) > (change.from as number)
            ? "in"
            : "out") as MoneyDirection,
          phase: tintPhase,
        }
      : null;

  return { display, tint };
}
