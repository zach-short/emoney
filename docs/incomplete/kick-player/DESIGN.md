<!-- personal-config v0.2.1 · 2026-09-16 · config 1e21aa31 · standard v1.0.3 -->
# DESIGN — kick a player

**Status: RATIFIED — GATE 1 *and* GATE 1b, both answered 2026-09-16. Stage 3 complete.
`GATE 2` (the plan) pending.**

**Renamed from `SCOPE.md` on 2026-09-16** by the Stage 3 session, per `docs/AGENT-PRACTICES.md`
§2.2. A plain `mv`, not `git mv`: `docs/` is entirely untracked
(`git status --short docs/` → `?? docs/`, 2026-09-16), so git records no transition and nothing
in this folder has ever been committed.

**Sections 1–7 are the Stage 1/2 record, preserved as written.** §1b amends §1 where the Stage 3
pass re-verified it and found it moved or incomplete. §8 is the frozen decisions `D1…D9` (GATE
1). §9 is what does *not* change. §10 is the **GATE 1b question set** — asked and **answered
2026-09-16**; its questions, options and arguments-against are preserved as the record of what
was weighed, and every one of them is **superseded by §11**, which freezes the answers as
`D10…D18`. §12 is the final ground-truth re-check that closed Stage 3. `PLAN.md` beside this
file is Stage 4.

**The whole decision set is `D1…D18` and it is frozen.** It changes by amendment — a new dated
`D<n>` or an `As built:` note under the decision it deviates from — never by editing a decision
in place. **Nothing in §3, §4, §6 or §10 is open any more**; read them as history.

**Opened 2026-09-16** out of Zach in chat: *"i need to deploy emoney with the
ability to kick a player to fix that"* — "that" being a stuck player named `Claude` in
production room `game`, blocking unrelated work in the portfolio repo. This folder is the
`/scope` session board item 2 (`HANDOFF.md` step 6) recommended before building idea 1: **"it
cannot be built without first deciding where a removed player's money and deeds go — which is
exactly the decision a scope session exists to take, and exactly what board item 2 was told not
to decide alone."**

**No worktree used for Stage 2 or Stage 3** — read-only research plus this doc and `PLAN.md`; no
source file touched in either session (`git status --short`, 2026-09-16, shows only the
pre-existing untracked/modified entries plus this untracked `docs/` tree).

---

## 1. What exists, verified 2026-09-16

All re-verified this session at `main` `e837445` — HANDOFF step 6 (`HANDOFF.md:498-805`) is a
lead, not fact, and moved main from `591599f` in the time since; three of its citations are
corrected below (rows 3, 9, 11).

| # | Claim | Verified state | Citation |
|---|---|---|---|
| 1 | No kick/remove-player route exists | `backend/routes/routes.go:9-41` has `room.DELETE("")` (whole room) and `property.DELETE("")` (unowns one property). No `player.DELETE`. | Read in full, 2026-09-16 |
| 2 | `Player.IsActive` exists, is written `true` on create/join, read nowhere | `backend/models/playerModel.go:12`; writes at `roomControllers.go:60,185`; `grep -rn "IsActive" backend frontend` returns only those two writes and the struct field | 2026-09-16 |
| 3 | **Corrected from HANDOFF step 6.** `DeleteRoom` is NOT in `roomControllers.go:229-243` — it is in `backend/controllers/playerControllers.go:229-243`. `roomControllers.go` (233 lines) holds only `CreateRoom`, `JoinRoom`, `CheckIfRoomCodeExists`. | `grep -rn "func DeleteRoom" backend/` → one hit, `playerControllers.go:229` | 2026-09-16 — HANDOFF step 6's file:line for this fact was wrong; the behavior it described (deletes one `Room` doc, orphans `Player`/`Property`/`EventHistory`, ignores `DeletedCount`) is otherwise unchanged, read in full |
| 4 | The room-join 409 is a **name-or-color collision, not a player cap** | `roomControllers.go:166-178`: `CountDocuments` with `$or: [{name: X}, {color: Y}]` scoped to the room; `>0` → 409 "Name or color already taken". No count-of-players check anywhere in `JoinRoom`. | 2026-09-16, read in full — **this settles the portfolio repo's open question** ("ask whether that is a player cap"): it never was one. Rejoining as `Claude` 409s because a player named `Claude` (or holding `Claude`'s color) already exists in the room, not because the room is full. |
| 5 | **Live production state of room `game`, right now.** 5 active players: `Zach` (banker, $1500, 0 properties), `John` ($30,840, 4 properties), **`Claude` ($1500, 0 properties)**, `sally` ($1340, 2 properties), `Jacob` ($1500, 0 properties). | `GET https://api.emoney.club/v1/rooms/game/players`, run 2026-09-16, read-only | **`Claude` — the stuck player — owns zero properties and holds only its starting cash.** The general "what happens to a kicked player's estate" question has a live blast radius of zero properties and $1500 for this specific case. |
| 6 | `RoomManager` cannot look up a live socket by player id | `backend/websocket/websocketManager.go:22-25`: `clients map[string]map[*Client]bool` — keyed by room code then by `*Client` pointer, no index by `PlayerID`. `Client.PlayerID` exists (`types.go:5-10`) but nothing scans for it. | Read in full, 2026-09-16 |
| 7 | Closing a client's socket from the manager already triggers full cleanup, for free | `handler.go:31-64`: the per-connection goroutine blocks in `conn.ReadJSON` (`:67`); the `defer` at `:54-64` already calls `Manager.RemoveClient`, `conn.Close()`, and broadcasts `PLAYER_LEFT` with `client.PlayerID`/`client.PlayerName`. Force-closing a target's `conn` from another goroutine (`gorilla/websocket`'s underlying `net.Conn.Close` is documented safe to call concurrently with a blocked `Read`) makes that goroutine's next `ReadJSON` return an error, `break`, and run the existing defer unchanged. | Read in full, 2026-09-16. **A kick therefore does not need new cleanup/broadcast logic** — only (a) a way to find the right `*Client` by `PlayerID` in the room's map, and (b) whatever new Mongo write disposes of the kicked player's data. |
| 8 | No per-player-targeted send exists — `Broadcast` is room-wide only | `websocketManager.go:53-65` is the only fan-out method | Re-confirmed, 2026-09-16. Relevant only if a kicked player should see a distinct message before disconnect (open question below); not required for a bare kick. |
| 9 | **Corrected from HANDOFF step 6's framing.** Nothing enforces `isBanker` server-side, on any action, including the one that exists today | `grep -rn "IsBanker\|isBanker" backend/` → 3 hits total: the struct field and the two writes at room-create/join time (`roomControllers.go:59,184`). Zero reads. | Re-grepped 2026-09-16, unchanged from HANDOFF step 6 fact 9 — carried forward as confirmed, not corrected; listed to make clear it was independently re-verified, not assumed. **A kick endpoint inherits this: whoever holds the room code can call it, same as every other action today.** |
| 10 | "Delete My Player this Game" / "Delete My Players in All Games" are local-only, by design, and unrelated to a real kick | `frontend/lib/utils/playerHelpers.ts:23-35` (`clearPlayerDataForRoom`, `clearAllPlayerData`) only touch `localStorage`; wired from `navbar.tsx:205-229`. A comment already left in the code (`navbar.tsx:215-217`) documents a prior fix to *which* key gets cleared — this control was never meant to reach the server. | Read in full, 2026-09-16. **This is not the bug to fix** — it's a working "forget this device" reset. The portfolio repo's read of it as broken (`PASSOFF.md`) is not wrong about the symptom (no network request) but wrong about the cause: it isn't misfiring, it's scoped to the browser on purpose. |
| 11 | **Corrected from HANDOFF step 6's fact 7 citation, one detail.** `SELL` reassigns a property to the bank (`playerId: nil`) and exists server-side; the frontend control for it is commented out. | `backend/manager/propertyManager.go:52` (`case "SELL"`); `frontend/components/players/manage-properties.tsx:138-141,346` (`handleSellToBank` and its button, both commented). Re-verified 2026-09-16, matches HANDOFF step 6 fact 7 exactly — no correction needed, confirmed independently. | This is the existing, working shape for "give a kicked player's properties back to the bank" — reuse, not new code, per §3 Option A. |
| 12 | **Two other sessions are live in this repo right now, one on a file this work would need.** | `git worktree list`: `.claude/worktrees/agent-a2f874037ff8da39a` (locked) has an uncommitted +134/−19 diff against `main` in `backend/websocket/websocketManager.go` **and** `websocketManager_test.go` — matches board item 3, "close `freeParking`'s missing `default` arm" (`PASSOFF.md` row 3, `OPEN`, files `websocketManager.go`, `websocketManager_test.go`). `.claude/worktrees/error-toast` has an uncommitted +22/−4 diff in `frontend/app/room/[code]/page.tsx` — matches idea 10 from HANDOFF step 6 (the `ERROR` toast). | `git worktree list` and `git diff main --stat` in each, 2026-09-16 | **Hazard — see §5.** A kick feature's natural home is exactly these two files. |
| 13 | Websocket contract is typed frontend-only, nothing checks the two sides agree | `backend/websocket/types.go:12-15` (`Message{Type string, Payload interface{}}`); `frontend/types/events.ts`, `payloads.ts` declare the real shapes | Re-confirmed 2026-09-16 | A new `KICK_PLAYER` type/payload is new surface on both sides with the same unchecked-agreement cost as every existing message type. |
| 14 | Deploy is two separate, manual-ish paths | `CLAUDE.md`: frontend auto-deploys to Vercel on push to `main`; backend is "one GCP e2-micro VM behind Caddy, deployed **by hand**" via `EMONEY_HOST=emoney ./backend/deploy/deploy.sh`, and **exactly one backend instance, ever** — room membership lives in process memory (`websocketManager.go:23`), so a second replica silently splits rooms. | `CLAUDE.md` *Architecture* and *Never do this*, re-read 2026-09-16 | A kick feature needs both a backend deploy (new route, new websocket case) and a frontend deploy (new UI, new message type) to actually reach production — "deploy emoney" is two steps, not one, and the backend step is never automatic. |

**Rows 5 and 12 have moved, and rows 2, 4 and 11 were incomplete in a way that changes the
design. See §1b.**

---

## 1b. Stage 3 re-verification, 2026-09-16 — supersedes §1 where they differ

Run by the Stage 3 session at `main` `5f30fd9`, three commits past the `e837445` §1 was written
against. Everything below was read, grepped or requested in this session; §1 is a lead, not
fact (R3).

| # | Claim | Verified state | Citation |
|---|---|---|---|
| 1b-1 | **§1 row 12 has moved twice.** `main` is now `5f30fd9`. The `error-toast` worktree is **gone and its work is merged** (`13615b0`, merged as `5f30fd9`). The other worktree is **no longer uncommitted and no longer locked**: `.claude/worktrees/agent-a2f874037ff8da39a` holds a clean tree at commit **`c202745`** ("reject an unknown free parking type, and check every payload string" — HANDOFF 10), which is **not merged into `main`**. | `git log --oneline -6 main`; `git worktree list --porcelain`; `git status --short` in that worktree (empty); `git log --oneline main..worktree-agent-a2f874037ff8da39a` → `c202745` | 2026-09-16. `git diff main --stat` in that worktree: `websocketManager.go` +134/−20, `websocketManager_test.go` +219/−2 **ahead** of main, and `frontend/app/room/[code]/page.tsx` and `scripts/emoney` **behind** main. So merging it is a real merge in both directions, not a fast-forward. **The hazard is no longer "a diff will drift" — it is "the precedent this build must copy only exists on that branch."** See `PLAN.md` §5. |
| 1b-2 | **§1 row 2 is right that `IsActive` is read nowhere, and that is a bigger problem than "the field already exists for this".** Marking a player inactive changes **nothing** anywhere: three separate read paths ignore it. | `grep -rn "isActive\|IsActive"` over `backend/` and `frontend/` (excluding `node_modules`) → 5 hits total: the Go struct field (`models/playerModel.go:12`), the two writes (`controllers/roomControllers.go:60,185`), and two frontend type declarations (`frontend/types/schema.ts:10,21`). **Zero reads, in either tree.** | 2026-09-16. The three paths that would each need teaching: (a) `handler.go:73-101` — `JOIN` fetches the player by id and broadcasts `PLAYER_JOINED` with no active check, and the browser reconnects 1s after any close (`frontend/app/room/[code]/page.tsx:219-223`), so **a mark-only kick undoes itself**; (b) `controllers/playerControllers.go:34,62` — `GetPlayersInRoom` queries `{"roomId": room.ID}` with no filter, so the gone player's card keeps rendering; (c) see 1b-3. |
| 1b-3 | **§1 row 4 is right about the 409 and wrong about the fix.** `JoinRoom`'s collision count has **no `isActive` filter**, so flipping `IsActive` to `false` does **not** free the kicked player's name or colour — §2's stated goal ("freeing their name/color") is not achieved by a mark alone. | `controllers/roomControllers.go:167-173`: `CountDocuments` on `{"roomId": room.ID, "$or": [{"name": …}, {"color": …}]}` | 2026-09-16, read in full. **This is why §10 `GATE 1b` Q2 (delete the document, or mark it and teach the reads) is a real fork and not a build detail.** |
| 1b-4 | **§1 row 11 is right that `SELL` is the return-to-bank shape, and incomplete about what it leaves behind.** `SELL` sets `playerId: nil` **and** `isMortgaged: false` — and leaves `developmentLevel` exactly as it was. | `backend/manager/propertyManager.go:53`: `bson.M{"$set": bson.M{"playerId": nil, "isMortgaged": false}}`; `developmentLevel` appears only in `HandleHouseManagement` (`:25`) | 2026-09-16, read in full. Consequence: a kicked player's hotel-bearing property returned to the bank **keeps its hotel**, reappears in `GetAvailableProperties` (which filters `{"roomId", "playerId": nil}`, `controllers/propertyControllers.go:75-78`), and the next buyer pays only the payload's `price` (`websocketManager.go:266-293` → `controllers.PurchaseProperty`) — **inheriting the development for free.** Pre-existing in `SELL`, unreachable from the UI as of 2026-09-16 (HANDOFF 6 fact 7); **a kick makes it reachable for the first time.** §10 Q4. |
| 1b-5 | **The money path a live auction would settle through is non-transactional and has no floor.** `controllers.PurchaseProperty` is two separate `UpdateOne` calls with no session and no balance check: it `$set`s `playerId` then `$inc`s `balance` by `-price`. | `backend/controllers/propertyControllers.go:19-35`, read in full | 2026-09-16. Two consequences for the auction: a winner can be taken **negative**, and a failure between the two writes leaves the deed transferred and unpaid. The app's *other* multi-document write does use a session — `freeParking` (`websocketManager.go:176-243`) — so a correct precedent exists in the same file. **Do not reuse `PurchaseProperty` for auction settlement.** |
| 1b-6 | **`Broadcast` writes to the clients map while holding only a read lock.** On a write error it calls `client.Conn.Close()` and `delete(clients, client)` inside `rm.mu.RLock()`. | `backend/websocket/websocketManager.go:53-65`, read in full; `RemoveClient` at `:42-51` takes `rm.mu.Lock()` | 2026-09-16. **This is the sharpest hazard in the whole feature.** A kick force-closes the target's socket; the target's own goroutine then runs the defer at `handler.go:54-64`, which calls `RemoveClient` (write lock) — while any concurrent `Broadcast` may be deleting from that same map under a read lock. Concurrent map read and write is a Go **runtime panic**, and `gin`'s Recovery middleware does not cover a panic in a non-request goroutine: **the process dies and every room in it goes down.** At `5f30fd9` the delete branch is rare (it needs a failing `WriteJSON`); a kick is the first feature that deliberately produces exactly that condition. `PLAN.md` Phase 1 names this and assigns it a narrow Deep review. |
| 1b-7 | **There is not one goroutine or timer in the Go tree**, so an auction countdown would be the first time-driven behaviour in this backend. | `grep -rnE "time\.(After\|Tick\|NewTimer\|NewTicker\|Sleep)\|go func\|goroutine" --include="*.go" backend` → 4 hits, **all** `context.WithTimeout` (`config/db.go:98`, `controllers/roomControllers.go:223`, `controllers/playerControllers.go:232`, `services/room.go:14`). No `go func`, no timer, no ticker. | 2026-09-16 | Bears directly on §10 Q5 (does the auction close on a clock, or does the banker close it). |
| 1b-8 | **Re-confirmed: no auction or bidding concept exists anywhere.** | `grep -rniE "auction\|\bbid\b\|bidding\|highestBid" --include="*.go" --include="*.ts" --include="*.tsx"` over `backend/` and `frontend/` (excluding `node_modules`) → **zero hits** | 2026-09-16, my own grep, not carried from §7 | Same for the kick itself: `grep -rniE "kick\|eject\|removePlayer\|deletePlayer\|setBanker\|promote"` returns only test-function names containing "Rejects". |
| 1b-9 | **§1 row 5 re-verified, unchanged.** Room `game` (`6aab31d558af4d619906fa91`), 5 players, all `isActive: true`: `Zach` (banker, $1500, 0 props), `John` ($30,840, 4), **`Claude` ($1500, 0 props, id `6aab34db58af4d619906fab6`)**, `sally` ($1340, 2), `Jacob` ($1500, 0). Free parking $0, 9 event rows. | `GET https://api.emoney.club/v1/rooms/game/players`, run 2026-09-16 by this session, read-only | The blast radius of the immediate blocker is still **zero properties and $1500**. |
| 1b-10 | **Gates on `main` `5f30fd9`, run by this session.** `cd backend && go build ./...` exit 0; `go vet ./...` exit 0; `go test ./...` exit 0 — **35 tests** (`websocket` 26, `manager` 9), `[no test files]` for the other 8 packages; `gofmt -l .` lists **`models/roomModel.go`** and nothing else. | Run 2026-09-16 from `backend/`. **Note:** run from the repo root instead, all three print `pattern ./...: directory prefix . does not contain main module` — and `go build`/`go vet` still reported exit 0 through the pipeline. Run them from `backend/`. | Frontend gates **not re-run this session**; `PLAN.md` §0 carries them as unverified-since-`5f30fd9`. The branch at `c202745` carries **49** tests (`websocket` 40), which is what a Phase 1 branch cut from it starts with. |

---

## 2. What this is / what this is not

**This is:** a banker-only capability to remove a player from a room they're still a member of —
whether they left, quit, went AFK, or (as with `Claude`) got created by mistake and can't
un-join — freeing their name/color and stopping their client from acting further in the room.
It covers: marking the player gone in Mongo, disposing of whatever they hold (cash and
properties), and force-ending their live connection so a reconnect doesn't resurrect them.

**This is not:**

- **Not the estate/trade system.** Idea 3 ("give, move, or return a deed") and idea 4
  ("banker-executed trade") from HANDOFF step 6 are separate features. Kick reuses the same
  underlying primitive as idea 3 (return-to-bank) for its own estate disposal, but does not
  build a general "move a deed anywhere" capability.
- **Not a way to hand off the banker role.** Idea 12 (HANDOFF step 6) is explicitly named there
  as *"the safety net under idea 1 — otherwise kicking the banker is unrecoverable."* Whether the
  banker can be kicked at all, and whether that leaves the room bankerless, is an open question
  below — building idea 12 itself is out of scope here regardless of the answer.
- **Not adding server-side `isBanker` enforcement.** Row 9 stands as a named, not closed, gap —
  same as every other action in the app today (`HANDOFF.md`, *Settled*, "no auth, deliberately").
  A kick endpoint is exactly as trust-the-client as `BANKER_TRANSACTION` is today.
- **Not a fix to "Delete My Player this Game."** Row 10 — that control does something different
  on purpose and is not broken.
- **Not a general targeted-websocket-send primitive**, unless the "does a kicked player see a
  message first" question below is answered yes in a way that needs one (Option discussion in
  §3, question 2).
- **Not a change to `JoinRoom`'s conflict behavior** (row 4) — freeing the name/color by removing
  the old player is the fix; the 409 logic itself is correct as written.

---

## 3. Options

Two decisions drive the whole shape: **(A) what happens to a kicked player's cash and
properties**, and **(B) can the banker kick themselves, and what happens to the room if they do**.
A third, smaller one — **(C) does a kicked player get any warning** — is closer to a dial than a
real fork, covered in §4.

### A. The kicked player's cash and properties

**A1 — Return everything to the bank. (Recommended.)** Properties: reassign to no owner, the
same state `SELL` already produces (row 11) — `playerId: nil`, available for anyone to buy
again. Cash: simply discarded — the player document is what's being removed, so there is no
"the bank's balance" to credit it to (this app has no bank balance field, only `Room.FreeParking`
and per-player `Balance`; row 1500-column checked in `models/roomModel.go`). Reuses an existing,
tested code path for the property half (row 11) and needs no new Mongo write shape.
*Argument against:* if a player is kicked mid-game holding real value (say John's $30,840 and
four properties, not `Claude`'s empty hand), the other players lose access to money and property
that was "in play" a moment before — an eliminated Monopoly player's assets normally go to
whoever bankrupted them, or back to the bank if nobody did. Silently vaporizing $30k reads as a
bug to anyone who didn't design it, even though it is exactly what "the bank" means when there's
no bankruptcy-by opponent to credit.

**A2 — Freeze in place (do nothing to balance/properties, just mark them gone).** The simplest
possible write: flip `IsActive` to `false` (the field already exists for this, row 2) and stop
there. Properties keep `playerId` pointing at a document that no longer answers. *Argument
against:* this is worse than A1 for exactly the case that motivated the feature — `app/help/page.tsx:51-53`'s
own FAQ copy is *"if a player leaves, their assets remain in the game unless the banker removes
them,"* i.e. A2 **is** the manual workaround already described as a problem, just automated. It
also leaves 28 "ownable" properties (row/board-size limited) permanently unavailable if enough
players get frozen this way over a game's life.

**A3 — Let the banker choose per-kick (return to bank vs. freeze), or name a specific creditor
player.** Most flexible; matches how a table actually plays a real elimination (a bankrupt player
routinely owes a specific opponent, not the bank). *Argument against:* real size cost — a
creditor-transfer needs the same multi-document Mongo session `handleBankTransaction` already
uses (`websocketManager.go:307-376`) but for two players and N properties at once, and it is a
UI decision made under time pressure at the table, which is exactly when a banker is least likely
to want three radio buttons. This is idea 4 (banker-executed trade) wearing a kick-shaped hat,
not a small addition to idea 1.

**For `Claude` specifically, all three options are behaviorally identical** — 0 properties, $1500
of starting cash nobody has spent — because there is nothing to move (row 5). This decision only
has visible stakes for a future kick of a player already holding assets.

### B. Kicking the banker

**B1 — Refuse it. (Recommended.)** The endpoint checks `targetPlayer.IsBanker` and rejects with
an error if true, same shape as every other rejection in this codebase
(`handleBankTransaction`'s `default:` arm, `websocketManager.go:336-338`). Cheapest, and matches
row 9's reality: since nothing enforces who's allowed to call the endpoint, the one thing worth
guaranteeing is that the room can never end up bankerless by accident. *Argument against:* a
banker who genuinely needs to leave (their phone dies, mid-HANDOFF-step-6's idea 12) has no
banker-kick path and no hand-off path either — idea 12 not existing means B1 makes that scenario
strictly unrecoverable except by deleting the whole room (`DeleteRoom`, row 3).

**B2 — Allow it; the room has no banker until someone is (manually, later) made one.** No new
code beyond not special-casing `IsBanker`. *Argument against:* "no banker" isn't a state anything
in this app handles — the balance +/− control (`player-card-content.tsx:140-152`) simply
disappears for everyone, permanently, until idea 12 exists to fix it. Silent and hard to notice
until someone needs the banker to act.

**B3 — Allow it, and auto-promote another player to banker in the same write.** Closes the gap
B2 opens, without waiting on idea 12's own scope. *Argument against:* "which other player"
is itself a decision (oldest? first joined? does the app even track join order distinctly from
insertion order — `models/playerModel.go` has no timestamp field, only `RoomID`), and it's
picking idea 12's shape by accident, inside a different feature, which is exactly the kind of
silent scope-merge the standard's R9 exists to prevent.

---

## 4. Dials

| Dial | What it controls | Recommended default |
|---|---|---|
| Warning before disconnect (open question C, §6) | Whether a kicked player's client shows a message before the socket closes | **No** for v1 — needs the targeted-send primitive (row 8) that does not exist; ship the bare kick first, add messaging as a follow-on if it's missed in practice |
| Confirmation dialog on the banker's side | Whether tapping "kick" needs a second tap | **Yes** — every other banker action with a Mongo write that isn't easily undone (`BANKER_TRANSACTION`) is one tap plus a `Confirm` button today (`player-card-content.tsx:105-135`); kick should be at least that guarded, arguably more since it's less reversible |
| Event history entry | Whether a kick is logged like every other money-moving action | **Yes** — `rm.CreateEventHistory` is one call, already the pattern for all five other mutation paths (row citations in HANDOFF step 6 fact 3); skipping it would make kick the one action the audit log (FAQ-promised, `app/help/page.tsx:17`) can't see |
| Where the kicked player's browser lands | What `PLAYER_KICKED` (or reusing `PLAYER_LEFT`) does client-side once the socket drops | **Redirect to `/`**, same destination "Delete My Player this Game" already uses (`navbar.tsx:212`) |

---

## 5. Hazards this work walks into

- **File collision, live, right now (row 12).** `backend/websocket/websocketManager.go` has an
  uncommitted 134-line diff in a locked worktree matching board item 3. A `KICK_PLAYER` case
  added to this file in a separate worktree will not conflict at the git level (different
  regions, probably) but **will drift out of sync with whatever item 3 lands**, and the two
  should not be built as concurrent edits to the same file without coordinating merge order.
  Same risk, smaller, for `frontend/app/room/[code]/page.tsx` against the `error-toast` worktree.
  **Recommendation carried into the eventual PLAN.md:** sequence kick-player's build phase after
  item 3 merges, or take it in the same worktree as item 3 back-to-back, not in parallel.
- **No auth means no real access control on "who can kick."** Row 9. Whatever ships here is as
  spoofable as `BANKER_TRANSACTION` is today — anyone with the room code and a `BANKER_ADD`-style
  payload already moves money; a `KICK_PLAYER` payload is the same trust model, not a new hole.
  Named so it isn't mistaken for one when this is reviewed later.
- **Exactly one backend instance, ever (row 14).** The kick's cleanup path (row 7) depends on
  the target's live socket being held by *this* process's in-memory `clients` map. This is
  already the app's standing constraint, not new — flagged only because it's the first feature
  in this scope whose correctness depends on that map directly, rather than only on Mongo.
- **The immediate driver (unsticking `Claude`) doesn't need the full feature.** Row 5: `Claude`
  has nothing to dispose of. It would be possible to unblock the portfolio repo today with a
  narrower, one-off fix (a direct Mongo delete of that one player document, or a minimal
  kick-with-no-estate-logic endpoint) without deciding §3A at all. Whether that's wanted instead
  of, or ahead of, the real feature is GATE 1 question 4 below.
- **Backend deploy is manual and un-scripted for correctness checks.** Row 14 — `deploy.sh` ships
  the process; nothing in the described tooling verifies the new route/websocket case actually
  reached the VM beyond the health check CLAUDE.md already names. Not new to this feature, but
  the first time this scope's own work depends on that path working.

---

## 6. Open questions — for the gate

1. **What happens to a kicked player's cash and properties — A1 (return to bank), A2 (freeze in
   place), or A3 (banker picks a disposition, including naming a creditor)?**
   *Recommendation: A1.* It's the smallest change, reuses code that already exists and is tested
   in production shape (`SELL`), and matches what "the bank" means in Monopoly when there's no
   specific opponent to credit. A3 is the right answer for a *trade/elimination* feature, which
   this scope explicitly is not (§2).

2. **Can the banker be kicked — B1 (no), B2 (yes, room goes bankerless), or B3 (yes, auto-promote
   someone)?**
   *Recommendation: B1.* Cheapest, matches the trust model already in place, and doesn't
   silently pre-empt idea 12 (hand over the banker role) by picking its promotion rule as a side
   effect of a different feature.

3. **Does a kicked player see anything before their connection drops, or does the app just go
   dark on them?**
   *Recommendation: nothing, for v1* (the "No" default in §4) — it needs new plumbing (row 8)
   that nothing else in the app has yet, and the FAQ's own description of the problem (row 10)
   is about the *banker's* side of this, not the kicked player's experience.

4. **Does unblocking the portfolio repo's stuck `Claude` player wait for this feature to be
   designed, built and deployed end to end — or is a narrower, immediate fix (a direct removal of
   that one player, no estate logic, run once by hand) wanted first, with the real feature
   following on its own timeline?**
   *No recommendation either way stated here* — this is a sequencing call, not a design one, and
   both paths reach the same place; naming it so it isn't picked implicitly by whichever gets
   built first.

---

## 7. GATE 1 — answered 2026-09-16

Asked and answered in chat, in one batch, this date. Recorded verbatim, then interpreted —
the interpretation is this document's, not a quote, and is where Stage 3 (`DESIGN.md`) must
start, not end.

1. **Estate (cash and properties).** Zach's answer, verbatim: *"Banker chooses goes back to
   bank, properties go up on auction where there should be a live auction everyone can
   participate in, or freeze per player."* **This supersedes options A1/A2/A3 above** — none of
   the three offered was chosen as-is. The real decision: **per kick, the banker picks one of
   three property dispositions** — return to bank (A1), **a live auction open to every remaining
   player** (a new option, call it **A4**, not previously scoped — no auction concept exists
   anywhere in this codebase, `HANDOFF.md` step 6 fact 11's grep for `auction|bid` returned
   nothing), or freeze in place (A2). **Not yet answered: what happens to the kicked player's
   cash specifically** — the question asked about "cash and properties" together; the answer
   addresses properties' disposition in detail but does not separately state whether cash follows
   the same three-way choice or has its own simpler rule (e.g. always to the bank, always frozen).
   **A4, the live auction, is a materially larger feature than anything else in this scope** — it
   needs bidding state, a close condition, a winner-determination rule, and money movement to the
   winner, none of which exist in this codebase in any form (`HANDOFF.md` step 6 fact 13: no
   per-player-targeted send; §1 row 8 above, same fact re-confirmed). This is now flagged in
   `DESIGN.md`/`PLAN.md` as its own phase, not a sub-case of kick.

2. **Banker kick.** Zach's answer, verbatim: *"Yes, but they choose so[me]one else to promote."*
   **Supersedes B1 (refuse) and B2 (bankerless) — closer to B3, with one addition B3 did not
   specify:** the promotion target is **chosen by the kicking banker**, not auto-selected by any
   rule (oldest, first-joined, etc. — moot, since `models/playerModel.go` has no join-order
   field anyway, §1 row 2). This also means **idea 12 (hand over the banker role) is now
   partially in scope** — not as its own general feature, but as the specific mechanic "kicking a
   banker requires naming a successor in the same action." **Not yet answered: can a banker kick
   *themselves* this way** (the original question asked "can the banker kick themselves (or
   another banker)" as one question) — i.e. is this a way for a banker to voluntarily leave the
   game while handing off, or does "kick" only ever target someone else, with a bankerself-leaving
   flow being a different, unscoped feature.

3. **Warning before disconnect.** Zach's answer: **"No warning for now (Recommended)"** — the
   recommended option, taken as-is. No supersession.

4. **Sequencing.** Zach's answer: **"Build and deploy the real feature first"** — no narrow
   one-off fix for `Claude`. Supersedes the "no recommendation" framing: **this scope proceeds
   straight to Design → Plan → Build → Deploy for the real feature; `Claude` is unblocked only
   when the feature ships**, not before. Named consequence, not yet asked: because `Claude` owns
   nothing (§1 row 5), the *first* build phase that includes a working return-to-bank-or-freeze
   path already resolves the actual blocker, even before A4's live auction exists — this is a
   sequencing opportunity for `PLAN.md`'s phase order, not a reason to reopen this question.

**Two follow-up questions the answers themselves raised, not yet asked — carried into `DESIGN.md`
as its own gate (GATE 1b) rather than guessed at:** the kicked player's cash rule (question 1),
and whether a self-kick-with-succession is in scope alongside kicking another banker (question
2). Per `docs/AGENT-PRACTICES.md` R6/R7, real product ambiguity gets asked, not assumed —
especially here, where the answer touches live money movement (R-tier discriminator in Part 4:
"can the failure be silent").

**Stage 3 note, 2026-09-16: that list of two grew to eight.** §1b's re-verification found four
more genuine forks that GATE 1's answers do not settle and a build would otherwise settle
silently — the delete-vs-mark question (1b-2, 1b-3), whether development comes off a deed that
leaves its owner (1b-4), and two auction mechanics the answer named but did not specify. All
eight are §10.

---

## 8. Decisions — frozen 2026-09-16

Every decision below is Zach's, given in chat on 2026-09-16 and recorded in §7 verbatim before
being interpreted here. The date on each is the date it was ratified, not the date it was
written down (same day for all nine). **Amendments append; nothing here is edited in place.**

### D1 — A kick is one banker action that both disposes of the estate and ends the connection.

Ratified 2026-09-16 (implicit in §2's definition of the feature, which GATE 1 did not dispute,
and required by §7 point 4's "the real feature").

One `KICK_PLAYER` action does three things atomically from the caller's point of view: writes
the estate disposition, marks the player gone, and force-closes that player's live websocket.

**Defense.** Without the socket close the feature does not work at all: the kicked browser
reconnects one second after any close (`frontend/app/room/[code]/page.tsx:219-223`) and
re-sends `JOIN`, which `backend/websocket/handler.go:73-101` honours unconditionally. And the
close is nearly free — `handler.go:54-64`'s existing `defer` already calls `RemoveClient`,
`conn.Close()` and broadcasts `PLAYER_LEFT`, so closing the target's `conn` from the hub makes
its blocked `ReadJSON` (`:67`) error, `break`, and run that cleanup unchanged (§1 row 7).

**Argument against, recorded.** A Mongo-only kick (write the disposition, let the player's
client discover it) is smaller and touches no concurrency. It was rejected because the client
would discover nothing: `GetPlayersInRoom` has no active filter (1b-2) so the room still renders
them, and their next action still writes money. **The counter-argument is not that it is harder
— it is that it does not remove anyone.**

### D2 — Per kick, the banker picks one of three dispositions for the kicked player's properties: **return to bank**, **live auction**, or **freeze in place**.

Ratified 2026-09-16. Zach, verbatim (§7 point 1): *"Banker chooses goes back to bank, properties
go up on auction where there should be a live auction everyone can participate in, or freeze per
player."*

**Supersedes §3A entirely.** A1 (always bank), A2 (always freeze) and A3 (banker picks, including
naming a creditor) were all offered and **none was chosen as offered**. What survives of each:
A1 becomes one of the three options, A2 becomes another, and A3's *shape* — a per-kick banker
choice — is adopted while A3's *content* — naming a creditor player — **dies**: no creditor
disposition exists in this design and none is built (§9, and `PLAN.md` §4). The third option,
the live auction, is new at GATE 1 and was never scoped as an option; §7 calls it **A4**.

**Defense.** The right disposition is a fact about the table, not about the software: a player
who quits on turn three and a player bankrupted on turn forty want different answers, and the
banker is the only one who knows which. The alternative — one global rule — is wrong for one of
those two cases every time.

**Argument against, recorded (from §3A3).** Three choices under time pressure is exactly when a
banker least wants a decision. Mitigations, decided at Stage 4 not here: the picker appears only
in the kick confirmation the banker already has to pass through, and one option is preselected
(`PLAN.md` §3 dial — default **return to bank**).

### D3 — "Return to bank" is the existing `SELL` write, reused, not a new one.

Ratified 2026-09-16 as the mechanism of D2's first option.

The property write is `{"$set": {"playerId": nil, "isMortgaged": false}}` — byte for byte what
`manager.HandlePropertySaleMortgage`'s `SELL` case already does
(`backend/manager/propertyManager.go:52-53`).

**Defense.** It is the one estate-disposal path in this codebase that already exists and is the
only one whose *effect* is already proven: `GetAvailableProperties` filters on `playerId: nil`
(`backend/controllers/propertyControllers.go:75-78`), so this write and no other is what makes a
deed reappear in Bank's Properties. A new "release" write would duplicate it and could diverge.

**Two stated consequences, not open questions.** (a) A mortgaged deed comes back to the bank
**unmortgaged** — that is what `SELL` does, and it is right: the mortgage is a debt to the bank
and the bank now holds the deed. (b) `developmentLevel` is **not** touched by that write
(verified 1b-4), which is a live problem and is §10 Q4 — **the only part of this decision that
is not settled.**

### D4 — "Freeze in place" writes nothing to the player's balance and nothing to any property.

Ratified 2026-09-16 as the mechanism of D2's third option. The player is marked gone (see §10
Q2 for what "marked" means) and every `Property.playerId` and the `balance` field are left
exactly as they are.

**Defense.** It is the disposition the app's own shipped copy already describes — `app/help/page.tsx:51-53`:
*"if a player leaves, their assets remain in the game unless the banker removes them"* — so it
is the state a table already understands, and it is the only option that keeps a record of who
held what.

**Argument against, recorded (§3A2).** Frozen deeds are unavailable for the rest of the game;
enough frozen players and the board stops being playable. Accepted, because under D2 the banker
chooses it deliberately per kick rather than inheriting it as a global rule.

**Consequence.** Freeze requires the kicked player's document to survive, because
`Property.playerId` must still resolve to a name — which is half of §10 Q2's answer.

### D5 — A banker can be kicked, and only by naming a successor in the same action.

Ratified 2026-09-16. Zach, verbatim (§7 point 2): *"Yes, but they choose so[me]one else to
promote."*

Kicking a player whose `isBanker` is true requires a `successorPlayerId` in the same
`KICK_PLAYER` action. The kick and the promotion are one write: target `isBanker: false`,
successor `isBanker: true`.

**Supersedes B1 and B2 outright** — the room may not be left bankerless, and the kick is not
refused. **Partially supersedes B3:** B3's "allow it and auto-promote" survives, B3's
*auto*-selection rule dies. The promotion target is named by the kicking banker, never derived.

**Defense.** A bankerless room is a state nothing in this app handles: the only banker control
in the product is gated on `currentPlayer?.isBanker`
(`frontend/components/players/player-card-content.tsx:140-152`), so losing the banker removes
the ⊖/⊕ balance controls from every client permanently. And a derived rule is not available even
if it were wanted: `models/playerModel.go` has **no join-order or timestamp field** (re-read
2026-09-16), so "oldest" or "first joined" would be insertion order dressed up as a rule.

**Argument against, recorded (§3B3).** Building succession inside a kick picks part of idea 12's
(hand over the banker role) shape as a side effect of a different feature — the merge R9 exists
to prevent. Answered: it is in scope **only** as "kicking a banker requires naming a successor",
and a general `SET_BANKER` is explicitly not built (§9, `PLAN.md` §4). If idea 12 is later
scoped, it inherits this mechanic rather than being constrained by it.

### D6 — The kicked player is told nothing before their socket closes.

Ratified 2026-09-16. Zach chose the recommended option as offered (§7 point 3: *"No warning for
now"*); §6 question 3 and §4's first dial. **No supersession.**

**Defense.** It needs plumbing nothing in the app has: `Broadcast` (`websocketManager.go:53-65`)
is the only fan-out and it is room-wide; the only other write is back to the *sending* socket
for errors (`handler.go:104-136`). A targeted send is a new primitive, and this feature does not
need one.

**Argument against, recorded.** The kicked player's screen simply goes dark, which reads as a
bug to them. Accepted for v1 and revisitable: the by-`PlayerID` scan Phase 1 adds is the same
scan a `SendTo` would use, so the follow-on is small (`PLAN.md` §4).

### D7 — The live auction is open to every remaining player in the room.

Ratified 2026-09-16 as the explicit content of §7 point 1's answer: *"a live auction everyone
can participate in"*. Every player still in the room may bid, including the banker (who is an
ordinary player with a balance) and including the successor if this kick promoted one. The
kicked player may not, having no live socket after D1.

**Defense.** It is what the answer says, and it is the real Monopoly rule — a bankrupt player's
property is auctioned to the remaining players, not sold to a nominee. **Argument against:**
"everyone" makes the auction the first feature in this app where several clients act on one
shared object concurrently, which is the whole reason `PLAN.md` Phase 3 is sequenced last and
carries a Deep review.

### D8 — Build and deploy the real feature. No one-off removal of `Claude`.

Ratified 2026-09-16. Zach, verbatim (§7 point 4): *"Build and deploy the real feature first"*.
**Supersedes §6 question 4's deliberate no-recommendation.** No direct Mongo delete, no
throwaway endpoint, no hand-run script against production.

**Defense.** It is his call and he took it. The sequencing consequence, which §7 names and
`PLAN.md` §2 acts on: because `Claude` holds zero properties and $1500 (re-verified 1b-9), the
**first** phase that ships a working kick with the return-to-bank-or-freeze path already removes
the real-world blocker. Waiting for the auction is not required, and nothing about this decision
asks for that.

**Argument against, recorded.** The blocker is live and unrelated work is waiting on it; a
one-line Mongo delete would end it in a single command. Rejected on his instruction, and the mitigation is
phase order, not a shortcut.

### D9 — The live auction is its own build phase, sequenced after a shippable kick.

Ratified 2026-09-16 as the operative reading of §7 point 1's own framing (*"a materially larger
feature than anything else in this scope … its own phase, not a sub-case of kick"*).

**Defense.** Nothing in either tree has a bid, an auction, a timer or a goroutine (1b-7, 1b-8,
both my own greps), and the money path an auction would settle through is unsafe as written
(1b-5). A kick that offers only bank and freeze is already a complete, deployable feature that
answers the FAQ's admitted gap and clears the blocker; an auction bolted into the same phase
would hold that hostage.

**Argument against, recorded.** Shipping a kick whose disposition picker has one option greyed
out ("auction — coming soon") is a visibly unfinished product, and D2 reads as one decision, not
two. Answered at Stage 4: Phase 2's picker ships **without** the auction option at all rather
than with a dead one, and the option appears when Phase 4 lands (`PLAN.md` §2).

---

## 9. Rules that survive unchanged

Listing what is *not* changing, so a build phase does not helpfully rewrite it.

1. **No auth, and no server-side `isBanker` check.** `grep -rn "IsBanker\|isBanker" backend/` →
   the struct field and two writes, zero reads (§1 row 9, re-confirmed 2026-09-16). `KICK_PLAYER`
   is exactly as trust-the-client as `BANKER_TRANSACTION` is at `5f30fd9`: whoever holds the room code
   can send it. This is settled in `HANDOFF.md` *Settled — no auth, deliberately* and is **not**
   this feature's job to fix. Do not add a check here; it would be the first one in the app and
   would read as a security model that does not exist.
2. **`JoinRoom`'s 409 logic is correct as written** (§2, §1 row 4). The name-or-colour collision
   count is right; what a kick changes is the *data* it counts, not the rule. Do not touch
   `roomControllers.go:167-178` except for whatever §10 Q2's answer requires.
3. **"Delete My Player this Game" and "Delete My Players in All Games" stay local-only.**
   `playerStore.clearPlayerDataForRoom` / `clearAllPlayerData`
   (`frontend/lib/utils/playerHelpers.ts:23-35`, wired at `navbar.tsx:105-120`) touch
   `localStorage` and nothing else, on purpose (§1 row 10). They are a "forget this device"
   reset, not a broken kick. Do not make them call the server.
4. **Another player's Properties drawer stays a read-only deed browser.** Settled 2026-09-16
   (`HANDOFF.md` *Settled*, HANDOFF 7). A kick adds no controls inside that drawer; the kick
   control is a card-level banker action, not a property-level one.
5. **Exactly one backend process, forever.** `clients map[string]map[*Client]bool`
   (`websocketManager.go:23`) is in-process room membership, and the kick's socket-close depends
   on the target's connection being held by *this* process. Do not add a replica, an autoscaler,
   or a "why not two instances" note.
6. **The websocket contract is typed on the frontend only, and both sides move in one commit.**
   Go's `Payload` is `interface{}` (`websocket/types.go:12-15`); the real shapes live in
   `frontend/types/payloads.ts` and `events.ts`. Nothing checks they agree. A new message type is
   new surface on both sides with that same unchecked cost — which is a reason to change both in
   one commit, not a reason to introduce a codegen step here.
7. **Prose event history.** `EventHistory.Event` is a `fmt.Sprintf` string
   (`models/eventHistoryModel.go:13`, built at `websocketManager.go:514-546`) and a kick writes
   one like every other mutation. Structured, machine-reversible events are idea 5 and are not
   this feature.
8. **`models/roomModel.go` stays unformatted.** `gofmt -l .` lists it and only it (1b-10); it is
   board item 5 and belongs to another session. Do not format it in passing — that is the
   drive-by rule, and it would put another session's file in this feature's diff.

---

## 10. GATE 1b — the question set. **Asked and answered 2026-09-16. SUPERSEDED by §11.**

**This section is history, kept deliberately.** It was written before the answers and says
"nothing here is decided"; that was true when written and is no longer true. **Every question
below was answered by Zach in chat on 2026-09-16** and is frozen in §11 as the `D<n>` it
reserved. The questions, their options and — the reason they are kept — their
**arguments-against** stay visible here, so that when one of those arguments comes back in three
weeks it arrives as a known cost that was weighed, not as new evidence (§8.4 of the standard).

| Question | Answer | Frozen as |
|---|---|---|
| Q1 — the kicked player's cash | **"To the bank"**, which in this schema means *out of circulation*: the balance stays on the inactive document and nothing reads or credits it. **Not** Free Parking. Recommended option (a), with his framing | **D10** |
| Q2 — delete the document, or mark it gone | **Mark `isActive: false` and teach the three reads.** Recommended option (a) | **D11** |
| Q3 — can a banker kick themselves | **Yes**, through the same successor-naming flow as kicking another banker. Recommended option (a) | **D12** |
| Q4 — development on a deed that leaves its owner | **Raze it** — houses and hotels sold back to the bank — **on the kick-to-bank path only**, not by changing `SELL`. Recommended option (a), including its explicit "leave `SELL` alone" | **D13** |
| Q5 — how the auction closes | **The banker closes each lot manually.** Recommended option (a) | **D14** |
| Q6 — opening bid and increment | **$0 opening, $1 minimum raise.** Recommended option (a) and the recommended dial | **D15** |
| Q7 — lot shape and the no-bid case | **One deed at a time in board order; an unsold deed returns to the bank.** Recommended option (a) | **D16** |
| Q8 — bidding above your balance | **Rejected when placed, and re-checked at settlement.** Recommended option (a) | **D17** |
| Where auction state lives (raised in `PLAN.md` as a GATE 2 item, not a numbered question here) | **Persisted on the `Room` document in Mongo**, not process memory | **D18** |

**Every answer took the recommended option.** That is worth recording rather than glossing:
seven of the nine were taken as offered, and the two that were not — Q1 and Q4 — were *narrowed*
rather than redirected. Q1 chose the recommended behaviour under a different name ("to the
bank"), and the follow-up that produced D10's wording is itself the evidence that the naming
mattered and the behaviour did not. Q4 took option (a) *and* its explicit carve-out against
option (d).

### Q1 → reserves **D10**. What happens to the kicked player's **cash**?

§7 point 1 is a detailed answer about *properties*; it does not say what happens to the money.
The facts that constrain it: there is **no bank balance field** — `Room` has `freeParking`,
`roomRules`, and nothing else that holds money (`models/roomModel.go:9-23`, read 2026-09-16) —
and there is no creditor concept (D2 killed A3's).

- **(a) Cash is left on the player's document, untouched, in all three dispositions.
  *Recommended.*** The simplest write (none), the only option that neither creates nor destroys
  money, and reversible: if the banker kicked the wrong person, the balance is still there and
  `BANKER_ADD`/`BANKER_REMOVE` can move it. Mental model: *the kick disposes of deeds; the cash
  leaves play with the player.* **Against:** money sitting on a gone player's document is money
  the table can see and cannot reach, which is the same complaint §3A2 makes about frozen deeds.
  It also means "kick" never rebalances the economy, so kicking `John` removes $30,840 from play.
- **(b) Cash goes to Free Parking.** One `$inc` on a field that already exists
  (`roomModel.go:13`) and already has a UI (`components/navbar/free-parking.tsx`), it puts the
  money back in play, and "the pot" is a rule real tables use. **Against:** $30,840 into Free
  Parking is a game-altering event disguised as an administrative one, and Free Parking is
  collected by whoever lands on it — so a kick would hand one random player a jackpot.
- **(c) Cash follows the disposition three ways: bank → discarded, freeze → untouched, auction →
  to Free Parking.** Most consistent with D2's shape. **Against:** "auction the cash" is not a
  thing, so the third arm is really (b) wearing D2's clothes, and it triples the paths to test
  for a field nobody is arguing about.
- **(d) Cash is set to 0 on the kick.** Explicit destruction; nothing can reach it later.
  **Against:** it destroys the audit trail for no gain over (a), and it is the one option that
  cannot be undone by hand.

### Q2 → reserves **D11**. Does a kick **delete** the player document, or **mark it gone** and teach the three reads that ignore `isActive` at `5f30fd9`?

This is the fork 1b-2 and 1b-3 uncovered and it is not a build detail: `isActive` is read
**nowhere** in either tree, so a mark-only kick leaves the card rendering, leaves the name and
colour taken, and **lets the kicked client rejoin one second later**.

- **(a) Mark gone (`isActive: false`) and teach the reads. *Recommended.*** Three edits, all
  small and all named: `handler.go:73-101` refuses `JOIN` for an inactive player;
  `playerControllers.go:34,62` either filters them out of `GetPlayersInRoom` or returns them
  flagged so the frontend can render them as removed; `roomControllers.go:167-173` adds
  `isActive: true` to the collision count so the name and colour free up. **Why this and not
  (b):** it is the only option compatible with D4 — a frozen estate needs an owner document for
  `Property.playerId` to resolve against — and it keeps every past `EventHistory` row's player
  name meaningful. **Against:** four files instead of one, and `isActive` becomes load-bearing
  for the first time, so every later read of the `Player` collection has to remember it exists.
- **(b) Hard-delete the `Player` document.** Frees the name and colour for free, the card
  disappears for free, no read paths to teach, and the kicked client's own refetch 404s
  (`playerControllers.go:50-56`) which is a natural "you are out" signal. **Against:** it makes
  **D4 impossible** — frozen deeds would point at a deleted `_id`, and `GetPlayersInRoom`'s
  per-player property loop (`:72-88`) would simply never find them, silently hiding the deeds
  rather than freezing them. It also forces Q1 to (d) whether or not that is wanted.
- **(c) Per disposition: delete on bank/auction, mark on freeze.** Each disposition gets the
  cheapest correct write. **Against:** two kick shapes to build, test and reason about forever,
  and "was this player deleted or frozen" becomes a question every later feature must ask.

### Q3 → reserves **D12**. Can a banker kick **themselves**, naming a successor?

§7 point 2 answered "can the banker be kicked" and did not separate "by someone else" from "by
themselves"; the original §6 question 2 bundled both.

- **(a) Yes — a self-kick with a named successor is the same action. *Recommended.*** It is the
  same code with `targetPlayerId == the caller's own id`, and it is the **only exit a banker
  has**: at `5f30fd9` a banker who must leave has no hand-off path and no kick path, so the only
  recovery is deleting the whole room (`playerControllers.go:229-243`, which orphans every other
  document — HANDOFF 6 fact 19). **Against:** "kick" is the wrong word for a voluntary
  departure, and a mis-tap on your own card while holding the banker role is the most expensive
  mis-tap in the app.
- **(b) No — kick targets other players only.** The banker's own departure stays idea 12's
  problem, unscoped. **Against:** it leaves the exact scenario §3B1's argument-against named as
  unrecoverable, while the mechanism to fix it (D5's succession) is already being built two feet
  away.
- **(c) Yes, but as a separate control** — "Leave and hand off" in the navbar's Danger Zone
  (`navbar.tsx:88-123`) rather than a kick on your own card. Clearer words, same write.
  **Against:** it is a second entry point to one action, and the Danger Zone is three
  purely local `localStorage` controls (§1 row 10), so it would be the first row there that
  reaches the server — a meaningful change to what that menu means.

### Q4 → reserves **D13**. When a deed leaves the kicked player, do its **houses and hotels** come off?

The finding is 1b-4: `SELL` clears `playerId` and `isMortgaged` and leaves `developmentLevel`
alone, so a returned deed keeps its buildings and the next buyer pays only `price` and inherits
them free. Pre-existing, unreachable from the UI as of 2026-09-16 (HANDOFF 6 fact 7) — **a kick makes
it reachable for the first time**, which is why it is this design's question and not a bug
report.

- **(a) Raze to `developmentLevel: 0` on the kick path — bank and auction both. *Recommended.***
  It is the real rule (buildings return to the bank when a player is out), it closes a free-value
  transfer, and it is one extra `$set` in a write the kick already makes. Leave
  `manager.HandlePropertySaleMortgage` itself **alone** — no drive-by (`CLAUDE.md`). **Against:**
  the kick path and `SELL` then disagree about what "back to the bank" means, in the same file,
  which is exactly the kind of drift that costs a later session an hour.
- **(b) Keep the development; the next buyer inherits it.** No new write at all, and it is
  arguably generous-but-harmless in a friendly game. **Against:** it is a silent money bug of
  the sort this repo has already been bitten by twice — a hotel on Boardwalk is $2,000 of
  development handed over for the $400 face price.
- **(c) Keep the development and reflect it in the price.** Most faithful to value. **Against:**
  the asking price is sent by the *client* (`handlePropertyPurchase` reads `payload["price"]`,
  `websocketManager.go:266-270`), so "reflect it in the price" means trusting the browser to
  compute a number that is now large — new surface, for a case that only arises after a kick.
- **(d) Fix `SELL` itself, so both paths agree.** The tidy answer. **Against:** it changes
  behaviour outside this feature and is a drive-by unless it is asked for explicitly — which is
  what this option is doing.

### Q5 → reserves **D14**. How does the auction **close**?

1b-7: there is not one goroutine, timer or ticker in the Go tree. Whatever this answer is, the
mechanism it needs does not exist yet.

- **(a) The banker closes it manually — "Sold". *Recommended for v1.*** No clock, no timer
  goroutine, no server-side deadline; the banker is already the arbiter at the table and calls
  the auction with their voice anyway. A stalled auction is visible in the room rather than
  silently expiring. **Against:** an auction stays open forever if the banker's phone dies —
  mitigated only because D5 means the table can kick and replace them.
- **(b) A countdown from the last bid (default 15s, reset on each bid).** The real auction feel,
  and it ends without anyone having to be trusted. **Against:** it is the first time-driven
  behaviour in this backend (1b-7) — a timer per room, in a process that is deployed by hand and
  restarted with no drain — and a countdown that fires while a bid is in flight is precisely a
  failure that compiles, passes every gate and is wrong in production.
- **(c) Both: a countdown with a banker override.** Best product, largest surface, and the
  two close paths can race each other — two ways to determine one winner.

### Q6 → reserves **D15**. What is the **opening bid**, and the minimum **increment**?

- **(a) Opens at $0; any bid above the current high bid wins the lead; minimum increment $1.
  *Recommended.*** It is the Monopoly rule, it needs no per-property data, and $0 makes the
  "nobody wants it" case (Q7) visible instead of hiding it behind an unmet reserve.
- **(b) Opens at the deed's face `price`** (`models/propertyModel.go:17`, already on every
  property document). A floor that stops a $1 steal. **Against:** it is a reserve price the real
  rules do not have, and it guarantees the no-bid case for any over-priced deed.
- **(c) The banker sets the opening bid per lot.** Flexible and matches a real auctioneer.
  **Against:** one more thing for the banker to type at the moment they are already mid-kick.
- Increment is a dial either way (`PLAN.md` §3): **$1** recommended; $10 if bid-spam becomes the
  problem in practice.

### Q7 → reserves **D16**. One **lot** or one deed at a time — and what if **nobody bids**?

- **(a) One deed at a time, in `propertyIndex` order, the banker advancing; an unsold deed goes
  back to the bank. *Recommended.*** It is how a table actually does it, each bid is about one
  deed's value, and the settlement is one purchase-shaped write per lot. The no-bid fallback is
  a disposition the banker could have chosen anyway (D2), so it needs no new concept, and it
  leaves the board playable. **Against:** kicking `John` (4 deeds, 1b-9) means four auctions in
  a row, which is a long ceremony for an administrative action.
- **(b) All of the kicked player's deeds as one lot, one winning bid.** Fast — one auction per
  kick regardless of size. **Against:** it prices a portfolio, which nobody at a table can do
  quickly, and it hands a whole colour group to one player in a single action.
- **(c) The banker groups the lots** (by colour group, say). **Against:** a third decision at
  kick time, and `Property.Group` (`propertyModel.go:15`) makes it easy to *build* and no easier
  to *use*.
- **No-bid alternatives** if (a) is taken but the fallback is not wanted: freeze the unsold deed
  with the kicked player instead, or let the banker re-run the auction. Both are one branch;
  neither is recommended, because both leave the board with an unreachable deed.

### Q8 → reserves **D17**. May a player bid **more than their balance**?

Money stakes, and this is the question that decides whether Phase 3 needs a Deep review of the
settlement path or merely of the close path. Facts: **nothing in this app floors a balance** —
`UpdatePlayerBalanceByBanker` (`controllers/playerControllers.go:201-227`) and
`PurchaseProperty` (`controllers/propertyControllers.go:19-35`) both `$inc` with no check, and
`PurchaseProperty` is not even in a session (1b-5).

- **(a) Reject a bid above the bidder's current balance when it is placed, **and** re-check at
  settlement. *Recommended.*** An auction is the first path in this app where two clients race
  to spend money, and a balance that was sufficient when the bid was placed can be insufficient
  when the hammer falls (the bidder can pay rent in between). Checking once is the bug; checking
  twice is the decision. **Against:** it is the first affordability check in the product, so it
  will be the only rejection a player meets that the rest of the app would have allowed —
  inconsistent, and invisible unless the `ERROR` toast renders it (which it now does, HANDOFF 9,
  merged as `5f30fd9`).
- **(b) Allow any bid; the banker polices it at the table.** Consistent with every other trust
  decision in this app (§9 rule 1), and zero new code. **Against:** the winner is then taken
  negative by the settlement write with no warning, and a negative balance is a state no screen
  in this app is designed to show.
- **(c) Allow the bid, reject at settlement only.** Half of (a) for half the work. **Against:**
  the auction can be won by someone who cannot pay, and then what — re-auction, next-highest
  bidder, or bank? That is a whole new decision, created by not making this one.

---

## 11. Decisions — frozen 2026-09-16 (the GATE 1b round)

Answered by Zach in chat on 2026-09-16, in one batch, and frozen here the same day. The `D<n>`
numbers are the ones §10 reserved, so every existing citation of "Q4 → reserves D13" resolves.
**Amendments append; nothing here is edited in place.**

### D10 — The kicked player's cash goes "to the bank", which in this schema means **out of circulation**: the balance stays on the inactive document and nothing reads, moves or credits it.

Ratified 2026-09-16, answering §10 Q1. Zach's answer was "to the bank", then clarified on a
follow-up: there is no literal bank-balance line to credit, so the concrete rule is that the
money simply stops being anybody's — **not** added to `Room.FreeParking`, which already means
something else. Functionally it is the cash equivalent of a deed reverting to "no owner".

**The write is: none.** `balance` is left exactly as it stands, on a document that D11 has marked
inactive.

**Defense.** It is the only option that neither creates nor destroys money, and it is
reversible: if the banker kicked the wrong person the balance is still there and
`BANKER_ADD`/`BANKER_REMOVE` can move it. The schema decides the rest — `Room` carries
`freeParking`, `roomRules` and nothing else that holds money
(`backend/models/roomModel.go:9-23`, read 2026-09-16) — so "credit the bank" has no destination
and inventing one would be a schema change inside a kick.

**Supersedes nothing in §3** (§3A treated cash and properties as one question). **Argument
against, recorded from Q1(a):** money the table can see and cannot reach, and kicking `John`
takes $30,840 out of play. Both accepted. **And the rejected alternative is worth naming
because it is the one that will be re-proposed:** routing the cash to Free Parking (Q1(b)) puts
it back in play but hands a jackpot to whoever next lands there, turning an administrative action
into a game-altering one.

**Interaction with D11, and this is why the two had to be answered together.** This decision is
only coherent because the document survives. Had Q2 gone the other way — a hard delete — the
cash would have been destroyed unconditionally and this decision could not have been taken.

### D11 — A kick marks the player `isActive: false`. It does **not** delete the `Player` document. The three reads that ignore `isActive` are taught to respect it.

Ratified 2026-09-16, answering §10 Q2 with its recommended option (a).

The three edits, named so a build phase cannot skip one:

1. **`handler.go:73-101`** — `JOIN` refuses an inactive player, so the kicked client's reconnect
   cannot re-seat them.
2. **`controllers/playerControllers.go:34,62`** — `GetPlayersInRoom` either filters inactive
   players out or returns them flagged for the frontend to render as removed.
3. **`controllers/roomControllers.go:167-173`** — `JoinRoom`'s name-or-colour collision count
   adds `isActive: true`, so a kicked player's name and colour free up.

**Defense.** It is the only option compatible with **D4** (freeze) — a frozen estate needs an
owner document for `Property.playerId` to resolve against — and with **D10**, which leaves cash
on a document that must therefore exist. It also keeps every past `EventHistory` row's player
name meaningful. A hard delete would have been cheaper at the write and would have broken both.

**This is the decision that makes the feature work at all.** Without the three edits the kick is
a no-op with green gates: `isActive` is read **nowhere** in either tree (§1b-2 — 5 grep hits, all
writes or type declarations), so marking alone leaves the card rendering, leaves the name taken
(§1b-3), and lets the kicked client rejoin one second later
(`frontend/app/room/[code]/page.tsx:219-223`).

**Argument against, recorded from Q2(a):** four files instead of one, and `isActive` becomes
load-bearing for the first time — every later read of the `Player` collection now has to
remember it exists. Accepted. **Consequence for later work:** that is a new invariant and it
belongs in `HANDOFF.md` *Invariants* when Phase 1 lands.

**As built, 2026-09-16 (HANDOFF 15) — D11's edit list was wrong in two ways, found by reading the
code it cites. The decision itself stands; only the sites change.**

**Edit 2 is "flagged", not "filtered", and it needs no backend change.** D11 offered either and
cited `playerControllers.go:34,62` — which is the players-list query, i.e. the *filter* reading.
**Filtering is not available:** `GetPlayersInRoom` nests each player's deeds inside the player
object (`playerControllers.go:87`, `players[i].Properties = properties`), so dropping an inactive
player from the list also drops their properties from the response — and `PLAN.md` Phase 1's
done-when (d) requires that under **D4** (freeze) "the deed still shows against the gone player,
house intact". Filtering would make D4 unobservable and quietly narrow it. The flag, meanwhile,
is **already on the wire**: `models.Player.IsActive` is tagged `json:"isActive"` with no
`omitempty`, so every client already receives it and has since before this feature existed.
Decided by Zach 2026-09-16: **flag.** The consequence is that rendering a removed player as
removed is **frontend work in Phase 2**, which Phase 2's scope did not previously mention and now
does.

**A trap at the exact line D11 cites, worth stating because the naive edit is silent.** `query` is
declared at `playerControllers.go:34` and used **twice** — at `:62` for the players `Find` and at
`:92` for the **EventHistory** `Find`. `models.EventHistory` has no `isActive` field, so adding
`"isActive": true` to `query` at line 34 — the literal reading of D11's citation — would match
zero event-history documents and blank the room's entire audit log, with every gate green. Anyone
who revisits the filter option must build a **separate** filter for the players `Find` and leave
`query` alone.

**There is a fourth re-seat path, and D11 does not name it. Built 2026-09-16 on Zach's decision.**
`GetPlayerDetails` (`playerControllers.go:119`) did `FindOne` on `_id` alone. `components/room/
join.tsx:71-73` calls it with the id in `localStorage` and routes straight into the room if a
player comes back — so a kicked player rejoins through the **Join screen** while D11 edit 1 bolts
only the **websocket** door. `isActive: true` is now part of that filter, so a kicked player takes
the *existing* 404 and is sent to the create-a-player form, where their old name and colour are
free by edit 3. **No new player-facing copy was written** — `PLAN.md`'s dial "where the kicked
browser lands" deferred that for v1, and reusing the existing not-found path honours it.

**So D11 is four sites, not three:** `handler.go`'s `JOIN` (**built 2026-09-17, HANDOFF 19**),
`GetPlayerDetails` (**built**), `JoinRoom`'s collision count (**built**), and `GetPlayersInRoom`
(**no change needed** — flag already on the wire; the work moved to Phase 2's frontend).

**All four are now closed, and `isActive` became a standing invariant when the last one landed** —
`HANDOFF.md` *Invariants* 6, which D11 predicted it would. `JOIN`'s refusal is a check on the
decoded player rather than part of the filter, unlike `GetPlayerDetails`: the read is
`controllers.GetPlayer`, which is shared with five other handlers and must not start refusing
inactive players for all of them. Nothing is sent back to the refused connection and its socket is
left open, per **D6** — the client reacts to its own room fetch failing, which is what the
"nothing new for v1" dial chose.

**One thing the build changed that D11 did not anticipate.** `handler.go` used to set
`client.PlayerID` *before* reading the player; it now sets it after, through `RoomManager.SeatClient`,
so a connection whose `JOIN` is refused stays unseated. That was forced by the force-close, not by
D11: `CloseClientByPlayerID` scans `PlayerID` from another player's goroutine, which turned the old
bare assignment into a data race, and an unseated conn carrying the id it claimed would also have
made the scan's empty-id guard meaningless.

### D12 — A banker can kick themselves, through the same successor-naming flow as kicking another banker.

Ratified 2026-09-16, answering §10 Q3 with its recommended option (a). One action, one code path:
`targetPlayerId` equal to the caller's own id is allowed, and the `successorPlayerId` requirement
of **D5** applies identically.

**Defense.** It is the only exit a banker has. Before this, a banker who has to leave had no
hand-off path and no kick path, so the only recovery was deleting the whole room
(`controllers/playerControllers.go:229-243`) — which orphans every `Player`, `Property`,
`EventHistory` and `Transfer` document (HANDOFF 6 fact 19). And it costs nothing to build: D5's
succession write is the same write with the same validation.

**Argument against, recorded from Q3(a):** "kick" is the wrong word for a voluntary departure,
and a mis-tap on your own card while holding the banker role is the most expensive mis-tap in the
app. Accepted, and it is what the confirmation dialog (`PLAN.md` §3) and R7's copy variants exist
to blunt. **Explicitly not taken: Q3(c)** — a separate "Leave and hand off" control in the
navbar's Danger Zone. One entry point, not two.

### D13 — A deed going back to the bank on a kick is **razed**: its houses and hotels are sold back to the bank and `developmentLevel` goes to 0. `manager.HandlePropertySaleMortgage`'s `SELL` case is **not** changed.

Ratified 2026-09-16, answering §10 Q4 with its recommended option (a), including the carve-out
option (a) named: **leave `SELL` alone.** Option (d) — fix `SELL` so both paths agree — was
explicitly not taken.

**Defense.** It is the real rule (buildings return to the bank when a player is out) and it
closes a hole a kick would newly open: `SELL` sets only
`{"playerId": nil, "isMortgaged": false}` (`backend/manager/propertyManager.go:53`) and leaves
`developmentLevel` alone, so a returned hotel-bearing deed reappears in
`GetAvailableProperties` (`controllers/propertyControllers.go:75-78`) at its face `price` and the
next buyer inherits ~$2,000 of development for free (§1b-4). That hole is pre-existing but
**unreachable from the UI as of 2026-09-16** (HANDOFF 6 fact 7); the kick is the first thing that
would make it reachable, and the decision is to not ship the hole rather than to fix a path
nothing calls.

**Argument against, recorded from Q4(a), and it is real:** the kick path and `SELL` now disagree
about what "back to the bank" means, in the same file. Accepted deliberately — changing `SELL`
would be the drive-by `CLAUDE.md` forbids, and it would change behaviour outside this feature.
**The mitigation is a comment at the kick's raze write naming `propertyManager.go:53` and saying
why the two differ**, so the next reader finds the disagreement documented rather than looking
like drift. **Scope note:** whether the mortgage clears is unchanged and not part of this — D3
already settled that a returned deed comes back unmortgaged, because that is what `SELL` does
and the bank now holds the deed.

### D14 — The banker closes each auction lot manually. There is no timer.

Ratified 2026-09-16, answering §10 Q5 with its recommended option (a).

**Defense.** No clock, no timer goroutine, no server-side deadline — and **there is not one
goroutine, timer or ticker anywhere in the Go tree** (§1b-7: four `context.WithTimeout` calls and
nothing else), so a countdown would be the first time-driven behaviour in a process that is
deployed by hand with no drain. The banker is already the arbiter at the table and calls the
auction out loud; a stalled auction is then visible in the room rather than silently expiring.

**Argument against, recorded from Q5(a):** an auction stays open forever if the banker's phone
dies. Mitigated, and only because of decisions taken in the same batch: **D5 and D12** mean the
table can kick and replace a banker who has gone dark. **Explicitly not taken: Q5(c)**, a
countdown with a banker override — two close paths that can race to determine one winner is
precisely the silent failure this phase's Deep review exists to catch, and the way to not have it
is to not build it.

### D15 — Auction lots open at $0, and the minimum raise is $1.

Ratified 2026-09-16, answering §10 Q6 with its recommended option (a) and the recommended
increment dial.

**Defense.** It is the Monopoly rule — any player may bid any amount above the current bid — and
it needs no per-property data. $0 matters specifically: it makes the "nobody wants this deed"
case **visible**, as a lot that closes with no bid (which D16 then disposes of), instead of
hiding it behind an unmet reserve price.

**Argument against, recorded from Q6(b):** no floor means a $400 deed can be stolen for $1 when
the table is not paying attention. Accepted — that is the auction working, and the banker closes
the lot (D14) so there is a human in the loop. **Increment stays a dial** (`PLAN.md` §3) at $1;
$10 only if bid-spam turns out to be a real problem in play.

### D16 — One deed at a time, in board order, the banker advancing. A deed nobody bids on returns to the bank.

Ratified 2026-09-16, answering §10 Q7 with its recommended option (a) — both halves.

Board order is `Property.PropertyIndex` (`backend/models/propertyModel.go:13`), which is already
the ordering over the 28 ownable properties.

**Defense.** It is how a table actually runs an elimination: one deed, one price, hammer, next.
Each bid is about one deed's value rather than a portfolio nobody can price quickly, and the
settlement is exactly one purchase-shaped write per lot. The no-bid fallback needs no new
concept at all — it is **D2**'s return-to-bank disposition, which the banker could have chosen
for the whole estate anyway, so the auction's failure mode is a state the design already handles
and the board stays playable.

**Argument against, recorded from Q7(a):** kicking `John` (4 deeds, §1b-9) means four auctions
back to back, a long ceremony for an administrative action. Accepted. **Explicitly not taken:**
Q7(b) all-deeds-as-one-lot (hands a colour group to one player in a single action) and Q7(c)
banker-grouped lots (a third decision at kick time). **And note what the no-bid answer rules
out:** freezing an unsold deed, or re-running the auction — both leave the board with an
unreachable deed, which is the one outcome this decision refuses.

### D17 — A bid above the bidder's balance is rejected when it is placed, and the balance is re-checked at settlement.

Ratified 2026-09-16, answering §10 Q8 with its recommended option (a). **Both checks, not one.**

**Defense.** The auction is the first path in this app where two clients race to spend money, and
a balance that was sufficient when the bid was placed can be insufficient when the hammer falls —
the bidder can pay rent to someone in between. **Checking once is the bug; checking twice is the
decision.** It also lands in a codebase where nothing floors a balance at all:
`UpdatePlayerBalanceByBanker` (`controllers/playerControllers.go:201-227`) and
`PurchaseProperty` (`controllers/propertyControllers.go:19-35`) both `$inc` with no check, and
the latter is not even in a session (§1b-5).

**Argument against, recorded from Q8(a):** it is the first affordability check in the product, so
it is the only rejection a player will meet that the rest of the app would have allowed —
inconsistent, and invisible unless the `ERROR` toast renders it. That last risk is now closed:
the `ERROR` branch is on `main` (HANDOFF 9, merged; verified on `main` 2026-09-16).
**Explicitly not taken: Q8(c)**, settlement-only rejection, because an auction won by someone who
cannot pay forces a whole new decision — re-auction, next-highest bidder, or bank — that this
answer exists to avoid.

### D18 — Auction state is persisted on the `Room` document in Mongo, not in process memory.

Ratified 2026-09-16. Raised as a GATE 2 item in `PLAN.md` Phase 3 rather than as a numbered §10
question; answered in the same batch, so it is frozen here with the rest.

**Defense.** It matches how the rest of the app already works: every client refetches room state
on every broadcast (`frontend/app/room/[code]/page.tsx:121-132`), and
`GET /rooms/:code/players` already returns the whole `Room` document
(`controllers/playerControllers.go:102-106`) — so **the live auction reaches every client with no
new REST route**. And it survives the two things in-memory state does not: a client that
reconnects mid-auction (phones lock; the 1s reconnect at `page.tsx:219-223` fires constantly, and
React 19 StrictMode double-mounts it in dev — HANDOFF 7) would otherwise receive only *future*
broadcasts and could never learn the current high bid; and a backend deploy, which is by hand,
one process, with no drain, would silently drop an auction mid-flight.

**Argument against, recorded:** it grows the `Room` schema for state that is transient by nature,
and every existing `Room` read now decodes fields it does not care about. Accepted — `Room` is a
small document, the reads are `FindOne`s that already decode the whole thing, and the alternative
trades a schema field for a class of bug no gate in this repo can see.

---

## 12. Ground truth re-checked at the close of Stage 3 — 2026-09-16

Run after §11 was written, because the repo moved twice while this document was being written and
`PLAN.md` §5's first hazard was about to be wrong.

| Claim | Verified state | Citation |
|---|---|---|
| **`c202745` is merged. `PLAN.md` §5 hazard 1 is closed.** `main` is `87e0821`; HANDOFF 10's work landed as `49594c1` ("merge freeparking default-arm fix and payload assertion hardening"). `origin/main` is level with `main` — it was pushed. | `git log --oneline -4 main`; `git log --oneline main..worktree-agent-a2f874037ff8da39a` → **empty**; `git branch -vv` | 2026-09-16. **So Phase 1 now cuts from `main`, and the precedent it must copy is there:** `websocketManager.go` on `main` has **19** checked two-value payload assertions and **zero** unchecked single-value ones (`grep -cE ', ok := payload\[[^]]*\]\.\(string\)'` → 19; the unchecked-form grep returns nothing). |
| **A different branch now owns the same file, and it is unmerged.** `.claude/worktrees/agent-abffee6918b2a9e0e`, clean, at `d1d6038` ("refuse to broadcast a notification nobody wrote" — `PASSOFF.md` board item 8), cut from `49594c1`. | `git worktree list --porcelain`; `git log --oneline main..worktree-agent-abffee6918b2a9e0e` → `d1d6038`; `git diff main --stat` → `websocketManager.go` **+55**, `websocketManager_test.go` **+213** ahead, `frontend/app/manifest.ts` and two `logo-maskable-*.png` behind | 2026-09-16. **The hazard did not go away; it changed hands.** `PLAN.md` §5 hazard 1 is rewritten to name this branch instead. |
| That merge is **clean** | `git merge-tree --write-tree main worktree-agent-abffee6918b2a9e0e` → exit 0, one tree OID (`edb0b95…`), no conflict block | 2026-09-16. The same read-only check HANDOFF 5 used before its merge. The command for Zach is in `PLAN.md` §5. |
| Backend gates on `main` `87e0821` | `go build ./...` 0 · `go vet ./...` 0 · `go test -count=1 ./...` 0 with **49 tests** (`websocket` 40, `manager` 9) · `gofmt -l .` lists `models/roomModel.go` only | Run 2026-09-16 from `backend/`. Supersedes `PLAN.md` §0's 35-test figure, which was `main` before the merge. |
