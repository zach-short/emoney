# TRIAGE — banker powers

**What this is.** The fifteen ideas from `HANDOFF.md` step 6, plus the five bugs that step found
and deliberately did not fix, turned into a dependency graph and a build order. **It is not a
scope document and it decides nothing about how anything is built.** Its whole job is to make the
field visible so that each `/scope` session afterwards is about one thing.

Written 2026-09-17. Every fact below is either carried from HANDOFF 6 — which walked the live app
and cited file and line — or re-verified in the tree on 2026-09-17, and which one is marked.

**Read `HANDOFF.md` step 6 before any scope session that starts from this file.** The twenty facts
there are the evidence; this is only the index over them.

---

## 1. Decisions taken, 2026-09-17

Twelve, asked and answered in three batches at the top of this session. They are the reason this
file can propose an order at all. **A scope session may not quietly overturn one of these** — it
may argue against one, in which case it stops and asks.

| # | Question | Decision |
|---|---|---|
| D1 | How to deliver the scoping | **Triage first**, then `/scope` one item at a time |
| D2 | Is `isBanker` enforced? | **Server-enforce it.** Banker-only actions check the sender's `isBanker` in Go |
| D3 | What is "Request"? | **Peer confirms** — the payer receives the claim and confirms it |
| D4 | Cards: one player or the table? | **Both, phased.** Single-target first, table-wide second |
| D5 | Trades | **Finish peer-to-peer properly**, reusing D3's machinery |
| D6 | Undo | **Structured event records, phased in** |
| D7 | Can a player go broke? | **A room rule the banker sets before the game starts** |
| D8 | The five found bugs | **Board rows now, fixed before the features** |
| D9 | Which room rules are real? | **All four** — starting cash, negative balances, building supply, Free Parking behaviour |
| D10 | Banker handover | **Hand off to one person.** Exactly one banker at all times |
| D11 | End game | **Archive with a final summary**, then a real cascade delete on a timer |
| D12 | Deed control | **Bare correction — no money moves with the deed** |

**D7 is the decision that reorganised this list.** It was asked as a straight three-way (allow
negative / refuse / refuse-for-players-only) and the answer was none of them: it belongs in room
settings. That pulls three previously unrelated items — fact 16 (rules stored, never read), fact 17
(the Game Rules screen is a dead end), fact 18 (the `startingCash` money bug) — into a single
cluster that **must work before any rule can mean anything**, and it promotes that cluster ahead of
every feature that consults a rule.

---

## 2. What is already built

Struck off before planning, because both were open when HANDOFF 6 was written and are not now.

- **Idea 1, kick a player — BUILT.** Phases 1-3 are on `main` and deployed (`ed87ac8`, `6a614a3`,
  `b4ea04c`). Phases 4 (auction UI) and 5 (ship/walk/record) remain; they are board row 7 and are
  **not re-planned here**.
- **Idea 10, the `ERROR` toast — BUILT.** Verified 2026-09-17: `app/room/[code]/page.tsx:129`
  handles `message.type === "ERROR"`. HANDOFF 6 called this a precondition for every other idea;
  it is discharged.

---

## 3. The foundations

Five pieces of shared machinery. **None of them is a feature**, every one is depended on by
several, and each is cheaper built once than discovered three times.

### F1 — Server-enforce `isBanker` (D2)

Today `IsBanker` is written at room creation (`controllers/roomControllers.go:59`) and `false` for
joiners (`:184`), and **read nowhere in the Go tree** — HANDOFF 6 fact 9, re-grepped 2026-09-17.
So the single banker-gated action that exists is gated in the browser only.

D2 makes that real. Note what it is and is not: it does **not** add auth — a room code is still
the only credential, which is settled and deliberate — it stops a client that *has* the code from
sending a frame the UI would not offer them.

**Blocks:** every banker power below. Do it before the first one, not after the third.
**Size:** small-medium. **Tier:** Deep — a wrong check here passes every gate and is wrong in
production, and it is the authority boundary the rest of the list assumes.

### F2 — A targeted send, `SendTo(room, playerID, msg)`

`Broadcast` (`websocketManager.go`) is the only fan-out on `RoomManager`; the only other write is
back to the *sending* socket for errors. Each `Client` already carries `PlayerID`, so this is a
short addition — HANDOFF 6 fact 13, which noted **nothing needed it yet**. D3 and D5 both need it
now.

**Blocks:** Request (D3), trades (D5), and telling a kicked player why they were removed — the
question kick Phase 1 left open.
**Size:** small. **Tier:** Default.

### F3 — Make `RoomRules` real (D7, D9)

The cluster D7 created, and the largest foundation. Three coupled problems:

1. **The Game Rules screen is a dead end** (fact 17): starting cash, and the two sliders, with no
   Back and no Create Room button — the only escape is a reload, which clears the form. *Partly
   stale:* a "Create Room" control exists at `create.tsx:143` as of 2026-09-17, but it sits
   **above** the Game Rules block, so it belongs to the previous step. **A scope session must
   re-walk this screen rather than trust either claim.**
2. **`startingCash` is computed before the body is bound** (fact 18, re-verified 2026-09-17,
   `roomControllers.go:29-33`): the `if requestBody.StartingCash == 0` test runs against a
   zero-valued struct, so `RoomRules.StartingCash` is unconditionally `1500` while the creator's own
   balance is written from the bound value at `:61`. **Pick $3000 and the host starts with $3000
   and every joiner with $1500.** Latent only because of (1).
3. **Nothing reads any rule.** `MaxHouses` / `MaxHotels` are stored and never read anywhere
   (re-grepped 2026-09-17: the model field and the write, nothing else).

**(1) and (2) must be fixed in the same change.** Fixing the screen without fixing the bind order
ships the money bug to every room created with non-default cash — this is the single most
dangerous item on the list, because it is a one-line reordering guarding a silent wrong number.

D9 then adds two new rules to the four-field set: **can players go negative**, and **Free Parking
behaviour**.

**Blocks:** the negative-balance rule everywhere (cards, transfers, rent, banker transactions),
building supply, mid-game rule adjustment.
**Size:** medium. **Tier:** Deep for the bind-order and the balance rule; Default for the screen.

### F4 — Structured event records (D6)

`EventHistory.Event` is a prose string built with `fmt.Sprintf`, so **nothing is
machine-reversible**. `Transfer` rows are structured but cover only `SEND` and are read by no route
(facts 5, 14). D6 takes the schema change.

**Blocks:** undo, the audit view, and any ledger worth reading.
**Size:** large — it touches every mutation. **Tier:** Deep.
**Known cost, stated up front:** existing rooms cannot be backfilled and keep their prose-only
history.

### F5 — Log transfers in the event history

One `rm.CreateEventHistory` call in `handleTransfer`, beside the existing `Broadcast`.
**Re-verified 2026-09-17: still zero calls** — every other money handler writes a row and this one
does not, while `app/help/page.tsx:17` tells the banker to audit the game with that log. The most
common money movement in the game is the one thing missing from the record the FAQ points at.

**Cheapest item on the entire list.** Do it early for the honesty, and accept that F4 may later
change its shape.
**Size:** trivial. **Tier:** Mechanical.

---

## 4. The bugs, per D8

D8 says board rows now, fixed before the features. Five from HANDOFF 6, plus four raised during
this session's other work.

| ID | Bug | Couples to | Tier |
|---|---|---|---|
| B1 | `startingCash` read before `ShouldBindJSON` — host and joiners get different money | **F3, same change as B2** | Deep |
| B2 | Game Rules screen has no way out — no room has ever been created with non-default rules | **F3, same change as B1** | Default |
| B3 | `REQUEST` shows money moving the wrong way — preview, payload and affordability guard all inverted (fact 5) | Request (D3). **Fixing the backend first would ship money moving the wrong way** | Deep |
| B4 | `DeleteRoom` deletes one document and orphans Players, Properties, EventHistory, Transfers; ignores `DeletedCount` | End game (D11) | Default |
| B5 | Three dead frontend files: `p2p-custom-transfer.tsx`, `ui/reason-select.tsx`, the commented-out `handleSellToBank` | **Existing board row 1** — the sweep already owns this lane | Mechanical |

Already on the board from this session, same family, listed so they are not triaged twice: **row
14** (`PlayerTransfer` takes a negative amount and has no balance check of any kind — the worst of
the three), **row 15** (`UpdatePlayerBalanceByBanker`, same sign flip), **row 16** (may a frozen
player move money), **row 17** (20 Dependabot advisories).

**Rows 14 and 15 are the same shape as F3's balance rule and should be taken with it, not before
it** — a floor written twice, once as a hard-coded guard and once as a room rule, is a floor that
will disagree with itself.

---

## 5. The features, with their dependencies

| # | Feature | Depends on | Size | Tier |
|---|---|---|---|---|
| 12 | **Hand over the banker role** (D10) | F1 | small | Default |
| 3 | **Give / move / return a deed** (D12) | F1 | small | Default |
| 2a | **Cards, single target** (D4) | F1, F3 | small | Default |
| 2b | **Cards, whole table** (D4) — *this is also idea 7* | 2a | medium | Deep |
| 8 | **Banker payout presets** ("Pass GO +$200") | F1 | small | Mechanical |
| 11 | **The bank's building supply** | F3 | medium | Default |
| 9 | **Request, peer-confirmed** (D3) | F2, B3 | medium | Deep |
| 4 | **Peer-to-peer trades** (D5) | F2, 9 | medium-large | Deep |
| 5 | **Undo / reverse an event** (D6) | F4 | large | Deep |
| 13 | **Banker audit view** | F5, ideally F4 | medium | Default |
| 14 | **End the game** (D11) | B4 | medium | Default |
| 15 | **Adjust room rules mid-game** | F3 | small | Default |

Two notes that change the arithmetic:

- **Idea 7 ("everyone pays $50") is not a separate feature.** It is 2b under another name; HANDOFF
  6 listed them separately and they are one handler.
- **Idea 3 is the best value on the list.** `SELL` already works server-side and is unreachable
  from the UI, and `AssignOwnerShipProperty` is written and called by nothing (facts 7, 8) — so
  most of it exists. D12 keeps it a bare correction, which means it composes with trades instead of
  becoming a second way to move money.

---

## 6. Proposed order

Six waves. **Within a wave, items are independent; across waves, they are not.** Each wave is a
place to stop.

**Wave 0 — the money bug, and the authority the rest assumes.**
B1 + B2 together · F1 · F5 · then 12 (hand over the banker), as the first consumer of F1 and the
safety net under kicking a banker.

*Why first:* B1 moves money wrongly and B2 is the only reason it has not yet. F1 is the boundary
every later item is defined against; adding three banker powers and then enforcing is three
rewrites.

**Wave 1 — room rules become real.**
F3 · then 11 (building supply) and 15 (mid-game adjustment), which fall out of it · take rows 14
and 15 here, so there is exactly one balance floor.

**Wave 2 — the first features the table will feel.**
3 (deed control) · 2a (single-target cards) · 8 (presets).

*Why here:* all three are small, all three are visible at the table, and none needs the peer
machinery. This is the wave that makes the app feel different.

**Wave 3 — the peer machinery.**
F2 · 9 (Request, with B3 fixed in the same change) · 4 (trades).

*Why together:* D3 and D5 are one body of work wearing two names — a pending store, a targeted
send, an inbox. Built separately, the second one rewrites the first.

**Wave 4 — records.**
F4 · 5 (undo) · 13 (audit view).

*Why last of the building:* it is the largest change and the only one whose value is entirely in
what it unblocks. Everything above ships without it.

**Wave 5 — closing the loop.**
14 (end game, archive + summary) with B4's cascade · 2b (whole-table cards) can land any time
after 2a.

---

## 7. What this file does not do

- **It does not scope anything.** Every row above still needs its own session, its own ground-truth
  pass and its own gate on Zach's decisions. The twelve decisions in §1 are the *product* answers;
  none of them says how a thing is built.
- **It does not add board rows.** Prioritising is Zach's call — the same rule HANDOFF 6 followed.
  §6 is a proposal.
- **It does not re-verify HANDOFF 6.** Facts marked re-verified were checked on 2026-09-17; the
  rest are carried, and fact 17 is explicitly flagged as possibly stale.
- **It does not touch kick Phases 4-5.** That is board row 7 and has its own plan.
