<!-- personal-config v0.2.1 · 2026-09-16 · config 1e21aa31 · standard v1.0.3 -->
# PLAN — kick a player

**Status: IN FLIGHT — PHASES 1-4 BUILT. AMENDED 2026-09-17 (HANDOFF 35): Phase 2 and Phase 3
are committed and on `main` (Phase 3 as `6a614a3`, deployed in HANDOFF 22); PHASE 4 IS BUILT AND
UNCOMMITTED in worktree `.claude/worktrees/kick-phase4`; PHASE 5 NOT STARTED.** The line this
header carried until then — "PHASE 2 BUILT, UNCOMMITTED; PHASE 3 BUILT, UNCOMMITTED; PHASES 4-5
NOT STARTED" — is spent. **What is left is Phase 5: the commit, the push, and the walk.**
**No phase has had its device walk, and the backend deploy is owed.**
§5 hazard 1's trigger fired; that blocker is closed and the paragraph is kept only as the record
of why the wait was right. **Phase 2 is the next one and is unblocked** — it consumes Phase 1's
wire contract, which is now written down in Phase 1's header. `GATE 2` was answered 2026-09-16 and
`DESIGN.md` D10–D18 are frozen; no gate and no design question is open.

**`GATE 2`, answered 2026-09-16, in three parts.** (1) **The five phases, their order and their
drivers are approved as written** — including both non-optional Deep (Fable 5.1) reviews, each in
its own worktree, verdict-only. (2) **Phase 1 waits for the in-flight `Broadcast` lock rewrite to
land** rather than starting now and merging after; the reasoning is in §5 hazard 1. (3) **Every
dial in §3 is accepted at its recommended default**, including *no* "you were removed" screen in
v1. Nothing in this plan is now waiting on Zach except the trigger in §5 hazard 1.

Written 2026-09-16, Stage 4 per `docs/AGENT-PRACTICES.md` §2.2, against `DESIGN.md` beside this
file. The design is *what and why*; this is *in what order, by whom, done when*.

**No gate stands between this plan and source any more.** `GATE 1b` was **answered 2026-09-16**
and is frozen as `DESIGN.md` §11, `D10…D18`; `GATE 2` was **answered 2026-09-16** and is recorded
above. **No design question is open and no approval is outstanding.** What remains is a
sequencing trigger, not a decision: §5 hazard 1. Amended 2026-09-16.

---

## 0. Facts verified 2026-09-16 — these supersede `DESIGN.md` where they differ

Verified by the Stage 3/4 session, **re-verified at `main` `87e0821` after GATE 1b was answered**,
and **re-verified again at `main` `76e9b6c` when `GATE 2` was answered 2026-09-16** — the rows
below carry the `76e9b6c` numbers. `main` moved four times while this plan was being written and
gated: `5f30fd9` → `49594c1` → `87e0821` → `76e9b6c`. That is the argument for this section
existing: **a phase that starts on a later day re-runs it first** (§2.4 of the standard: days-old
plans have stale numbers). The
full close-of-Stage-3 re-check is `DESIGN.md` §12.

| Fact | Value | How checked |
|---|---|---|
| `main` | **`76e9b6c`** — "merge the empty-notification broadcast guard". Level with `origin/main`. **Amended 2026-09-16 at `GATE 2`: it was `5f30fd9` when this table was first written and has moved three times since.** Phase 1 branches from here or later. | `git log --oneline -1 main`; `git branch -vv`, 2026-09-16 |
| Live worktrees | **Four, and every one of them is fully merged — `main..<branch>` is empty for all four.** `agent-a2f874037ff8da39a` (spent, HANDOFF 10) · `agent-abffee6918b2a9e0e` (spent, board item 8, merged as `76e9b6c`) · `agent-a6f8242b5ef1ea6fc` (empty) · **`agent-b62fc66a24e0336cfb0a` at `76e9b6c` — NOT spent: it holds uncommitted edits to `websocketManager.go` plus an untracked `broadcast_race_test.go`.** That is the live hazard, and it is uncommitted work rather than an unmerged branch, so a `git log main..` check will not show it. **The same uncommitted change is also sitting in the main checkout's working tree** (byte-identical diff, 2026-09-16). See §5 hazard 1. | `git worktree list`; `main..<branch>` empty for all four; `git status --short` per tree, 2026-09-16 |
| What is on `main` that Phase 1 must copy | **The payload-assertion precedent, no longer stranded.** `websocketManager.go` on `main` has **19** checked two-value assertions and **zero** unchecked single-value ones. | `grep -cE ', ok := payload\[[^]]*\]\.\(string\)' backend/websocket/websocketManager.go` → 19; the unchecked-form grep returns nothing. 2026-09-16 |
| ~~What `d1d6038` holds that `main` does not~~ | **Nothing — closed 2026-09-16.** Board item 8 merged as `76e9b6c` (HANDOFF 11) and its gates were re-run on the merged tree. `git diff 76e9b6c d1d6038 -- backend/` is empty, so `main`'s backend now contains the empty-notification guard. **Phase 1 inherits it: `Broadcast` refuses any payload whose `notification` is present but empty, so `PLAYER_KICKED` must set one.** | `git diff --stat 76e9b6c d1d6038 -- backend/` → empty, 2026-09-16 |
| ~~Backend gates on `main` `76e9b6c`~~ **STALE — corrected 2026-09-17, HANDOFF 19: the count is 71 at `main` `c2ce09c` (`websocket` 62, `manager` 9), not 58. Items 9 and 10's race tests landed in between. Re-run this row, do not read it.** | `go build ./...` 0 · `go vet ./...` 0 · `go test -count=1 ./...` 0 with **58 tests** (`websocket` **49**, `manager` 9), `[no test files]` for the other 8 packages · `gofmt -l .` lists **`models/roomModel.go`** only. **Supersedes the 49-test figure**, which predated board item 8's 9 tests. Phase 1's done-when count must beat **58**. Run in the clean `agent-abffee6918b2a9e0e` tree, whose `backend/` is byte-identical to `main`'s — **not** in the main checkout, which carries another session's uncommitted edits. | Run 2026-09-16 **from `backend/`**. **Supersedes the 35-test figure this row carried before the merge.** From the repo root all three print `pattern ./...: directory prefix . does not contain main module` and two of them still reported exit 0 — run them from `backend/`. |
| Frontend gates | **Not run this session.** Last verified green 2026-09-16 at `e837445` (HANDOFF 7) and in the `error-toast` worktree (HANDOFF 9). | R10: this is "not seen running", not "green" |
| Production room `game` | **Re-checked at `GATE 2`, unchanged:** 5 players, all `isActive: true`; `Claude` = `6aab34db58af4d619906fab6`, $1500, **0 properties**; `Zach` is the banker; room `6aab31d558af4d619906fa91`; free parking $0. **The driver for this whole item is still live.** | `GET https://api.emoney.club/v1/rooms/game/players`, read-only, 2026-09-16 |
| Fixed reading overhead per build session | `docs/AGENT-PRACTICES.md` ~14k · `HANDOFF.md` ~22k · `CLAUDE.md` ~3k · `DESIGN.md` ~15k · this file ~8k ≈ **62k tokens before one source file is opened** | `wc -c … | awk '{print $1/4000}'`, 2026-09-16. **`HANDOFF.md` and `CLAUDE.md` both grew during this session** (88,314 and 13,995 bytes, up from 86,626 and 13,083 an hour earlier) — another session is editing them. Re-measure; do not trust these figures on a later day. |
| Deploy | Frontend auto-deploys on push to `main` (`frontend/vercel.json` skips commits that miss `frontend/`); **backend ships by hand**, `EMONEY_HOST=emoney ./backend/deploy/deploy.sh`, one process on one VM | `CLAUDE.md` *Architecture*; HANDOFF 8 ran it |

---

## 1. Decisions taken since ratification — build-level

These implement `DESIGN.md` §8; none of them changes a decision. Each carries its reversal so a
later deviation can cite it (R12).

- **BD-1 — The kick is a websocket message (`KICK_PLAYER`), not a REST route.** *Reversal:* add
  `player.DELETE` to `routes.go` later; the handler body moves unchanged. *Why:* every mutation
  in this app is a websocket action, and the kick must close a socket held in this process's
  `clients` map (`websocketManager.go:23`) — a REST route would have to call into the hub anyway.
- **BD-2 — The disposition is a field on one payload (`disposition: "BANK" | "FREEZE"`, plus
  `"AUCTION"` from Phase 3), not three message types.** *Reversal:* split the switch into three
  cases in `handler.go`. *Why:* it mirrors `managementType` (`websocketManager.go:418-427`) and
  `freeParkingType`, which is this file's established shape for "one action, several arms", and
  it keeps the rejection of an unknown arm in one place.
- **BD-3 — Estate write, mark-gone and banker succession happen inside one
  `session.WithTransaction`.** *Reversal:* unwrap into sequential writes. *Why:* the precedent
  is `freeParking` (`websocketManager.go:176-243`), the app's only multi-document transaction —
  **not** `PurchaseProperty` (`propertyControllers.go:19-35`), which is two bare writes and is
  the wrong model for anything that must not half-apply. A kick that promotes a successor and
  then fails to demote the old banker would leave two bankers.
- **BD-4 — The by-`PlayerID` socket close is a new method on `RoomManager`, beside `Broadcast`,
  and it does *not* call `RemoveClient`.** *Reversal:* named; it is ~10 lines. *Why:* the
  per-connection `defer` at `handler.go:54-64` already calls `RemoveClient` (which takes the
  write lock) and broadcasts `PLAYER_LEFT`. Calling it from the hub as well would either
  double-broadcast or deadlock against the lock the caller is holding. **See Phase 1's *watch
  for*: the lock discipline here is the feature's sharpest hazard.**
- **BD-5 — Every payload field is read with the checked two-value assertion, and every rejection
  precedes the first Mongo call.** *Reversal:* none wanted. *Why:* that is the shape HANDOFF 10
  put on all 19 sites in `websocketManager.go`, and "reject before the first database call" is
  what makes a rejection reachable in a test at all on this machine (`config.DB` is nil in a test
  binary — `CLAUDE.md` *gates that lie*). **That precedent is on `main` as of `49594c1`** — 19
  checked assertions, zero unchecked (§0). It was stranded on an unmerged branch when this
  BD was written; see §5 hazard 1 for what replaced that problem.
- **BD-6 — Phase 2 ships a disposition picker with two options, not three-with-one-disabled.**
  *Reversal:* one boolean in the picker. *Why:* `DESIGN.md` D9's recorded argument-against; a
  dead "auction — coming soon" row is a promise in the UI with nothing behind it, which is
  exactly the complaint the FAQ's trade copy already earns (HANDOFF 6 fact 9).

---

## 2. Phases

| # | Phase | Driver | Subagents | Est. context | Why that shape |
|---|---|---|---|---|---|
| 1 | Kick core — backend | **Default** (Opus 5) | One **Deep** (Fable 5.1) review of the socket-close lock discipline, in its own worktree | **full** | Go only. Backend and frontend ship separately here (Vercel auto vs. a hand-run script), which is the standard's own split signal. ~5 files, one new handler, one new hub method, the tests. |
| 2 | Kick UI — frontend | **Default** (Opus 5) | none | comfortable | React only, ~7 files, no new decisions — it consumes Phase 1's contract. Second because the wire shape is settled by then. |
| 3 | Live auction — backend | **Default** (Opus 5) | One **Deep** (Fable 5.1) adversarial review of bid-close + winner-determination + settlement | **tight** | New state, a close condition, concurrent bidders and a money write, in a codebase with no precedent for any of them. Tight even alone, which is why the UI is Phase 4. |
| 4 | Auction UI — frontend | **Default** (Opus 5) | none | full | A live bidding panel is the first screen in this app that renders shared, fast-changing state; it also has to stop the message handler toasting every bid. |
| 5 | Ship, walk, record | **Default** (Opus 5) | none | comfortable | The backend deploy is a person's, the runtime pass is Zach's, and the as-built write-up needs the judgement to say what deviated. Mechanical is acceptable if by then only doc reconciliation is left. |

**Ordering, and where the blocker actually clears.** Phase 1 + a backend deploy is enough to
remove `Claude`: a banker client can send a `KICK_PLAYER` frame by hand — HANDOFF 7's set-up
note records opening a second `WebSocket` in the page and sending the app's own frames, and that
is all this needs. **That is a fact about the phase order, not a licence to stop there**:
`DESIGN.md` D8 says build the real feature, and Phases 2–4 are the real feature.

**Lanes.** All five phases are **one lane, in order** — Phases 1 and 3 both own
`backend/websocket/websocketManager.go`, Phases 2 and 4 both own
`frontend/app/room/[code]/page.tsx`, and 2 depends on 1's contract while 4 depends on 3's. This
is board lane **D** (`PASSOFF.md` row 7). Nothing here runs in parallel with board item 8
(`PASSOFF.md`), which owns the same Go file.

---

### Phase 1 — Kick core, backend

**Status: BUILT 2026-09-17 — HANDOFF 19. MERGED TO `main` 2026-09-17 as `ed87ac8`, merge
`a1d7b49`** — this line asked to be filled in when it landed and was filled by the Phase 2
session, which found it landed mid-build (HANDOFF 20; `git show HEAD:backend/websocket/handler.go
| grep -c KICK_PLAYER` → 1). It was built in worktree `.claude/worktrees/kick-phase1-websocket`,
cut from `main` at `c2ce09c`. **Merged is not deployed** — `EMONEY_HOST=emoney
EMONEY_HEALTH_URL=https://api.emoney.club ./backend/deploy/deploy.sh` is still owed, and until it
runs, a kick sent from the UI does nothing and says nothing (`handler.go`'s switch has no
`default` arm). The `controllers/` half was built
first and separately — HANDOFF 15, merged as `4ecfb3c`, deployed in `c2ce09c`. §5 hazard 1's
trigger fired and was re-checked rather than assumed before any source was opened.

**Gates green; the device walk below is still owed.** Those are different claims. `go build`,
`go vet`, `go test` (**115**, up from the branch's 71), `go test -race` over 3 reps and `gofmt`
all pass, in the worktree and again in a `git archive HEAD` isolation tree; five mutation checks
bite; the Deep review returned **SOUND WITH CAVEATS** with no defect in the new code. **None of
that is evidence that a valid kick moves the right estate** — nothing in this repo asserts that
about any action. Done-when (a)–(g) is the only evidence that will exist, and it needs a backend
deploy first.

**What this phase settled that the plan left open, for the phases that consume it:**

- **The wire contract.** Inbound `KICK_PLAYER` payload: `roomId`, `targetPlayerId`,
  `disposition` (`"BANK"` | `"FREEZE"`), and optional `successorPlayerId` — absent, `null` and
  `""` all read as "no successor named". Outbound `PLAYER_KICKED` payload: `notification` and
  `playerId` (the removed player's). **Phase 2 builds against exactly this.**
- **Phase 3 has two edits here, not one.** Adding `AUCTION` means adding it to
  `handleKickPlayer`'s disposition guard **and** to `kickNotification`'s switch, whose default
  arm is currently unreachable-by-construction. Miss the second and the auction kick broadcasts
  the generic sentence.
- **The event icon's key is load-bearing on the copy.** `eventTypeFor` matches the kick on the
  substring **`"from the game"`**, and that arm must stay **first** — the notification has
  "Banker" as its subject and would otherwise be drawn as a balance change. Changing the kick
  copy means changing the key with it. `eventTypeFor` was extracted from `CreateEventHistory`
  solely so this is testable at all.
- **A new invariant, now in `HANDOFF.md`.** `Client.PlayerID` / `PlayerName` are written only
  through `RoomManager.SeatClient`, under `rm.mu`. The force-close reads them from another
  player's goroutine, which turned a previously safe bare assignment into a data race.
- **A build-level rule `GATE 2` never faced,** decided by Zach 2026-09-17: a successor named when
  the target turns out **not** to be the banker is **refused**, not silently dropped.

**Scope.** Executable without re-reading the design.

1. **Cut the branch from `main` at `76e9b6c` or later** — both precedents are there now: the 19
   checked assertions (`c202745`, merged as `49594c1`) and the empty-notification guard in
   `Broadcast` (`d1d6038`, merged as `76e9b6c`). Still check `git log --oneline -1` against
   `main` before reading a source file — that is HANDOFF 10's lesson, earned by a worktree that
   arrived four commits stale, and HANDOFF 11's worktree arrived stale the same way.
   **Additionally check every tree's `git status --short`, not just `main..<branch>`:** §5
   hazard 1's blocker is uncommitted work, which a branch comparison does not see.
2. **`case "KICK_PLAYER"` in `handler.go`'s switch** (`handler.go:72-137`), in the same shape as
   the other five: call the handler, and on error reply `Message{Type: "ERROR", Payload:
   err.Error()}` on the caller's own connection. The browser renders that as a red toast as of
   `5f30fd9` (HANDOFF 9). **Which write call to use is not settled — re-read the switch.** On
   `main` at `76e9b6c` it is `conn.WriteJSON`; board item 9's in-flight work replaces every one of
   them with `client.WriteJSON`, a new per-client serialized write path (§5 hazard 1). **Copy the
   neighbouring cases as they actually read when you open the file**, not as this line describes
   them — getting this wrong reintroduces the panic item 9 exists to close.
3. **`handleKickPlayer` in `websocketManager.go`.** Read `roomId`, `targetPlayerId`,
   `disposition`, and `successorPlayerId` with the checked two-value form (BD-5). Reject, before
   the first Mongo call: a non-object payload; any non-string field; a malformed ObjectID; an
   unknown `disposition`; a banker target with no successor (D5); a successor equal to the
   target; a successor not in this room or not active. **Do not reject a target equal to the
   caller** — a banker kicking themselves is allowed and takes the same successor-naming path
   (D12), so `targetPlayerId == client.PlayerID` is a valid action, not an error.
4. **The estate write, inside one `session.WithTransaction`** (BD-3, the `freeParking` shape at
   `:176-243`). `BANK` → `{"$set": {"playerId": nil, "isMortgaged": false, "developmentLevel": 0}}`
   over the target's properties, filtered `{"roomId", "playerId"}` — D3's `SELL` shape
   (`manager/propertyManager.go:52-53`) **plus the raze D13 adds**. Write it at the kick site;
   **do not change `manager.HandlePropertySaleMortgage`** — D13 took that carve-out explicitly.
   Leave a comment at the raze naming `propertyManager.go:53` and saying why the two paths
   differ, or the next reader reads it as drift. `FREEZE` → no property write at all (D4).
5. **No cash write at all** (D10). `balance` is left exactly as it stands, on the document step 6
   marks inactive; nothing is credited anywhere and **nothing touches `Room.freeParking`**. The
   thing to get right here is *not writing*, which is easy to "improve" by accident.
6. **Mark the player `isActive: false`** (D11). **The read edits are four sites, not three, and
   two of them are already built** — amended 2026-09-16 (HANDOFF 15) after reading the code D11
   cites; see `DESIGN.md` D11's *As built* block for the full reasoning.
   - `JOIN` refuses an inactive player (`handler.go:73-101`) — **still to build, this phase.**
   - `GetPlayerDetails` (`controllers/playerControllers.go:119`) refuses one — **BUILT.** This is
     the fourth site D11 never named: the Join screen re-seats through it, bypassing the
     websocket door.
   - `JoinRoom`'s collision count is scoped to active players
     (`controllers/roomControllers.go:167-173`) — **BUILT.**
   - `GetPlayersInRoom` — **no change, by decision.** "Flag" beat "filter": the flag is already on
     the wire (`models.Player.IsActive` is `json:"isActive"`), and filtering would drop a frozen
     player's nested deeds and make done-when (d) unobservable. **Do not add `isActive` to
     `query` at `:34`** — it is shared with the EventHistory find at `:92` and would blank the
     audit log with every gate green.

   **These are the difference between a kick and a no-op with green gates** (`DESIGN.md` 1b-2,
   1b-3, D11).
7. **Banker succession** (D5), same transaction: target `isBanker: false`, successor
   `isBanker: true`.
8. **`CloseClientByPlayerID(room, playerID string)` on `RoomManager`**, beside `Broadcast`
   (`:74-90` at `76e9b6c`; item 9 moves it). Scan `rm.clients[room]` for `PlayerID == playerID`
   under the lock, call `client.Conn.Close()`, and **return without touching the map** (BD-4) —
   the target's own goroutine defer does the removal. **A close is not a write**, so it does not
   go through item 9's new `Client.WriteJSON` and does not take `writeMu` — but **confirm that
   against the landed code** (§5 hazard 1), because if closes ever route through the client too,
   BD-4 needs re-checking for lock ordering between `writeMu` and `rm.mu`. `gorilla/websocket`'s
   underlying `net.Conn.Close` is safe
   to call concurrently with a blocked `Read`, which is what makes this work at all (§1 row 7).
9. **Event history + broadcast.** `rm.CreateEventHistory(notification, roomObjID)` then
   `rm.Broadcast(client.Room, Message{Type: "PLAYER_KICKED", Payload: {"notification": …,
   "playerId": …}})`. **`Broadcast` now refuses a payload whose `notification` is present but
   empty or non-string** (HANDOFF 11, on `main` as of `76e9b6c`) — so a `PLAYER_KICKED` that
   forgets to set one is dropped silently to the room and logged on the VM, not toasted blank.
   Set it on every arm of the disposition switch, and note that this makes an "it worked but
   nobody saw it" failure look like nothing happening at all. **Order matters:** broadcast
   *before* the close, or the kicked client may miss nothing (it is told nothing anyway, D6) but
   the *other* clients' refetch can race the
   `PLAYER_LEFT` the close produces. Note `CreateEventHistory`'s icon switch is substring-based
   (`:517-532`): a notification containing "Banker" gets 🏦 — see §3.
10. **Tests in `websocketManager_test.go`**, following `bankTransactionOutcome` /
    `freeParkingOutcome`. One per rejection in step 3, plus a `…RejectsBeforeReadingThePlayer`
    twin so a later refactor cannot slip validation below the first Mongo call.

**Subagents.** One **Deep (Fable 5.1)** subagent, **in its own worktree**, given exactly two
files (`websocketManager.go`, `handler.go`) and one question: *does the force-close path ever put
a concurrent read and write on `rm.clients`, and is every lock released on every return?* Its
whole output is a verdict — which is the one job that survives a subagent boundary (Part 4). Do
**not** give it the shared worktree: an analytical subagent in the shared tree edits source
(Part 4), and check its diffstat when it returns. No Mechanical sweep is needed; the file
inventory is already in this plan.

**Done when.**

- `cd backend && go build ./... && go vet ./...` both exit 0.
- `cd backend && go test ./...` passes, with a count **greater than the 58 the branch starts
  with** (§0 — it was 49 before board item 8 merged), and `ok` for `manager` and `websocket`.
- `cd backend && gofmt -l .` lists **`models/roomModel.go` and nothing else** (§5 — it can never
  be empty until board item 5 lands).
- **Mutation-checked, not just run**, the way HANDOFF 5 and 10 did it: rename the `BANK` case and
  watch a test fail; move the validation below the first Mongo call and watch the
  before-the-read test fail; restore each with `cp` from a scratchpad copy — never
  `git checkout --`, never `git stash`.
- **Proof a green gate cannot supply.** Nothing in this repo asserts that a valid action moves
  the right money (`config.DB` is nil in a test binary). So: **after the backend is deployed,
  in a throwaway production room with two devices** — banker on one, victim on the other, the
  victim holding one deed **with a house on it** — a kick must produce, observed on screen:
  (a) the victim's socket closes and the victim does **not** reappear after the 1s reconnect;
  (b) the victim's name **and** colour can be joined with again — that is the whole point of
  D11's third read edit, and it is the one the gates cannot see; (c) with `BANK`, the deed
  appears in Bank's Properties **with no house on it** (D13) at its face price; (d) with
  `FREEZE`, the deed still shows against the gone player, house intact, and their balance is
  **unchanged** — read the number, do not assume (D10); (e) `Room.freeParking` is **unchanged**
  by the kick, which is D10's explicit negative; (f) a banker kicking **themselves** with a named
  successor moves the ⊖/⊕ controls to the successor's screen **without a reload** (D12), and so
  does a banker kicking another banker; (g) `systemctl show emoney` still reports
  `SubState=running` afterwards — see *watch for*. Then delete the room, and record that its
  non-`Room` documents are orphaned (HANDOFF 6 fact 19).

**Watch for.**

- **The concurrent map panic, and it kills the process, not the request. Read §5 hazard 1
  first — another session may have already fixed this, and if so the shape below is out of
  date.** As of `76e9b6c`, `Broadcast` deletes
  from `rm.clients[room]` while holding only `rm.mu.RLock()` (`:74-90`), and `RemoveClient`
  writes the same map under `Lock()` (`:42-51`). A force-close is the first feature that
  *deliberately* produces the failing `WriteJSON` that triggers `Broadcast`'s delete, at the same
  moment as the target's own defer calls `RemoveClient`. Concurrent map read and write is a Go
  runtime throw, and `gin`'s Recovery does not cover a panic in a non-request goroutine: **the
  process dies and every room on the VM goes down.** This is why the Deep review exists and why
  (f) is in the done-when. Fixing `Broadcast` is arguably a drive-by — **raise it, do not fold
  it in** (`CLAUDE.md`), and if the review says the kick cannot be made safe without it, that is
  a `GATE`-level question, not a build call (R12).
- **A mark-only kick that silently does nothing.** D11 is exactly that shape, so this is the
  live risk, not a hypothetical: skip any **one** of the three read edits and the gates stay
  green while the kicked player walks back in a second later, or keeps their name locked, or
  keeps rendering a card. The done-when's (a) and (b) are the only things that catch it — and
  they catch different ones, which is why both are listed.
- **Two bankers.** A succession write that half-applies leaves `isBanker: true` on two
  documents, and nothing in the app rejects that. BD-3's single transaction is the guard; verify
  the transaction actually wraps both updates rather than looking like it does.

---

### Phase 2 — Kick UI, frontend

**Status: BUILT 2026-09-17 — HANDOFF 20. Commit hash: none yet, the work is uncommitted in the
primary checkout on `main`; fill this line in when it lands.** All nine scope items below are
built. **Gates green and the UI driven in a browser at 375px; the device walk is owed** — those
are different claims and the *Done when* block below says which is which.

**What this phase settled or changed, for the phases that consume it:**

- **Phase 4 inherits the same threading.** `onKickPlayer` runs
  `page.tsx` → `room.client.tsx` → `player-card.tsx` → `player-card-content.tsx`, and it is passed
  to **both** cards, unlike `onManageProperties` (there is a comment in `room.client.tsx` saying
  why, so the next reader does not "fix" the asymmetry). An auction panel that needs a per-card
  control follows this path; one that needs room-level state does not, and should not be bolted
  onto it.
- **Phase 3 has a frontend edit it did not know about.** Adding `AUCTION` means adding it to
  `KickPlayerPayload["disposition"]` in `frontend/types/payloads.ts` **and** to the picker in
  `frontend/components/players/remove-player.tsx`, in the same commit as the Go guard. The union
  deliberately has two members today (BD-6), so the picker cannot grow a third option by accident.
- **`PLAYER_KICKED` now refetches properties as well as players** (`page.tsx`) — a `BANK` kick
  frees deeds that `GetAvailableProperties` filters on. Phase 3's auction messages will need the
  same judgement call, message by message, and `BID_PLACED` is the one §3's last dial says must
  **not** toast.
- **`isRemoved` is `player?.isActive === false`**, in `player-card-content.tsx`. Any later screen
  that lists players — an auction bidder list included — has to make the same test, because the
  API returns kicked players forever by design (D11, *flag not filter*).

**Scope.**

1. **`KickPlayerPayload` in `frontend/types/payloads.ts`**, added to the `WebSocketPayload`
   union (`:1-7`) — `disposition` typed `"BANK" | "FREEZE"` (BD-6), `successorPlayerId` optional.
2. **`"KICK_PLAYER"` added to both type unions in `frontend/lib/utils/sendWsMessage.ts`**
   (`:9-15` and `:32-38`) — there are two, and missing the second is a type error that reads like
   a payload problem.
3. **A handler in `frontend/app/room/[code]/page.tsx`** beside `handleBankerTransaction`
   (`:70-85`), returning early when `player?.id` or `room?.id` is missing, the way the others do
   (that early return is HANDOFF 2's work — do not remove it).
4. **A `getIconForType` case for `PLAYER_KICKED`** (`:298-314`), or it falls through to `ℹ️`.
5. **Prop threading through `room.client.tsx`** to `player-card-content.tsx`. `room.client.tsx`
   carries a comment about a deliberately-unpassed prop (HANDOFF 7) — do not "fix" that one
   while adding this one.
6. **The control itself** in `player-card-content.tsx`, gated on `currentPlayer?.isBanker`
   exactly as the ⊖/⊕ controls are (`:140-152`) — **on every card including the banker's own**,
   because D12 puts a self-kick in scope through the same flow. The card-identity test already
   in that file (`currentPlayer?.id !== player?.id`, `:182`) gates "Pay or Request"; the kick
   control must **not** reuse it.
7. **A confirmation `Dialog`** following the precedent already in that file (`:105-135`), showing
   the target's name, the disposition picker (default per §3), and the successor picker **only**
   when the target `isBanker` — R7 applies to every word of it; bring variants, do not pick.
   **A self-kick needs its own copy**: "remove yourself and hand the bank to …" is a different
   sentence from "remove Rita", and it is the most expensive confirmation in the app (D12's
   recorded argument-against).
8. **Render an inactive player as removed.** Added 2026-09-16 (HANDOFF 15): D11's edit 2 was
   decided as *flag, not filter*, so the API keeps returning kicked players with
   `isActive: false` and **the frontend is the only thing that can distinguish them.** Without
   this the kick looks like a no-op on every other player's screen — the card still renders, at
   full balance under D10. Under `FREEZE` the card must keep showing the player's deeds (D4, and
   Phase 1's done-when (d) is exactly this check); under `BANK` it shows a player with no
   properties. `isActive` is already typed on both `frontend/types/schema.ts:10` and `:21`, so
   this is render logic only — no type or payload work.
9. **Nothing else in that file.** Its `DrawerTitle` renders a raw Mongo id (`:169`) — that is
   board item 1's, named in HANDOFF 7's found-not-fixed list.

**Subagents.** None. Seven files, all named.

**Done when.** `cd frontend && bun run lint` 0 · `bunx tsc --noEmit` 0 · `bun run build` 0 with
its route count. **All three green 2026-09-17** (HANDOFF 20), build listing the same ten routes as
before the change — this phase adds no route. **A local build is not evidence about production, and — corrected 2026-09-17,
HANDOFF 19 — there is no longer any tell in the output.** This line used to say the `apiUrl`
line in the build output prints `http://localhost:8080` when the env vars are unset; **that
`console.log` was removed in `bec2350` and no such line exists** (`CLAUDE.md` *gates that lie*,
corrected there 2026-09-16). The hazard is unchanged and the signal is gone: a local build bakes
in `localhost:8080` whenever `NEXT_PUBLIC_API_URL` is unset, which is the normal local state, and
nothing in a green build says so. Read `frontend/lib/utils/api.ts:3` against the environment you
built in, or run through `scripts/emoney dev`, which passes both variables explicitly. **Proof a green gate cannot supply:** the runtime entries
in Phase 1's done-when, walked with the real UI instead of a hand-sent frame, at 375px — the
device width every other walk in this ledger used — including the confirmation dialog's copy
being readable and the successor picker appearing **only** for a banker target. **STILL OWED, and
it needs the backend deployed first.** What *was* done 2026-09-17 is a render pass, and the
difference matters: a throwaway route rendering three cards from fabricated props was driven in a
browser at 375×812 and then deleted, which proved the dialog fits and reads, the successor picker
gates on `isBanker`, an inactive player is excluded from it, Confirm stays disabled until a
successor is picked, the removed card renders, and the two dispositions emit
`("me","BANK","sam")` and `("rita","FREEZE",undefined)`. **No kick was sent to a real backend and
no production room was created or joined.**

**Watch for.** **Scope 8 is the one that is easy to skip and invisible in a gate** — it is render
logic with no type change behind it, so nothing fails if it is forgotten; the kick simply looks
like it did nothing. And the websocket contract is typed on the frontend only, with nothing
checking the two sides agree (`CLAUDE.md`; `DESIGN.md` §9 rule 6) — `disposition` spelled
`"bank"` compiles on both sides and is rejected at runtime. Grep Go for the literal before trusting a string. And
`page.tsx` is the file the `error-toast` work just landed in and the file Phase 4 needs: re-read
`git status --short` immediately before any commit block.

---

### Phase 3 — Live auction, backend

**Status: BUILT 2026-09-17, UNCOMMITTED. Its device walk is owed and needs the backend deploy
first.** Amended 2026-09-16: this header previously named `GATE 1b` Q5–Q8.

**As built.** Six files: `backend/models/auctionModel.go` (new, the `Auction` struct),
`backend/models/roomModel.go` (+1 field), `backend/websocket/websocketManager.go`,
`backend/websocket/handler.go` (the two new cases), `backend/websocket/websocketManager_test.go`,
`backend/websocket/auction_test.go` (new). Plus the two frontend files BD-6 requires in the same
commit as the Go guard: `frontend/types/payloads.ts` and
`frontend/components/players/remove-player.tsx`. Backend suite **115 → 186**, `-race -count=3`
clean, nine mutation checks all bit, and the diff builds in a tree extracted from `HEAD`.

**Three decisions the scope did not name, all taken by Zach 2026-09-17 rather than assumed.**

1. **A lot that sells is razed and unmortgaged too**, not only one that goes unsold. Scope 5 read
   literally is `{playerId: winner}` and nothing else, which leaves D13's hole open on the sold
   path: bidding $1 on an unwanted hotel deed would be strictly better than letting it go unsold,
   because the unsold path razes and the sold path would not. Recorded under `DESIGN.md` D13.
2. **A winner who cannot pay at settlement loses the deed to the Bank**, on D16's no-bid write.
   D17 requires the second check and never says what a failed one does. A high bidder who has
   been removed from the room since bidding takes the same path. Recorded under `DESIGN.md` D17.
3. **`CLOSE_AUCTION` names the auction-lot it is closing — both `propertyId` and
   `kickedPlayerId` — and both are required.** Not in the scope, and a correctness fix rather
   than a nicety. Without the lot: two close frames — a double tap, or one retried by a flaky
   reconnect — both read the same open lot, one loses the write conflict, `WithTransaction`
   retries it, the retry re-reads and finds the *next* lot open, and settles that. One banker,
   one intention, two deeds sold, nothing reporting a problem. Without the kicked player: a
   property id identifies a *deed*, not an auction lot, and the same deed can be the open lot of
   two different auctions — so a close frame queued on a device that missed two broadcasts can
   hammer a later auction's first lot to the Bank with nobody able to bid. `(kickedPlayerId,
   propertyId)` is a unique auction-lot identity, because a player cannot be kicked twice. The
   first came out of writing the handler; the second out of the Deep review.

**The Deep (Fable 5.1) review this phase requires was run, in its own worktree, verdict only, and
it edited nothing** — its worktree was auto-cleaned and every other tree verified clean. Its
question was the plan's: *name every interleaving of two bids and a close that produces a wrong
winner, a double settlement, or a deed transferred without payment.* **Verdict: SOUND WITH
CAVEATS**, and it read the mongo-driver and gorilla source rather than reasoning from memory. Two
findings were acted on and it re-checked both:

- **A wrong winner needing no concurrency at all.** The winning bidder's read treated *every*
  error as "they left the game", so one transient read failure would send the deed to the Bank
  with the real high bidder neither charged nor given it, and the room told they had left. Fixed
  as `winnerStatus`, a pure function so a test can reach it, distinguishing `ErrNoDocuments` from
  a failed read and aborting on the latter.
- **The pin's comment described a mechanism that does not exist.** Every read in the settlement
  callback is on the session context, so they share one snapshot and the pin *cannot* fail on the
  attempt that read it — what actually protects the close is the `WriteConflict` plus
  `WithTransaction`'s retry. The consequence is that **a bid committing before the close commits
  wins**, which is right, and the opposite of what the comment claimed. Code unchanged, comment
  and both `MatchedCount` messages rewritten to say what is true.

**Raised, not folded** (`PASSOFF.md` row 13 for the first): three money bugs in `freeParking` —
a balance read outside its own transaction and re-checked stale on retry, `CreateEventHistory`
called inside the callback on `context.Background()` so a retry double-logs and an abort logs
money that never moved, and `strconv.Atoi` accepting a negative amount so `ADD -50` moves $50
*out* of Free Parking. Also: the close's deed write does not pin the current owner, so the
pre-existing unchecked `PURCHASE_PROPERTY` lets someone buy the open lot mid-auction, be charged,
and then have the close overwrite the deed to the winner — pinning it here would create a lot
that can never be closed, so it is raised rather than fixed. And `BID_PLACED` broadcasts are sent
from each bidder's own goroutine, so toasts can arrive out of commit order even though every
client's refetched high bid is correct — Phase 4's to handle.

**Two things the scope asks for that were deliberately not built, so the omissions are not
mistaken for oversights.** (a) **No `eventTypeFor` arm for the auction's rows** — scope 7 asks
for event-history rows, not an icon, and §3's icon dial was answered for the kick alone. The rows
take the neutral ℹ️ pair, which is pinned by a test rather than left to chance, and adding an arm
later means adding it *below* the kick arm. (b) **No frontend change beyond the two BD-6 files** —
`AUCTION_LOT_CLOSED` still needs adding to `page.tsx`'s `refetchProperties` list (a close moves a
deed, so without it Bank's Properties goes stale), and `frontend/types/schema.ts` needs the
`auction` field by hand. Both are Phase 4's, and both are named here so Phase 4 does not have to
rediscover them.

**Driver: Default (Opus 5) — with a Deep (Fable 5.1) subagent review that is not optional.**
Part 4's discriminator is *can the failure be silent*, and this phase answered that in three
places when it was written. **D14 closed one of them:** with the banker closing each lot by hand
there is no timer that can fire while a bid is in flight. **Two remain** — winner determination
can pick the wrong bid when two `PLACE_BID` frames arrive on separate goroutines, and settlement
moves money in a codebase whose nearest precedent is non-transactional and unfloored
(`DESIGN.md` 1b-5). Both compile, pass every gate in this repo, and are wrong in production, and
the repo has **no test that asserts any valid transaction moves the right money** (`CLAUDE.md`).
Opus 5 drives because the phase is a build and a Deep subagent never builds (Part 4); the Deep
review is where the tier's judgement is actually spent.

**Scope.** Every mechanic below is `DESIGN.md` D14–D18, not a proposal.

1. **Auction state on the `Room` document in Mongo** (D18). `Room` is
   `models/roomModel.go:9-17`; `GET /rooms/:code/players` already returns it whole
   (`playerControllers.go:102-106`), so every client's existing refetch-on-broadcast
   (`page.tsx:121-132`) sees the live auction with **no new REST route** — and a client that
   reconnects mid-auction learns the current high bid, which in-memory state could never give it.
   Shape needed: the lot (one `propertyId`), the current high bid and bidder, the remaining
   queue, and which player was kicked. **Nothing about the auction lives in `RoomManager`.**
2. **`START_AUCTION`** — written by `handleKickPlayer` when Phase 1's disposition is `AUCTION`
   (BD-2's third arm). It enqueues the kicked player's deeds **in `PropertyIndex` order** and
   opens the first one (D16). `PropertyIndex` is `models/propertyModel.go:13`.
3. **`PLACE_BID`** — opens at **$0**, any raise of at least **$1** above the current high bid
   (D15). **Reject a bid above the bidder's current balance at bid time** (D17). Also reject:
   a bid at or below the current high, a bid on a closed or non-existent lot, a bid from a player
   not in this room or not active, and a bid from the kicked player (who has no socket after D1
   and no standing after D11).
4. **`CLOSE_AUCTION`** — **the banker closes each lot by hand** (D14). No timer, no goroutine, no
   deadline: D14 chose this precisely so that this phase does not introduce the first time-driven
   behaviour into a backend that has none (`DESIGN.md` 1b-7). Closing advances the queue to the
   next deed, or ends the auction when the queue is empty.
5. **Settlement, inside one `session.WithTransaction`**: deed to the winner, winning bid off the
   winner's balance, lot cleared. **Re-check the winner's balance here** — D17's second check,
   which exists because the winner can have paid rent between bidding and the hammer. **Do not
   call `controllers.PurchaseProperty`** — two bare writes, no session, no floor (1b-5). Write
   the pair explicitly, or extract a session-taking version and leave the existing function
   alone.
6. **A lot nobody bid on returns to the bank** (D16) — the same write Phase 1 makes for the
   `BANK` disposition, **including D13's raze**, so an unsold developed deed does not arrive in
   Bank's Properties with a hotel on it.
7. Event history rows for start, each close and each settlement — prose, like everything else
   (`DESIGN.md` §9 rule 7).
8. Tests for every rejection in step 3, plus close-by-a-non-banker and close-of-an-empty-queue.
   All before the first Mongo call, so they are reachable at all (`config.DB` is nil in a test
   binary).

**Subagents.** One **Deep (Fable 5.1)**, own worktree, given the close/winner/settlement code
and this question: *name every interleaving of two bids and a close that produces a wrong winner,
a double settlement, or a deed transferred without payment.* Bounded to those functions, verdict
only. **D14 shrinks its job usefully** — with no timer there is no clock-versus-bid race to
reason about, so the remaining surface is two concurrent `PLACE_BID`s and a `CLOSE_AUCTION`, all
of which arrive as websocket frames on separate goroutines. One **Mechanical (Sonnet 5)** sweep
to enumerate every existing `Room` read and decode before the schema grows — now warranted
rather than optional, since D18 grows `Room`.

**Done when.** The backend gates as in Phase 1, plus a test count above Phase 1's. **Proof a
green gate cannot supply:** three devices in a throwaway production room — kick a player holding
**two deeds, one of them developed**, with `AUCTION`, and observe on screen that (a) the deeds
come up **one at a time in board order** (D16); (b) a $0 opening accepts a $1 bid and **rejects**
a bid equal to the current high (D15); (c) a bid above the bidder's balance is **refused with a
red toast** rather than accepted (D17 — the `ERROR` path is on `main`); (d) two bidders bidding
against each other end with exactly **one** winner charged exactly the winning bid and the
loser's balance **unchanged** — record all balances before and after, by number; (e) the banker's
close is what ends each lot, and nothing ends it on its own if everyone waits (D14); (f) a bidder
who reloads mid-auction sees the current high bid, not a blank panel (D18 — this is the check
that justifies the schema field); (g) the developed deed, if it goes unsold, lands in Bank's
Properties **razed** (D16 + D13). Then the same run with **nobody** bidding on either deed.

**Watch for.** Everything in the driver note above, plus: **a bid is the first message type in
this app that is not idempotent and not rare.** `page.tsx:121-132` toasts *every* non-`ERROR`
message and refetches players on each one — twenty bids is twenty toasts and twenty room-wide
refetches from every client (§3's dial). And the auction is the first shared mutable object in
the product: two clients acting on one document at once is a thing this codebase has never had
to be correct about.

---

### Phase 4 — Auction UI, frontend

**Status: BUILT 2026-09-17 — HANDOFF 35. UNCOMMITTED**, in worktree
`.claude/worktrees/kick-phase4` (branch `worktree-kick-phase4`, cut from `f999188` and
fast-forwarded onto `main` at `dafd16d`, which is backend-only and shares no file with this
diff). Eight files, six modified and two new, 241 insertions. Frontend gates green in that
worktree: `tsc --noEmit` 0, `bun run lint` 0 (76 files, 0 errors, 0 warnings — two more files
than before, which is exactly the two new components), `bun run build` 0 with 10 routes.
**Its three-device walk is owed**, and so is one wire check no gate here can make; see *As
built* below.

**As built.** `frontend/types/schema.ts` (the `Auction` type, `Room.auction`),
`frontend/types/events.ts` (`AuctionStarted`, `BidPlaced`, `AuctionLotClosed`),
`frontend/types/payloads.ts` (`PlaceBidPayload`, `CloseAuctionPayload`, both added to the
`WebSocketPayload` union), `frontend/lib/utils/sendWsMessage.ts` (the two new type literals, in
both unions), `frontend/app/room/[code]/page.tsx` (the two senders, the `BID_PLACED` branch,
`AUCTION_LOT_CLOSED` added to `refetchProperties`), `frontend/components/room/room.client.tsx`
(the bar, inside the sticky header), and two new files —
`frontend/components/room/auction-bar.tsx` (the strip, the drawer, and the id → deed resolution)
and `frontend/components/room/auction-panel.tsx` (the lot, the bid control, the hammer).
Phase 3 already landed the `AUCTION` option in the disposition picker, so BD-6 needed nothing
here.

**Both jobs Phase 3 named were done:** `AUCTION_LOT_CLOSED` is in `page.tsx`'s
`refetchProperties` list, and `schema.ts` carries the `auction` field.

**Four decisions the scope did not name.** Each is reversible and each is recorded because a
later reader should not have to guess whether it was considered.

1. **The panel does not say "deed 2 of 4", and cannot.** The scope asks for the position in the
   queue. The total is only ever stated in `AUCTION_STARTED`'s `lotCount`, which is a transient
   broadcast; `models.Auction` carries the remaining queue and nothing else, so a client that
   reloads mid-auction has no way back to it — and a reload mid-auction is this phase's own
   *watch for*. A count derived from `1 + queue.length` is right for the first lot and wrong for
   every one after it. The panel names the deeds still to come instead ("Still to come: Park
   Place · Boardwalk", or "Nothing — this is the last lot"), which is reload-safe and answers
   the question the scope gives as its reason: *a player needs to know what is still coming*.
   **Reversal:** add a lot number to `models.Auction`; it is a backend change, a redeploy, and
   `websocketManager.go` again, so it was raised rather than folded (`PASSOFF.md` row 27).
2. **The bid control refuses an unaffordable bid on the client**, rather than sending it and
   rendering the server's red `ERROR` toast. The three rules mirror `bidRejection`'s in its own
   order. `free-parking.tsx` is the precedent — it pre-checks insufficient funds before sending.
   **Consequence for the walk:** Phase 3's done-when (c) expects a red toast for a bid above the
   bidder's balance, and through this UI that bid can no longer be sent. The server path is
   still reachable and still worth walking: bid the whole balance, pay rent, then hammer (D17's
   *second* check), or let two bidders race. **Reversal:** delete the third clause of `refusal`.
3. **The Banker's close is one tap plus an inline Confirm**, matching the dial that made the
   kick itself a confirmed action and `player-card-content.tsx`'s banker writes. The confirm is
   held against the high bid it was opened on, not a boolean, so a bid landing while it is open
   closes it and the Banker taps again against the new number — there is no window in which
   Confirm means something other than the sentence above it. **Reversal:** call `closeLot`
   straight from the button.
4. **The sheet does not open itself when an auction starts.** `AUCTION_STARTED` already toasts,
   and a sheet thrown over someone mid-transfer is a worse interruption than a bar they can see.
   **Reversal:** one effect on `room.auction` appearing.

**How the no-toast, no-refetch bid works, since it is the one non-obvious thing here.**
`BID_PLACED` leaves `handleWebSocketNotification` before both the toast and `refetchPlayers`
(§3's dial). The panel stays current because the payload carries the whole of the new high bid
and is applied as an overlay, taken as the **larger** of the overlay and the room's stored
`highBid` — monotone within a lot, so a refetch landing between a broadcast and its write cannot
walk the number backwards, and an overlay from a closed lot is ignored rather than cleaned up.
The overlay is keyed on `(kickedPlayerId, propertyId)`, captured from the auction the bid was
accepted against rather than from the payload, which does not carry the kicked player: the same
deed can be the open lot of two different auctions, and a stale high bid from the first would
otherwise sit on top of the second's $0 opening. **The phase's *watch for* is answered
explicitly**: a `BID_PLACED` naming a lot this client does not have open is the one bid that
does pay for a `refetchPlayers`, because it means a broadcast was missed and the fetch is the
only transport that can repair it. It cannot loop — the refetch brings the lot the bids are on.

**Verified in a browser at 375px, and what that does and does not prove.** The deployed
backend's CORS and websocket origin allowlist name `http://localhost:3000` literally
(`backend/main.go:22-26`, `websocket/handler.go:18`) and port 3000 on this machine was held by an
unrelated project, so the run was driven against a **throwaway local stub** of the five endpoints
this screen touches, not against `api.emoney.club`. **What it proves:** every branch of the panel
renders and every interaction fires — a bid raises the high bid with no toast and, confirmed
against the network log, **no refetch**; the field follows the minimum and stops on a keystroke;
all three refusals disable the button with their sentence; the Banker's button reads "Sold to
Carol for $4" and "No bids — return it to the Bank" as the state changes; the confirm settles and
advances the lot in `PropertyIndex` order (Oriental 6 → Park Place 37 → Boardwalk 39); the
"bidding on the deed alone" note appears on the developed, mortgaged lot; closing the last lot
ends the auction and the bar and sheet disappear together; a full reload mid-auction shows the
live high bid (D18's whole justification); the kicked player is told "This is your estate. You
can't bid on it."; a non-Banker is told the Banker closes each lot; and a `BID_PLACED` for an
unknown lot recovers by refetching. **What it does not prove:** that the real Go handlers accept
these two payloads. **Nothing in this repo has checked that the frontend's `PLACE_BID` /
`CLOSE_AUCTION` payload keys match the server's**, in either direction — that is `CLAUDE.md`'s
standing warning about this contract, and it is the first thing the device walk will find if a
key is wrong.

**Scope.** The bidding panel (a `Drawer`, following the house pattern); the bid control, opening
at $0 with $1 raises (D15); the live high bid and bidder, and the position in the queue — *deed 2
of 4* — because D16 auctions them one at a time and a player needs to know what is still coming;
**the banker's "Sold" control**, which is the only thing that closes a lot (D14) and therefore
the one control the auction cannot work without; the **suppression of the per-bid toast** and of
the per-bid `refetchPlayers`, which is a real change to `page.tsx`'s message handler and not a
cosmetic one; the `AUCTION` option finally appearing in Phase 2's disposition picker (BD-6);
payload and event types on both sides.

**Subagents.** None.

**Done when.** The frontend gates as in Phase 2. **Proof a green gate cannot supply:** Phase 3's
three-device run, driven entirely through the UI at 375px, plus one deliberate reload mid-auction
on a bidder's device.

**Watch for.** This is the first screen that renders fast-changing shared state, and its only
transport is a broadcast plus a refetch — so a dropped message means a stale panel with no
visible error. Decide what the panel does when it has no auction state and a `BID_PLACED` arrives
(refetch, then render) rather than discovering it on a device.

---

### Phase 5 — Ship, walk, record

**Status: NOT STARTED. Waits on Phase 4, and on Zach for both deploys.**

**Scope.** Print the deploy commands — never run them (`CLAUDE.md`: wrangler-equivalent rules
apply here too; the backend script deploys for real, and a push to `main` deploys the frontend).
Walk `RUNTIME-PASS.md` with Zach. Write the `As built:` paragraphs under each `D<n>` that the
build deviated from, and the `HANDOFF.md` steps at the **next free number read from the file**.
Then Stage 8: grep for referrers, archive the folder to
`~/Projects/archive/emoney/kick-player/`, index it, and move anything in `DESIGN.md` §9 that
still governs the code into "standing rules that outlived their doc".

**Subagents.** None; Mechanical (Sonnet 5) is fine for the referrer grep if the phase runs long.

**Done when.** `api.emoney.club` answers before and after the backend deploy; `systemctl show
emoney` reports a fresh `ActiveEnterTimestamp`; the runtime pass is walked and its findings are
**new board items, not ad-hoc fixes**; `PASSOFF.md` row 7 reads `DONE — HANDOFF <n>`.

**Watch for.** "Merged is not shipped" for the backend. And a deploy drops every socket in every
live room — clients reconnect in 1s, but a game mid-turn sees toasts and a flicker.

---

## 3. Dials

Every number and default the design leaves open. Each was a `GATE 2` item; none is a decision.

**All eight accepted at their recommended defaults, 2026-09-16 at `GATE 2`** — including the
fifth row's "nothing new for v1" for the kicked browser, which means **no "you were removed"
screen in Phase 2** even though D11 made one buildable. A phase that wants to deviate from a
column below is deviating from an answered gate (R12), not filling in a blank.

| Dial | What it controls | Recommended default |
|---|---|---|
| Confirmation before a kick | Whether the kick needs a second tap | **Yes, a `Dialog` naming the target** — every banker action with a Mongo write is already one tap plus a `Confirm` (`player-card-content.tsx:105-135`), and a kick is less reversible than a balance change |
| Preselected disposition | What the picker opens on | **Return to bank** — the only option that leaves the board fully playable, and the one D2's argument-against exists to make cheap |
| Event history entry for a kick | Whether a kick is logged | **Yes** — `rm.CreateEventHistory` is one call and is the pattern for all six other mutation paths; skipping it makes the kick the one action the FAQ-promised audit log (`app/help/page.tsx:17`) cannot see |
| The kick's event icon | Which colour/emoji pair the row gets | **Add a case** to `CreateEventHistory`'s switch (`websocketManager.go:517-532`). Left alone, a notification containing "Banker" falls into the 🏦 arm by substring — which is not wrong, just indistinguishable from a balance change |
| Where the kicked browser lands | What the client does once its socket is gone | **Nothing new for v1** — D6 means no message arrives, so the only honest behaviour is what already happens. ~~Note what D11 makes of that: the reconnect's `JOIN` is refused and `GetPlayersInRoom` no longer returns them, so the room fetch fails and `DataState` renders its error branch.~~ **Corrected 2026-09-17 (HANDOFF 20): the second half is wrong.** D11 was built as *flag, not filter* (Phase 1 step 6), so `GetPlayersInRoom` still returns a kicked player and **their room fetch succeeds** — the room renders, and their own card shows Phase 2's "No longer in the game" treatment like everyone else's view of it. The `JOIN` refusal half stands. The dial's answer is unchanged and no new screen was built; what changed is what "nothing new" actually looks like, and it is more honest than the error branch would have been. A "you were removed" screen is now buildable (D11 settled the data question) but is **not** in scope for Phase 2 |
| Auction minimum increment | The smallest raise | **$1 — decided, not a dial any more** (D15). Kept in this table so that "why $1" resolves to a decision rather than to nothing. $10 would be a dated amendment to D15, not a config change |
| ~~Auction countdown~~ | ~~Seconds from the last bid~~ | **Removed by D14 — there is no countdown.** The banker closes each lot by hand. Struck rather than deleted so that a later reader does not re-propose a timer thinking it was simply never considered |
| Toast per bid | Whether each `BID_PLACED` raises a toast | **No** — `page.tsx:121-132` toasts and refetches on every message at `87e0821`; twenty bids would be twenty of each, from every client. Still a dial: D14 means bids arrive only while a lot is open, so the blast radius is bounded but not small |

---

## 4. Seams reserved, deliberately not built

So the next effort need not guess whether an omission was considered.

- **`SendTo(room, playerId, msg)` — a targeted send.** Not built: D6 means the kicked player is
  told nothing. Phase 1's `CloseClientByPlayerID` is the same scan over the same map, so the
  follow-on is a few lines in the same place (HANDOFF 6 fact 13 argued the same).
- **A general `SET_BANKER` / hand-over-the-banker-role action** (idea 12). Not built: D5 scopes
  succession to *inside a kick only*. Phase 1's write is the mechanism a later `SET_BANKER`
  would reuse.
- **A general "move a deed to any player"** (idea 3). Not built. `AssignOwnerShipProperty`
  (`controllers/propertyControllers.go:53-60`) stays uncalled; the auction's settlement writes
  its own pair inside a transaction rather than adopting it.
- **A creditor disposition** — "the kicked player's estate goes to player X". Not built: D2
  superseded A3's creditor arm. It would be a third message shape and is really idea 4.
- **Server-side `isBanker` enforcement.** Not built (`DESIGN.md` §9 rule 1), on purpose, and the
  kick inherits the same trust model as every other action.
- **Structured, machine-reversible event rows** (idea 5). Not built; the kick writes prose.
- **Filtering inactive players out of the *frontend* independently of the API.** Not built:
  D11 puts the decision in one place — `GetPlayersInRoom` — rather than two, so the frontend
  renders what the API gives it.

---

## 5. Repo hazards, with live numbers

1. **Another session is rewriting `Broadcast`'s lock discipline right now, uncommitted, in Phase
   1's exact file. This is the fourth instance of the same hazard in one day — rewritten
   2026-09-16 at `GATE 2`.**

   *What this entry used to say, and why both earlier versions are closed:* `c202745`
   (HANDOFF 10, the checked-assertion precedent) merged as `49594c1`; `d1d6038` (board item 8,
   the empty-notification guard) merged as `76e9b6c` with its gates re-run on the merged tree
   (HANDOFF 11). **Every worktree branch in this checkout is now 0 commits ahead of `main`**, so
   there is no unmerged branch to sequence against and **Phase 1 branches from `main` at
   `76e9b6c` or later.** Both earlier sequencing calls were resolved the same way — land the
   finished work first — and both were right.

   *The live version, and it is a different shape:* the collision is **uncommitted working-tree
   work, not a branch**, which is why `git log main..<branch>` shows nothing and why checking
   only branches would miss it. Worktree `.claude/worktrees/agent-b62fc66a24e0336cfb0a` (at
   `76e9b6c`) holds a modified `backend/websocket/websocketManager.go` plus an untracked
   `backend/websocket/broadcast_race_test.go`; **the byte-identical change also sits in the main
   checkout's working tree** (`git diff` of the two is empty, 2026-09-16). What it does:
   `Broadcast` stops deleting from `rm.clients[room]` under `RLock()` — it collects failed
   clients into a local slice, releases the read lock, and deletes them under `Lock()`,
   re-checking that the room key still exists between the two. The new test file upgrades real
   `httptest` websockets because `Client.Conn` is a concrete `*websocket.Conn` with nothing to
   mock, and is written to be run under `go test -race`.

   **That is hazard 2 below — Phase 1's single sharpest named risk — being fixed by someone
   else, before Phase 1 starts.** It is not this item's to touch (`CLAUDE.md`: never edit another
   session's uncommitted work), and it is not this item's to claim.

   **`GATE 2`'s answer, 2026-09-16: Phase 1 waits for it to land.** The reasoning, recorded so a
   later session does not re-litigate it: a force-close is the first feature that *deliberately*
   produces the failing `WriteJSON` that triggers `Broadcast`'s delete, at the same moment as the
   target's own defer calls `RemoveClient` — so Phase 1 is the single largest consumer of exactly
   this fix, and building against the old lock discipline means the Deep review's verdict is
   stale before it is written. Starting now would also mean a real merge in `websocketManager.go`
   between two sessions' new code, which is the collision the board's "files it owns" column
   exists to prevent.

   **The trigger, for whoever starts Phase 1:** that work is committed and merged to `main`, and
   `git status --short` in the main checkout no longer lists `backend/websocket/websocketManager.go`.
   Check `git worktree list` **and** every tree's `git status --short` — not just `main..<branch>`,
   which would have shown nothing here. Then re-run §0 before reading a source file.

   **What Phase 1 should expect to find when it does start.** `Broadcast` will take the read lock
   twice-over — RLock for the fan-out, then Lock for the reap — so `CloseClientByPlayerID` (BD-4)
   must not be called while holding either, and the Deep review's question sharpens rather than
   disappears: *given the new two-phase lock in `Broadcast`, does the force-close path ever put a
   concurrent read and write on `rm.clients`, and is every lock released on every return?*
   Re-read `Broadcast` before writing `CloseClientByPlayerID`; do not copy the shape from this
   paragraph.

   **Amended 2026-09-16, second check, when Phase 1 was asked to start and the trigger had not
   fired.** `main` was still `76e9b6c` and the work was still uncommitted — **and it had grown
   past board item 9's own "files it owns" column**, which named `websocketManager.go`'s
   `Broadcast` func and one new test file. It now also modifies **`types.go`** and
   **`handler.go`**. That is the column doing its job: the row said one file, the tree says three,
   and Phase 1 owns all three.

   *What the expanded work appears to be doing* — read-only, from its worktree, and **it was
   mid-mutation-check at the time** (`Client.WriteJSON`'s body carried a
   `// TEMPORARILY REVERTED - restore before finishing.` line with the lock not taken), so none of
   this is settled and **all of it must be re-read against `main` when it lands**:

   - `types.go` grows `Client` a `writeMu sync.Mutex` and a `Client.WriteJSON(Message) error`
     method, documented as **the only write path in the package** — the rationale being that a
     `*websocket.Conn` permits exactly one concurrent writer and gorilla panics
     `"concurrent write to websocket connection"` otherwise.
   - `handler.go` rewrites **every** `conn.WriteJSON(...)` in the message switch to
     `client.WriteJSON(...)`.

   **Two direct consequences for Phase 1, both of which invalidate a step as written:**

   1. **Step 2's `case "KICK_PLAYER"` must use `client.WriteJSON`, not `conn.WriteJSON`.** Copying
      the shape of the neighbouring cases is the plan's own instruction and would, after item 9
      lands, reintroduce exactly the bug item 9 closed. Re-read the switch before copying it.
   2. **BD-4 and step 8 stand, with one thing to confirm.** `CloseClientByPlayerID` calls
      `client.Conn.Close()`, which is a close and not a write, so it does not go through
      `WriteJSON` and does not take `writeMu`. **Verify that against the landed code** rather than
      against this paragraph — if item 9 ends up routing closes through the client too, BD-4's
      "does not call `RemoveClient`" reasoning needs re-checking for lock ordering between
      `writeMu` and `rm.mu`.

   **Do not touch that work to find out.** It is another session's uncommitted tree
   (`CLAUDE.md`), and it was observed mid-mutation-check — a state that is *supposed* to be
   broken.

2. **`Broadcast` writes the clients map under a read lock** (`websocketManager.go:53-65`), and a
   force-close is the first thing that makes that path likely. Phase 1's *watch for* has the
   mechanism; the consequence is a dead process, not a failed request.
3. **`gofmt -l .` can never be empty.** It lists `models/roomModel.go` (board item 5, another
   session's file). Every phase's gate is "lists only that file"; do not format it in passing.
4. **Frontend gates only work from `frontend/`.** There is no root `package.json`; from the root
   they fail with a missing-script error that reads like a broken config. The Go gates have the
   mirror problem — from the root they print `directory prefix . does not contain main module`
   and `go build`/`go vet` **still reported exit 0** through a pipeline (observed 2026-09-16).
5. **A local `bun run build` says nothing about production.** `NEXT_PUBLIC_API_URL` lives only on
   Vercel and is inlined at build time; the tell is the `apiUrl` line in the build output.
6. **Nothing in this repo asserts that a valid action moves the right money.**
   `controllers/` has no test file, and `config.DB` is nil in a test binary, so any test that
   reaches Mongo panics rather than failing. Two existing tests *assert* that panic as their only
   signal that a value was accepted. Every money claim in this feature therefore rests on a
   device walk, not a gate.
7. **`PurchaseProperty` is two bare writes with no session and no floor**
   (`controllers/propertyControllers.go:19-35`). It is the *wrong* precedent and it is the
   nearest one; `freeParking` (`websocketManager.go:176-243`) is the right one.
8. **One backend process, deployed by hand, no drain.** A deploy drops every socket; an
   in-memory auction would not survive one.
9. **`HANDOFF.md`, `PASSOFF.md` and `docs/AGENT-PRACTICES.md` are gitignored**
   (`~/.config/git/ignore:4-6`). Never in a `git add` block. `docs/incomplete/kick-player/` is
   **not** gitignored — it is untracked and commit-able, and needs `git add -N` before a
   path-scoped commit will take it.
10. **Other sessions are editing this checkout right now, and this is not a background worry.**
    Inside one session: `HANDOFF.md` and `CLAUDE.md` both grew; `main` advanced three commits;
    one worktree's work merged; **a new worktree appeared** (`agent-abffee6918b2a9e0e`) holding
    unmerged edits to the exact file Phase 1 needs. Re-read `git status --short` **and**
    `git worktree list` immediately before printing any commit block, and never
    `git checkout --` or `git stash` to undo an experiment — copy the file aside and restore with
    `cp`.

---

## 6. Session protocol

- **`GATE 2` is answered (header). Do not re-ask it.** The one thing outstanding is §5 hazard 1's
  trigger, which is an event to observe, not a decision to request.
- The process standard is `docs/AGENT-PRACTICES.md`; read it, `CLAUDE.md` and `HANDOFF.md` before
  a phase, and this file plus `DESIGN.md` §8–§10 instead of re-deriving the decisions. That is
  ~62k tokens of fixed overhead (§0) — for a phase that runs long, `HANDOFF.md`'s standing
  sections plus steps 6 and 10 are the parts that bear on this work.
- **Ask Zach in one batch, in chat, before building** (R6). He is interactive.
- **For every word a player reads, bring 2–3 variants in different registers** (R7). The kick
  confirmation copy is unwritten.
- **Stop mid-file if something contradicts a settled decision** (R12) — `DESIGN.md` §8 and §9 are
  the settled set, and §9's whole job is to be the list a build phase does not rewrite.
- **No drive-by fixes.** This plan names five things that are wrong and not ours: `Broadcast`'s
  lock, `PurchaseProperty`'s missing session and floor, **`SELL` leaving `developmentLevel`**,
  `models/roomModel.go`'s formatting, and the raw Mongo id in the properties `DrawerTitle`.
  Raise them; do not fold them in. **The third one is the subtle case:** D13 razes development on
  the *kick* path and deliberately leaves `manager.HandlePropertySaleMortgage` alone, so
  "fixing `SELL` while I'm in here" is a drive-by that also contradicts a ratified decision
  (R12) — two reasons, not one.
- **Never run `git commit`, `git push`, `git add -A` or `git add .`** — print the two blocks.
- **Never run the deploy.** Print it; Zach runs it.
