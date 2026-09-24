// A panel that replaces another inside the same sheet -- the navbar's menu
// views, make-offer's steps -- enters with a short slide, so the swap reads as
// navigation rather than as a repaint (DESIGN.md D5, moment 4; PLAN.md Phase 6
// item 3). CSS only, through `tailwindcss-animate`: this moment is outside
// `motion`'s remit by design (DESIGN.md section 3-E).
//
// 240 ms ease-out is the "Panel/drawer transition" dial (PLAN.md section 3),
// held once as the `panel` duration token in `tailwind.config.ts`.
//
// Reduced motion keeps the state and removes the movement: every animation
// class is `motion-safe:`, so under `prefers-reduced-motion: reduce` the new
// panel is simply there. The media query is live, so turning the setting on
// or off takes effect on the next swap without a reload.
//
// Enter only. The panel being left is unmounted at once; the new one slides in
// from the side the user is heading -- forward from the right, back from the
// left. `null` is "nothing was navigated yet", so a sheet opening does not
// replay a slide on top of the drawer's own.

export type PanelMove = "forward" | "back" | null;

const FORWARD =
  "motion-safe:animate-in motion-safe:fade-in-0 motion-safe:slide-in-from-right-4 motion-safe:duration-panel motion-safe:ease-out";
const BACK =
  "motion-safe:animate-in motion-safe:fade-in-0 motion-safe:slide-in-from-left-4 motion-safe:duration-panel motion-safe:ease-out";

export const panelEnter = (move: PanelMove): string =>
  move === "forward" ? FORWARD : move === "back" ? BACK : "";
