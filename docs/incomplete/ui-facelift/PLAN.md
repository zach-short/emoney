# PLAN — UI facelift: two lanes, six phases

**Status: `PLANNED`. GATE 2 approved 2026-09-22.** Written by an Opus 5 session in the worktree
`.claude/worktrees/ui-facelift` (branch `worktree-ui-facelift`, cut from `main` at `c14faa5`).
Stage 4 of `docs/AGENT-PRACTICES.md` §2.2. Its input is `DESIGN.md` beside this file, whose §0
holds the ten decisions ratified 2026-09-22.

**GATE 2 — approved by Zach 2026-09-22.** Per `AGENT-PRACTICES.md` §2.2 the approval authorizes
the whole run: phases do not each need re-approval. It also carries the two installs this plan
was pre-approved only to *propose* — `@radix-ui/react-popover` (Phase 4) and `motion`
(Phase 5) — which may now be installed inside their own phase and nowhere earlier.
**Lane 2's hold is not what GATE 2 releases**: it waits on board row 29 landing on
`app/room/[code]/page.tsx`, and that wait is still live.

**The session that wrote this plan wrote no code** — no component file, no CSS and no dependency
was touched by it.

**`DESIGN.md` is *what and why*. This file is *in what order, by whom, done when*.** Where a
phase's scope and the design disagree on a number, §0 below wins and says so.

---

## 0. Facts verified 2026-09-22 — these supersede the design where they differ

A second, independent verification pass, run against this worktree at `c14faa5` on 2026-09-22 —
not carried from `DESIGN.md` (R3). `DESIGN.md`'s own claims were walked or read on the same
date, so most confirm exactly; the rows marked **SUPERSEDES** are where this pass found the
design's citation wrong, and **this table is the one to build against**.

| # | Claim | Verified state, 2026-09-22 | Citation | Verdict |
|---|---|---|---|---|
| 0.1 | Worktree base | `c14faa5`, tree clean but for the untracked `docs/incomplete/ui-facelift/` | `git rev-parse --short HEAD`, `git status --short` | confirms |
| 0.2 | The ground is hardcoded | `<body className={`bg-black text-white`}>` | `frontend/app/layout.tsx:21` | confirms |
| 0.3 | No `ThemeProvider` exists | 0 hits across `app/ components/ hooks/ lib/ types/` | grepped 2026-09-22 | confirms |
| 0.4 | Both token blocks, and where | `:root` (stock light zinc) `globals.css:6-32`; `.dark` `:33-58`. **Both are indented inside `@layer base`** — a `^\.dark` grep misses them | `frontend/app/globals.css:5-59` | confirms |
| 0.5 | `next-themes` line number | Declared at **`package.json:22`** | `frontend/package.json:22` | **SUPERSEDES** `DESIGN.md` §1.1, which cites `package.json:19` |
| 0.6 | `tailwindcss-animate` line number | Declared at **`package.json:28`** | `frontend/package.json:28` | **SUPERSEDES** `DESIGN.md` §1.4, which cites `package.json:22` |
| 0.7 | `next-themes`' only consumer | `useTheme` imported `:3`, called `:9` | `frontend/components/ui/sonner.tsx:3,9` | confirms |
| 0.8 | Radix deps installed | accordion, dialog, slider, slot — and nothing else | `frontend/package.json:12-15` | confirms |
| 0.9 | No motion or popover dependency | No `motion`, no `framer-motion`, no `@radix-ui/react-popover` | `frontend/package.json`, grepped 2026-09-22 | confirms |
| 0.10 | The drawer scrim | `"fixed inset-0 z-50 bg-black/80"` | `frontend/components/ui/drawer.tsx:31` | confirms |
| 0.11 | The sonner offset block | Comment `:15-16`, **`offset={76}` at `:17`**; the light-pinning class at `:21` | `frontend/components/ui/sonner.tsx:15-21` | **SUPERSEDES** `DESIGN.md` §1.5's `:16-19` (the `:21` citation in §1.1 is exact) |
| 0.12 | No mobile offset, no toast cap | Neither `mobileOffset` nor `visibleToasts` is set anywhere in the file | `frontend/components/ui/sonner.tsx` (34 lines, read in full 2026-09-22) | confirms |
| 0.13 | Nothing is frosted | 0 hits for `backdrop-blur` or `backdrop-filter` in `app/` + `components/` | grepped 2026-09-22 | confirms |
| 0.14 | Zero reduced-motion handling | 0 hits for `prefers-reduced-motion`, `motion-safe`, `motion-reduce` | grepped 2026-09-22 | confirms |
| 0.15 | Unsystematic elevation | **14** shadow utilities, no scale, no token | `grep -rno "shadow-\(sm\|md\|lg\|xl\|2xl\)"`, 2026-09-22 | confirms exactly |
| 0.16 | Tokens unreachable by feature code | Every token class sits inside `components/ui/*`; the only two outside are `globals.css:63` (`@apply border-border`) and `:66` (`@apply bg-background text-foreground`). **Zero in any feature component** | grepped 2026-09-22 | confirms in substance. The design's total of "24" is grep-pattern-dependent (this pass's pattern returns 17); the load-bearing half — *zero in feature code* — is exact |
| 0.17 | The two invisible strings | `text-black` on the standalone Home link and on the iOS instructions `<p>` | `frontend/app/install/page.tsx:55,70` | confirms |
| 0.18 | A dead font weight | `sulpherLight` defined `:19`, exported `:28`, **no other reference in the tree** | `frontend/components/ui/fonts.ts:19,28`, grepped 2026-09-22 | confirms |
| 0.19 | A dead toast case | `PROPERTY_CHANGE` appears **exactly once in the whole repository** | `frontend/app/room/[code]/page.tsx:419` | confirms |
| 0.20 | The loader's second `:root` | `--length`, `--color-theme: #ffff00`, `--color-base: #000` | `frontend/app/globals.css:168-172` | confirms |
| 0.21 | Event history keyed on index | `key={index}` on the `eventHistory.map` | `frontend/components/navbar/navbar.tsx:113` | confirms |
| 0.22 | Toast styling and timing | `duration: 4000` `:200,208`; `position: "top-center"` `:201,210`; `text-xs` `:202,211` | `frontend/app/room/[code]/page.tsx` | confirms |
| 0.23 | The fixed-width player card | `snap-center w-[360px] border bg-white border-black aspect-[3/4]` | `frontend/components/players/player-card.tsx:91` | confirms |
| 0.24 | The sticky header and the `lg` grid | Header `sticky top-0 z-50 bg-black` at `:97`; the `lg:` wrap classes at **`:122`** | `frontend/components/room/room.client.tsx:97,122` | confirms (design cites `:120-123` for the block; the classes are on `:122`) |
| 0.25 | **The type work is bigger than the design priced** | **100 `.className` interpolations across 36 files** | `grep -rno "josephin[A-Za-z]*\.className\|sulpher[A-Za-z]*\.className"`, 2026-09-22 | **SUPERSEDES** `DESIGN.md` §1.2's "~70 … across 30 files". Phase 2 is sized against 100/36 |
| 0.26 | **`components/ui/` is mostly not generated** | 17 files; **only 7** have the generated shadcn shape (`accordion`, `button`, `dialog`, `drawer`, `input`, `slider`, `sonner`). The other 10 are hand-written house files | `ls frontend/components/ui/`, heads read 2026-09-22 | **NEW** — not in the audit. Strengthens D8 and is why the exception is worded as it is |
| 0.27 | **The popover targets do not touch row 29's file** | `page.tsx` is 440 lines and renders exactly one thing — `<RoomView>` at `:386`, imported `:11`. The card strip is `room.client.tsx` (174 lines); every F1/F2 target is under `components/property/` or `components/players/` | read 2026-09-22 | **NEW** — this is the evidence for `BD-1` |
| 0.28 | **What Lane 2 is waiting on has not moved** | Board row 29 is `HELD — GATE 1`; `docs/incomplete/room-state-sync/SCOPE.md` still reads `Status: SCOPING` (written 2026-09-17, Fable 5.1) and all five §6 questions are unanswered | `PASSOFF.md` row 29; `docs/incomplete/room-state-sync/SCOPE.md:3` | confirms — the hold is live |
| 0.29 | **Row 29 §6.3 is unanswered**, so D10 stays dormant | §6.3 is the `PLAYER_LEFT` / known-`PLAYER_JOINED` question whose recommendation carries "the explicit resync on `onopen`". Unratified | `docs/incomplete/room-state-sync/SCOPE.md` §6.3 | confirms — **no copy is added by this effort** |
| 0.30 | Fixed reading overhead per session | `AGENT-PRACTICES.md` ~13.8k + `CLAUDE.md` ~3.5k + `DESIGN.md` ~17.4k ≈ **34.6k tokens** before any work | `wc -c … \| awk '{print $1/4}'`, 2026-09-22 | **NEW** — the number the `Est. context` bands are computed against |
| 0.31 | The `components/ui/` rule, and that it is safe to edit | The rule is one table cell at `CLAUDE.md:90`; the primary checkout's `CLAUDE.md` is **clean** (no concurrent uncommitted edit) | `git -C /Users/zachshort/Projects/emoney status --short CLAUDE.md`, 2026-09-22 | confirms — D8's guard is satisfied |

**Not re-verified by this pass, and still standing on the audit alone:** every *walked* claim —
the live DOM readings, the measured rectangles, the computed colours, the 390×844 overlap
measurement — because they need a browser against `https://emoney.club` and this session did not
open one. They are dated 2026-09-22 and were walked that day. Any phase that acts on one
re-walks it first; this is stated again in each phase's `done when`.

---

## 1. Decisions taken since ratification

Build-level calls that *implement* `DESIGN.md` §0 rather than change it (R12). Each carries its
reversal.

**BD-1 — D6 (popovers F1/F2) is assigned to Lane 1, as Phase 4.**
D7 splits this work into two lanes by naming options A, C, D for Lane 1 and option E for Lane 2.
**It does not name option F at all.** Assigning it is therefore a gap to fill, not a decision to
reopen. It goes to Lane 1 because D7's stated reason for holding Lane 2 — "binds to
`page.tsx`'s message handling" — does not apply: row 0.27 above verifies that `page.tsx` renders
only `<RoomView>`, and every F1/F2 target lives under `components/property/` or
`components/players/`. Phase 4 is sequenced last within Lane 1 because it consumes Phase 1's
tokens and Phase 3's overlay elevation step.
*Reversal:* move Phase 4 to Lane 2 and re-sequence it behind Phase 5. Nothing in Phases 1–3
depends on it.

**BD-2 — Phase 1 removes the `next-themes` dependency in the same phase that deletes `.dark`.**
D1 calls `next-themes` "dead weight to be removed" without saying when. Doing it inside Phase 1
rather than as a later cleanup is cheapest: its only consumer is the `useTheme` call at
`sonner.tsx:3,9` (row 0.7), and Phase 1 already rewrites that file to stop pinning the toast
light. Removing the dep separately would mean touching `sonner.tsx` twice.
*Reversal:* leave `"next-themes": "^0.4.4"` in `package.json:22` and delete only the `useTheme`
call. The dependency is inert either way; nothing else imports it.

---

## 2. Phases

| # | Phase | Driver | Subagents | Est. context | Why that shape |
|---|---|---|---|---|---|
| **Lane 1 — buildable now; GATE 2 approved 2026-09-22. No collision with board row 29.** |
| 1 | The theme substrate | Opus 5 | none | full | One CSS file, one `layout.tsx` line, six literal deletions — small in diff, widest blast radius in the document, and invisible when correct. The cost is the walkthrough, not the edit |
| 2 | Type, and one money formatter | Opus 5 | **Sonnet 5 × 1** — call-site inventory only | full | 100 interpolations across 36 files (0.25) is a *reading* cost, not a reasoning one. Delegating the map is what keeps this phase under its ceiling |
| 3 | Materials, elevation and the toast | Opus 5 | none | comfortable | Four surfaces, three files, one dial table. The narrowest phase here |
| 4 | Popovers F1 and F2, at `lg` | Opus 5 | none | full | A new dependency plus two new surfaces, consuming Phases 1 and 3. Sequenced last in the lane for that reason (BD-1) |
| **Lane 2 — `HELD`. Waits on board row 29 landing on `app/room/[code]/page.tsx`.** |
| 5 | Cash count-up | Opus 5 | **Fable 5.1 × 1** — narrow adversarial review of the count-up hook | full | The one path here whose failure is *silent*. Part 4's sanctioned shape: a Deep **review**, not a Deep driver — the whole output is a verdict on three rules |
| 6 | Deed, offer badge and panel transitions | Opus 5 | none | comfortable | The remaining three motion moments, all component-local once Phase 5 has established the reduced-motion gate and the keying discipline |

**Bands.** Against the Opus 5 ceiling of ~400k (`AGENT-PRACTICES.md` Part 5, measured
2026-08-16), with the ~34.6k fixed reading overhead measured in row 0.30 subtracted before the
work starts. `comfortable` / `full` / `tight` as Part 5 defines them. **Nothing here is planned
at `tight`** — a `tight` phase is one that needs splitting.

---

### Phase 1 — The theme substrate

**Status: `PLANNED`. Lane 1. Driver: Opus 5. Waits on: nothing — GATE 2 approved 2026-09-22.**
Implements **D1** and **D9(c)**; carries **BD-2**.

**Scope.**
1. Replace the stock zinc `:root` block (`globals.css:6-32`) with e-money's own single dark token
   set, and **delete** the `.dark` block (`:33-58`) rather than populating it. One ground, on
   `:root`. Keep the `@layer base` wrapper — both blocks live inside it (0.4).
2. Set the token values to the ground the app already renders, so the visible result is
   unchanged where it is already correct: near-black background, near-white foreground. This
   is a *port*, not a re-colour — D2's palette work is Phase 3's, not this phase's.
3. Remove `bg-black text-white` from `<body>` (`layout.tsx:21`) and let the ground come from
   `globals.css:65-67`'s existing `@apply bg-background text-foreground`.
4. Delete the six pasted `bg-black text-white` literals that cancel `bg-background` on
   `DrawerContent`: `navbar.tsx:69`, `player-card.tsx:115`, `player-card-content.tsx:181`,
   `player-card-content.tsx:222`, `color-select-drawer.tsx:89`, `reason-select.tsx:51`.
5. Stop `sonner.tsx` pinning the toast light: remove `group-[.toaster]:bg-background` and the
   sibling class overrides at `:21` that defeat the resolved theme, and remove the `useTheme`
   call at `:3,9`.
6. Remove `"next-themes": "^0.4.4"` from `package.json:22` (BD-2) and re-run `bun install` so
   `bun.lockb` moves in the same commit.
7. Delete `text-black` at `install/page.tsx:55` and `:70` — the two strings that are invisible
   *because* of this bug (D9(c)).
8. Do **not** touch `globals.css:70-256` — the `.font` treatment, the scrollbar rules and the
   dice loader's own `:root` at `:168-172`. They are out of this phase's scope and D2 governs the
   `#ffff00` in them.

**Subagents.** None. Nine files, all small (0.4, and `globals.css` is 256 lines); there is no
sweep to delegate and the judgment is all in one CSS block.

**Done when.**

```bash
cd frontend && bun install && bun run lint
```
```bash
cd frontend && bunx tsc --noEmit
```
```bash
cd frontend && bun run build
```

…**and** — because every one of those is green at `c14faa5` and will be green for a bundle that points
at `localhost:8080` (`CLAUDE.md`, *The gates that lie*) — the proof a green gate cannot supply,
which for this phase is the whole of it:

- **The banker's Add-Money dialog renders dark.** Run the app against the real API
  (`scripts/emoney dev`, which passes both API variables — **not** a bare `bun run build`), open
  a room as banker, open a player card, tap Add Money. The dialog
  (`player-card-content.tsx:124-126`) must render on the app's own dark ground, not as a white
  card. Its Confirm button must not be a filled black `bg-primary` rectangle.
- **Every drawer still looks exactly as it did.** Open the navbar menu, a player card, the
  colour-select drawer and the reason-select drawer. Each must be visually unchanged from before
  the phase — that is the *point* of deleting the six literals: they were cancelling the token
  this phase fixes. Any drawer that changes appearance means a token value is wrong.
- **`/install` shows its text.** With a desktop browser, `/install` rendered (walked 2026-09-22) an empty
  black screen (walked 2026-09-22). After this phase the standalone "Home" link and the iOS
  instructions must be legible when their branch renders.
- **A toast is readable.** Trigger any room notification; the toast must render on the dark
  ground rather than as a white box.

**Watch for.**
- **This phase is invisible when it works, and there is no test runner that can see it** (0.14,
  and `CLAUDE.md`: no frontend test runner, no CI). A green gate proves only that it compiles.
  The four walkthrough items above are the entire proof and none of them may be skipped.
- **Deleting a literal that was not cancelling the token.** Each of the six is a `bg-black
  text-white` on a `DrawerContent`. If a seventh literal is found that is doing something else —
  setting a ground the token set does not provide — leave it and say so; do not widen the
  deletion to match the pattern.
- **`text-black` appears legitimately elsewhere** — on the white property deed, which D4 keeps as
  paper. Only `install/page.tsx:55,70` are in scope. Do not sweep the tree for `text-black`.
- **The `a11y` `text-black` hacks at `navbar.tsx:71,88` and `player-card-content.tsx:183` are
  *not* this phase's** (`DESIGN.md` §0, *Rules that survive unchanged* #7). They will look like
  the same bug. They are raised, not fixed.

---

### Phase 2 — Type, and one money formatter

**Status: `PLANNED`. Lane 1. Driver: Opus 5. Waits on: Phase 1.**
Implements **D3**.

**Scope.**
1. Add the two new faces to `components/ui/fonts.ts` — a body/UI face and a numeral face
   (tabular figures at minimum, a real mono preferred) — keeping `Josefin_Sans` as the display
   face (D3(c)). Delete the dead `sulpherLight` **only if** board row 1's sweep has not already
   claimed it; if it has, leave it (0.18, and *Rules that survive unchanged* #3).
2. Set the body/UI face on `<body>` in `layout.tsx` so the system stack stops being the default.
3. Introduce one shared money formatter and route both existing call sites through it:
   `player-card-content.tsx:161` (`$1875`) and `make-offer/amount.tsx:61` (`1,500`). One output
   shape, in the numeral face, with tabular figures.
4. Apply the numeral face to the remaining numeric surfaces: the room code, the rent ladders in
   `components/property/cards/`, and the counts on the player card.
5. Replace the display-face interpolations where the new body face is now correct. **Sized
   against 100 interpolations across 36 files (0.25), not the design's ~70/30.**

**Subagents.** **One, Sonnet 5, inventory only** — and it is what keeps this phase inside its
band. Brief: *enumerate every `josephin*.className` / `sulpher*.className` occurrence with its
file, line, and what the element is (heading, label, numeral, body copy); classify each as
display / body / numeral / unclear; return the table and nothing else.* It gets **its own
worktree** and the diffstat is checked when it returns (`AGENT-PRACTICES.md` Part 4: a subagent
in the shared worktree will edit source even when asked only to review). Its model is passed
explicitly; it never inherits the driver's tier.

**Done when.**

```bash
cd frontend && bun install && bun run lint
```
```bash
cd frontend && bunx tsc --noEmit
```
```bash
cd frontend && bun run build
```

…**and** the proof no gate can give:

- **A changing balance does not reflow.** Against the real API, move money so a balance crosses a
  digit-width boundary — `$1500 → $1750` is the case walked on 2026-09-22 — and confirm the
  glyphs do not shift horizontally. This is the load-bearing half of D3 and the one thing that
  must be true before Phase 5 animates that number at all.
- **Both money formats now agree.** The player card and the offer panel must render the same
  amount identically; at `c14faa5` one gives `$1875` and the other `1,500`.
- **The 12px toast and the event log are legible on a phone.** At 390×844, read a room
  notification and the navbar event history. This is the argument the dedicated body face was
  ratified on (D3(b)) and it is judged by eye, not by a number.
- **Font payload is not worse than it looks.** Record the three faces actually shipped and their
  weights. §1.9 of the design confirms no weight measurement has ever been taken here; this phase
  takes the first one so the D3 fallback (Josefin + `tabular-nums` + one mono) can be judged on
  evidence if it is ever needed.

**Watch for.**
- **The subagent's own budget.** Give it the bounded list and the stop-and-hand-off contract
  (`AGENT-PRACTICES.md` Part 5, *The relay*). A returned pass-off prompt is a normal result, not
  a failure — relay it to a fresh Sonnet 5 rather than absorbing the sweep inline.
- **`next dev` rewrites `frontend/AGENTS.md` on every run** (its own header says so). It will
  appear in `git status` during this phase's walkthrough. It is not this session's change and
  must never reach a `git add` block.
- **Three faces is three payloads on the one page that must stay snappy.** If the measurement
  above comes back bad, the fallback is a dial (§3, *Numeral face*), not a reopening of D3.

---

### Phase 3 — Materials, elevation and the toast

**Status: `PLANNED`. Lane 1. Driver: Opus 5. Waits on: Phase 1.**
Implements **D4**, **D2** and **D9(a)**.

**Scope.**
1. **The scrim.** Replace `bg-black/80` (`drawer.tsx:31`) with the §3 dial values: black at 55%
   plus a real `backdrop-filter: blur(12px)`. This is the one place where the current state is a
   measured defect rather than a preference.
2. **The header.** Make `room.client.tsx:97`'s `bg-black` translucent with an 8px blur, so the
   card strip visibly passes under it.
3. **The elevation scale.** Four steps as tokens — flat, raised, overlay, modal — replacing the
   14 unsystematic shadow utilities (0.15). On a near-black ground these carry elevation with a
   hairline top highlight and a wide soft shadow, not with the stock Tailwind shadows, which are
   tuned for white pages.
4. **D2's palette, applied.** Greyscale chrome; `#ffff00` retained only on the wordmark and the
   primary action on `/`, `/create`, `/join`; the raw `!important` at `globals.css:79-81` removed;
   red/green promoted from the literals at `make-offer/amount.tsx:71-74` and
   `free-parking.tsx:81` into tokens, **each with a redundant sign glyph**.
5. **The toast.** Set `mobileOffset` on `sonner.tsx` to clear the `h-16` header (the existing
   `offset={76}` at `:17` does not apply at phone widths — 0.11, 0.12); raise the toast to
   `text-sm` at `page.tsx:202,211`; set `visibleToasts` to 3. Leave `duration: 4000` alone.
6. **Do not glass the player card or the property deed.** They are paper (D4).

**Subagents.** None. Three files plus a token block; the judgment is visual and belongs in one
context.

**Done when.**

```bash
cd frontend && bun install && bun run lint
```
```bash
cd frontend && bunx tsc --noEmit
```
```bash
cd frontend && bun run build
```

…**and**:

- **Opening a drawer no longer deletes the room.** Against the real API, open the navbar menu.
  The room behind must remain perceptible through the scrim — the specific defect measured on
  2026-09-22 was that `bg-black/80` over black leaves nothing to see, so the drawer reads as a
  full-screen page. Judge it by whether you can tell a sheet is over a room.
- **The toast no longer covers the header, measured not eyeballed.** At 390×844, trigger a
  notification and read the toast's and the header's rectangles from the DOM. The audit measured
  toast `top: 20, bottom: 72` against header bottom `64`. They must no longer overlap.
- **The scrim does not stutter on a real phone.** Open and close a drawer repeatedly on a
  mid-range Android device. This is the one hazard with no baseline (§5) and the blur dial is set
  low because of it. **If it stutters, lower the dial — do not raise it without a measurement.**
- **Nothing yellow survives outside the three allowed places.** Walk `/`, `/create`, `/join` and
  a room; `#ffff00` must appear on the wordmark and those three primary actions and nowhere else.
- **Red/green never carries meaning alone.** Every money-direction indicator must also show a
  sign glyph. This is D2's accessibility hedge and it is invisible to every gate.

**Watch for.**
- **`backdrop-filter` is the most expensive thing in this document** and there is no performance
  baseline anywhere in the repo (design §1.9). The device check above is not optional.
- **The deed and the player card will be tempting.** They are explicitly paper; frosting them is
  the decoration this effort exists to avoid.
- **The five button languages are not collapsed here** (*Rules that survive unchanged* #9).
  Phase 3 touches colour tokens, not button structure. Collapsing them mid-phase is the drive-by
  `CLAUDE.md` forbids.

---

### Phase 4 — Popovers F1 and F2, at `lg`

**Status: `PLANNED`. Lane 1. Driver: Opus 5. Waits on: Phases 1 and 3. Carries BD-1.**
Implements **D6**.

**Scope.**
1. Install `@radix-ui/react-popover` — **pre-approved to propose, and GATE 2 is the approval to
   install.** Not before.
2. **F1 — property details.** A property's name, wherever it renders, opens a popover showing the
   deed's terms in place with the room still visible. The current cost is four interactions deep
   inside an opaque modal (design §1.6).
3. **F2 — player at a glance.** Balance, property count, banker status and pending-offer count,
   without opening the card's drawer stack.
4. **At `lg` and above only.** Below `lg` the existing drawer path is left **completely
   unchanged** — not refactored, not shared, not "improved while we're here".
5. Popover surfaces use Phase 3's `overlay` elevation step and are one of D4's four glass
   surfaces. No fifth glass surface is introduced.
6. **F3 (quick actions) is not built** and is recorded in §4 as a reserved seam.

**Subagents.** None.

**Done when.**

```bash
cd frontend && bun install && bun run lint
```
```bash
cd frontend && bunx tsc --noEmit
```
```bash
cd frontend && bun run build
```

…**and**:

- **A rent ladder is readable without leaving the room.** At ≥`lg`, click a property name and read
  its terms with the room still visible behind. Count the interactions: it must be one, against
  the four the audit measured.
- **Below `lg`, nothing changed at all.** At 390×844, walk menu → Bank's Properties → colour group
  → card. It must behave exactly as before this phase. Two implementations of one feature is the
  accepted cost of D6; a regression in the drawer path is not.
- **The popover reads as floating, not as a replacement.** It must sit above the room on the
  `overlay` elevation step, not read as another sheet.

**Watch for.**
- **Popovers are a mouse idiom and this is a phone-first product.** The `lg` gate is the whole
  mitigation. Any temptation to "just let it work on touch too" is a change to D6's ratified
  breakpoint and needs a supersession, not a judgement call.
- **Bundle weight lands on the room page**, the one page that must stay snappy, and nothing in
  this repo measures bundle size (design §5). Record the delta.

---

### Phase 5 — Cash count-up

**Status: `HELD`.** **Waits on: board row 29 landing on `app/room/[code]/page.tsx`.**
Lane 2. Driver: Opus 5, with a Fable 5.1 review subagent. Implements **D5**, items 1 and its
mitigation.

**What it is held on, stated exactly.** Board row 29 owns `app/room/[code]/page.tsx` and its
option B rewrites `handleWebSocketNotification` and `hooks/use-public-fetch.ts` — the exact seam
this phase hooks into. `AGENT-PRACTICES.md` §2.1: two items naming the same file do not run at
the same time, whatever their lanes say. **As of 2026-09-22 row 29 is `HELD — GATE 1` itself**
(0.28): its scope doc still reads `Status: SCOPING` and all five of its §6 questions are
unanswered. So this phase is held behind a gate that is itself held. **Re-read row 29's status
before starting, and re-run §0 against the live tree** — `AGENT-PRACTICES.md` §2.4: a resumed
effort re-runs Stage 4 §0 before executing a single phase.

**Scope.**
1. Install `motion` (pre-approved to propose; GATE 2 approves the install).
2. A count-up on a player's balance, and a brief directional tint on the card — credit one way,
   debit the other.
3. **The three D5 rules, implemented as code rather than as intentions:** duration hard-capped at
   600 ms; the animation always targets the authoritative server value, never a computed
   sequence; a new value arriving mid-animation **snaps** rather than re-eases.
4. `prefers-reduced-motion` from the first commit, both as a Tailwind `motion-safe:` variant and
   as `motion`'s `useReducedMotion`. **Reduced motion removes movement and keeps state**: no
   count, but the number still changes and the tint still appears as a static state.

**Subagents.** **One, Fable 5.1, review only** — the single sanctioned use of the Deep tier here
(`AGENT-PRACTICES.md` Part 4: a Deep subagent is for a narrow adversarial review of one short
load-bearing artifact, never for building, because the boundary keeps only the final text).
Brief: *given this count-up hook and these three rules, construct the message interleavings that
break each one; say for each whether the code prevents it and how you know.* Its own worktree;
diffstat checked on return; model passed explicitly.

**Done when.**

```bash
cd frontend && bun install && bun run lint
```
```bash
cd frontend && bunx tsc --noEmit
```
```bash
cd frontend && bun run build
```

…**and** — no gate in this repository can see any of the following, which is the entire reason
the Deep review exists:

- **Two transactions in quick succession never leave a wrong number on screen.** With two browser
  profiles in one room, fire two balance changes inside 600 ms. The final displayed balance must
  equal the server's, and no intermediate frame may show a value the server never sent for longer
  than the cap.
- **A mid-flight value snaps.** Land a second change while a count is running. The number must
  jump to the new authoritative target, not re-ease toward it from wherever the interpolation had
  reached.
- **Reduced motion keeps the information.** With `prefers-reduced-motion: reduce` set at the OS
  level, the balance must still change and the tint must still appear — as a state, not an
  animation. "Animations off" is a failure of this phase, not a pass.
- **The Fable 5.1 review returned a verdict on all three rules**, and it is pasted into the
  phase's close-out rather than summarised.

**Watch for.**

> **The D5 mitigation, carried here as its own named hazard and not folded into scope prose.**
> **Motion can make a stale screen look live.** This client refetches the whole room on every
> socket message with no ordering guard and no coalescing (`page.tsx:189-243`), so two messages
> in flight can resolve out of order. A 600 ms count-up is 600 ms during which the screen shows a
> number the server never sent — and the smoothness itself is what reads as "this is up to date".
> The blunt refetch as it stands is ugly and *instantaneous*, and instantaneous is worth
> something in a money app. **The three rules in scope item 3 are what make the count-up honest,
> and this codebase has no gate that can enforce any of them.** That is the whole reason this
> phase carries a Deep review and four walkthrough proofs.

- **Every refetch replaces the data object whole**, so every derived array is a new identity on
  every message. A `motion` list keyed on array index re-runs its enter animation for every
  player on every frame of any kind. **Key on `player.id` and `property.id`, never on index** —
  and note `navbar.tsx:113` keys the event history on `index` as of 2026-09-22 (0.21).
- **Reserve space before animating anything conditional.** Card contents appear and disappear on
  state (`player-card-content.tsx:193-239`) inside a fixed `w-[360px] aspect-[3/4]` card.
  Animating entry without reserving space pushes the rest of the card while someone is reading a
  balance or tapping Accept. Animate opacity and transform only.

---

### Phase 6 — Deed, offer badge and panel transitions

**Status: `HELD`.** **Waits on: board row 29, and Phase 5.** Lane 2. Driver: Opus 5.
Implements **D5**, items 2, 3 and 4.

**Scope.**
1. **A deed changing hands.** On `PURCHASE_PROPERTY` and `OFFER_ACCEPTED`, the property count on
   the affected cards animates and the receiving card acknowledges it.
2. **An offer arriving.** The `waitingOnMe` badge (`player-card.tsx:105-111`) arrives rather than
   appearing between frames — the one place a persistent signal beats a four-second toast.
3. **Panel transitions.** The navbar's three-boolean body swap (`navbar.tsx:52-54,83-234`) and
   make-offer's step change get a real transition, so a replaced panel reads as navigation rather
   than as a repaint. **CSS only** — design §3-E puts item 4 outside `motion`'s remit.
4. No fifth motion moment (D5).

**Subagents.** None.

**Done when.**

```bash
cd frontend && bun install && bun run lint
```
```bash
cd frontend && bunx tsc --noEmit
```
```bash
cd frontend && bun run build
```

…**and**:

- **A deed changing hands is visible without reading the toast.** Complete a purchase in a
  two-profile room and confirm the receiving card acknowledges it.
- **An arriving offer is still visible after four seconds.** Send an offer from the second
  profile; the badge must persist after the toast has gone. This is the item's whole argument.
- **The navbar no longer repaints between frames.** Walk menu → Bank's Properties → back. The swap
  must read as navigation.
- **Reduced motion still keeps every one of these as state**, per Phase 5's gate.

**Watch for.**
- **The badge's enter animation and the identity churn hazard are the same bug in two costumes.**
  Phase 5's keying discipline applies here unchanged.
- **Item 4 is CSS, not `motion`.** Reaching for the library because it is now installed is how
  this phase grows past its band.
- **The layout-shift hazard applies hardest to the badge**, which is exactly a conditional
  element inside the fixed card. Reserve its space.

---

## 3. Dials

Carried **verbatim** from `DESIGN.md` §4 (extracted mechanically, not retyped). Every one
already has a recommended default; each is destined for one named token or constant, not a
literal typed twice. A dial is the thing a phase may move on evidence **without** reopening a
decision — `DESIGN.md` §0 is what may not move.

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

**Dials this plan has already moved, on evidence:** none. Every value above is as ratified.
The two most likely to move during the build, and what would move them: the **scrim blur
radius** (lower it if Phase 3's device check stutters — never raise it without a measurement)
and the **numeral face** (fall back to Josefin + `tabular-nums` + one mono if Phase 2's weight
measurement comes back bad). Both are dial moves, not supersessions of D3 or D4.

---

## 4. Seams reserved, deliberately not built

Recorded so the next effort need not guess whether an omission was considered. Each was
considered and parked, with its reason.

- **F3 — quick-action popovers on a player.** Parked at GATE 1 (D6). Not weak because it is hard,
  but because it is *wrong*: the actions are all consequential — money moves, kicks — and a
  popover is the wrong container for something that should be deliberate. The seam Phase 4 leaves
  behind is the popover primitive itself, so building F3 later is a component, not an
  architecture. **Do not re-propose this as a new idea.**
- **Real 3D for the property deed** — a deed that tilts with genuine perspective, or flips to show
  its reverse. Parked before the audit began; the CSS-only constraint is ratified (*Rules that
  survive unchanged* #4). Note that CSS `perspective` + `rotate3d` already reaches most of it and
  the technique is in-house: the dice loader at `globals.css:156-166` does exactly this. A WebGL
  renderer for one card would be a new dependency, a new bundle and a new class of device failure,
  for an effect that is decoration by the brief's own test.
- **A light theme.** D1 deletes `.dark` rather than populating it. The seam reserved is the token
  set itself: one set on `:root` means a second set is a block, not a refactor — but every
  hardcoded `bg-black` / `text-white` / `text-black` literal still in the tree would each need its
  own decision first (design §5).
- **Collapsing the five button languages.** Left whole (*Rules that survive unchanged* #9). It
  stays on the board as its own row; the facelift ships with the inconsistency visible. The seam
  Phase 3 leaves is the token layer the collapse would be built on.
- **Applying motion below `lg`, or popovers on touch.** D6 fixes the breakpoint. Changing it is a
  supersession, not a dial.

---

## 5. Repo hazards, with live numbers

Carried from `DESIGN.md` §5 and **re-checked against this worktree on 2026-09-22** — the numbers
below are this pass's, not the audit's.

- **Motion can make a stale screen look live.** The top hazard; stated in full as Phase 5's
  `watch for` and again in `DESIGN.md` D5. `page.tsx:189-243` refetches the whole room on every
  message with no ordering guard and no coalescing. **Rule for the build: animate to the
  authoritative value, never through a computed one, and snap rather than re-ease when a new
  value lands mid-flight.**
- **This work and board row 29 both own `app/room/[code]/page.tsx`** (440 lines, 0.27). Row 29 is
  `HELD — GATE 1` and unratified as of 2026-09-22 (0.28). D7's lane split is the mitigation and
  Lane 2's hold is live, not a formality.
- **There is no frontend test runner and no CI.** Verified in `CLAUDE.md` (*Stack*, *The gates
  that lie*) and unchanged here: `frontend/package.json` has no test script and no runner
  dependency. **Every claim this work makes is proved by a walkthrough or not at all** — which is
  why every phase above carries walkthrough proofs and why Phase 1's are the whole of its
  done-when.
- **`bun run build` is green while baking `localhost:8080` into the bundle, and there is no
  tell.** `api.ts:3` and `wsHelpers.ts:6` fall back to localhost when the env vars are unset,
  which is the normal local state; the `console.log` that used to name the chosen URL was removed
  in `bec2350`. **Verify visual work through `scripts/emoney dev` or against the deployed site,
  never against a local build alone.**
- **`backdrop-filter` is the expensive thing here** (Phase 3), on an app that must stay instant on
  a mid-range phone, and **no performance baseline of any kind exists** — no bundle size, no frame
  timing, no Lighthouse run (design §1.9). The blur dials are set low for this reason.
- **Added JS weight lands on the room page**, the one page that must stay snappy. Both proposed
  dependencies (`@radix-ui/react-popover`, `motion`) are additive, and nothing in this repo
  measures bundle size.
- **Every refetch replaces the data object whole**, so every derived array is a new identity on
  every message — `navbar.tsx:113` keys on `index` as of 2026-09-22 (0.21). Key on `id`, never on index.
- **`components/ui/` is 17 files of which only 7 are generated shadcn** (0.26). D8's exception
  narrows the rule; the ten hand-written files in that directory were always outside its spirit.
- **`next dev` rewrites `frontend/AGENTS.md` on every run** — its own header says so. It will show
  up in `git status` during any phase's walkthrough. **It is never this session's change and never
  goes in a `git add` block.**
- **Several sessions run against this checkout.** Never `git add -A` or `git add .`; never
  `git checkout --` or `git stash` to undo an experiment — copy the file aside and restore with
  `cp`. Commits are the owner's: print the two blocks and stop.

---

## 6. Session protocol

- **The standard is `docs/AGENT-PRACTICES.md`**, read in full before writing code (R11 — not
  optional, not conditional on phase size). It is **gitignored**, so it exists only in the primary
  checkout at `/Users/zachshort/Projects/emoney/docs/AGENT-PRACTICES.md` and **not in this
  worktree** — verified 2026-09-22. Read it from there.
- **This worktree.** `.claude/worktrees/ui-facelift`, branch `worktree-ui-facelift`, cut from
  `main` at `c14faa5`. Work only inside it; never touch the primary checkout except to read.
  On entry, before believing any gate: `( cd frontend && bun install )` — `node_modules` is
  gitignored and a fresh checkout has none.
- **`HANDOFF.md` and `PASSOFF.md` are also gitignored** and live only in the primary checkout.
  Read `HANDOFF.md` first, every session. Never list any of the three in a `git add` block.
- **Order of reading for a phase session:** `docs/AGENT-PRACTICES.md`, then `CLAUDE.md`, then this
  file's §0 and the phase, then `DESIGN.md` §0 for the decisions the phase implements. ~34.6k
  tokens of fixed overhead before any work (0.30).
- **A resumed phase re-runs §0 first** (`AGENT-PRACTICES.md` §2.4). Days-old numbers go stale and
  another effort may have taken one — Lane 2's hold in particular is a live status to re-check,
  not a fact to carry forward.
- **Close-out is Part 7, unprompted:** the phase header becomes `BUILT <date>, commit <hash>`, any
  deviation is written back into `DESIGN.md` **under the decision it deviates from**, and the
  hand-back carries the three blocks — the pass-off prompt, the runtime entries, and the next
  session's model on its own line.
- **Gates, from `frontend/` and never the repo root** (there is no root `package.json`):
  `bun run lint`, `bunx tsc --noEmit`, `bun run build`. No Go gate applies to any phase in this
  plan — nothing in `backend/` is in scope.
