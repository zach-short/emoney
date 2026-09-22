# DESIGN — UI facelift: materials, type, motion and the surfaces that carry them

**Status: RATIFIED 2026-09-22.** Written as `SCOPE.md` on 2026-09-22 by an Opus 5 session and renamed to `DESIGN.md` on 2026-09-22 when GATE 1 closed; the ten ratified decisions are **§0**, which is the part of this document that is frozen. This effort entered
`docs/AGENT-PRACTICES.md` Stage 2 as an audit (§2.4: "an audit produces its findings as Stage 1
output and then enters at Stage 2") and is now at **Stage 3**. §1 is the Stage 1 output and §2–§5
still propose and decide nothing — they are evidence, not authority. **§0 is the authority**;
where §3 offers options that §0 has since decided between, §0 wins. GATE 1 was §6, closed
2026-09-22. `PLAN.md` beside this file is Stage 4 and stops at GATE 2.

**What this is about.** Zach asked for a visual and motion pass that makes e-money feel like
Raycast and like `~/Projects/furlough`'s "Ember Glass" system — glass materials with real depth,
motion tied to state rather than decoration, a tight type hierarchy, restraint that saves
boldness for the moments that matter. This document is the reading that has to come first: what
the app actually renders as of 2026-09-22, measured in the deployed product and cross-read
against the source, so that every option in §3 is priced against the real thing rather than
against a screenshot.

**Read first:** `HANDOFF.md` (the ledger), then `CLAUDE.md`, then this file. Line numbers are
against `main` at `c14faa5` unless stated. The working tree was clean at the start of this
session (`git status --short`, 2026-09-22).

**How the live claims were made (R10).** Two kinds appear below and each is labelled. *Walked*
means driven in a browser against `https://emoney.club` on 2026-09-22 at 1440×900 and 390×844,
with a real room (`UIAUD2`) created for the purpose and real money moved through it. *Read*
means taken from the source with a `file:line`, not observed. Where a number is given —
a rectangle, a computed colour, a font size — it was read out of the live DOM on 2026-09-22
and the reading is quoted.

**What could not be reached, and why.** The auction surfaces (`AUCTION_STARTED`, `BID_PLACED`,
`AUCTION_LOT_CLOSED`) need a kick with an `AUCTION` disposition and at least three players; they
are **read, not walked**. A *received* offer sitting in an inbox was not reached: `playerStore`
keys identity by room code in one `localStorage` (`room_<code>_playerId`, per
`components/navbar/navbar.tsx:215-217`), so one browser profile is one player at a time. The
compose form and the empty inbox were walked by clearing that key and rejoining as a second
player; the pending-incoming state was not. `MANAGE_PROPERTIES`, `PURCHASE_PROPERTY` and the
kick/removal flows were reached as screens but not driven to completion.

---

## 0. Decisions — ratified 2026-09-22

GATE 1 closed 2026-09-22, in chat, in one batch (R6): all ten of §6's questions were answered
and **every one took this document's own recommendation**. Each decision below states what was
decided, the defense it rests on — carried from the §3 option that argued it, not re-derived —
what it supersedes, and the date.

**From here this design is frozen** (`docs/AGENT-PRACTICES.md` Stage 3). It changes by
amendment — a new dated `D<n>`, or an `As built:` note written under the decision it deviates
from — never by editing a decision in place. §1–§5 stand as written on 2026-09-22 and are the
evidence these decisions rest on; §6 is closed and kept for the record rather than deleted (R5,
8.4: docs are append-and-amend).

**Supersession pointers, all ten: none.** This is the first ratification of this effort. No
decision here replaces an earlier one, and none is partial.

---

### D1 — Dark theme only. Ratified 2026-09-22. (§6.1 → §3-A)

The `.dark` block (`globals.css:33-58`) is **deleted, not populated**. One token set is defined
on `:root` — the block now at `globals.css:6-32` holding the stock light zinc values — and
redefined to e-money's actual ground. `layout.tsx:21` stops hardcoding `bg-black text-white`
and takes its ground from the tokens. The six pasted `bg-black text-white` literals cancelling
`bg-background` on `DrawerContent` (`navbar.tsx:69`, `player-card.tsx:115`,
`player-card-content.tsx:181,222`, `color-select-drawer.tsx:89`, `reason-select.tsx:51`) come
out. `next-themes` (`package.json:22`) becomes dead weight to be removed, and with it the
`useTheme` call that is its only consumer (`components/ui/sonner.tsx:3,9`).

**Defense.** Nothing else in this document can be built on the state §1.1 measured. A glass
material is a translucency over a *ground*; with the ground hardcoded on `<body>` and the tokens
resolving to their light values, there is no ground to be translucent over, and every new
component inherits the same bug. Dark-only is also the smallest version of the fix — one token
set rather than two — and lets the design commit to a single ground. E-money is played in a room
with other people around a table; a light theme is a feature nobody has asked for, and no code
has ever rendered one.

**The argument it beat.** "Both themes" keeps `next-themes` meaningful and needs a toggle placed
somewhere. It also roughly doubles this work: every surface in §1 would have to be checked
twice, and roughly half carry hardcoded `bg-black` / `text-white` / `text-black` literals that
would each need their own decision (§5).

**The cost accepted with it.** §3-A's own strongest argument against stands and is not waved
away: this change is invisible when it works, it has the widest blast radius of anything here,
and there is no frontend test runner to catch a regression in it (§5). That is precisely why its
`done when` in `PLAN.md` is a described walkthrough and not a green gate.

### D2 — Greyscale chrome; the yellow is identity only. Ratified 2026-09-22. (§6.2 → §3-B: B3 + B1)

The chrome goes greyscale on a near-black ground. **Player colour and property-group colour are
the only saturated colour inside the room.** The existing `#ffff00` (`globals.css:75` and the
loader's second `:root` at `:168-172`) is kept strictly as identity — the wordmark and the
primary action on `/`, `/create` and `/join` — and **nowhere else**; it stops appearing as a
generic border colour and stops being applied through raw `!important` (`globals.css:79-81`).
Red and green are reserved for money direction only, promoted from the ad-hoc literals at
`make-offer/amount.tsx:71-74` and `navbar/free-parking.tsx:81` into real tokens, and **always
carry a redundant sign glyph**.

**Defense.** This is the most disciplined answer and the one closest to the reference points.
The screen's colour then always *means* something — whose money, which group — and never means
"this is a button". It also solves a problem §1 did not go looking for: with six players, chrome
chroma and player chroma compete. Keeping the yellow as identity retains the one genuinely
distinctive thing the product has (the outlined-text treatment at `globals.css:70-78`) while
removing the use that was undisciplined.

**The arguments it beat.** B1 alone — keep and discipline the yellow everywhere — loses because
pure yellow on black reads as hazard-tape rather than money, has no usable dark-on-light inverse
for the white property cards, and fails legibility at `text-xs` well before it fails contrast
maths. B2 — money green as the accent — loses on the one pairing that fails for the most common
colour-vision deficiency, in an app that puts money direction at its centre; the redundant sign
glyph this decision keeps is B2's honest mitigation, adopted without adopting B2.

**The cost accepted with it.** B3's own argument against: a greyscale chrome risks the
"templated shadcn dark app" look this effort exists to escape. The retained yellow is the hedge,
and it is deliberately narrow.

### D3 — Three type jobs, split; numerals first. Ratified 2026-09-22. (§6.3 → §3-C)

Three sub-answers, all taking the recommendation:

- **(a) Numerals — yes, and this is the load-bearing half.** Tabular figures are the floor, a
  real mono is the intent. Every dollar amount, the room code, rent ladders and counts use it.
  The two money formats found at `player-card-content.tsx:161` (`$1875`) and
  `make-offer/amount.tsx:61` (`1,500`) collapse onto one shared formatter.
- **(b) Body / UI face — a dedicated face**, set on `<body>` so the system stack stops being the
  default. Not Josefin everywhere.
- **(c) Display face — keep Josefin Sans.** It is the voice of the wordmark and the property
  cards and it is good at size.

**Defense.** The numeral half is not a taste call. §1.2 measured `font-variant-numeric: normal`
on the card balance. Every amount in this game is a number that changes while someone is looking
at it, and proportional digits mean it *reflows* as it changes — the `1` is narrower than the
`8`, so `$1500 → $1750` shifts the glyphs. Any motion applied to that number under D5 sits on
top of a layout that is already moving, and the two are indistinguishable to the eye. The
dedicated body face is argued separately and on its own terms: Josefin's very low x-height is
what makes the 12px toast and the event log hard to read.

**The argument it beat.** Three families is three font payloads on a page that must stay snappy
on a phone on a table in a pub, and §1.9 confirms no weight measurement was ever taken. The
cheaper version — keep Josefin everywhere, add `tabular-nums`, add one mono for amounts — was
rejected as the *target* but is explicitly the fallback if a weight measurement during the build
says so; that fallback is a `PLAN.md` dial, not a reopening of this decision.

### D4 — Exactly four glass surfaces. Ratified 2026-09-22. (§6.4 → §3-D)

Four, and only four: **the drawer scrim** (`ui/drawer.tsx:31`, `bg-black/80` as of 2026-09-22), **the
sticky header** (`room.client.tsx:97`, `bg-black` as of 2026-09-22), **popover and menu surfaces** (the new
ones D6 adds), and **a 3–4 step elevation scale** replacing the 14 unsystematic shadow utilities
counted in §1.3 (re-counted 2026-09-22: still 14).

**Explicitly not glassed: the player card and the property title deed.** They are *paper*.

**Defense.** Glass is applied where there is something behind it worth seeing and withheld
everywhere else. Four surfaces is a system; twelve would be a style. The scrim is the one place
where the current state is a defect rather than a preference — §1.3 measured that `bg-black/80`
over a black ground does not make the room recede, it deletes it, so a drawer reads as a
full-screen page rather than a sheet over a room. The deed stays paper because §1.3 found it to
be the best asset in the product and the physical-card metaphor is the product's best idea;
frosting it would be decoration in exactly the sense this effort exists to avoid.

**The cost accepted with it.** `backdrop-filter` is the most expensive thing in this document, a
known compositor cost on low-end Android, in an app whose whole job is to be instant during a
board game — and §1.9 confirms no performance baseline exists to say whether that risk is real
here or theoretical. The blur dial is set low for this reason (§4: 12px scrim, 8px header) and
§5 carries the hazard. Raising it requires a device measurement first.

### D5 — Cash count-up, with its mitigation as a first-class constraint. Ratified 2026-09-22. (§6.5 → §3-E)

The four motion moments are confirmed and **there is no fifth**: cash changing, a deed changing
hands, an offer arriving, and panel/drawer transitions. `prefers-reduced-motion` is part of this
decision and not a follow-up — **reduced motion removes movement and keeps state**: no count-up,
no slide, no parallax; the number still changes, the tint still appears as a non-animated state,
the badge still renders.

**The cash count-up is ratified, and it is the single riskiest item in this document.** Its
mitigation is carried forward as a constraint of equal standing with the decision itself, not as
a footnote:

1. **Hard-cap the duration at 600 ms** (§4's dial: 450 ms default, 600 ms cap).
2. **Always animate *to* the authoritative server value**, never through a computed sequence.
3. **A new value arriving mid-animation snaps rather than re-eases.**

**Defense.** This is the part of the brief with the most to gain and the clearest test of
whether it was done right: does the motion tell you something you could not otherwise know? A
count-up on a balance encodes *direction and magnitude*, which the current design throws away
entirely and replaces with a 12px sentence that disappears in four seconds
(`page.tsx:200,202,208,211`, re-verified 2026-09-22). §1.4's walkthrough is the evidence: a
banker Add-Money of $250 moved `$1500` to `$1750` with no highlight, no count, no flash and no
movement.

**Why the mitigation is load-bearing.** §3-E's argument against is serious and was not
overruled, only bounded. This client refetches the whole room on every socket message with no
ordering guard and no coalescing (`page.tsx:189-243`), so two messages in flight can resolve out
of order. A 600 ms count-up is 600 ms during which the screen shows a number the server never
sent, and worse, **motion makes a stale screen look live** — the very smoothness is what reads
as "this is up to date". The blunt refetch as it stands is ugly and instantaneous, and
instantaneous is worth something in a money app. The three rules above are what make the
count-up honest; the codebase has no gate that can enforce them, which is why they appear again
as a named `watch for` in `PLAN.md` rather than as scope prose.

### D6 — Popovers for F1 and F2 only, at `lg` first. Ratified 2026-09-22. (§6.6 → §3-F)

**F1 (property details) and F2 (player at a glance) are in. F3 (quick actions) is out** and is
recorded as parked so it is not re-proposed as a new idea. The popover path is built **at the
`lg` breakpoint first**; the existing drawer below `lg` is left unchanged.

**Defense.** A popover is the right control precisely when the answer is *reference* rather than
*action*, and F1 and F2 are both reference. §1.6 measured F1's case rather than assuming it:
reading one property's rent ladder, walked 2026-09-22, is menu → Bank's Properties → colour group → horizontal
scroll, four interactions deep inside a modal sheet, with two stacked back affordances and the
room entirely hidden behind an opaque scrim throughout. F3 is excluded on its own terms — its
actions all move money or kick players, and a popover is the wrong container for something that
should be deliberate.

**The cost accepted with it, named at ratification rather than discovered mid-build.** Popovers
are a mouse idiom and this is a phone-first product (§1.5 measured a layout built around a 360px
card and a swipe strip). A popover at `lg` with a drawer below it is **two implementations of
one feature**, and that roughly doubles F1's cost. That trade is made here, deliberately, with
the `lg`-first ordering as the hedge: the drawer path already exists and works, so the popover
is additive rather than a replacement.

### D7 — This work splits into two lanes against board row 29. Ratified 2026-09-22. (§6.7 → §5)

- **Lane 1 — theme substrate, type, materials** (§3-A, §3-C, §3-D). **Zero collision with board
  row 29.** Can be planned and built immediately.
- **Lane 2 — motion** (§3-E). Binds to `app/room/[code]/page.tsx`'s message handling, which row
  29 also owns. **Waits on row 29 landing there first.**

**Defense.** `AGENT-PRACTICES.md` §2.1: two items naming the same file do not run at the same
time, whatever their lanes say. Row 29's option B rewrites `handleWebSocketNotification` and
`hooks/use-public-fetch.ts` — the exact seam D5's motion hooks into. The split puts the
cheapest, highest-leverage change (D1's substrate) in flight immediately with no collision, and
lets the motion work be written against whatever `page.tsx` actually looks like after row 29
lands, rather than against a shape that is about to change.

**Live status of what Lane 2 waits on, re-verified 2026-09-22:** board row 29 is `HELD — GATE 1`.
Its scope doc (`docs/incomplete/room-state-sync/SCOPE.md`, written 2026-09-17 by a Fable 5.1
session) is still `Status: SCOPING`, and its five §6 questions are unanswered. **Row 29 has not
been ratified, let alone built.** Lane 2's hold is therefore live and is not a formality.

### D8 — A dated `components/ui/` exception is recorded in `CLAUDE.md`. Ratified 2026-09-22. (§6.8 → §5)

`CLAUDE.md`'s directory map declares `ui/` generated shadcn — "regenerate, do not hand-edit"
(`CLAUDE.md:90`, re-verified 2026-09-22). D1 and D4 both need to edit `drawer.tsx`, `dialog.tsx`,
`sonner.tsx` and `button.tsx`. **Ratified: narrow the rule with a dated exception in `CLAUDE.md`
— "regenerate the primitives; the theming layer over them is ours."** This was done as part of
the same item that recorded it, not left owed.

**Defense.** The alternative — wrapping every primitive — adds a file per component to avoid a
rule whose purpose is already served once the tokens are correct. The rule exists so nobody
hand-patches generated markup and loses it at the next regeneration; a token and className layer
is exactly the part that must not be regenerated away.

**A verified fact that strengthens it, found while checking the citation on 2026-09-22 and not
part of the original audit:** `frontend/components/ui/` holds 17 files, of which only seven have
the generated shadcn shape (`accordion.tsx`, `button.tsx`, `dialog.tsx`, `drawer.tsx`,
`input.tsx`, `slider.tsx`, `sonner.tsx` — Radix/vaul/cva + `cn`). The other ten — including
`button-custom.tsx`, `link.tsx`, `cusotm-link.tsx` (the typo is in the filename on disk),
`reason-select.tsx`, `toasts.tsx`, `fonts.ts`, `helper-funcs.ts`, `install-app-button.tsx`,
`return-to-menu.tsx`, `loader.tsx` — are hand-written house files that the rule as worded
already forbids editing, which nobody has been observing. The exception therefore also corrects
a rule that was over-broad as written.

**R12 note.** This decision contradicts a standing instruction in `CLAUDE.md`, which is why §6.8
asked rather than decided. It was ratified by the owner on 2026-09-22.

### D9 — Two of the three found defects are in scope; the third is not. Ratified 2026-09-22. (§6.9)

- **(a) The mobile toast covering the header — IN.** Folded into Lane 1's materials work.
  §1.5 measured it at 390×844: toast `top: 20, bottom: 72` against a `header` bottom of `64`.
  `offset={76}` exists with a comment saying it is there to clear the `h-16` header
  (`ui/sonner.tsx:15-17`, re-verified 2026-09-22 — the audit cited `:16-19`) but does not apply
  at phone widths, and no `mobileOffset` is set.
- **(c) The two invisible `text-black` strings on `/install` — IN.** `app/install/page.tsx:55`
  and `:70`, both re-verified 2026-09-22. Fixed by D1 as a consequence rather than as a
  drive-by, because they are invisible *for exactly that reason*.
- **(b) The toast icon map's dead `PROPERTY_CHANGE` case and five missing event types — OUT**,
  to board row 1's frontend sweep. Re-verified 2026-09-22: `PROPERTY_CHANGE` appears exactly
  once in the entire repository, at `app/room/[code]/page.tsx:419`, and no Go site sends it.

**Defense.** (a) is a measured defect in the very surface this work redesigns, and (c) is fixed
for free by D1. (b) is excluded because it is a *correctness* question about event names, and
`CLAUDE.md` warns that the websocket contract is the one place the frontend and Go can silently
disagree — the frontend types are the only typed side. Changing an event name from the visual
lane is how that disagreement gets introduced.

**R9.** None of the three was dropped silently; (b) is `raised, not folded in`, and its home is
named.

### D10 — The resync toast string is chosen, and dormant. Ratified 2026-09-22. (§6.10 → R7)

The string is the **plain** register: **"Reconnected. The room is up to date."**

**Defense.** It says the two things a player needs — the connection returned, and what they are
looking at is current — which the terse version ("Reconnected.") leaves them to infer at the one
moment they have reason to doubt it. The warm version ("Back in the room — you're all caught
up.") carries a cheerfulness the moment does not warrant.

**It is dormant, and this is a live condition, not a formality.** The string only has a home if
board row 29's explicit reconnect resync is itself ratified. Re-verified 2026-09-22: row 29's
§6.3 — which recommends skipping the refetch for `PLAYER_LEFT` and already-known `PLAYER_JOINED`
"with the explicit resync on `onopen`" — is **unanswered**, and row 29 is `HELD — GATE 1`.
**Therefore no copy is added by this effort.** If row 29 ratifies the resync, this string is the
one to use and needs no second copy decision; if it does not, D10 expires unused and nothing
about it is re-litigated.

---

### Rules that survive unchanged

Listed in full so no build phase helpfully rewrites one (`AGENT-PRACTICES.md` Stage 3). These
are carried from §2's non-scope list and remain true after ratification:

1. **No backend change of any kind.** No game rule, handler, payload or route. Nothing in
   `backend/` is in scope. No event name in §1.4 is renamed anywhere; `CLAUDE.md`'s standing
   warning about the untyped websocket contract applies and this work does not touch it.
2. **No restructure of the room's data flow.** The refetch-on-every-message design and its two
   known client bugs are board row 29 (`docs/incomplete/room-state-sync/SCOPE.md`), still
   awaiting its own GATE 1 as of 2026-09-22. This work consumes that behaviour as it stands.
3. **Not the dead-code sweep.** `sulpherLight` (defined `fonts.ts:19`, exported `:28`, no other
   reference — re-verified 2026-09-22), the `PROPERTY_CHANGE` case, `p2p-custom-transfer.tsx`
   and `ui/reason-select.tsx` belong to board row 1 and TRIAGE B5. Raised here, not fixed here.
4. **No WebGL, three.js or any real-3D renderer.** Ratified as a constraint before the audit
   began. Depth is CSS only. §3-G parks the one case that might argue otherwise.
5. **Not furlough's palette or its fonts.** Ember `#E5563D`, Amber `#F59E4A` and the Bricolage
   Grotesque / Onest / Geist Mono trio were built for a screen-time app. What is borrowed is the
   discipline, not the values.
6. **No rewrite of the property title deed.** §1.3 found it the best asset in the product. It is
   preserved, and D4 explicitly leaves it as paper.
7. **Not an accessibility audit.** The a11y hacks noticed in passing — `DrawerTitle` rendered
   `text-black` on black at `navbar.tsx:71,88` and `player-card-content.tsx:183`,
   `className={"hidden"}` at `player-card.tsx:117` — are raised, not fixed. A proper pass is its
   own row.
8. **Not the Pay-or-Request flow's missing custom-amount path.** A functional gap belonging to
   the banker-powers triage, not to a facelift.
9. **The five button languages are not collapsed by this work** (§1.7). Doing it as a sweep
   inside a visual pass is exactly the drive-by `CLAUDE.md` forbids. The facelift ships with the
   inconsistency still visible, and it stays on the board.

---

## 1. What exists, verified 2026-09-22

### 1.1 Theming: the tokens exist and nothing can reach them

This is the single largest finding, and it explains most of the others.

| Claim | Verified state | Citation |
|---|---|---|
| The app's ground is hardcoded, not tokenised | `<body className={`bg-black text-white`}>` | `frontend/app/layout.tsx:21` |
| There is no `ThemeProvider` | Zero hits for `ThemeProvider` across `app/`, `components/`, `hooks/`, `lib/`, `types/` | grepped 2026-09-22 |
| The `.dark` token block exists | 25 custom properties defined | `frontend/app/globals.css:33-58` |
| …and is never applied | Live read on `emoney.club`: `html.className` = `""`, `html[data-theme]` absent | walked 2026-09-22 |
| …so the light tokens are what resolve | Live read: `--background` = `0 0% 100%` (white) while `body` computes `rgb(0, 0, 0)` — under `prefers-color-scheme: dark` | walked 2026-09-22 |
| No Tailwind `dark:` variant is used anywhere | Zero hits in `app/` or `components/` | grepped 2026-09-22 |
| `next-themes` is installed and inert | Declared at `package.json:19`; its only consumer is the Toaster | `frontend/components/ui/sonner.tsx:3,9` |
| The tokens are used only by generated primitives | 24 token-class occurrences, all inside `components/ui/*` plus `globals.css:63-66`; **zero** in any feature component | grepped 2026-09-22 |

The consequence is concrete and visible. **Every shadcn surface renders white on a black app.**
Walked 2026-09-22: the banker's Add-Money dialog (`components/players/player-card-content.tsx:124-126`)
renders as a white card floating on the black room, and stays white with
`prefers-color-scheme: dark` emulated. Its Confirm button is `bg-primary`
(`components/ui/button.tsx:13`), which resolves near-black — a filled black button that appears
nowhere else in the product.

Every other drawer avoids this only by hand-overriding the token with a literal: `bg-black
text-white` is pasted onto `DrawerContent` at `navbar.tsx:69`, `player-card.tsx:115`,
`player-card-content.tsx:181,222`, `color-select-drawer.tsx:89` and `reason-select.tsx:51`,
each one cancelling `bg-background` from `components/ui/drawer.tsx:49`.

**The toast is pinned light even when sonner resolves dark.** `components/ui/sonner.tsx:21`
forces `group-[.toaster]:bg-background`. Measured live on 2026-09-22 with
`prefers-color-scheme: dark`: the `[data-sonner-toaster]` element carried `data-theme="dark"`
— sonner resolved the system preference correctly — while the toast itself computed
`background-color: rgb(255, 255, 255)`, `color: rgb(9, 9, 11)`. The class defeats the theme.

**Two strings are invisible.** `app/install/page.tsx:55` and `:70` set `text-black` on a page
whose body is black: the standalone-mode "Home" link and the iOS "tap the share button"
instructions. Walked 2026-09-22 on a desktop browser, `/install` renders as an empty black
screen with only the corner wordmark — the route's three branches (`isStandalone`,
`deferredPrompt`, `isIOS` at `:49,66,69`) were all false, and nothing is rendered for that case.

### 1.2 Type: the fonts are real, the hierarchy is not

**Two of the leads this audit was given are wrong, and the disproofs are recorded here (R5).**

- *"No custom web fonts are loaded (no `next/font` usage in `app/layout.tsx`)."* **Wrong.**
  `frontend/components/ui/fonts.ts:1` imports `Josefin_Sans` and `Sulphur_Point` from
  `next/font/google` and exports five weights. The lead is right only about `layout.tsx`, which
  has no font import — the fonts are loaded, just never made global.
- *"`app/globals.css` holds the default, unmodified shadcn 'zinc' theme tokens."* **Half right.**
  Lines 6-58 are the stock zinc block, unmodified. Lines 70-256 are 186 lines of bespoke CSS the
  default does not ship: the outlined-text `.font` treatment (`:70-85`), the scrollbar rules
  (`:87-141`), and the 3D dice loader (`:147-256`) with its own second `:root` block at `:168-172`
  declaring `--color-theme: #ffff00`.

| Claim | Verified state | Citation |
|---|---|---|
| The default face is the system stack | Live read: `body` computes `ui-sans-serif, system-ui, sans-serif, …` | walked 2026-09-22 |
| Faces are opted into per element | ~70 `${josephinX.className}` / `${sulpherX.className}` interpolations across 30 files | grepped 2026-09-22 |
| One declared weight is dead | `sulpherLight` is defined at `fonts.ts:19` and exported at `:28`; no other file references it | grepped 2026-09-22 |
| There is no monospace or tabular face | Live read of the card balance `$1875`: family `"Josefin Sans"`, `font-variant-numeric: normal` | walked 2026-09-22 |
| Money is formatted two different ways | Card: `${player?.balance \|\| 0}` → `$1875`. Offer panel: `balance.toLocaleString()` → `1,500` | `player-card-content.tsx:161`; `make-offer/amount.tsx:61` |
| The size scale is ad-hoc | `text-[.5rem]`, `text-[1.4rem]`, `text-xs` … `text-3xl`, each chosen at its call site | e.g. `property/cards/common-card.tsx:88`; `app/page.tsx:53`; `page.tsx:202` |

Josefin Sans is a geometric humanist face with a very low x-height and single-storey lowercase
`a` and `g`. It carries the wordmark and the property cards well. It is doing all four jobs at
once — display, body, UI label and numerals — and the numerals are the job it does worst:
proportional digits mean the balance reflows as it changes, which is exactly the moment the eye
is on it.

### 1.3 Materials: there is no glass, and depth is drawn with borders

| Claim | Verified state | Citation |
|---|---|---|
| Nothing is frosted | Zero `backdrop-blur`, `backdrop-filter` or equivalent anywhere | grepped 2026-09-22 |
| Elevation is unsystematic | 14 shadow utilities total across the tree — `shadow-sm`/`md`/`lg`/`xl`, no scale, no shared token | grepped 2026-09-22 |
| Depth is borders on black | The player card is a white rectangle inside two nested black borders | `components/players/player-card.tsx:91-93` |
| The drawer scrim is opaque-ish black | `bg-black/80` over a black page | `components/ui/drawer.tsx:31` |
| Real 3D already exists, in one place | The dice loader: `@keyframes roll` rotating on three axes, `perspective: 500px`, `transform-style: preserve-3d` | `globals.css:156-166, 178, 205`; `components/loaders/dice.tsx` |

The scrim is the sharpest material problem. `bg-black/80` over a black ground does not make the
room recede — it deletes it. Walked 2026-09-22: opening any drawer leaves the space above it
indistinguishable from the drawer's own background, so the drawer reads as a full-screen page,
not as a sheet over a room. This is the one place where the requested glass treatment has an
unambiguous job.

The asset worth protecting is the **property title deed** (`components/property/cards/`):
`card-container.tsx:12` builds a `bg-white border border-black` card at
`.property-card-aspect-ratio` (69/100, `globals.css:143-145`), and `common-card.tsx` lays in the
colour band, the rent ladder, mortgage value and house costs. Walked 2026-09-22 — it is a
faithful, legible reproduction and it is the best-looking thing in the product.

### 1.4 Motion: almost none, and none of it is tied to state

| Claim | Verified state | Citation |
|---|---|---|
| The entire animation surface | `tailwindcss-animate` driving accordion open/close and the dialog's `animate-in`/`animate-out` — and nothing else | `package.json:22`; `ui/accordion.tsx:31,37,49`; `ui/dialog.tsx:24,41` |
| Plus a handful of colour fades | 14 × `transition-colors`, 3 × `transition-transform`, 1 × `duration-200`, 2 × `duration-300` | counted 2026-09-22 |
| No motion library is installed | No `framer-motion`, no `motion` | `package.json`, grepped 2026-09-22 |
| No popover primitive is installed | Radix deps are accordion, dialog, slider, slot only | `package.json:12-15` |
| **Zero reduced-motion handling** | No `prefers-reduced-motion`, `motion-safe` or `motion-reduce` anywhere | grepped 2026-09-22 |
| The only bespoke motion is a hover | `text-shadow: 4px 4px 0 #fff; transform: translate(-2px,-2px)` over 250ms | `globals.css:77, 82-85` |
| The best existing motion is a toggle | A sliding white pill, `transition-transform duration-300` | `players/pay-req-toggle-switch.tsx:24`; same shape at `navbar/free-parking.tsx:45` |

**The finding that matters most: every state change in this game is a toast and a silent number
swap.** `app/room/[code]/page.tsx:189-243` answers every inbound socket frame the same way —
toast the server's sentence, then `refetchOffers()`, then `refetchPlayers()` unless the frame is
one of the three private offer types, then `refetchProperties()` for three more. Nothing on the
card acknowledges anything.

Walked 2026-09-22 in room `UIAUD2`: a banker Add-Money of $250 took the balance from `$1500` to
`$1750`. The digits changed between frames. There was no highlight, no count, no flash, no
movement — the only evidence anything happened was a 12px sentence in a white box at the top of
the screen, gone in four seconds (`duration: 4000` at `page.tsx:200,208`). This is the whole of
the product's feedback for every transfer, purchase, mortgage, rent payment and trade.

Three defects sit inside that one mechanism, all verified 2026-09-22:

1. **The toast is 12px.** Live read of the title element: `font-size: 12px`. Set by
   `text-xs` at `page.tsx:202` and `:211`.
2. **The icon map has a dead case and five gaps.** `getIconForType` (`page.tsx:411-440`) has a
   `case "PROPERTY_CHANGE"` at `:419`; grepping the whole repository for that literal on
   2026-09-22 returns **that line and nothing else** — no Go site sends it. Meanwhile
   `FREE_PARKING`, `PURCHASE_PROPERTY`, `AUCTION_STARTED`, `BID_PLACED` and
   `AUCTION_LOT_CLOSED` have no case and fall through to `default: "ℹ️"` (`:437-438`). Walked:
   adding $5 to Free Parking toasted `ℹ️ Rival added $5 to Free Parking`.
3. **Toast position is inconsistent.** Every room toast sets `position: "top-center"`
   (`page.tsx:201,210`). `navbar.tsx:186`'s `toast.error("Failed to copy room code")` sets
   nothing and takes sonner's default. Measured at 390×844 on 2026-09-22: that toast landed at
   `top: 771, bottom: 824` — the bottom of the screen, while every other message appears at the
   top.

### 1.5 The mobile layout, measured

The room is a phone-first design that has one measured hole in it.

| Claim | Verified state | Citation |
|---|---|---|
| The player card is a fixed 360px | `w-[360px] … aspect-[3/4]` | `players/player-card.tsx:91` |
| At 390px, the second player is **0 pixels visible** | Live measurement: card rects `left 15 / right 375` and `left 391 / right 751`; strip `scrollWidth 768` vs `clientWidth 390` | walked 2026-09-22 |
| …and the scrollbar is hidden on touch | `.hide-scrollbar` sets `scrollbar-width: none`; given back only at `@media (pointer: fine)` | `globals.css:87-107`; live read `scrollbar-width: none` |
| The desktop case was already fixed | Wraps to a centred grid at `lg` | `room/room.client.tsx:120-123` and its comment at `:116-119` |
| The mobile toast covers the header | Live measurement at 390×844: toast `top: 20, bottom: 72`; `header` bottom `64`; `overlapsHeader: true` | walked 2026-09-22 |
| …despite an offset written to prevent exactly that | `offset={76}`, with a comment saying it exists to clear the `h-16` header | `ui/sonner.tsx:16-19` |

The offset does not apply at phone widths — sonner switches to its own mobile offset below its
mobile breakpoint, and `sonner.tsx` sets no `mobileOffset`. So the fix that was written for the
desktop case is absent in the case that matters more, and the room name and the menu button are
covered every time anything happens.

**Drawers are tall and mostly empty.** Declared heights: `h-[80vh]` (`navbar.tsx:69`),
`h-[90vh]` (`player-card-content.tsx:222`), `min-h-[90vh]` (`player-card.tsx:115`), `h-[600px]`
(`player-card-content.tsx:181`), `h-[70vh]` (`reason-select.tsx:51`), `max-h-[80vh]`
(`color-select-drawer.tsx:89`), plus `.drawer-content { height: 75vh }` at `globals.css:110-113`.
Walked at 390×844: the Pay-or-Request drawer rendered a toggle and the single sentence "Auditor
doesn't have any properties to pay rent for" above roughly 700px of empty black. The menu drawer
renders four rows, then a ~200px void, then the Danger Zone pinned at `absolute bottom-5`
(`navbar.tsx:197`).

### 1.6 Navigation: the drawer navigates inside itself

`components/navbar/navbar.tsx:52-54` holds three booleans — `showProperties`, `showFreeParking`,
`showEvents` — and `:83-234` swaps the drawer's entire body between them, with a manual
`ReturnToMenu` button rendered on top (`:74-82`). There is no transition on the swap; the content
is replaced between frames.

Walked 2026-09-22, the cost of that: reading one property's rent ladder is **menu → Bank's
Properties → colour group → horizontal scroll to the card**, four interactions deep inside a
modal sheet, with two stacked back affordances (the `ReturnToMenu` bar and an inner chevron) and
the room entirely hidden behind an opaque scrim the whole time. This is the concrete case for
popovers, and it was found rather than assumed.

The same shape appears on the player card: `Properties` opens a second drawer from inside the
first (`player-card-content.tsx:170-192`), and the name bar opens a third
(`player-card.tsx:94-140`).

### 1.7 Five button languages

All hand-rolled, none shared, verified 2026-09-22:

1. **Yellow-bordered, outlined text** — `ui/link.tsx:20` and `ui/button-custom.tsx:12`, both
   pulling `.font` (`globals.css:70-78`: `color: transparent`, `-webkit-text-stroke: 1px #ffff00`,
   `text-shadow: 0 0 0 #fff`). This is the only thing in the product with a designed identity.
   Home, create and join only.
2. **White-bordered pill** — `make-offer/make-offer.tsx:334`, `offers-inbox.tsx:125`.
3. **shadcn filled `bg-primary`** — `ui/button.tsx:13`, imported at exactly two sites
   (`player-card-content.tsx:17`, `remove-player.tsx:10`). Renders black-on-white in the banker
   dialog.
4. **White pill with coloured text** — `navbar/free-parking.tsx:80-81`, green for Add, blue for
   Collect.
5. **Sliding-pill segmented toggle** — `pay-req-toggle-switch.tsx:20-24`,
   `free-parking.tsx:41-45`.

### 1.8 The newest surface is the best one

The trades work (`c14faa5`) produced `components/players/make-offer/`. Walked 2026-09-22: "Make
an Offer" gives a two-sided structure ("I'm offering" / "I Would Like", each with Cash and
Properties), a note field with real explanatory copy and a `0/280` counter
(`make-offer.tsx:323`), and a disabled Send button that **says why** — "Add cash or a property on
either side to send." The percent chips (`amount.tsx:64-92`) colour red for offer, green for
request, and the comment at `:79-81` explains the flooring.

That is the standard the rest of the product should be levelled up to, and it is house
precedent rather than an import. Its own gaps: the chips are `border-white` until selected with
no hover state, the layout is top-aligned in a `min-h-[90vh]` sheet, and there is no motion
between the compose step and the amount step — walked 2026-09-22, the panel is replaced
instantly, same as the navbar.

### 1.9 What was not verified

No auction state was reached (§ preamble). No room with more than two players was observed, so
the `lg` wrap grid was seen with two cards and not with six. No performance measurement of any
kind was taken — no bundle size, no frame timing, no Lighthouse run — so every weight claim in
§3 and §5 is an argument, not a measurement. The `prefers-reduced-motion` absence is a grep, not
an observed failure.

---

## 2. What this is / what this is not

**This is:** a visual and motion system for the existing surfaces — a reachable theme, a type
hierarchy with a numeral face, a material and elevation scale, motion bound to the websocket
events that already arrive, and popovers where §1.6 showed a drawer is doing a tooltip's job.

**This is not**, whoever asks mid-build:

- **Not a change to any game rule, handler, payload or route.** Nothing in `backend/` is in
  scope. The event names in §1.4 are renamed nowhere; `CLAUDE.md`'s standing warning about the
  untyped websocket contract applies and this work does not touch it.
- **Not a restructure of the room's data flow.** The refetch-on-every-message design and its two
  known client bugs are **board row 29**, scoped at `docs/incomplete/room-state-sync/SCOPE.md`
  and awaiting its own GATE 1. This work consumes that behaviour as it stands. Where a motion
  option depends on what row 29 decides, §5 says so.
- **Not the dead-code sweep.** `sulpherLight` (§1.2), the `PROPERTY_CHANGE` case (§1.4) and
  `p2p-custom-transfer.tsx` / `ui/reason-select.tsx` belong to the existing **board row 1**
  frontend sweep and to TRIAGE B5. They are recorded here because they were found here; they
  are not folded in. (R9: raised, not fixed.)
- **Not WebGL, three.js or any real-3D renderer.** Ratified as a constraint before this audit
  began. Depth is CSS only. §3-E parks the one case that might argue otherwise.
- **Not furlough's palette or its fonts.** Ember `#E5563D`, Amber `#F59E4A` and the Bricolage
  Grotesque / Onest / Geist Mono trio were built for a screen-time app. What is borrowed is the
  discipline; §3-B proposes e-money's own.
- **Not a rewrite of the property title deed.** §1.3 found it to be the best asset in the
  product. It is preserved.
- **Not an accessibility audit.** Several a11y hacks were noticed in passing — `DrawerTitle`
  rendered `text-black` on black at `navbar.tsx:71,88` and `player-card-content.tsx:183`,
  `className={"hidden"}` at `player-card.tsx:117` — and they are raised, not fixed. A proper
  pass is its own row.
- **Not the Pay-or-Request flow's missing custom-amount path.** Walked 2026-09-22: with neither
  player owning property, the Send side offers nothing at all. That is a functional gap and
  belongs to the banker-powers triage, not to a facelift.

---

## 3. Options

Each carries its defense and the strongest argument against it, so the losing argument stays
written down. **Recommendations are marked.** The answer to any of them may be no.

### A — The theme substrate: make the tokens reachable  ← **recommended, and first**

Put the ground back into `globals.css` as tokens, redefine the `.dark` block to e-money's actual
values, and have `layout.tsx` declare the theme rather than hardcode `bg-black text-white`.
Every literal `bg-black` pasted onto a `DrawerContent` (§1.1) then comes out, and the shadcn
dialog stops being white.

**Defense.** Nothing else in this document can be built on top of the state §1.1 measured. A
glass material is a translucency over a *ground*; with the ground hardcoded on `<body>` and the
tokens resolving to their light values, there is no ground to be translucent over and every new
component inherits the same bug. This is also the cheapest item here by a wide margin: it is one
CSS file, one line of `layout.tsx`, and the deletion of six pasted literals. It fixes two
invisible strings (`install/page.tsx:55,70`) as a side effect rather than as a drive-by, because
they are invisible *for this reason*.

**Strongest argument against.** It is invisible when it works. A whole change that a walkthrough
cannot distinguish from no change is a change with no proof, and §5 notes there is no frontend
test runner to catch a regression in it. It also touches every drawer in the app at once, which
is the widest blast radius of anything proposed here, for a benefit the owner will not see. The
counter to that counter: the blast radius is *deletions* of overrides that are cancelling the
token being fixed, and each one is verifiable by eye in the same walkthrough.

**Open sub-question, for §6:** whether to keep a light theme at all. Answering "dark only" makes
this smaller — one token set, `.dark` deleted rather than populated — and lets the design commit
to a single ground. Answering "both" keeps `next-themes` meaningful and needs a toggle placed
somewhere.

### B — The colour direction: pick e-money's own accent

Three real candidates, all keeping a near-black ground.

- **B1 — Keep and discipline the existing yellow.** `#ffff00` already exists twice
  (`globals.css:75, 170`) and is the wordmark, the outlined buttons and the dice. Give it a
  proper ramp (it is unusable as a text colour at small sizes at full chroma), reserve it for
  identity and primary action, and let it stop appearing as a raw `!important` (`globals.css:79-81`).
  *Defense:* it is already the brand; nobody has to be told what the app is. It is also the most
  honest reading of "restraint" — the accent exists, it is just undisciplined.
  *Against:* pure yellow on black is the highest-contrast pair available and reads as
  hazard-tape, not money; it has no usable dark-on-light inverse, which the white property cards
  need; and at `text-xs` it fails legibility well before it fails contrast maths.
- **B2 — Money green as the accent, yellow demoted to the wordmark.** A single deep green for
  affirmative money movement, with the existing red/green semantics from `amount.tsx:71-74` and
  `free-parking.tsx:81` promoted into real tokens.
  *Defense:* the product is about money moving in two directions, and it already reaches for red
  and green ad hoc in three places. Making that the system means the accent *carries meaning*
  rather than decorating — which is the Raycast/furlough discipline stated precisely.
  *Against:* green/red as the primary semantic axis is the one pairing that fails for the most
  common colour-vision deficiency, and this app puts money direction at its centre. It would
  need a redundant encoding (sign, arrow, position) to be honest, which is extra work this
  option's defense does not price.
- **B3 — A neutral chrome with player colour as the only chroma.** The UI goes greyscale on
  near-black; the twenty-nine player swatches (walked 2026-09-22) and the property group bands
  become the *only* saturated colour on screen.
  *Defense:* this is the most disciplined answer and the one most like the reference points.
  The screen's colour then always means something — whose money, which group — and never means
  "this is a button". It also solves a problem §1 did not look for: with six players, chrome
  chroma and player chroma compete.
  *Against:* it throws away the only identity the product has. The yellow-on-black wordmark is
  recognisable and the outlined-text treatment is genuinely distinctive; a greyscale chrome
  risks the "templated shadcn dark app" look that this whole effort exists to escape.

**Recommendation: B3 for the chrome, with B1's yellow retained as the identity accent** — the
wordmark, the primary call to action on `/`, `/create`, `/join`, and nothing else. That keeps
the distinctive thing, stops it being used as a generic border colour, and leaves player colour
as the meaningful chroma inside the room. B2's red/green stays as a *semantic* pair for money
direction only, with a redundant sign glyph.

### C — Type: a three-face hierarchy with numerals separated  ← **recommended**

Set a face on `<body>` so the system stack stops being the default (§1.2), and split the four
jobs Josefin is doing:

- **Display / headings** — a face with real character, carrying the wordmark, room name and
  drawer titles.
- **Body / UI** — a clean, high-legibility face at small sizes, for labels, help copy, the event
  log and toast text.
- **Numerals** — a monospaced or tabular-figure face, used for every dollar amount, the room
  code, rent ladders and counts.

Candidates are a §6 question, not a unilateral pick. The disciplined options: keep **Josefin
Sans** as display only (it is already the brand voice and it is good at large sizes), pair it
with a workhorse UI face, and add a mono. Alternatively retire Josefin for something with more
presence at display sizes and keep the change to one decision.

**Defense.** The numeral half is not a taste call and is the strongest argument in this
document. §1.2 measured `font-variant-numeric: normal` on the balance. Every amount in this game
is a number that changes while someone is looking at it, and proportional digits mean it
*reflows* as it changes — the `1` is narrower than the `8`, so `$1500 → $1750` shifts the
glyphs. Any motion applied to that number in §3-E sits on top of a layout that is already
moving, and the two are indistinguishable to the eye. `font-variant-numeric: tabular-nums` on
the existing face is the floor; a real mono is the intent. It also fixes §1.2's two money
formats by giving them one formatter to share.

**Strongest argument against.** Three families is three font payloads on a page that has to stay
snappy on a phone on a table in a pub, and §1.9 says no weight measurement was taken. The
cheaper version of this option — keep Josefin for everything, add `tabular-nums`, and add one
mono for amounts only — captures most of the benefit for one extra face. The full three-face
split should be defended on how the app *reads*, not smuggled in behind the numerals argument,
which is the only part of it that is load-bearing.

### D — Materials: glass where it has a job, borders where it does not  ← **recommended, narrowly**

Four surfaces, and only four:

1. **The drawer scrim.** Replace `bg-black/80` (`drawer.tsx:31`) with a scrim that actually
   recedes: a lower-opacity ground plus a real `backdrop-filter: blur()`. §1.3 measured why —
   opaque black over black deletes the room instead of layering over it. This is the single
   highest-value material change and the only one where the current state is a defect rather
   than a preference.
2. **The sticky header** (`room.client.tsx:97`, `bg-black`) — translucent with a blur, so the
   card strip visibly passes under it.
3. **Popover and menu surfaces** — §3-F's new ones, which need to read as floating above the
   room rather than replacing it.
4. **An elevation scale** — three or four steps as tokens, replacing the 14 unsystematic shadow
   utilities (§1.3). On a near-black ground, elevation is carried mostly by a hairline top
   highlight and a wide soft shadow, not by the stock Tailwind shadows, which are tuned for
   white pages and are nearly invisible here.

**Explicitly not glassed:** the player card and the property title deed. They are *paper* — the
metaphor is a physical card on a table, and it is the product's best idea. Frosting them would
be decoration in exactly the sense this effort is meant to avoid.

**Defense.** This is the narrow reading of the brief and the one that survives review: glass is
applied where there is something behind it worth seeing, and withheld everywhere else. Four
surfaces is a system; twelve would be a style.

**Strongest argument against.** `backdrop-filter` is the most expensive thing proposed in this
document, it is a known compositor cost on low-end Android, and this is an app whose whole job
is to be instant during a board game. A large blurred scrim animating in on every drawer open,
on a phone, is precisely the case where it stutters — and §1.9 confirms no performance
measurement exists to say whether that is a real risk here or a theoretical one. §4 sets a
blur radius dial low for this reason, and §5 carries the hazard.

### E — Motion: bind it to the events that already arrive  ← **recommended, and the reason to add `motion`**

§1.4 established the gap: sixteen websocket message types reach the browser and every one of
them is rendered as a toast and a silent number swap. The proposal is that motion answers
**state**, in four places and no others:

1. **Cash changing.** When a player's balance changes, the number counts from old to new and the
   card carries a brief directional tint — credit one way, debit the other. This is the moment
   the entire product exists for and it has no visual treatment at all.
2. **A deed changing hands.** On `PURCHASE_PROPERTY` and `OFFER_ACCEPTED`, the property count on
   the affected cards animates and the card the deed landed on acknowledges it.
3. **An offer arriving.** The `waitingOnMe` badge (`player-card.tsx:105-111`) appears
   between frames, with no transition. It should arrive — and it is the one place a *persistent* signal beats a
   four-second toast.
4. **Drawer and panel transitions.** The navbar's three-boolean body swap (§1.6) and
   make-offer's step change (§1.8) get a real transition, so a replaced panel reads as
   navigation rather than as a repaint.

`motion` (the `framer-motion` successor, pre-approved to propose) earns its place on items 1–3:
`layout` animations and enter/exit on lists are the parts that are genuinely unpleasant to
hand-roll, and a count-up needs a driver. Items 4 and everything else stays CSS.

**`prefers-reduced-motion` is not optional and is part of this option, not a follow-up.** §1.4
found zero handling. The rule: reduced motion removes *movement* — no count-up, no slide, no
parallax — and keeps *state*: the number changes instantly, the tint still appears as a
non-animated state, the badge still renders. Every animation added lands behind that gate from
the first commit, both as a Tailwind `motion-safe:` variant and as `motion`'s own
`useReducedMotion`. This is also the honest answer to the item below.

**Defense.** This is the part of the brief with the most to gain and the clearest test of whether
it was done right: does the motion tell you something you could not otherwise know? A count-up
on a balance is not decoration — it encodes *direction and magnitude*, which the current design
throws away entirely and replaces with a 12px sentence that disappears.

**Strongest argument against, and it is serious.** *Animating a value that arrives over a
websocket can lie.* §1.4 and row 29's audit both establish that this client applies state by
refetching the whole room on every message, with no ordering guard and no coalescing — so two
messages in flight can resolve out of order. A 600ms count-up is 600ms during which the screen
shows a number that is not the server's number, and if a second refetch lands mid-count the
animation either fights it or is cancelled and the player sees a value that never existed.
Worse, motion makes a *stale* screen look like a *live* one: the very smoothness is the thing
that reads as "this is up to date". The blunt refetch as it stands is ugly and instantaneous,
and instantaneous is a property worth something in a money app. Mitigation — cap the duration hard
(§4), always animate *to* the authoritative value rather than through a computed sequence, and
make a new value arriving mid-animation snap rather than re-ease — but the mitigation is a
discipline the codebase has no gate for, and §5 carries it as the top hazard.

### F — Popovers: three earned locations, found not assumed  ← **recommended for F1, F2**

`@radix-ui/react-popover` (shadcn's Popover, pre-approved to propose). The candidates §1.6 and
§1.7 actually turned up:

- **F1 — Property details.** The strongest case, and the one Zach described. §1.6 measured the
  current cost: four interactions inside an opaque modal to read a rent ladder. A property's
  name anywhere — the bank list, a player's holdings, an offer's contents — opens a popover
  showing the deed's terms in place, with the room still visible.
- **F2 — Player at a glance.** Balance, property count, banker status and pending-offer count
  without opening the card's drawer stack. Useful the moment there are more than two players,
  which is every real game.
- **F3 — Quick actions.** A small action menu on a player. **Weaker, and not recommended:** the
  actions are all consequential (money moves, kicks), and a popover is the wrong container for
  something that should be deliberate. Recorded so it is not re-proposed as new.

**Defense.** A popover is the right control precisely when the answer is *reference* rather than
*action* — you want to know something and then carry on. F1 and F2 are both reference. It also
directly serves the "restraint" half of the brief by removing modal sheets rather than
decorating them.

**Strongest argument against.** Popovers are a mouse idiom. This is a phone-first product — §1.5
measured a layout built around a 360px card and a swipe strip — and on a touch screen a popover
is a small modal with worse ergonomics than the drawer it replaced: harder to dismiss, easier to
mis-tap, and it cannot be dragged away. The honest version of F1 is **a popover at `lg` and a
drawer below it**, which is two implementations of one feature and roughly doubles its cost.
That trade should be made deliberately at GATE 1, not discovered during the build.

### G — Parked: real 3D for the property deed

A deed that tilts on hover with genuine perspective, or flips to show its reverse. **Parked, with
its cost, not scoped** — the ratified constraint is CSS-only for this pass, and CSS
`perspective` + `rotate3d` already reaches most of it (the dice at `globals.css:156-166` proves
the technique is in-house). A WebGL renderer for one card is a new dependency, a new bundle, and
a new class of device failure, for an effect that is decoration by the brief's own test.
Recorded here so it is not re-proposed as a new idea.

---

## 4. Dials

Every number this design leaves open, with a recommended default. Each is destined for one named
token or constant, not a literal typed twice.

| Dial | Recommended default | Why |
|---|---|---|
| Scrim opacity | `black / 55%` | Enough to subdue the room, not enough to delete it — the defect §1.3 measured is that `/80` on black leaves nothing to see |
| Scrim blur radius | **12px** | Reads clearly as glass; low enough to stay off the expensive end of the compositor cost §3-D warns about. Raise only after a device measurement |
| Header blur radius | 8px | The header is thin; more blur reads as smear, not depth |
| Elevation steps | **4** — flat, raised, overlay, modal | Three is too few to separate a popover from a drawer; six invites arbitrary picks, which is the state §1.3 found |
| Balance count-up duration | **450 ms**, hard cap 600 ms | Long enough to read direction, short enough that a second socket frame rarely lands inside it (§3-E's argument against) |
| Balance tint hold | 900 ms, then fade over 300 ms | Outlives the count so the card still says "this one changed" after the number settles |
| Standard transition | **180 ms, ease-out** | Matches the existing `duration-200` (`accordion.tsx:37`) rather than introducing a third timing |
| Panel/drawer transition | 240 ms | Slower than a control, faster than the 300 ms the pill toggle uses (`pay-req-toggle-switch.tsx:24`) |
| Reduced-motion behaviour | **State kept, movement removed** | §3-E. Not "animations off" — the tint and the badge are information |
| Toast font size | **14 px** (`text-sm`) | 12px is too small for the app's only feedback channel (§1.4). Anything larger crowds the 390px width |
| Toast mobile offset | **`mobileOffset` set to clear the `h-16` header** | §1.5 measured the current overlap; `offset={76}` alone does not apply at phone widths |
| Toast duration | Keep **4000 ms** | Already deliberate at `page.tsx:200,208`; changing it is a separate argument |
| Max simultaneous toasts | **3** | A burst of socket frames stacks without limit; `visibleToasts` is unset at `ui/sonner.tsx` as of 2026-09-22 |
| Popover breakpoint | `lg` and above; drawer below | §3-F's argument against. Flipping this to "popover everywhere" is a real option, not a detail |
| Card corner radius | 8px chrome / keep deed square | The deed is paper and paper has square corners |
| Numeral face | Tabular figures minimum; mono preferred | §3-C. `tabular-nums` on the existing face is the floor, not the goal |

---

## 5. Hazards this work walks into

- **Motion can make a stale screen look live.** The top hazard, stated in full at §3-E. This
  client refetches the whole room on every message with no ordering guard and no coalescing
  (`page.tsx:189-243`; `docs/incomplete/room-state-sync/SCOPE.md` §1.2). Any animation that
  interpolates *toward* a value is showing a number the server never sent for the duration of
  the animation. **Rule for the build: animate to the authoritative value, never through a
  computed one, and snap rather than re-ease when a new value lands mid-flight.**
- **This work and board row 29 both own `app/room/[code]/page.tsx`.** Row 29 is scoped and
  awaiting GATE 1; its option B rewrites `handleWebSocketNotification` and
  `hooks/use-public-fetch.ts` — the exact seam §3-E's motion hooks into. Per
  `AGENT-PRACTICES.md` §2.1, two items naming the same file do not run at the same time,
  whatever their lanes say. **Sequencing is a GATE 1 question (§6.7), not a build-time
  discovery.**
- **Layout shift during a live trade.** Cards are a fixed `w-[360px] aspect-[3/4]`
  (`player-card.tsx:91`) but their *contents* are conditional — the Cash King tag, "Pay or
  Request", the offer badge, "No longer in the game" all appear and disappear on state
  (`player-card-content.tsx:193-239`). Animating entry on those without reserving their space
  will push the rest of the card while someone is reading a balance or tapping Accept. Reserve
  the space; animate opacity and transform only.
- **Every refetch replaces the data object whole**, so every derived array is a new identity on
  every message. A `motion` list keyed on array index will re-run its enter animation for every
  player on every frame of any kind. **Key on `player.id` and `property.id`, never on index** —
  and note that `navbar.tsx:113` keys the event history on `index` as of 2026-09-22.
- **`backdrop-filter` is the expensive thing here** (§3-D), on an app that must stay instant on a
  mid-range phone, and §1.9 confirms no performance baseline exists. Measure on a real device
  before raising the blur dial.
- **Added JS weight lands on the room page**, which is the one page that must stay snappy. Both
  proposed dependencies are additive. Nothing in this repo measures bundle size and there is no
  CI to notice it growing (`CLAUDE.md`, *Stack*).
- **There is no frontend test runner and no CI** (`CLAUDE.md`, *The gates that lie*). Every claim
  this work makes is proved by a walkthrough or not at all, and §3-A is explicitly a change a
  walkthrough cannot see. Its done-when has to be written as a proof a green gate cannot supply
  (`AGENT-PRACTICES.md` Stage 4).
- **`bun run build` is green while baking `localhost:8080` into the bundle**, with no tell since
  `bec2350` (`CLAUDE.md`). Verify visual work against the deployed site or through
  `scripts/emoney dev`, never against a local build alone.
- **`components/ui/` is generated shadcn** (`CLAUDE.md`, *Directory map*: "regenerate, do not
  hand-edit"). §3-A and §3-D both change `drawer.tsx`, `dialog.tsx`, `sonner.tsx` and
  `button.tsx`. Either the no-hand-edit rule gets a recorded exception for the theming layer, or
  the changes go in wrappers outside `ui/`. **This is a real conflict with a standing rule and
  is §6.8.**
- **Five button languages will not collapse into one without touching many files** (§1.7).
  Doing it as a sweep inside this work risks exactly the drive-by that `CLAUDE.md` forbids;
  doing it as its own row means the facelift ships with the inconsistency still visible.
- **The app has never rendered in a light theme.** If §6.1 answers "both themes", every surface
  in §1 has to be checked twice, and roughly half of them carry hardcoded `bg-black` / `text-white`
  / `text-black` literals that will each need a decision. That answer roughly doubles this work.
- **`next dev` rewrites `frontend/AGENTS.md`** on every run (its own header says so). Do not
  treat that as this session's change.

---

## 6. Open questions — the GATE 1 batch  ·  **CLOSED 2026-09-22**

**This section is closed. All ten were answered on 2026-09-22, every one taking the recommendation; the answers are §0 as `D1`–`D10`.** It is kept as written rather than deleted, because the question a decision answered is part of the record of why it came out that way (8.4). Do not re-answer anything here — a change to a ratified decision is a dated supersession in §0 (R8).

Ten, batched per R6. Every one carries a marked recommendation; the answer to any may be no.

**6.1 — One theme or two?** Dark-only makes §3-A small and lets the design commit to one
ground; it also makes `next-themes` dead weight to be removed. Both themes keeps it meaningful,
needs a toggle placed somewhere, and roughly doubles the surface to check (§5).
**Recommended: dark only**, with the `.dark` block deleted rather than populated and the single
token set defined on `:root`. E-money is played in a room with other people around a table; a
light theme is a feature nobody has asked for, and no code has ever rendered one.

**6.2 — Ratify the accent direction (§3-B).** **Recommended: B3 + B1** — greyscale chrome on a
near-black ground, player colour and property-group colour as the only chroma inside the room,
and the existing yellow kept as identity only (wordmark and the primary action on `/`, `/create`,
`/join`). Red/green reserved strictly for money direction, always with a redundant sign glyph.
Say plainly if the yellow is precious beyond the wordmark — it is the one thing here with an
established identity and B3 demotes it.

**6.3 — Ratify the type direction (§3-C), and how far.** Three sub-answers:
(a) **the numerals** — tabular figures minimum, a real mono preferred. **Recommended: yes**, this
is the load-bearing half and §1.2 measured the defect;
(b) **the body face** — a dedicated UI face set on `<body>`, or keep Josefin everywhere.
**Recommended: a dedicated body face**, because Josefin's low x-height is what makes the 12px
toast and the event log hard to read;
(c) **the display face** — keep Josefin as display, or replace it. **Recommended: keep it.** It
is the voice of the wordmark and the property cards and it is good at size.

**6.4 — Confirm the four glass surfaces, and only four (§3-D).** Scrim, header, popovers,
elevation scale — with the player card and the property deed explicitly left as paper.
**Recommended: yes.** If the deed should get a material treatment after all, say so now; it is
the one place where reversing later means redoing the work.

**6.5 — Confirm the four motion moments, and no fifth (§3-E).** Cash changing, a deed changing
hands, an offer arriving, panel transitions. **Recommended: yes**, with the hard rule that
reduced motion removes movement and keeps state. The thing to react to is whether **cash
count-up** is wanted at all given §3-E's argument against it — a tint with an instant number is
the conservative version and is a legitimate answer.

**6.6 — Ratify popovers for F1 and F2, and the breakpoint (§3-F).** **Recommended: yes for F1
(property details) and F2 (player at a glance), no for F3 (quick actions).** The real question is
the breakpoint dial: popover at `lg` with a drawer below is two implementations of one feature.
**Recommended: build the popover path first at `lg`** and keep the existing drawer below it
unchanged, rather than building both at once.

**6.7 — Sequencing against board row 29.** Both own `app/room/[code]/page.tsx` (§5). Options:
this work waits for row 29; row 29 waits for this; or this work is split so the parts that do
not touch `page.tsx` (§3-A theming, §3-C type, §3-D materials) go first in their own lane and
the motion work (§3-E) waits. **Recommended: the split** — it puts the cheapest, highest-leverage
change (§3-A) in flight immediately with no collision, and lets the motion work be written
against whatever `page.tsx` looks like after row 29 lands.

**6.8 — `components/ui/` is declared hand-edit-forbidden, and this work needs to edit it
(§5).** `drawer.tsx`, `dialog.tsx`, `sonner.tsx` and `button.tsx` all need changes.
**Recommended: record a dated exception in `CLAUDE.md`** narrowing the rule to "regenerate the
primitives; the theming layer over them is ours" — because the alternative, wrapping every
primitive, adds a file per component to avoid a rule whose purpose is already served once the
tokens are correct. This contradicts a standing instruction, so per R12 it is asked rather than
decided.

**6.9 — Confirm the three found defects are in scope or out (R9 — none is dropped
silently).** All three were found by this audit, all three are visual, none is a drive-by if
ratified here:
(a) the mobile toast covering the header (§1.5, measured);
(b) the toast icon map's dead `PROPERTY_CHANGE` case and five missing event types (§1.4);
(c) the two invisible `text-black` strings on `/install` (§1.1).
**Recommended: (a) and (c) in** — (a) is a measured defect in the surface this work is redesigning,
and (c) is fixed for free by §3-A. **(b) out**, to board row 1's sweep: it is a correctness
question about event names, which `CLAUDE.md` warns is the one place the frontend and Go can
silently disagree.

**6.10 — Toast copy for the reduced-motion and failure cases, per R7.** The facelift changes no
existing game copy; §3-E adds one new string, shown when the socket has been down and the room
has just resynced. Three registers:
- **Plain:** *"Reconnected. The room is up to date."*
- **Warm:** *"Back in the room — you're all caught up."*
- **Terse:** *"Reconnected."*
**Recommended: plain.** It says the two things a player needs — the connection returned, and what
they are looking at is current — which the terse version leaves them to infer at the one moment
they have reason to doubt it. Note this string only has a home if row 29's explicit reconnect
resync is ratified (`room-state-sync/SCOPE.md` §6.3); if it is not, this question is moot and
no copy is added.
