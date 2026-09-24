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

**Corrections to this table, found by Phase 3 while building, 2026-09-22 (R5).** The rows are
left as written; these are the disproofs, recorded where the wrong claim lives. Seven of the eight
are line numbers that moved when Phases 1 and 2 landed — this table was verified at `c14faa5` and
Phase 3 ran at `e144d6f`, so **cite the file, re-grep the line**. The eighth is not a line number
and is the one that changed a scope item.

| Row / plan text | Said | Actually, at `e144d6f` | How found |
|---|---|---|---|
| 0.11 | `offset={76}` at `sonner.tsx:17` | **`:23`** — Phase 2 inserted the body-face comment above it | read 2026-09-22 |
| 0.20 | the loader's second `:root` at `globals.css:168-172` | **`:151-155`** before this phase's edit | read 2026-09-22 |
| Phase 3 item 4 | the raw `!important` at `globals.css:79-81` | **`:62-64`** (`.color { color: yellow !important }`) | read 2026-09-22 |
| Phase 3 item 4 | red/green literals at `make-offer/amount.tsx:71-74` | **`:76,77`** (the percent borders) and **`:101`** (the `$` icon) | read 2026-09-22 |
| Phase 3 item 4 | red/green literals at `free-parking.tsx:81` | **`:83`**, and they are **green/blue**, not red/green — `text-green-600` for ADD against `text-blue-600` for Collect, which encodes no direction at all | read 2026-09-22 |
| Phase 3 item 7 | player-tags' four `text-black` children at `:279,283,289,293` | **`:282,286,292,296`**, *plus* the `DialogContent`'s own `text-black` at `:267`, which the item did not name | read 2026-09-22 |
| 0.15 | **14** shadow utilities | 14 confirmed exactly by the same grep — but the pattern `shadow-\(sm\|md\|lg\|xl\|2xl\)` **cannot see a bare `shadow`**, and there is one, at `ui/button.tsx:13` on the default variant. **15.** All 15 are on the scale now | grepped 2026-09-22 |
| Phase 3 item 5 | set `mobileOffset` on `sonner.tsx` | **That prop does not exist in the installed sonner.** `sonner@1.7.1` declares `offset?: string \| number` and `visibleToasts?: number` and nothing of the kind (`node_modules/sonner/dist/index.d.ts:93,98`); its mobile block hardcodes `top: 20px` and defines a single `--mobile-offset` used only for left/right (`dist/styles.css:369-404`). `mobileOffset`, and the `top: var(--mobile-offset-top)` that consumes it, are **sonner 2.x** (confirmed against the library's current docs, 2026-09-22). See `BD-9` | read + Context7, 2026-09-22 |

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

**BD-3 — Phase 1 removes the banker dialog's own `text-black` (`player-card-content.tsx:125`).**
Taken 2026-09-22 while building. The phase's done-when walks this dialog and requires it to render
on the dark ground; its `DialogContent` carried `text-black`, which was correct while
`bg-background` was white and is black-on-black after. It is the same species as D9(c)'s two
`/install` strings — invisible *because of* this phase — so it is a consequence, not a drive-by.
Its contents are an `Input` and a `Button`, both token-driven, so nothing else in it assumes a
light card.
*Reversal:* put `text-black` back at `player-card-content.tsx:125`; the dialog's title and labels
go black-on-black again.

**BD-4 — the other two dialogs are pinned light with `bg-white`, deliberately, for Phase 3 to
clear.** `player-tags.tsx:267` and `remove-player.tsx:139`. **Zach's call, asked and answered in
chat 2026-09-22**, against fixing them dark now or leaving them broken. Both were written for a
white card and both were confirmed unreadable after the token change in the running app:
player-tags' two `h4`s and two `p`s compute `rgb(0,0,0)` on an `rgb(0,0,0)` card; remove-player's
title, body and *unselected* options likewise, while its selected row (`bg-black text-white`)
survives only by accident. Fixing them dark means re-styling `border-neutral-300`,
`hover:bg-black/5` and the selected row's inversion — palette calls D2 reserves for Phase 3. The
pin keeps both exactly as they render today and is one class each, with a comment at each site
naming Phase 3.
*Reversal:* delete the two `bg-white` classes and their comments. Do not do it without restyling
the interiors — that is Phase 3's item 5 below.

**BD-5 — Sulphur Point leaves the tree entirely, in Phase 2.**
Taken 2026-09-22 while building. D3 names three jobs and one face for each, and D3(c) makes
Josefin *the* display face; a fourth family has no job under it. `sulpherBold` had five call
sites and `sulpherLight` none. Of the five, four are in files nothing renders —
`ui/reason-select.tsx` and `players/p2p-custom-transfer.tsx`, both named in *Rules that survive
unchanged* 3 as board row 1's to delete (`grep -rn "reason-select\|ReasonSelect\|CustomTransfer"
app components hooks lib` finds only their own definitions and one commented-out import,
2026-09-22). The fifth, `color-select-drawer.tsx:89`, is a `DrawerContent` whose only children are
an `sr-only` title and a grid of colour swatches — **no text renders under it at all**, so the
face was painting nothing while costing 11,008 B. Those four files are **not deleted here**; only
their font class changed, which is what lets the family go.
*Reversal:* re-add `Sulphur_Point` to `fonts.ts` and put `sulpherBold.className` back at the five
sites. It costs 22,288 B of basic-latin woff2 and changes nothing visible.

**BD-6 — the weight mapping when a display interpolation becomes body.**
Taken 2026-09-22 while building. `josephinBold.className` sets `font-weight: 700` as well as the
family (verified from the generated rule in `.next/dev/static/chunks/[next]_internal_font_google_josefin_sans_*.css`,
2026-09-22), so dropping it drops the weight too — and Josefin 700 at its low x-height carries
roughly the emphasis Manrope 600 does, not Manrope 700. Blanket-preserving 700 would have made
the app markedly heavier than it is. The mapping used: **body copy** — paragraphs, descriptions,
notes, empty states, fine print, event rows, toast detail lines — drops the class and inherits
Manrope 400; **controls and labels** — buttons, toggles, row labels, badges, toast messages —
drops the class and gains `font-semibold`; `josephinNormal` and `josephinLight` on body drop to
400. `josephinLight`'s one non-deed site (`my-rooms/page.tsx:71`) is a room code and took the
numeral face instead.
*Reversal:* the mapping is one Tailwind utility per site and is visible in the diff as
`font-semibold`. To go heavier everywhere, `font-semibold` → `font-bold`; to go lighter, delete it.

**BD-7 — the scrim covers `ui/dialog.tsx` as well as `ui/drawer.tsx`. Zach's call, asked and
answered in chat 2026-09-22.**
D4 names "the drawer scrim" and cites `drawer.tsx:31`, and D4's whole point is that there are
*exactly four* glass surfaces. `dialog.tsx:24` carried the identical `bg-black/80` and therefore
the identical measured defect, and Phase 3 item 7 turns both pinned dialogs into black cards — so
without this a dialog over a black room reads as a full-screen page, which is the state D4 exists
to fix. Asked rather than taken because a fifth glass surface would contradict a ratified decision
(R12). The answer: **"the scrim" is one surface kind, not one file**; the count stays at four. The
two alternatives offered and refused were leaving `dialog.tsx` opaque, and separating the dialog by
elevation alone.
*Reversal:* put `bg-black/80` back at `dialog.tsx:24` and drop the `backdrop-blur-[12px]`. The
drawer is unaffected.

**BD-8 — the dice loader's `#ffff00` goes greyscale.**
D2 names the loader's second `:root` as one of the two homes of the literal and then allows the
yellow only on the wordmark and the primary action of `/`, `/create` and `/join`. A full-screen
splash is neither, and the loader is on the walk path — `components/containers/data-state.tsx:55`
makes `DiceLoader` the default loading component, so it renders inside the room. `--color-theme`
and `#loading p`'s colour are now `hsl(0 0% 88%)`; the pips stay `#000` and still read against it.
Taken rather than asked because D2's sentence is not ambiguous and re-opening it would be R8.
*Reversal:* two values in `globals.css` — `--color-theme` and the `#loading p` colour — back to
`#ffff00`, plus `border-neutral-600` back to `border-yellow-100` at `loaders/dice.tsx:13`.

**BD-9 — the toast's mobile offset is a CSS rule, not the `mobileOffset` prop item 5 names.**
The prop is sonner 2.x and this repo is on `1.7.1` (see the §0 correction table). Upgrading a major
version is not something this phase may do: GATE 2 released exactly two installs,
`@radix-ui/react-popover` and `motion`, each inside its own phase, and neither is sonner. So the
dial is applied against the version that is actually installed — a `--toast-mobile-offset: 76px`
token on `:root` and one rule overriding sonner's hardcoded `top: 20px` inside the same
`@media (max-width: 600px)`. It is selected as `html [data-sonner-toaster][data-y-position="top"]`,
specificity (0,2,1) against the vendor's (0,2,0), so it wins on specificity rather than on
stylesheet order — which matters because the vendor CSS is bundled, not authored here. `76` is the
same number the desktop `offset={76}` uses: one dial, two places it is applied.
*Reversal:* delete the `@media` block and the token from `globals.css`. When sonner next goes past
2.0, delete it anyway and pass `mobileOffset={{ top: 76 }}` — the comment at the rule says so.

**BD-10 — what a field is (item 8): a recessed well, and the document declares its colour scheme.**
Item 8 made this phase decide before touching `ui/input.tsx`, and Phase 1's R1.5 could not verify
how the field painted. **A field is the one place the page asks for something back, so it reads as
a hole in the surface, not a plane on top of it.** `bg-transparent` gave it no extent at all and
left the hairline border doing the whole job; it is now `bg-white/[0.04]` with the same border and
the `flat` elevation step — **no fifth token**, because the recess is carried by the fill, which
keeps D4's four-step dial intact. The second half is `color-scheme: dark` on `:root`: until a
document declares its scheme the native controls, the caret, autofill and the scrollbars all paint
from the light default however the CSS is written, and that is the real cause of the white-box
rendering R1.5 could not get a trustworthy picture of. Verified live 2026-09-22: the banker's
Add-Money field renders dark with white text in a real browser.
*Reversal:* `bg-transparent` and `shadow-sm` back at `input.tsx:11`; drop `color-scheme: dark`.
The raw `<input>`s that do not go through `ui/input.tsx` were left alone — see *Raised, not folded
in* below.

**BD-11 — how far D2's palette reaches, and what it deliberately does not touch.**
Item 4 names two files for the red/green promotion, but its own done-when — "every money-direction
indicator must also show a sign glyph" — is app-wide, and a token applied at two of seven sites for
one meaning is a third literal, not a palette. So:
**tokens + a sign glyph at every money-direction site** — `make-offer/amount.tsx` (percent borders
and the `$` icon; `\u2212`/`+` on every percent button, not only the selected one, because the
glyph is what says which side of the trade the column is), `navbar/free-parking.tsx`,
`make-offer/make-offer.tsx` (five borders), `make-offer/select-properties.tsx`,
`players/pay.req.rent.component.tsx` and `players/purchase-properties-bank.tsx`. On the last two
the glyph goes on a **signed delta beside the balance**, not on the balance itself, because the
balance is not negative.
**Saturated chrome that is not a direction goes greyscale**, which is D2's "colour never means
'this is a button'": offers-inbox's green Accept and red Decline/Withdraw, the blue Confirm buttons
at `pay.req.rent.component.tsx` and `manage-properties.tsx`, the green Confirm Purchase, the yellow
Mortgage, and `ui/slider.tsx`'s two yellows. `manage-properties.tsx`'s Buy/Sell Confirm went
greyscale rather than to a token because the line above it already reads `Buy 2 houses (-$200)`.
**Red that stays**: the error-toast icons (`free-parking.tsx:94,99,105`,
`manage-properties.tsx:263,312`), the delete-room Danger Zone (`navbar.tsx:202-228`) and the
`not-found` error line. Those are an error axis, not money direction, and Phase 1's as-built kept
`--destructive` bright deliberately.
**The six `player-tags.tsx` badge colours are parked — Zach's call, asked and answered in chat
2026-09-22**, against greyscaling them and against collapsing them to a rarity ramp. They are
identity, like player colour, and item 4 does not name them. They are the reason the tag banner in
that dialog keeps its colour after the pin came off.
*Reversal:* each is one Tailwind utility, and the tokens are two values on `:root`.

**BD-12 — the identity treatment becomes opt-out rather than unconditional.**
`ui/link.tsx` and `ui/button-custom.tsx` *are* the primary action on `/`, `/create` and `/join`, so
the `.font` stroke plus `border-yellow-200` stays their default — but the same two components are
also used on `/my-rooms`, `/install` and the error fallback, which are not among D2's three places.
They take a `tone?: "identity" | "plain"` prop, exported as `toneClasses` from `link.tsx`, and the
three off-brand call sites pass `tone="plain"`. **This is not collapsing the five button
languages** (*Rules that survive unchanged* #9): no button language was merged, no structure
changed, and the prop is a colour variant — which is what the phase's own *watch for* says Phase 3
touches.
*Reversal:* delete the prop and inline `font border-yellow-200` in both components again; the three
call sites then go back to yellow.

**BD-13 — what F1 actually triggers from, because the design's premise is wrong. Zach's call,
asked and answered in chat 2026-09-23.**
D6 and §3-F both say "a property's name anywhere — the bank list, a player's holdings, an offer's
contents — opens a popover". **Grepped 2026-09-23: a property name renders in only five places and
three of them are the deed itself** (`common-card.tsx:63`, `utility-card.tsx:18`,
`railroad-card.tsx:18`), which already shows the full ladder. The only bare-name sites are
`offers-inbox.tsx:45` (names joined into one prose string) and `purchase-properties-bank.tsx:103`
(inside the same drawer, *after* the deed). So "a property's name" had almost nowhere to attach.
Put to Zach as a real question with three options. **Chosen: the player card's "Properties N" row
at ≥`lg`, plus each deed named in an offer.** Explicitly **not** the navbar's "Bank's Properties":
that row lives inside a `DrawerContent`, so a popover there floats over an open sheet, and
`navbar.tsx` is the file board row 1's unmerged sweep (`5c311e7`) restructures into a view union.
*Reversal:* revert the two call sites; `deed-popover.tsx` is then unreferenced and can be deleted
whole.

**BD-14 — F2 ships thin, because the card face already shows half of it. Zach's call, chat
2026-09-23.**
D6 names four facts. Two are already on the card face — the balance (`player-card-content.tsx`'s
headline `<p>`) and the property count (the Properties row). Of the other two, **`isBanker`
renders nowhere in the app at all** (grepped 2026-09-23: it appears only as a gate on the banker's
own controls and in `remove-player.tsx:115`'s successor logic), and a *total* pending-offer count
for another player **is not on the client** — `offers` carries only the current player's offers,
both directions (`room.client.tsx`). Chosen against dropping the popover for a permanent banker
mark, and against restating all four as ratified. **Built:** banker status, offers pending
*between the two of you* (or waiting on you, on your own card), plus balance and property count
restated so the panel reads whole. The offers label says which of the two it is rather than
implying a number the client cannot know.
*Reversal:* delete `player-glance.tsx` and the `hidden lg:block` button in `player-card-content.tsx`;
the balance goes back to a single `<p>` and the `offers` prop on `PlayerDetails` becomes unused.

**BD-15 — for F1's single deed the deed IS the surface, and it stays paper. Zach's call, chat
2026-09-23.**
Phase 4's scope item 5 says popover surfaces are one of D4's four glass surfaces and take the
`overlay` step. D4 says the title deed is **explicitly not glassed** — "they are paper". F1's
content is the existing `PropertyCard`, so the two point different ways; asked rather than taken,
because "exactly four" is the load-bearing word in D4 (R12). **Chosen:** `PopoverContent`'s glass
skin is stripped for a single deed (`bg-transparent`, `p-0`, `border-0`, `backdrop-blur-none`) and
the paper deed is the whole surface. No fifth glass surface. **The deed-LIST popover keeps the
glass**, because there the panel is the surface and the deeds sit on it — the same rule read the
other way round. Verified live 2026-09-23: single-deed surface computes `rgba(0, 0, 0, 0)` /
`backdrop-filter: none` with the deed at `rgb(255, 255, 255)`; the list surface computes
`rgba(0, 0, 0, 0.55)` / `blur(12px)`.
*Reversal:* drop the four override classes in `DeedPopover`; the deed then sits on a glass tray.

**BD-16 — the popover ships with no entry animation.**
Taken 2026-09-23 while building. The stock shadcn popover carries four
`data-[state=*]:animate-*` classes. Shipping them here would put unguarded motion in the tree
*before* Phase 5 establishes the reduced-motion gate this plan ratified (§3, "state kept, movement
removed"; row 0.14 confirms nothing in the tree handles `prefers-reduced-motion` as of
2026-09-23), and **Phase 6 owns panel transitions** by name. So the popover appears instantly,
which is also what the rest of the app does today.
*Reversal:* add the four `data-[state=*]:animate-*` classes to `PopoverContent` — but do it in
Phase 6, behind the gate, not before it.

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

**Status: `BUILT` 2026-09-22, Opus 5, in worktree `.claude/worktrees/ui-facelift` (branch
`worktree-ui-facelift`, on top of `c896b0e`). **Committed by Zach as `6b02528`** — 17 files,
+235/-54. NOT merged to `main`.** Lane 1. Implements **D1** and **D9(c)**; carries **BD-2**, and took **BD-3** and
**BD-4** while building.

**Gates — all run from `frontend/` after the last edit, 2026-09-22.** `bun run lint`: 75 files,
0 errors, 0 warnings (counted from `eslint . -f json`, not inferred from a silent pass).
`bunx tsc --noEmit`: clean. `bun run build`: 11/11 static pages, 10 routes. `bun install`:
487 packages, one fewer than before, `next-themes` removed.

**As built — five things the plan did not have right.**

1. **The six literals are not six `bg-black text-white` strings.** The six *sites* and their line
   numbers were exactly right, but only three carry a `text-white` at all: `navbar.tsx:69`,
   `player-card.tsx:115` and `player-card-content.tsx:222` are `bg-black` alone, and
   `player-card-content.tsx:181` has the two non-adjacent. What came out at each site is
   `bg-black`, plus `text-white` where it was present; every other class was kept.
2. **The token values are a port of what renders, not of the `.dark` block.** `--background` is
   `0 0% 0%` and `--foreground` `0 0% 100%` — pure black and white, because `bg-black` *is*
   `#000` and `text-white` *is* `#fff`, and the done-when requires the drawers to be unchanged;
   `.dark`'s near-black `240 10% 3.9%` would have shifted all six by a visible step. `--primary`
   flips to `0 0% 98%` (the done-when's filled-button requirement) and `--ring` to
   `240 4.9% 83.9%`, because a near-black focus ring on a black ground is no ring. **Every other
   token keeps the value it renders with today** — `--border` and `--input` stay near-white
   `240 5.9% 90%`, `--muted` stays light so the drawer's grab handle is unchanged, `--destructive`
   stays the bright red. `--card` and `--popover` follow the ground; nothing consumes them
   (grepped 2026-09-22).
3. **Scope step 5 was wrong about what pins the toast light, so it was implemented by its stated
   goal, not its letter.** The `group-[.toaster]:bg-*` classes at `sonner.tsx:21` do not defeat
   the resolved theme — with one token set they *are* the dark ground, and deleting them would
   hand the toast to sonner's own palette. What pinned it light was `theme` resolving through
   `next-themes` with `:root` holding light values. So: the `useTheme` call is gone (BD-2) and
   `theme="dark"` is set literally; the token classNames stay.
4. **`reason-select.tsx` cannot be walked, because nothing renders it.** Its only import is
   commented out at `p2p-custom-transfer.tsx:4` (`grep -rn "reason-select\|ReasonSelect" app
   components hooks lib`, 2026-09-22, two hits: its own definition and that comment). The literal
   came out; the drawer is unreachable dead code, and both files are already board row 1 / TRIAGE
   B5's (`DESIGN.md` *Rules that survive unchanged* #3).
5. **The plan inventoried drawers and missed dialogs.** See BD-3 and BD-4 — three `DialogContent`
   sites exist (`grep -rn "<DialogContent"`, 2026-09-22) and all three were written for the white
   card `bg-background` used to give them.

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

**Status: `BUILT` 2026-09-22, Opus 5, in worktree `.claude/worktrees/ui-facelift` (branch
`worktree-ui-facelift`, on top of `d76e4b3`). **Committed by Zach as `34be5ac`** — 39 files,
+553/-184. NOT merged to `main`.** Lane 1. Implements **D3**; took
**BD-5** and **BD-6** while building. Frontend only; nothing in `backend/` was touched.

**Gates — all run from `frontend/` after the last edit, 2026-09-22.** `bun run lint`: **76 files,
0 errors, 0 warnings** (counted from `eslint . -f json`, not inferred from a silent pass).
`bunx tsc --noEmit`: clean. `bun run build`: 11/11 static pages, 10 routes. `bun install`: 535
packages, unchanged — **this phase adds no dependency**; all three faces come from
`next/font/google`, which was already in use.

**As built — what the phase actually did.**

1. **Three faces, not five.** `components/ui/fonts.ts` now exports Josefin Sans at its three
   existing weights (display, untouched), **Manrope** (body/UI, variable) and **JetBrains Mono**
   (numerals, pinned to weight 500), plus a `numeralFace` string that bundles the mono class with
   `tabular-nums`. `Sulphur_Point` is gone entirely — see BD-5. `sulpherLight` went with it; board
   row 1's sweep (`5c311e7`, unmerged) does not touch `fonts.ts`, checked before deleting.
2. **`app/layout.tsx:21`** carries `manrope.className` on `<body>`.
3. **`lib/utils/money.ts`** is new: one `formatMoney()`, pinned to `en-US`, whole dollars, sign
   before the `$`. Both named call sites go through it and so do eleven others found while
   applying the numeral face.
4. **100 interpolations became 19.** All 19 survivors are display: three deed files plus
   `card-container.tsx`, the wordmark, two page headings, the FAQ trigger, three offers-inbox
   headings, the offer form title, the player-card name bar, the tag dialog title, the error
   page's 4xl title, and the two conditional sites (`room.client.tsx:104`,
   `room-code-input.tsx:22`) that pick display *or* numeral by what they are rendering.
5. **`components/ui/sonner.tsx` gained the body face**, which was not in the phase's scope list
   and turned out to be required by it — see the walk below.

**The count in 0.25 is right about occurrences and one file too many.** `grep -rn
"josephin[A-Za-z]*\.className\|sulpher[A-Za-z]*\.className"` over `frontend/` returns **100
occurrences across 35 source files**, verified 2026-09-22. The 36th file the row counts is
`docs/incomplete/ui-facelift/DESIGN.md` itself, which quotes the pattern in prose. The sizing
conclusion is unaffected.

**The subagent did its whole work-list in one pass.** Sonnet 5, own worktree, model passed
explicitly; it returned all 100 rows classified plus a list of numeral sites carrying no font
class at all, and no pass-off prompt. Its worktree was clean on return — **it edited no source**.
That "outside the grep" list is what made items 3 and 4 above complete rather than partial; two
of its entries (`navbar.tsx:148`, and the toast) were still wrong after my first pass and were
caught by measuring rather than by looking.

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

**Status: `BUILT 2026-09-22`, commit `be5080e` (37 files, +738/-123). Lane 1. Driver: Opus 5 (the
assigned driver; the session checked its own model against this line before reading anything).
Waited on: Phase 1 — built on `e144d6f`, which is Phase 2.**
Implements **D4**, **D2** and **D9(a)**. Took **BD-7**…**BD-12** while building, all in §1 with
their reversals; **BD-7 and BD-11's tag-colour half are Zach's calls**, asked and answered in
chat 2026-09-22 before the files they gate were edited (R12, R6).

**34 files, all under `frontend/`. Nothing in `backend/` was touched.**

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
7. **Clear Phase 1's two pinned dialogs (BD-4), added 2026-09-22 — this is not optional, it is a
   debt with a date.** `player-tags.tsx:267` and `remove-player.tsx:139` each carry a deliberate
   `bg-white` and a comment naming this phase. Restyle both interiors for the dark ground, then
   delete the pin and the comment: in `player-tags.tsx` that is four `text-black` children
   (`:279,283,289,293`); in `remove-player.tsx` it is the `DialogContent`'s own `text-black`, the
   `OptionRow` unselected state (`border-neutral-300 text-black hover:bg-black/5`, `:55`) and the
   **selected** state (`border-black bg-black text-white`, `:55`), which inverts to white-on-black
   and therefore disappears on a black card — that inversion is the real decision here, and it is
   D2's to make. **Do not delete a pin without restyling the interior**; that is the state Phase 1
   measured and refused to ship.
8. **Do not restyle `input` here without deciding what a field is.** Phase 1 left
   `ui/input.tsx`'s `bg-transparent` untouched and could not verify how it paints (see Phase 1's
   runtime entry R1.5). A field on the dark ground is a materials decision and belongs to this
   phase if it belongs anywhere.

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

#### As built — 2026-09-22

**All eight scope items landed.** Items 1, 2, 5, 6, 7 and 8 as written; item 3 covers **15**
utilities rather than 14 (§0 corrections); item 4 reaches further than its two named files, and
**BD-11** is the argument for exactly how far.

**The token layer is where the phase actually lives.** `app/globals.css` gained six values on the
single `:root` — `--money-in`, `--money-out`, the four `--elevation-*` steps — plus
`--toast-mobile-offset` and `color-scheme: dark`; `tailwind.config.ts` exposes them as
`text-money-in` / `border-money-out` and `shadow-flat|raised|overlay|modal`. **Nothing in a feature
component names a colour or a shadow value any more**; they name a token. That is what row 0.16
("zero tokens in feature code") was measuring the absence of.

**Elevation on a near-black ground is a highlight, not a shadow.** Tailwind's ramp is tuned for a
white page: a black drop shadow on `#000` draws nothing, which is why the 14 utilities the audit
counted were unsystematic *and* invisible. Each step here is an `inset 0 1px 0` white hairline on
the top edge — light falling on a raised plane — plus a wide, very dark, heavily-spread shadow that
only separates. The mapping: controls → `raised` (the two toggle tracks and thumbs, all four
`button.tsx` variants, `return-to-menu`, the Pay-or-Request button, the slider thumb, the two dead
files' shadows); the toast → `overlay`; `dialog.tsx` → `modal`. **One shadow was added rather than
replaced**: `DrawerContent` had none at all and is the `modal` step by definition — it is most of
what makes the sheet read as a sheet, and the scrim was doing the whole job alone.

**The scrim is the item the phase existed for, and it works.** `bg-black/55` +
`backdrop-blur-[12px]` on both `drawer.tsx:31` and `dialog.tsx:24` (BD-7). Verified live: the room
name and the player card are legible-but-subdued through it, which is the "can you tell a sheet is
over a room" test. The blur radius is a **pre-authorised dial and was not moved** — no device
measurement was taken (see *Done when*, and R3.3 in `RUNTIME-PASS.md`).

**The header's translucency needed a border the opaque one never did.** `bg-black/60` +
`backdrop-blur-[8px]` + `border-b border-white/10` at `room.client.tsx:97`. Without the hairline the
header had no bottom edge at all once it stopped being opaque, and the card strip passing under it
read as a rendering fault rather than as depth. Walked at 390×520 with the strip scrolled under it.

**The toast's own transform is why the first three measurements looked like a failure.** sonner
positions a top toaster by its `top` and then animates the toast in from `translateY(-100%)`. A
measurement taken inside a polling loop catches the enter transition mid-flight and reports the
toast ~55px (its own height) above where it settles. **Settled, measured at 390×844: toast
`top: 76, bottom: 131` against header `top: 0, bottom: 65` — an 11px gap.** The audit's figures
were `top: 20, bottom: 72` against `64`. Measure a toast after it has settled, not while it moves.

**`text-xs` → `text-sm` is two call sites, both in `app/room/[code]/page.tsx`** (the `ERROR` branch
and the success branch), and `visibleToasts={3}` is on the Toaster. `duration: 4000` untouched, as
the dial says. Phase 2's `manrope.className` on the Toaster was kept — verified still computing
`Manrope` at 14px after the edit.

**BD-4's two pins are cleared, interiors first, and the inversion was the real work.**
`remove-player`'s selected `OptionRow` now goes `bg-white text-black` with a `raised` step against
an unselected `border-white/25` — measured live as `rgb(255,255,255)` on `rgb(0,0,0)` selected and
`rgba(255,255,255,0.25)` hairline unselected. `player-tags`'s dialog dropped `bg-white text-black`
and its five `text-black` children; its tag-colour banner stayed (BD-11).

**Est. context was `comfortable` and that was wrong — call it `full`.** The band was set against a
narrow reading of item 4. 34 files is Phase 2's size, not "three files plus a token block". The
work was not harder, but a future plan should price "apply a palette decision" by the number of
literals it has to find, not by the number of surfaces the decision names.

**Raised, not folded in** (R9 — none dropped silently, none fixed here):
- **The raw `<input>`s do not go through `ui/input.tsx`** and so did not get BD-10's field
  treatment: `free-parking.tsx:74` (`bg-inherit`, measured transparent live), `amount.tsx:115`
  (which *was* changed, because it is inside a file item 4 already owned — it took
  `bg-white/[0.04]` and `pl-14` to clear the widened sign glyph),
  `pay.req.rent.component.tsx:200` and `room-code-input.tsx`. `color-scheme: dark` improves all of
  them; the fill does not reach them. A "make every field go through `ui/input.tsx`" row is worth
  having.
- **`manage-properties.tsx:263` formats money by hand** — `` `You need $${totalCost - player.balance}` ``
  — bypassing Phase 2's `formatMoney()`. It is in a toast detail line, which is why the Phase 2
  sweep's `.className` grep could not see it. Board row 1 / the money-format follow-up.
- **`components/ui/loader.tsx` still carries `border-yellow-200`** and was left alone: it is
  imported nowhere (raised in HANDOFF 45), so it cannot be walked and it is board row 1's to
  delete. The live loader is `components/loaders/dice.tsx`.
- **The websocket to the production API drops during a local dev session** and raises Next's dev
  overlay as "1 Issue" — visible in every screenshot from this walk. Same artifact HANDOFF 45
  raised; not this phase's and not a regression.

---

### Phase 4 — Popovers F1 and F2, at `lg`

**Status: `BUILT` 2026-09-23, commit `dd63b15` — 11 files, +623/-14. Lane 1. Driver: Opus 5 (the
session checked its own model against this table before reading anything). Waits on: Phases 1 and
3, both landed.** Implements **D6**. Carries **BD-1**; took **BD-13**…**BD-16** while building,
all in §1 with their reversals. Frontend only; nothing in `backend/` was touched.

**As built — the design's premise for F1 was wrong, and that reshaped the phase (R5).** Scope
item 2 reads "a property's name, wherever it renders". A property name renders in five places and
**three of them are the deed itself**, which already shows the terms; see **BD-13** for the grep
and for what Zach chose instead. The consequence for this phase's own done-when is stated under
*Done when* below — **the four-interaction bank path is unchanged, by decision**, and the honest
counts are different numbers against different paths.

**What was built.** Three new files — `components/ui/popover.tsx` (the primitive, house-styled),
`components/property/deed-popover.tsx` (F1: `DeedPopover` for one deed, `DeedListPopover` for a
holding) and `components/players/player-glance.tsx` (F2) — plus three wired call sites in
`player-card-content.tsx`, `player-card.tsx` and `make-offer/offers-inbox.tsx`.

**The `lg` gate is CSS, not a hook.** Every gated surface renders twice, once with `lg:hidden` and
once with `hidden lg:…`. A `useMediaQuery` would disagree between the server and the first client
frame, because the app prerenders all 11 pages. It also means **below `lg` the popover trigger is
`display: none` and therefore not in the tab order at all** — a disabled-but-focusable control was
the specific mistake `property/cards/card-container.tsx` already documents.

**`describeSide` returns nodes now, not a string** (`offers-inbox.tsx`). Its prose is unchanged,
including the `$${amount}` form, which deliberately still bypasses `formatMoney()` because it
mirrors `sideDescription` in `backend/websocket/offers.go`, the side that writes the settled
record. Raised separately rather than fixed here.

**Two defects found by measuring rather than by eye, both fixed.** (1) **The deed list wrapped one
deed per row.** `overflow-y-auto` takes ~7px for the scrollbar, so `w-[35rem]` left a 521px content
box where two 256px deeds plus a 16px gap need 528 — seven short. Now `w-[37rem]`; measured
afterwards at `left: 566` and `left: 838`, same `top`. (2) **The `overlay` elevation was painting
twice on a single deed.** `cn()`'s `twMerge` does **not** know `shadow-overlay` is a shadow
utility — it is a custom `boxShadow` key, not one twMerge ships with — so `shadow-none` did not
cancel it and both classes applied. The wrapper's border box is exactly the deed's box (measured
identical at `[343, 300, 256, 371]`), so the step was drawn on the same rectangle twice and read
darker than an `overlay`. Fixed by letting the popover surface own the shadow and giving the deed
none. **This is a live trap for any future `shadow-*` override in this app**, not just here.

**The risk that did not materialise.** F1 inside the offers drawer means a Radix popover portalled
out of an open vaul sheet, which could have fought its focus trap or dismissed it on
pointer-down-outside. Walked 2026-09-23: the popover opens and **the drawer stays open**
(`data-state: "open"`, height 810). Radix's dismissable-layer stack and vaul's cooperate.

**`.claude/launch.json` in the primary checkout was NOT touched this phase**, unlike Phases 1, 2
and 3. The preview tool does read the primary copy and not the worktree's — confirmed again
2026-09-23, `preview_start` refused `ui-facelift-frontend` and listed only the primary's three —
but the dev server can simply be started directly from the worktree with the two API variables and
opened by URL, which touches no shared file at all. **Use that instead; the touch-and-restore dance
the three earlier phases did is avoidable.** The one hard constraint is the port: the backend's
CORS allowlist names `http://localhost:3000` literally (`backend/main.go:22-26`), so **port 3000 or
nothing** — port 3200 was blocked by CORS on every request before this was worked out.

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
  **MET, against different paths than this line assumed — read BD-13 first.** Walked 2026-09-23 at
  1280×900. One interaction to a holding's deeds (click "Properties 3" on another player's card),
  against two-to-three for the drawer it replaces; one interaction to a deed named in an offer,
  against **no path at all** — that is F1's real win and the audit never counted it, because a deed
  in an offer was simply unreachable. **The four-interaction bank path is unchanged and was not in
  scope**: Zach excluded the navbar (BD-13), so "four → one" is *not* a claim this phase gets to
  make. It was re-walked below `lg` and still measures four.
- **Below `lg`, nothing changed at all.** At 390×844, walk menu → Bank's Properties → colour group
  → card. It must behave exactly as before this phase. Two implementations of one feature is the
  accepted cost of D6; a regression in the drawer path is not.
  **MET.** Walked 2026-09-23 at 375×812 (the pane's mobile preset; 390×844 was not available as a
  preset and the gate is a breakpoint, not a width). Menu → Bank's Properties → orange → St. James
  Place, deed and rent ladder intact in the horizontal strip. Measured, not eyeballed: at 375px
  every popover trigger computes `display: none` and every drawer trigger `display: flex`/`block`,
  and the balance is a `<p>` again with its `<button>` twin unpainted.
- **The popover reads as floating, not as a replacement.** It must sit above the room on the
  `overlay` elevation step, not read as another sheet.
  **MET, measured.** The glass surfaces compute `rgba(0, 0, 0, 0.55)` + `backdrop-filter:
  blur(12px)` — the ratified scrim dials — with a `1px rgb(228, 228, 231)` token border and the
  `overlay` step exactly once: `rgba(255, 255, 255, 0.09) 0 1px 0 inset` plus
  `rgba(0, 0, 0, 0.75) 0 8px 24px -6px`. The single-deed surface is transparent with the paper deed
  carrying the step (BD-15). The list popover caps itself on Radix's own measurement
  (`--radix-popover-content-available-height: 463.5px`) and scrolls rather than leaving the
  viewport: `bottom: 888` against a 900px viewport.

**Watch for.**
- **Popovers are a mouse idiom and this is a phone-first product.** The `lg` gate is the whole
  mitigation. Any temptation to "just let it work on touch too" is a change to D6's ratified
  breakpoint and needs a supersession, not a judgement call.
- **Bundle weight lands on the room page**, the one page that must stay snappy, and nothing in
  this repo measures bundle size (design §5). Record the delta.
  **RECORDED, 2026-09-23. `@radix-ui/react-popover@1.1.23`; `bun install` reports 535 → 561
  packages (+26).** Total client JS across `.next/static/**/*.js`, same 23 chunks either side:
  **953,244 B → 1,022,858 B, +69,614 B (+7.3%)** raw; **282,932 B → 306,518 B, +23,586 B (+8.3%)**
  gzipped, which is the number that actually ships. Method: build this tree, then
  `git archive HEAD` into a `mktemp -d`, `bun install && bun run build` there, and compare — which
  is also Part 6's HEAD-isolation check, and HEAD built clean. **The room route's own
  `build-manifest.json` is useless for this**: it reported an identical 551,485 B across 6 chunks
  on both trees, because it lists shared chunks only. The dep is imported only by room-route
  components, so effectively all of the delta lands on the room page. **This is the first bundle
  measurement in the repo's history** — there is still no baseline and nothing enforces one.

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

**The numeral-face dial was tested and NOT moved — Phase 2, 2026-09-22.** The fallback (Josefin +
`tabular-nums` + one mono) is not needed. Measured on the basic-latin subset, the only one an
English UI fetches: **before `58,816 B`** (Josefin ×3 = 36,528 B, plus Sulphur Point 300 = 11,280 B
and 700 = 11,008 B), **after `82,980 B`** (Josefin ×3 unchanged, Manrope variable 24,576 B,
JetBrains Mono 500 = 21,876 B) — **+24,164 B, +41%**. That is the number to judge a future
fallback against. Two things shrank it before it was accepted: dropping Sulphur Point removed
22,288 B of which 11,280 B was never referenced, and **pinning the mono to one weight halved it**
— JetBrains Mono as a variable file is 40,480 B against 21,876 B for a single static instance.
Manrope goes the other way and must stay variable: pinned to 400+600+700 it is 24,576 B *per
weight*, 73,728 B in total, against 24,576 B for the one variable file. Method: `git archive HEAD`
into a temp dir, `bun install && bun run build`, then the `@font-face` rules in
`.next/static/chunks/*.css` joined to the file sizes in `.next/static/media/`, keeping only the
rule whose `unicode-range` is the minified `U+??` (that is `U+0000-00FF`).
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
