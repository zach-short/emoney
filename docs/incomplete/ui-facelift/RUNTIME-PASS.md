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

---

## Phase 2 — Type, and one money formatter. Entries written 2026-09-22.

**How the phase was walked.** A dev server from this worktree against the production API
(`.claude/launch.json` → `ui-facelift-frontend`; the entry had to be copied into the *primary*
checkout's `launch.json` for the preview tool to see it, and was restored afterwards — the
primary checkout is clean), in the Claude desktop browser pane at **390×844**, the width the
done-when names. Same throwaway production room **`UIP1CHK`** Phase 1 used, now with **two**
players: "Banker" (banker, `$1,825` after three banker adds of $150, $50 and $25 made during
this walk) and **"P2Check"** (`$1,500`), added by a direct `POST` to
`https://api.emoney.club/v1/rooms/UIP1CHK/players` because the offer panel cannot be reached
with one player in the room. **Both are real rows in the production database — delete the room
when you next open the app.** Every "verified" below is a screenshot or a `getComputedStyle` /
`Range.getBoundingClientRect()` read, named per entry.

### R2.1 — A changing balance does not reflow. **Verified — this is the load-bearing one.**
- **Where.** Any room, as banker → your own player card → ⊕ beside the balance → an amount →
  Confirm. Watch the number, not the toast.
- **Right answer.** The digits change and **nothing moves horizontally**. Verified 2026-09-22 by
  measuring the balance element across a real `$1,600 → $1,750` through the production API: width
  `86.40625px`, left `144.296875`, right `230.703125` — **identical before and after** — and a
  per-character `Range` measurement showing every glyph exactly `14.406px` wide, so glyph *n* sits
  at the same x whatever digit it holds. Computed style: `JetBrains Mono`, weight `500`,
  `font-variant-numeric: tabular-nums`. **This is what Phase 5's count-up needs to be true
  before it animates that number.**

### R2.2 — The player card and the offer panel render the same amount identically. **Verified.**
- **Where.** Note your own balance on your card. Then another player's card → the name bar →
  *I'm offering* → **Cash**. Compare the two.
- **Right answer.** Byte-identical strings, same face, same weight. At `c14faa5` the card gave
  `$1875` and the panel gave `1,500`. Verified 2026-09-22: card `$1,750`, panel `$1,750`, both
  `JetBrains Mono` / `500` / `tabular-nums`; the panel's "Your balance:" label is Manrope 600.

### R2.3 — The 12px room toast is in the body face and legible on a phone. **Verified.**
- **Where.** 390×844. Any room action that raises a notification — the banker add above is the
  easiest.
- **Right answer.** The toast text is **Manrope**, not the system stack and not Josefin. Verified
  2026-09-22 on `[data-sonner-toast] [data-title]`: `Manrope`, `12px`, weight `500`, plus a
  screenshot of "🏦 Banker has added $25 to Banker's balance". **It computed `ui-sans-serif`
  until `components/ui/sonner.tsx` named the face** — see the D3 as-built note. It is still 12px
  and still overlaps the header at 390: both are Phase 3's dials, deliberately untouched here.

### R2.4 — The event history is legible at 390×844. **Verified.**
- **Where.** 390×844, ☰ → **Event History**.
- **Right answer.** Every row reads cleanly at its 12px. Verified 2026-09-22: three rows,
  `Manrope` / `12px` / weight `400`, plus a screenshot — "🏦 Banker has added $150 to Banker's
  balance · 2 min ago".

### R2.5 — The navbar menu's four values are one numeral column. **Verified.**
- **Where.** ☰, the four rows: Bank's Properties, Free Parking, Event History, Room Code.
- **Right answer.** All four values in `JetBrains Mono` / `500`, labels in `Manrope` / `400`.
  Verified 2026-09-22 by reading all four. **Bank's Properties was still Manrope on the first
  pass** and was caught by measuring, not by looking — it is the one numeral surface in the app
  that carried no font class and no sibling to compare against.

### R2.6 — The room code reads as a code wherever it appears. **Not fully walked.**
- **Where.** Four places: ☰ → Room Code (**verified**, above); the sticky header of a room with
  **no name set** (**not walked** — `UIP1CHK` has the name "facelift phase 1", so the header
  correctly shows the *name* in Josefin and the numeral branch never rendered); `/join`'s Room
  Code field and `/create`'s New Room Code and Starting Cash fields (**not walked**).
- **Right answer.** A code or a cash figure is mono; a *name* is Josefin. The header at
  `room.client.tsx:104` switches on `room?.name`, so a named room showing Josefin is correct, not
  a miss. To check the unwalked branch, create a room with a code and no name.

### R2.7 — The property deed's rent ladder lines up. **Not walked.**
- **Where.** ☰ → Bank's Properties → a colour group → a deed. Also a player's Properties drawer.
- **Right answer.** The right-hand column of amounts — RENT, the four house rows, HOTEL, Mortgage
  Value, Houses/Hotels Cost — is a straight vertical edge, in the mono, while "With 1 House" and
  the fine print stay Josefin: the deed is still paper. **Not reached in this walk** — the check
  room has no properties dealt. Worth a look because it is the one place the numeral face was
  applied *inside* a preserved display artifact.

### R2.8 — Nothing lost its emphasis in the body sweep. **Not walked, judged by eye.**
- **Where.** Anywhere a button, a row label or a badge used to be bold: the Pay-or-Request
  button, offers-inbox Accept, the Cash King tag, Send offer, the free-parking toggle.
- **Right answer.** They still read as emphasised. `PLAN.md` BD-6 maps Josefin 700 → Manrope 600
  on these, because Josefin's 700 at its x-height is not Manrope's 700. **If any of them now
  looks heavier or lighter than it should, that is a one-utility fix and BD-6 says which.**
