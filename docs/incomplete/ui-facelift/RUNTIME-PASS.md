# RUNTIME-PASS — UI facelift

Stage 7 of `docs/AGENT-PRACTICES.md` §2.2. **Zach walks this; it does not block a phase or its
close-out.** What it blocks is a session claiming something was seen working when it was not
(R10). Each entry is three lines: the goal in product terms, where to look, and what the right
answer is.

Entries are numbered `R<phase>.<n>` and appended by the phase that created them.

---

## Phase 1 — The theme substrate. Entries written 2026-09-22.

**How the phase was walked.** A dev server from this worktree against the production API
(`.claude/launch.json` → `ui-facelift-frontend`, which is `scripts/emoney dev`'s two variables
with this worktree's path), in the Claude desktop browser pane at 1024×768, in a **throwaway
production room `UIP1CHK`, name "facelift phase 1", one player "Banker", $1600 after one $100
banker add.** Delete it when you next open the app — it is a real room in the production
database. Every "verified" below is either a screenshot or a `getComputedStyle` read, named per
entry.

### R1.1 — The banker's dialog is dark and its Confirm button reads. **Verified.**
- **Where.** Any room, as banker → a player card → the ⊕ beside the balance.
- **Right answer.** Black card, white title and white typed amount, light hairline border; the
  Confirm button is a near-white filled rectangle with dark text, **not** a black-on-black one.
  Verified 2026-09-22: dialog `rgb(0,0,0)`, text `rgb(255,255,255)`, Confirm
  `rgb(250,250,250)` on `rgb(24,24,27)`, plus a screenshot.

### R1.2 — Four drawers are unchanged by the phase. **Verified.**
- **Where.** The navbar menu (☰); a player card's name → offers; Select Colour on `/create`
  step 2; a card's Properties row.
- **Right answer.** Each identical to before: ground `rgb(0,0,0)`, text `rgb(255,255,255)`,
  border `rgb(228,228,231)`, grab handle `rgb(244,244,245)`. All four read 2026-09-22 with
  `getComputedStyle`; the deleted `bg-black`/`text-white` literals were `#000`/`#fff`, which is
  exactly what the tokens now resolve to, so a difference here would mean a token is wrong.

### R1.3 — A room notification is readable. **Verified.**
- **Where.** Any room, trigger any notification (the $100 banker add did it).
- **Right answer.** The toast is a black rounded box with a light border and white text at
  top-centre, **not** a white box. Screenshot 2026-09-22: "Banker has added $100 to Banker's
  balance", dark.

### R1.4 — `/install`'s two strings are legible. **NOT verified — needs a device.**
- **Where.** `/install`, on an iPhone (the iOS instructions branch) **and** from the installed
  PWA (the standalone "Home" link branch). Neither branch renders in a desktop browser:
  `useIsIOS` reads the user agent and `useIsStandalone` matches `(display-mode: standalone)`
  (`hooks/use-browser-env.ts`), so a desktop `/install` is a blank black page, exactly as the
  audit found.
- **Right answer.** Both legible. The `text-black` classes at `install/page.tsx:55,70` are gone
  and neither element sets a colour, so both inherit `--foreground` (`rgb(255,255,255)`, verified
  on `body`). That is a code-and-computed-style argument, not a sighting.

### R1.5 — The amount field's painted appearance. **NOT verified — browser artifact.**
- **Where.** The same Add-Money dialog, the "Enter amount" input, in a real browser.
- **Right answer.** It should be a transparent field on the black card with white text and a
  light hairline border. The CSS says exactly that (`bg-transparent`, colour
  `rgb(255,255,255)`, border `rgb(228,228,231)`, verified). **The Claude browser pane paints
  every native text input as a white box with dark text regardless of CSS** — reproduced
  2026-09-22 with three control inputs appended to the page, including one with
  `color-scheme: dark`, all of which compute transparent-with-white-text and all of which
  rasterise light. So the DOM is right and the picture cannot be trusted. If it *is* a white
  box in your browser, that is Phase 3's item 8, not a Phase 1 regression: `ui/input.tsx` was not
  touched by this phase.

### R1.6 — The two pinned dialogs look exactly as they did. **Verified.**
- **Where.** A player card → the "Cash King" tag (player-tags); a player card → Remove player.
- **Right answer.** Both still white cards with black copy, unchanged from before the phase.
  Verified 2026-09-22 after the pin: both `rgb(255,255,255)`, copy `rgb(0,0,0)`, remove-player's
  selected option still white-on-black. **Before the pin both were black-on-black and unreadable**
  — that is what BD-4 exists to prevent, and what Phase 3 must clear properly.

### R1.7 — The sixth drawer, Pay or Request. **NOT walked.**
- **Where.** A room with **two** players, as banker → another player's card → Pay or Request
  (`player-card-content.tsx:222`).
- **Right answer.** Identical to before: black ground, white text, light border. Not walked
  because it needs a second player in the room and the join form could not be driven from the
  browser pane (its controlled input did not take synthetic events; the React native-setter
  workaround filled it, but the step-2 click never advanced). Its literal was `bg-black` with no
  `text-white`, and `--background` is `#000`, so the argument is the same as R1.2's — but it is an
  argument, not a sighting.

### R1.8 — `reason-select.tsx` is unreachable. **Verified, by grep.**
- **Where.** Nowhere. `grep -rn "reason-select\|ReasonSelect" app components hooks lib`
  (2026-09-22) returns two hits: its own definition and a commented-out import at
  `p2p-custom-transfer.tsx:4`.
- **Right answer.** Nothing to walk. The literal came out with the other five; the file is dead
  code already owned by board row 1 / TRIAGE B5.
