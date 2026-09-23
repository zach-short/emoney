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

---

## Phase 3 — Materials, elevation and the toast. Entries written 2026-09-22.

**How the phase was walked.** A dev server from this worktree against the production API
(`.claude/launch.json` → `ui-facelift-frontend`; the entry had to be copied into the *primary*
checkout's `launch.json` for the preview tool to see it, and was restored from a backup afterwards
— the primary checkout is clean), in the Claude desktop browser pane at **390×844**, the width the
done-when names, and once at **390×520** to get the card strip to scroll under the header. Same
throwaway production room **`UIP1CHK`**, two players: "Banker" (`$1,850` after one $25 banker add
made during this walk) and "P2Check" (`$1,500`). **Both are real rows in the production database —
delete the room when you next open the app.** Every "verified" below is a screenshot or a
`getComputedStyle` / `getBoundingClientRect()` read, named per entry.

**One thing to know before re-walking R3.2.** sonner animates a top-positioned toast in from
`translateY(-100%)`. A rectangle read inside a polling loop catches that transition and reports the
toast about its own height above where it settles, which looks exactly like the fix having failed.
Wait two seconds, then measure.

### R3.1 — A drawer reads as a sheet over a room, not as a full-screen page. **Verified.**
- **Where.** Any room → the navbar menu (☰). Also the banker's ⊕ dialog and the two dialogs in
  R3.5, which now share the scrim.
- **Right answer.** The room stays *perceptible* behind the drawer: blurred and subdued, but you
  can see the room name and the top of the player card and tell there is something back there. It
  must not be a flat black band. Verified 2026-09-22: overlay computes `rgba(0, 0, 0, 0.55)` with
  `backdrop-filter: blur(12px)`, the sheet carries the `modal` step
  (`rgba(255,255,255,0.11) 0 1px 0 inset, rgba(0,0,0,0.85) 0 24px 56px -12px`), plus screenshots of
  the menu drawer and of the Add-Money dialog with the player card legible-but-blurred behind it.

### R3.2 — The toast no longer covers the header, measured not eyeballed. **Verified.**
- **Where.** 390×844, any room, any action that raises a notification — reloading the room raises
  the join broadcast, which needs no database write.
- **Right answer.** The two rectangles do not overlap. Verified 2026-09-22, settled (see the note
  above): **toast `top: 76, bottom: 131`; header `top: 0, bottom: 65`** — an 11px gap. The audit
  measured `top: 20, bottom: 72` against `64`. Also read at the same time: the toast title computes
  **`Manrope` at `14px`** (the `text-sm` dial, and Phase 2's face still winning), and the toast
  carries the `overlay` elevation step.

### R3.3 — The scrim does not stutter on a real phone. **NOT verified — needs a device.**
- **Where.** A mid-range Android, on `https://emoney.club` once this branch is merged and
  deployed. Open and close the navbar drawer ten or fifteen times in a row, then the banker's ⊕
  dialog, and watch the *close* in particular.
- **Right answer.** No dropped frames, no visible step as the blur comes off. **This is the one
  done-when no gate and no desktop browser can supply, and the repo has no performance baseline of
  any kind to fall back on** (design §1.9). `backdrop-filter` is the most expensive thing in this
  work. **If it stutters, lower the scrim dial — it is pre-authorised to move down.** It is
  `backdrop-blur-[12px]` at `components/ui/drawer.tsx:31` and `components/ui/dialog.tsx:24`; the
  header's is `backdrop-blur-[8px]` at `components/room/room.client.tsx:97`. **Never raise either
  without a measurement.**

### R3.4 — Nothing yellow survives outside the three allowed places. **Verified.**
- **Where.** `/`, `/create`, `/join` and a room, at 390×844.
- **Right answer.** `#ffff00` appears on the wordmark and on the primary action of those three
  pages, and nowhere else. Verified 2026-09-22 by reading `webkitTextStrokeColor`, `color`,
  `borderColor` and `backgroundColor` off **every element** on each page and counting
  `rgb(255,255,0)`: **`/` → 3** (E-Money, Join Room, Create Room), **`/create` → 2** (E-Money,
  Next), **`/join` → 2** (E-Money, Next), **a room → 0**. Re-run that count rather than looking.
  The player card's yellow name bar is the *player's own colour* and is not `#ffff00` — D2 keeps it.

### R3.5 — BD-4's two pinned dialogs are dark and readable. **Verified.**
- **Where.** A player card → the "Cash King" tag; and a player card → Remove player.
- **Right answer.** Both are black cards with white copy, on the blurred scrim. **player-tags**:
  white headings, grey body, and the tag's own colour still on the title banner (parked by Zach —
  `PLAN.md` BD-11). **remove-player**: the selection *inverts* — the selected option is a filled
  **white** row with black text and a `raised` step, unselected rows are a `rgba(255,255,255,0.25)`
  hairline with white text. Verified 2026-09-22: dialog `rgb(0,0,0)` / `rgb(255,255,255)`; selected
  row `bg rgb(255,255,255)` / `color rgb(0,0,0)`; three unselected rows reading back the hairline;
  the successor row's player-colour swatch intact; plus screenshots of both. **Before Phase 1's pin
  these were black-on-black and unreadable — that is what BD-4 existed to prevent.**

### R3.6 — Every money-direction indicator carries a sign glyph. **Partly verified.**
- **Where, verified.** Another player's card → the name bar → **I'm offering → Cash**, and
  **I Would Like → Cash**; then ☰ → **Free Parking**, toggling Add/Collect.
- **Where, NOT walked.** Pay or Request's *Updated Balances* (needs a property dealt and a rent
  calculation), *Balance After Purchase* (☰ → Bank's Properties → buy), Develop Property's
  Buy/Sell line, and the offer panel's **property** buttons (needs properties dealt — the check
  room has none).
- **Right answer.** Wherever red or green says which way money is going, a `−` or `+` says it too.
  Verified 2026-09-22: the offer side reads **`−10% −25% −50% −75% −100%`** with a `−$` glyph, the
  selected border and the glyph both `rgb(238, 94, 83)` (`--money-out`); the request side reads
  **`+10% … +100%`** with `+$` at `rgb(54, 211, 112)` (`--money-in`); the staged offer renders
  **`−$925`**; free parking reads **`− Add to Pot`** in money-out and **`+ Collect`** in money-in.
  On the unwalked screens the glyph is a **signed delta beside the balance** (`$1,300 (−$500)`),
  never on the balance itself.

### R3.7 — A field reads as a recessed well and paints dark. **Verified.**
- **Where.** The banker's ⊕ → "Enter amount"; the offer panel's amount field.
- **Right answer.** A faint filled well on the black card with white text and a hairline border —
  **not** a white box, and not a transparent gap with only a border. Verified 2026-09-22:
  `background rgba(255, 255, 255, 0.04)`, `color rgb(255, 255, 255)`, `box-shadow: none`, plus a
  screenshot of the Add-Money dialog. **This is the entry Phase 1's R1.5 could not get a
  trustworthy picture of**; `color-scheme: dark` on `:root` is what fixed the painting
  (`PLAN.md` BD-10). **The raw `<input>`s that do not go through `ui/input.tsx` did not get the
  fill** — `free-parking.tsx:74` measured transparent live. They are legible; they are just not on
  the new field treatment. Raised in `PLAN.md`.

### R3.8 — The card strip visibly passes under the header. **Verified.**
- **Where.** A room on a short viewport (walked at 390×520, where the card is taller than the
  screen), scrolled down. At 390×844 the card is centred and nothing scrolls under it, so this
  cannot be seen at that size.
- **Right answer.** The white card and the player's colour bar show *through* the header, blurred
  and darkened, with a hairline separating the two — not cut off at a hard black edge. Verified
  2026-09-22: header computes `rgba(0, 0, 0, 0.6)` with `backdrop-filter: blur(8px)` and
  `border-bottom rgba(255, 255, 255, 0.1)`, plus a screenshot with the card visible through it.

### R3.9 — The dice loader is no longer yellow. **Verified by code and computed value; not seen.**
- **Where.** `/my-rooms` while it reads localStorage, and anywhere
  `components/containers/data-state.tsx` is waiting on data.
- **Right answer.** Light-grey dice with black pips and a grey "LOADING", on black — no yellow.
  `--color-theme` and `#loading p`'s colour are `hsl(0 0% 88%)` in `app/globals.css` and
  `loaders/dice.tsx:13` now carries `border-neutral-600` (`PLAN.md` BD-8). **Not caught on screen**:
  the loader's window is ~500ms and no walk in this session landed inside it. The yellow count in
  R3.4 would have caught it had it been on screen during those reads, which is weaker than a
  sighting. Worth ten seconds on a slow connection.

---

## Phase 4 — Popovers F1 and F2, at `lg`. Walked 2026-09-23.

Walked from a dev server started **directly from this worktree** with
`NEXT_PUBLIC_API_URL=https://api.emoney.club NEXT_PUBLIC_API_URL_NO_PREFIX=api.emoney.club
npx next dev --port 3000`, against the production API, in the browser pane. **Port 3000 is not
optional** — the backend's CORS allowlist names `http://localhost:3000` literally
(`backend/main.go:22-26`), and a first attempt on 3200 was refused on every request.
No file in the primary checkout was touched this time; see `PLAN.md` Phase 4's as-built.

**Fixture.** The same throwaway production room **`UIP1CHK`** Phases 1-3 used. This phase changed
its data, through the app's own flows: **P2Check now owns Oriental Avenue, Vermont Avenue and
Connecticut Avenue** (bought from the bank, $1,500 → $1,180) and has **one PENDING offer to
Banker** offering Oriental Avenue for nothing. Banker is $1,850 with no deeds. **All of this is
real data in the production database — delete the room when convenient.** The two player ids are
`6ab2f0536bc35e3f6aa5f8d1` (Banker) and `6ab318a36bc35e3f6aa5f910` (P2Check); the walk switched
between them with `localStorage.setItem('room_UIP1CHK_playerId', <id>)`.

### R4.1 — A holding's deeds open in place, over the room. **Verified.**
- **Where.** ≥`lg` (walked 1280×900), a room with another player who owns deeds. Click that
  player's **"Properties N"** row. Do not open any drawer first.
- **Right answer.** A glass panel with the deeds on it as paper, the room still readable behind,
  in **one** interaction. Verified 2026-09-23: two deeds side by side (measured `left: 566` and
  `left: 838`, same `top: 442`), the third wrapped and centred; panel `rgba(0, 0, 0, 0.55)` +
  `blur(12px)`; capped at Radix's own `--radix-popover-content-available-height: 463.5px` and
  scrolling, `bottom: 888` inside a 900px viewport. Screenshot taken.

### R4.2 — A deed named in an offer opens its terms. **Verified. This is the one with no
previous path at all.**
- **Where.** ≥`lg`. Your own card → name bar → **Your offers** → an offer that names a property.
  Click the property name (dotted underline).
- **Right answer.** The paper deed floats over the **still-open** drawer, anchored under the name.
  Verified 2026-09-23 on the P2Check → Banker offer: deed visible, drawer `data-state: "open"`,
  height 810. The popover surface is transparent (`rgba(0, 0, 0, 0)`, `backdrop-filter: none`,
  `padding: 0`, `border: 0`) and the deed is `rgb(255, 255, 255)` — paper, not glass (BD-15).
  Screenshot taken.

### R4.3 — The `overlay` step is painted exactly once. **Verified — it was twice, and was fixed.**
- **Where.** Same as R4.2, on the single-deed popover.
- **Right answer.** `box-shadow` = `rgba(255, 255, 255, 0.09) 0 1px 0 inset,
  rgba(0, 0, 0, 0.75) 0 8px 24px -6px` on the surface, and `none` on the deed. It first measured
  the same shadow on **both**, on an identical rectangle `[343, 300, 256, 371]`, because
  `cn()`'s `twMerge` does not recognise the custom `shadow-overlay` key and `shadow-none` did not
  cancel it. **Worth re-checking after any future `shadow-*` override anywhere in this app.**

### R4.4 — Player at a glance, on another player's card. **Verified.**
- **Where.** ≥`lg`. Click another player's **balance figure**.
- **Right answer.** Their name, no Banker badge if they are not the banker, Balance, Properties,
  and **"Offers between you"** with the count of offers pending in either direction between you
  two. Verified 2026-09-23 on P2Check: `$1,180`, `3`, `Offers between you 1`.

### R4.5 — Player at a glance, on your own card, and the banker badge. **Verified.**
- **Where.** ≥`lg`. Click **your own** balance figure, as a banker.
- **Right answer.** A **"Banker"** pill beside your name — *the fact that rendered nowhere in the
  app before this phase* — and **"Offers waiting on you"**, matching the `N offer` badge on your
  own name bar. Verified 2026-09-23: `Banker`, `$1,850`, `0`, `Offers waiting on you 1`, against a
  name bar reading `1 offer`. Panel measured `rgba(0, 0, 0, 0.55)` + `blur(12px)`, border
  `1px rgb(228, 228, 231)`. Screenshot taken.

### R4.6 — Below `lg` nothing changed, and nothing new is focusable. **Verified by measurement.**
- **Where.** 375×812. The room, and menu → Bank's Properties → a colour group → a deed.
- **Right answer.** The drawer path behaves exactly as before — verified 2026-09-23 through to
  St. James Place's rent ladder in the horizontal strip. And measured at 375px: every popover
  trigger computes `display: none` (so it is **not in the tab order**, which is why they are two
  renderings rather than one disabled control), every drawer trigger `display: flex`, the balance
  is a `<p>` with its `<button>` twin unpainted, and a deed named in an offer is a plain
  `<span>` with `text-decoration: none`.

### R4.7 — The bank's own four-interaction path. **Verified UNCHANGED — and that is by decision,
not an oversight.**
- **Where.** Any width. Menu → Bank's Properties → colour group → scroll.
- **Right answer.** Exactly what §1.6 measured: four interactions inside a sheet. Zach excluded
  the navbar from F1 (`PLAN.md` BD-13) because that row sits inside a `DrawerContent`. **So this
  phase must not be described as turning four interactions into one** — it did that for a
  holding and for a deed inside an offer, neither of which is this path.

### R4.8 — The popover on a real touch device. **NOT WALKED.**
- **Where.** A tablet or a laptop with a touchscreen, at ≥`lg`.
- **Right answer.** D6's whole hedge is that popovers are a mouse idiom and the `lg` gate keeps
  them off phones — but **a tablet is ≥`lg` and is touch**. Nothing in this walk used a real
  touch input; the browser pane sends mouse events. Open a deed popover on a tablet and check it
  can be dismissed by tapping outside without mis-tapping a card control behind it. This is the
  nearest thing to a gap D6's stated cost leaves behind.

### R4.9 — Bundle weight on a mid-range phone. **NOT WALKED, and it is the standing hazard.**
- **Where.** The room page on a real mid-range Android over a slow connection.
- **Right answer.** +23,586 B gzipped is measured (`PLAN.md` Phase 4 *watch for*) but its effect
  on time-to-interactive is not. This joins R3.3's unmeasured scrim blur as the second thing this
  work has added to the room page that no device has judged.
