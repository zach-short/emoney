# DESIGN — room state sync: what the room page does with a socket message

**Status: RATIFIED 2026-09-23.** Was `SCOPE.md`, `Status: SCOPING`, written 2026-09-17 by a
Fable 5.1 session (the tier board row 29 names; checked against the running model before
anything was read). Renamed on GATE 1 ratification per `docs/AGENT-PRACTICES.md` §2.2 Stage 3 —
git records the transition (the file was committed as `SCOPE.md` in `89e470b`, 2026-09-22).
§1–§5 are unedited Stage 1/2 output; §6 below is rewritten from "Open questions" into `D1`–`D5`,
the ratified decisions, each with its defense, dated. GATE 1 ran in chat 2026-09-23 — one batched
question, four parts, every option carrying its own recommendation (R6, R7) — and every answer
matched the recommendation.

**What this is about.** `frontend/app/room/[code]/page.tsx` answers every inbound websocket
message the same way: toast the sentence, then `GET /rooms/:code/players` again, and for four
message types `GET /rooms/:code/properties` as well. Row 29 asked whether the page should
instead update from the payload, and named the reason it is a Deep question: a client-side
apply-from-payload path that drifts from Mongo fails silently. This document is the reading
that question said had to come first — every message type, what each payload actually carries,
and where the blunt refetch is load-bearing rather than lazy.

**Read first:** `HANDOFF.md` (the ledger), then this file. Every line number below is against
`main` at `a796509` unless a worktree is named; the three files most cited —
`backend/websocket/websocketManager.go`, `backend/websocket/handler.go`,
`frontend/app/room/[code]/page.tsx` — are byte-identical between `dafd16d` and `a796509`
(`git diff --stat dafd16d HEAD -- <the three>` is empty, 2026-09-17).

---

## 1. What exists, verified 2026-09-17

**Trees read.** `main` at `a796509`; `.claude/worktrees/kick-phase4` at `8848ec7` (row 7 Phase 4,
committed, unmerged); `.claude/worktrees/trades` (row 18, **uncommitted** on `0f6623a`);
`.claude/worktrees/frontend-sweep` at `5c311e7` (row 1, committed, unmerged);
`.claude/worktrees/broadcast-deadline` (row 11, uncommitted); `.claude/worktrees/mongo-v2`
(row 23, uncommitted). The room page the product will actually run is `main` plus the first
three, and no tree holds all three merged yet — row 7 Phase 5 is the merge.

### 1.1 The wire, message by message

Sixteen message types reach the browser once the three frontend branches land. Thirteen are
room-wide `Broadcast`s, three are targeted `SendTo`s (trades), and `ERROR` goes back on the
sender's own conn. Go builds every payload as an inline map at the send site
(`CLAUDE.md`, "The websocket contract is typed on the frontend only"). "Applicable" means the
payload carries enough to update the screen in place without a fetch.

| Type | Send site | Payload beyond `notification` | Mongo state the handler changed | Applicable from the payload? |
|---|---|---|---|---|
| `PLAYER_JOINED` | `handler.go:128-135` | `playerId`, `playerName` | **None** — `JOIN` reads (`:99`) and seats (`:126`); it writes nothing | For a player already on screen there is nothing to apply. For a **new** player (joined over REST, then `JOIN`ed) the payload lacks `color`, `balance`, `isBanker`, `isActive` — a card cannot be built from it |
| `PLAYER_LEFT` | `handler.go:72-80` | `playerId` | **None** | Nothing to apply; the app renders no presence |
| `TRANSFER` | `websocketManager.go:349-354` | **nothing** | Two balances (`controllers.PlayerTransfer`, `:323`), one `Transfer` row; **no `EventHistory` row** (TRIAGE F5) | **No** — neither id nor the amount is on the wire |
| `FREE_PARKING` | `:584-589` | **nothing** | `player.balance` and `room.freeParking`, one transaction (`:505-566`); `EventHistory` (`:582`) | **No** |
| `PURCHASE_PROPERTY` | `:663-668` | **nothing** | `property.playerId`, `buyer.balance` (`controllers/propertyControllers.go:19-35`); `EventHistory` (`:662`) | **No** |
| `BANKER_TRANSACTION` | `:782-787` | **nothing** | target `balance` (`controllers/playerControllers.go:215-241`); `EventHistory` (`:780`) | **No** |
| `MANAGE_PROPERTIES` | `:977-982` | **nothing** | deeds' `developmentLevel` / `isMortgaged` / `playerId` (`SELL` → nil), `player.balance` (`:908-928`); `EventHistory` (`:976`) | **No** |
| `PLAYER_KICKED` | `:1284-1290` | `playerId` | target `{isActive:false, isBanker:false}`; successor `{isBanker:true}` (`:1256-1268`); `BANK`: target's deeds `{playerId:nil, isMortgaged:false, developmentLevel:0}` (`:1182-1190`); `AUCTION`: `room.auction` (`:1217-1227`); `EventHistory` (`:1277`) | **Partial** — `isActive` yes; the successor's id and the disposition are **not** on the wire |
| `AUCTION_STARTED` | `:1299-1307` | `propertyId`, `kickedPlayerId`, `lotCount` | `room.auction` incl. the queue (`:1204-1207`); `EventHistory` (`:1298`) | **Partial** — the queue is absent (derivable: the kicked player's deeds by `propertyIndex`, which the client holds) |
| `BID_PLACED` | `:1745-1753` | `propertyId`, `bidderId`, `amount` | `room.auction.highBid` / `highBidderId`, one conditional `UpdateOne` (`:1721-1732`); **no `EventHistory`** (`:1568-1571`) | **Yes** — and it is already applied this way in `kick-phase4` (§1.6) |
| `AUCTION_LOT_CLOSED` | `:2039-2050` | `propertyId`, `winnerId?`, `amount?` | `room.auction` → next lot or `$unset` (`:1949-1962`); deed `{playerId: winner or nil, isMortgaged:false, developmentLevel:0}` (`:1995-2004`); winner `balance` (`:2006-2015`); `EventHistory` (`:2037`) | **Partial** — deed and balance yes; the next lot is derivable from the queue head (`advanceAuction`, `:1520-1535`) but only its *name* is on the wire, inside the prose |
| `OFFER_ACCEPTED` (trades) | `trades/.../offers.go:840-848` | `offerId`, `fromPlayerId`, `toPlayerId` | offer → `ACCEPTED`; deeds both ways, each pinned to its owner; one net cash `$inc` (`:772-822`); `EventHistory` (`:837`) | **No for third parties** — the deed lists and amounts are only in the two parties' inboxes |
| `OFFER_RECEIVED` / `OFFER_SENT` (trades, `SendTo`) | `offers.go:579-592` | `offer` — the whole `models.Offer` (`trades/backend/models/offerModel.go:55-71`) | `Offer` inserted; with `counterOf`, the original → `COUNTERED` (`:541-567`) | **Yes** — and the countered original is identifiable through `offer.counterOf` |
| `OFFER_RESOLVED` (trades, `SendTo`) | `offers.go:698-709` | `offerId`, `status: "DENIED"` | one `FindOneAndUpdate` (`:676-678`) | **Yes** |
| `ERROR` | `handler.go:145-203` (eight sites) plus the `default` arm on `cc22512` (row 25, unmerged, undeployed) | bare string | **None** | n/a — `page.tsx:129-138` already returns without refetching |

Two structural facts fall out of the table. **Five of the thirteen broadcasts carry nothing but
a sentence** — the five oldest money paths. And **the `EventHistory` row a handler writes is on
no payload at all**: `id`, `timestamp` and the `eventType` pair are generated in
`CreateEventHistory` (`websocketManager.go:2092-2108`) with `eventTypeFor`'s substring switch
(`:2064-2090`), and the only way any of that reaches the Event History pane
(`components/navbar/navbar.tsx:112-136`, count at `:168`) is the refetch.

### 1.2 The client's state model

- Room state lives in two `usePublicFetch` hooks (`page.tsx:31-58`; the hook is
  `frontend/hooks/use-public-fetch.ts`, whose only other user is the hook file itself — grepped
  2026-09-17). `data` is **replaced whole** by `applyResponse` (`use-public-fetch.ts:18-26`);
  the hook exposes no way to change a field. Everything the screen shows — `player`,
  `otherPlayers`, `room`, `eventHistory` — is a straight projection of the last response
  (`page.tsx:60-68`). The trades branch adds a third hook for the inbox.
- **`refetch` has no in-flight coalescing and no ordering guard** (`use-public-fetch.ts:76-86`):
  each call is an independent GET and each response is applied when it lands. Two refetches in
  flight — which every burst of messages produces — can resolve out of order, and the older
  response then overwrites the newer one with no signal. Read, not observed.
- **A failed refetch replaces the room with the error page.** `applyResponse` sets `data` to
  `null` and `error` to the failure (`:22-24`); `page.tsx:274-302` computes `error` from either
  hook and hands it to `DataState`, which renders `Fallback` on any truthy error
  (`components/containers/data-state.tsx:64-66`) — the full-screen "Something went wrong / Try
  Again" (`components/not-found/fallback.tsx`). The socket stays up, so the next message's
  refetch restores the room if it succeeds. One transient failure among the N-per-message
  fetches a room now makes is enough. Read, not observed.
- The one-way latch at `page.tsx:285-291` stops a refetch flashing the dice loader; `loading`
  is passed to `RoomView` and unused there (`room.client.tsx:36` declares it, the body never
  reads it).
- Local UI state keyed on fetched-array identity resets on every refetch:
  `components/players/manage-properties.tsx:57-71` re-seeds the house counter whenever
  `player.properties` is a new array, which after every message in the room it is. An
  observation about the current design, not a target.

### 1.3 What the refetch costs, in queries and bytes

- `GET /rooms/:code/players` (`controllers/playerControllers.go:18-118`) is one `Room`
  `FindOne` (`:28`), one `Player` `Find` (`:62`), **one `Property` `Find` per player**
  (`:72-88`), and one `EventHistory` `Find` sorted newest-first with no limit (`:91-92`):
  **P + 3 Mongo queries per fetch**, where P is the number of players, and the history part grows
  for the life of the room.
- `GET /rooms/:code/properties` (`controllers/propertyControllers.go:62-96`): two queries.
  Trades' `GET /rooms/:code/offers`: one, indexed.
- So one action fans out to **N × (P + 3)** queries for a room of N screens, plus N × 2 on the
  four property-changing types, plus N × 1 for the inbox once trades lands — and the same again
  on **every reconnect by any one player**, because their `JOIN` broadcasts `PLAYER_JOINED` to
  the whole room. For a five-player room that is roughly 40 queries per action and 50 per
  purchase, all from a single e2-micro against Atlas.
- Measured 2026-09-17, read-only, against production room `TRDCHK` (2 players, 0 deeds, 1
  event): `/players` **747 B JSON, 347 B gzipped, 0.48 s**; `/properties` **8,596 B JSON, 1,109 B
  gzipped, 0.33 s** (28 deeds with `images` and `rentPrices`). Caddy compresses
  (`backend/deploy/Caddyfile:15`). Scaling that per-property size, a five-player mid-game room
  is on the order of 20-25 KB of JSON per `/players` fetch. Bandwidth is not the cost; query
  count and latency are.
- **No measurement of the server under a real game exists**, and HANDOFF 13's five-player room
  `game` is gone (`/players` → 404, 2026-09-17). Row 29 was raised from reading code, not from
  a slow room. State that plainly: the fan-out is a scaling and battery concern with no incident
  behind it.

### 1.4 Where the refetch is a correctness backstop — by name

Row 29 asked whether kick, trades or the auction assume the refetch as a backstop rather than a
shortcut. **All three do, and they say so in their own comments.** Six places:

1. **Reconnect resync.** `page.tsx:248-252` reconnects a second after any close and
   `onopen` sends `JOIN` (`:240-242`); the server broadcasts `PLAYER_JOINED` to everyone in the
   room map, sender included (`AddClient` runs at upgrade, `handler.go:52`; `Broadcast` walks the
   whole room, `websocketManager.go:115-130`), and that echo's refetch is the **only** catch-up a
   client gets for everything it missed while disconnected. Nothing in `page.tsx` refetches on
   `onopen` itself.
2. **D18's defense** (`docs/incomplete/kick-player/DESIGN.md:1019-1021`): the auction lives on
   the `Room` document *because* "every client refetches room state on every broadcast", so the
   live auction needs no REST route.
3. **Trades' dropped-frame recovery.** `refetchOffers()` on every frame of any type
   (`trades` `page.tsx` diff; `types/events.ts` trades diff `:36-38`), and the inbox's Accept
   closes the drawer trusting "the refetch the `OFFER_ACCEPTED` broadcast triggers" to clear it
   (`offers-inbox.tsx:197-200`).
4. **The auction's unknown-lot fallback.** A `BID_PLACED` for a lot this client does not have
   open pays for a refetch, "the only transport that can repair it" (`kick-phase4` `page.tsx`
   diff, the `BID_PLACED` branch; `auction-panel.tsx:152-156`).
5. **Shipping a backend feature ahead of its UI.** HANDOFF 19: `PLAYER_KICKED` "already renders
   correctly before Phase 2 ships" because the page toasts and refetches on any type it does not
   recognise. The refetch-on-unknown is how the backend has been deployed ahead of the frontend
   all week.
6. **Ordering.** `BID_PLACED` broadcasts leave from each bidder's own goroutine, so toasts can
   arrive out of commit order (HANDOFF 21, finding 3); "every client's refetched high bid is
   still right" is the stated reason that was acceptable.

### 1.5 What no message carries: a version, a sequence, an event row

- **Nothing orders the messages.** No payload carries a sequence number; no document carries a
  version. `Room.updatedAt` exists (`backend/models/roomModel.go:16`) and is written once at
  creation (`controllers/roomControllers.go:53`), never by a handler (grepped `updatedAt`,
  `version`, `seq` across `backend/`, 2026-09-17). A client cannot tell a missed message from a
  quiet room, or an out-of-order pair from an in-order one.
- **Every handler broadcasts after its transaction commits** (each `rm.Broadcast` sits below the
  `WithTransaction` or the controller call that writes), so a refetch triggered by a broadcast
  reads committed state. But two handlers on two goroutines can commit in one order and broadcast
  in the other (§1.4 item 6) — so *any* apply-from-payload design needs post-state that is safe
  to apply out of order, or a sequence taken where the commit order is known.
- **The `EventHistory` row is server-generated and off the wire** (§1.1). An apply path either
  carries the row in the payload (a change at every send site that writes one), synthesises it
  (client clock, a client copy of `eventTypeFor` — drift by construction), or stops treating the
  pane as live.

### 1.6 The one precedent: `BID_PLACED` in `kick-phase4`

`components/room/auction-bar.tsx:27-80` and the `BID_PLACED` branch in that tree's `page.tsx`:
the payload is held as an **overlay** (`LiveBid`), merged by **`max`** so it is monotone within a
lot, **keyed on the lot's identity** `(kickedPlayerId, propertyId)` captured from the auction the
bid was accepted against, ignored rather than cleaned up when the lot changes, and **falls back
to a refetch** when the payload names a lot the client does not have open. The room fetch stays
the source of truth; the payload only ever moves the display forward. Any option in §3 has to be
consistent with this, and it is the house shape to copy.

### 1.7 Row 29's premise, corrected (R5)

The row says the payload "already carries what changed in every case row 1's session read". It
does not: **five of the thirteen broadcasts carry only the sentence** (§1.1), three more are
partial, and no event-history row is on any of them. HANDOFF 36 reasoned from `page.tsx` alone;
the Go send sites were not read there. The corrected claim is on the board row itself, dated.

Provenance, for the record: refetch-on-every-message entered the file in `3389b36`
(2025-05-21, "removing type errors") on a handler from `4e0b767` (2024-12-23). Nobody decided
it; it is what the first version did.

### 1.8 What was not verified

No browser was driven and no room was played. §1.2's two failure modes (out-of-order responses;
the error page on a failed refetch) and §1.3's fan-out are read from the code, with the byte
counts as the only live measurement. Nothing in this repo can observe server load.

---

## 2. What this is / what this is not

**This is:** a decision about how the room page keeps its state current after a socket message —
whether to keep the refetch and fix its failure modes, replace it with payload-driven updates,
or both in stages — and the server-side changes each answer needs.

**This is not**, whoever asks mid-build:

- Not a change to any money rule, any rejection, or any transaction — the handlers' writes are
  out of scope; only what they *broadcast* is in it.
- Not the toast copy, the icons, or the event-history pane's design.
- Not `GetPlayersInRoom`'s per-player property loop or the unbounded history read (§1.3). They
  are the biggest lever on server cost and they are a **companion row** (§3, option E), not a
  fold-in — `controllers/` is not this row's file.
- Not auth, presence, or a second backend process. A per-room sequence held in process memory
  (§3, option C) is legitimate **only** because invariant 1 says there is exactly one process.
- Not TRIAGE F4 (structured event records, D6). Option C overlaps it and §3 says how; this
  document does not design F4.
- Not the merge of the three frontend branches that all touch `page.tsx`. That is row 7 Phase 5,
  and nothing here starts until it has landed (§5).

---

## 3. Options

Each with its defense and the strongest argument against it, so the losing arguments stay
written down. **The recommendation is B now, E beside it, C deferred into F4 — marked below.**

### A — Leave it as it is (`SETTLED AS NO`)

**Defense.** No measured problem exists (§1.3). The refetch is the backstop six places lean on
(§1.4) and every feature built this week assumed it. A table game has two to eight screens.

**Strongest argument against.** It leaves two failure modes in place that a device walk would
never catch and that cost little to close: an older response overwriting a newer one, and one
transient failure among N fetches blanking a player's screen to "Something went wrong" (§1.2).
And N × (P + 3) grows with every player and every event for the life of a room, on hardware
with no headroom.

### B — Keep the refetch; make it coalesced, ordered and failure-tolerant  ← **recommended, now**

Client only: `page.tsx` and `hooks/use-public-fetch.ts`. Four changes, all of which keep every
backstop in §1.4:

1. **Coalesce.** At most one GET per resource in flight; a message arriving mid-flight sets a
   dirty flag and exactly one trailing refetch follows. A burst of five messages costs two
   fetches, not five, and the second one is guaranteed to see all five commits.
2. **Order.** Each request takes a rising number; a response older than the newest one applied
   is dropped. This can only remove a wrongness the code already has.
3. **Keep the last good room on a failed refetch.** Toast it, retry with backoff (§4), and leave
   the full-screen `Fallback` to the initial load, where it is right.
4. **Stop refetching where nothing changed.** `PLAYER_LEFT` always; `PLAYER_JOINED` when the id
   is already on screen and is not this client's own — and make the reconnect resync **explicit**
   (refetch on `onopen` after `JOIN`) instead of an echo side-effect, so §1.4 item 1 becomes a
   stated rule rather than an accident that would break the day someone stops broadcasting
   joins.

**Defense.** Closes both silent failure modes in §1.2, removes the no-op share of the fan-out and
the reconnect storm's worst case, changes **no wire contract**, and keeps the refetch as the
truth — which is exactly what option C needs as its fallback anyway, so nothing here is thrown
away later. Its failure modes are loud: a stuck dirty flag or a dropped-forever response shows
as a stale screen on the first device walk, never as wrong money. **Default tier.** Small: two
files, no Go.

**Strongest argument against.** N fetches per action remain, each still P + 3 queries. It answers
"how do we refetch well", not "should we refetch at all", and a reader in a month may call it a
patch. It is a patch, on purpose: the thing it patches is also the fallback every later design
keeps.

### C — Apply from the payload, with server-authored post-state and a sequence  ← **deferred into F4**

The full answer. **Server:** each mutating handler adds to its broadcast the documents it changed
*as they stand after commit* — `players: [{id, balance, isActive, isBanker}]`,
`properties: [{id, playerId, isMortgaged, developmentLevel}]`, `room: {freeParking, auction}`,
`event: {id, timestamp, event, eventType}` — as **typed Go structs, one per event**, replacing the
thirteen inline maps; plus `seq`, a per-room counter. **Client:** a reducer replacing the two
hooks' role for room state, applying by id (idempotent under redelivery), dropping anything with
`seq` at or below the last applied, and **refetching on a gap, on a reset, on reconnect, and for
any message without post-state** — the `BID_PLACED` shape (§1.6) generalised. Consumers under
`components/` are untouched if `page.tsx` keeps handing down the same props.

**Defense.** Closes the fan-out for the steady state; a missed message becomes a *detected* gap
instead of silent drift; the Go side finally has a typed contract for what it sends, which
`CLAUDE.md` has been warning about since the first day.

**Strongest arguments against — three, and the third is the one that decides the sequencing.**

- *Thirteen send sites, each an untyped map today.* A handler that forgets a field in its
  post-state produces a partial update the client applies and nothing flags until a refetch
  papers over it — the exact silent failure row 29 names. Typed structs and a package test that
  every `Broadcast` carries `seq` and its declared keys are the mitigation, but that is a
  discipline the package does not have and would have to be built first.
- *Commit order is not broadcast order* (§1.5). A `seq` stamped in `Broadcast` can order two
  messages opposite to their commits, so post-state for the same document can be applied
  newest-then-oldest. The three known ways to pin it — a per-room mutex held from before the
  commit to after the broadcast; `seq` taken inside every transaction, which puts a `Room`
  write in every money transaction and makes unrelated actions in one room conflict and retry;
  or per-document versions — each cost something, and choosing is a Stage 3 decision, not a
  build-level call.
- *It is F4 by another name.* TRIAGE D6 ratified structured event records — a machine-readable
  account of what each mutation changed — and F4 is Deep, large, and "touches every mutation".
  C's post-state **is** that account, sent over the socket. Built standalone it designs the
  structured record twice, in two places, by two sessions. **Recommendation:** C's server half
  is folded into F4's scope, so one design produces the stored record and the broadcast
  payload from the same struct; B's coalesced refetch remains the fallback either way.

Also colliding today: rows 11 and 23 both hold uncommitted rewrites of `websocketManager.go`
(§5). C cannot start before both land.

### D — Server-pushed document changes from Mongo change streams

The server watches `Player`, `Property`, `Room`, `EventHistory` (and `Offer`) and pushes each
change to the room, so no handler has to describe what it did. **Rejected, and recorded so it is
not proposed again as new:** it is a second push source whose ordering against the handlers'
own broadcasts is undefined; it needs per-room stream lifecycle in a process that already must
stay single (invariant 1); Atlas tier limits on open change streams are unverified; and it is
wholly outside row 29's file.

### E — Cut the cost of one fetch on the server  ← **recommended, as a companion row**

`GetPlayersInRoom`'s P + 3 becomes 3 (one `Property` find for the room, grouped in Go) and the
`EventHistory` read gets a cap or a page. Independent of A-C, the largest single lever on
server cost, and a pure read refactor with no wire change. **Not this row's file** — raised as its
own row rather than folded. Mechanical to Default tier; the failure is loud (a deed missing from a
card on the first walk).

---

## 4. Dials

Every number the design leaves open, with a recommended default; destined for one named
constant each, not a literal typed twice.

| Dial | Recommended default | Why |
|---|---|---|
| Coalescing shape | Trailing-edge, no delay: fire at once, one follow-up if anything arrived mid-flight | A game action should show promptly; a debounce window only helps bursts, and the one real burst (`BID_PLACED`) already does not refetch |
| Failed-refetch retry | 3 attempts, 1 s / 2 s / 4 s, then wait for the next message | Enough to ride out a phone's radio hiccup; bounded so a dead API does not hammer |
| Reconnect resync | Explicit refetch on `onopen`, after `JOIN` | Makes §1.4 item 1 a rule; the echoed `PLAYER_JOINED` then costs nothing extra because of item 4 below |
| Messages that skip the room refetch | `PLAYER_LEFT`; `PLAYER_JOINED` for a known id that is not this client's | Neither changes a document (§1.1). Everything else refetches until C exists |
| Event-history pane | Stays in the `/players` payload; E caps it | Lazy-loading on open trades one unbounded read for a visible pane count that goes stale |
| Failed-refetch toast copy | **Zach's, per R7** — variants in §6 | User-facing |

---

## 5. Hazards this work walks into

- **`page.tsx` is touched by three unmerged branches** — `kick-phase4` (`8848ec7`), `trades`
  (uncommitted), `frontend-sweep` (`5c311e7`) — and row 7 Phase 5 merges the first two by hand
  with a stated collision in `handleWebSocketNotification` (HANDOFF 35: two idioms for "do not
  toast this one"; Phase 5 is told to keep the early return). **Nothing in this scope edits
  `page.tsx` until all three are on `main`**, and B's coalescing is written against the merged
  handler, which will hold the `BID_PLACED` early return, the `OFFER_*` guard, and
  `refetchOffers`.
- **`use-public-fetch.ts` is used by `page.tsx` alone** (grepped 2026-09-17), so B can change
  its contract; `frontend-sweep` retypes `api.service.ts` in the same landing, so B builds on
  the typed `GetPlayersResponse` rather than the inline generics.
- **Rows 11 and 23 hold uncommitted rewrites of `websocketManager.go`** (`broadcast-deadline`:
  `Broadcast` and `types.go`; `mongo-v2`: 22 backend files, every transaction callback's
  signature). Any server-side option collides with both and waits for both.
- **`types/events.ts` declares each inbound shape flattened**, event name beside payload fields,
  which is not the `{type, payload}` the wire carries (the `kick-phase4` `page.tsx` comment says
  so). C would have to fix the declarations before adding to them, or the types lie twice.
- **No frontend test runner** (GATE 0 answer 2). B's coalescing and ordering logic would be the
  first frontend code in this repo whose correctness is worth a unit test and has no gate; the
  runtime pass is its only proof. Raising a runner is a GATE 0 reopen, not a fold-in.
- **Row 25's `default` arm is unmerged and undeployed**: a type the server does not know is still
  dropped in silence, so a client that stops refetching on some message must never be a client
  the server has stopped answering.
- **React StrictMode double-mounts the socket effect in dev** (HANDOFF 7's set-up note), so
  reconnect behaviour looks worse locally than in production; measure on a device.
- **The five-player production room is gone**; a walk needs a throwaway room. `TRDCHK` exists
  (Alice the banker, Bob; 2 players, no deeds — HANDOFF 30).
- **This audit ran the Deep tier past its ceiling** (about 440k of context, measured with the
  Part 5 script) by reading every send site inline. The next session reads this document and
  the ledger step, not the tree again.

---

## 6. Decisions — ratified 2026-09-23

Asked as one batched question, four parts, in chat (R6); every option carried its own
recommendation (R7 for the copy); Zach's answer matched the recommendation on all four.
`SCOPE.md`'s original question numbering is kept alongside each `D`-id so a cross-reference from
elsewhere in this doc or from `HANDOFF.md` still resolves.

**D1 — This is hygiene, not an incident.** (was Q1) No room has been observed to feel slow or go
stale; the case for acting rests on §1.2's two real client-side bugs (out-of-order refetch
responses; one transient failure blanking the whole screen to `Fallback`) and §1.3's unmeasured
scaling concern, not on anything seen in play. **Defense:** this is what keeps B ahead of C — C
earns its cost from a measured problem B cannot fix, and none exists yet. **Supersession:** if a
room is later seen stale or slow in real play, that evidence supersedes D1 and reopens the
question of moving C ahead of F4.

**D2 — The split is ratified: B now, E as a companion row, C folded into F4.** (was Q2) Board row
29 closes as `DONE — DESIGN.md, ratified 2026-09-23` once B is built (Stage 4/5 below), not on
ratification alone — ratifying a design is not shipping it. **B** ships as its own board item,
Default tier, client-only, two files (`page.tsx`, `use-public-fetch.ts`). **E** (§3) is raised as
its own board row, Mechanical/Default tier, `controllers/playerControllers.go` only, sequenced
after row 23's driver migration lands since both rewrite the same controller. **C**'s server half
is not designed standalone; TRIAGE F4 inherits it, and §3-C's commit-order question (a per-room
mutex vs. transaction-scoped `seq` vs. per-document versions) is decided there, not here.
**Defense:** avoids designing the structured-event record twice in two places by two sessions,
which is the argument that decided this over building C directly. **Supersession:** none; this is
the frozen split unless F4 itself reopens it.

**D3 — `PLAYER_LEFT` and an already-known `PLAYER_JOINED` stop refetching the room.** (was Q3)
Neither changes any document today (§1.1). Paired with an **explicit** refetch on the socket's own
`onopen`, after `JOIN` — so a client's own reconnect (§1.4 item 1) becomes a stated rule instead of
an accident that would silently break the day nobody broadcasts joins anymore. **Defense:** removes
the no-op share of the fan-out without touching the one place a skip would actually cost
correctness. **Supersession:** a future presence feature that needs to react to `PLAYER_LEFT` /
known-`PLAYER_JOINED` puts its own refetch or its own handling back — this decision does not bind
that feature, it only removes today's default.

**D4 — A failed refetch keeps the last good room, toasts, and retries; the full-screen error stays
for the initial load only.** (was Q4) Retry shape is dial §4's default: 3 attempts, 1 s / 2 s / 4 s,
then wait for the next message. **Copy — plain, ratified:** *"Couldn't refresh the room.
Retrying…"* The warm and terse variants are recorded in §4's dial table and are not used.
**Defense:** closes §1.2's silent screen-blanking failure mode without a wire change.
**Supersession:** none.

**D5 — E is wanted as a row.** (was Q5) Folded into D2 above rather than kept separate — same
ratification, same turn.

### Rules that survive unchanged

Listed so a later build phase does not "helpfully" relitigate them:

- **Every backstop in §1.4 stays exactly as built**, except item 1 (reconnect), which D3 makes
  explicit rather than removing. Kick's `isActive` partial payload, trades' `refetchOffers` on
  every frame, the auction's unknown-lot fallback, shipping-ahead-of-deploy via refetch-on-unknown,
  and the out-of-commit-order tolerance for `BID_PLACED` are all untouched.
- **No wire contract changes.** B is client-only; no Go file is touched by this decision.
- **`BID_PLACED`'s overlay-and-refetch-fallback shape (§1.6) stays the house precedent** for any
  future payload-driven surface, including whatever F4 eventually builds for C.
- **The event-history pane keeps reading only from the `/players` refetch** until E caps it or F4
  gives it its own payload — this decision does not touch it either way.
