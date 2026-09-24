# PLAN — room state sync, option B: a coalesced, ordered, failure-tolerant room refetch

**Status: `PLANNED` 2026-09-23. Awaiting GATE 2.** Stage 4 of `docs/AGENT-PRACTICES.md` §2.2,
written 2026-09-23 by a Default-tier session. The harness reported the model as Opus 5.5, and
the tier table (`docs/AGENT-PRACTICES.md` Part 4) names Opus 5 for Default. The orchestrating
session dispatched it for Stage 4 only (board row 29's status, `PASSOFF.md:46`, "CLAIMED
2026-09-23 for Stage 4 only"). No code was written or edited to produce this file.

**Scope is option B and nothing else**, per `DESIGN.md` D2 (`DESIGN.md:390-400`). B ships as
board row 29's own item. E is board row 33 and is not this plan's. C is deferred into TRIAGE F4
and is not designed here. D1–D5 are frozen (`DESIGN.md:375-419`) and this plan reopens none of
them (R8). It turns the ratified *what and why* into *in what order, by whom, done when*.

**Read first:** `HANDOFF.md`, then `DESIGN.md` §3-B (`:236-264`), §4 (`:325-337`), §5
(`:341-371`) and §6 (`:375-433`), then this file. Where this file's §0 disagrees with
`DESIGN.md`, **§0 wins**.

---

## 0. Facts verified 2026-09-23 (these supersede `DESIGN.md` where the two differ)

This is the second verification pass. `DESIGN.md`'s Stage 1 read is dated 2026-09-17 and was
taken against `main` at `a796509` (`DESIGN.md:20-24`). **Where this table and `DESIGN.md` §1 or
§5 disagree, this table wins.** A build session re-runs this pass before its first edit
(Phase 1, scope step 0; `docs/AGENT-PRACTICES.md` §2.4). This table's numbers are then leads,
not facts (R3).

| # | Claim | Verified state, 2026-09-23 | Citation |
|---|---|---|---|
| 0.1 | Where `main` is | `a628b3a` (2026-09-23, "ratify room-state-sync gate 1, and add missing drawer/dialog descriptions"). `a796509` is an ancestor. | `git log -1 --oneline main`; `git merge-base --is-ancestor a796509 main` → exit 0 |
| 0.2 | `worktree-kick-phase4` (`8848ec7`, row 7 Phase 4) is on `main` | **No.** It is not an ancestor of `main`. Its merge-base is `dafd16d`. Row 7's Phase 5 (commit, push, walk) is what lands it. | `git merge-base --is-ancestor worktree-kick-phase4 main` → exit 1; `git merge-base worktree-kick-phase4 main`; `PASSOFF.md:24` |
| 0.3 | `worktree-frontend-sweep` (`5c311e7`, row 1) is on `main` | **No.** It is not an ancestor. Its merge-base is `f999188`. | `git merge-base --is-ancestor worktree-frontend-sweep main` → exit 1; `PASSOFF.md:18` ("NOT YET MERGED") |
| 0.4 | Trades, the third `page.tsx` branch `DESIGN.md` §5 names | **Merged.** `worktree-trades` (`30feb3f`) is an ancestor of `main`. It no longer blocks B. This supersedes the trades half of `DESIGN.md:343-349`. | `git merge-base --is-ancestor worktree-trades main` → exit 0; `PASSOFF.md:35` |
| 0.5 | `frontend/app/room/[code]/page.tsx` changed since `a796509` | **Yes:** +93/−3, one commit, `30feb3f` (trades, 2026-09-22). `DESIGN.md`'s `page.tsx` line numbers are stale. The remap is below. | `git diff --stat a796509 main -- 'frontend/app/room/[code]/page.tsx'`; `git log a796509..main -- <same>` |
| 0.6 | `frontend/hooks/use-public-fetch.ts` changed since `a796509` | **No.** The diff is empty, so `DESIGN.md`'s hook citations still hold: `applyResponse` `:18-26`, null-on-failure `:22-24`, `refetch` `:76-86`. Neither unmerged branch touches the file either. | `git diff --stat a796509 main -- frontend/hooks/use-public-fetch.ts` (empty); same against each branch's merge-base (empty) |
| 0.7 | Who uses `use-public-fetch.ts`. **Correction (R5)** to `DESIGN.md:72-74` ("whose only other user is the hook file itself") and `:350` ("used by `page.tsx` alone") | The **module** has three importers. `page.tsx:22` imports `usePublicFetch`. `components/room/create.tsx:11` and `components/room/join.tsx:13` import `usePublicAction`, which is defined in the same file at `:99-144`. **`usePublicFetch` itself has one caller**, `page.tsx`, at three call sites (`:47`, `:62`, `:77`). The same was true at `a796509`, so the claim was wrong when it was written. **Consequence:** B may change `usePublicFetch`'s contract. B must not change `usePublicAction`. | `git grep -n use-public-fetch main -- frontend`; `git grep -n use-public-fetch a796509 -- frontend` |
| 0.8 | The trades code on `main` already carries a skip list | `page.tsx:219-227` skips `refetchPlayers()` for `OFFER_RECEIVED`, `OFFER_SENT` and `OFFER_RESOLVED`. `:214-217` calls `refetchOffers()` on every non-`ERROR` frame. The third hook, for the inbox, is at `:71-83`. `DESIGN.md:77` places it on the trades branch; it is now on `main`. | `page.tsx` read at `a628b3a` |
| 0.9 | The `refetchProperties` list | `main` has three types: `PURCHASE_PROPERTY`, `MANAGE_PROPERTIES` and `PLAYER_KICKED` (`page.tsx:235-241`). `kick-phase4` adds `AUCTION_LOT_CLOSED`, so after the merge there are four. `DESIGN.md:14`'s "four message types" describes the file after the merge. | `page.tsx:236`; `git diff dafd16d worktree-kick-phase4 -- 'frontend/app/room/[code]/page.tsx'` |
| 0.10 | The `BID_PLACED` early return | **Not on `main`.** It exists only on `kick-phase4`. That branch's handler returns before the toast: it sets `liveBid` for the open lot, and for an unknown lot it calls `refetchPlayers()` and returns. | Same diff as 0.9; `DESIGN.md:168-176` |
| 0.11 | What `frontend-sweep` does to `page.tsx` | (1) `storedPlayerId` comes from `useStoredPlayerId(code)`. That is a new file on the branch, `frontend/hooks/use-stored-player-id.ts`, which returns `null` on the server and on the hydrating render. (2) The players hook passes `resourceParams: [code]`. (3) The inline generics go, in favour of a typed `api.service.ts`, with `GetPlayersResponse` at `:108` on that branch. The branch's `api.service.ts` predates trades and has **no `getOffers`**, so the merge must reconcile it. | `git diff f999188 worktree-frontend-sweep -- 'frontend/app/room/[code]/page.tsx'`; `git show worktree-frontend-sweep:frontend/hooks/use-stored-player-id.ts`; `git show worktree-frontend-sweep:frontend/lib/utils/api.service.ts \| grep -c getOffers` → 0 |
| 0.12 | A fourth branch touches `page.tsx`, and `DESIGN.md` does not name it | `worktree-ui-facelift` (`a086e7f`, board row 32, unmerged) changes `page.tsx` by +2/−3. It edits the two toast `className`s in the `ERROR` and success branches (`:199-211` on `main`) and drops the `josephinBold` import. The commits are `34be5ac` (2026-09-22) and `be5080e` (2026-09-23). The branch did not exist on 2026-09-17. Row 32's Lane 2 (Phases 5–6) is `HELD` on row 29. | `git diff main...worktree-ui-facelift -- 'frontend/app/room/[code]/page.tsx'`; `git log c14faa5..worktree-ui-facelift -- <same>`; `PASSOFF.md:49` |
| 0.13 | Nothing else holds a change to either file | Only `frontend-sweep`, `kick-phase4` and `ui-facelift` have committed changes to `page.tsx`. No branch changes `use-public-fetch.ts`. Across all 23 worktrees, no uncommitted change touches either file. | `git diff --stat main...<b> -- <the two files>` looped over every local branch that is not an ancestor of `main`; `git -C <w> status --short` over `git worktree list` |
| 0.14 | The `PLAYER_JOINED` and `PLAYER_LEFT` payloads | `PLAYER_LEFT` is sent at `handler.go:72-80` with payload `playerId` (`:76`). `PLAYER_JOINED` is sent at `:128-135` with `playerId` and `playerName` (`:131-132`). `DESIGN.md`'s citations still hold. A kicked player's `JOIN` is refused at `:118-121` and broadcasts nothing. | `grep -n` over `backend/websocket/handler.go` at `a628b3a` |
| 0.15 | A kicked player's `/players` fetch | **It succeeds.** `GetPlayersInRoom` queries `{roomId}` with no `isActive` filter (`controllers/playerControllers.go:34`, `:61-62`). The kicked player renders with a "No longer in the game" banner (kick `DESIGN.md:448-457`, D6 As-built 2026-09-17). **So D4's keep-last-good change cannot hide a kick.** The comment at `handler.go:116-117` ("their own room fetch failing is what their client reacts to") is stale against that As-built note. That is raised in §4, not fixed here (no drive-by fixes). | `playerControllers.go:18-117` read; `docs/incomplete/kick-player/DESIGN.md:448-457` |
| 0.16 | The failure path on `main` | `DESIGN.md` §1.2 holds, with more detail. The hook nulls `data` on a failed response (`:22-24`) **and** in both `catch` blocks (`:57-61`, `:80-82`). `page.tsx:355` builds `error` from the players and properties hooks only, so an offers failure silently empties the inbox. `DataState` renders `Fallback` on any truthy error (`components/containers/data-state.tsx:64-65`). Its `loading` is `!initialLoadComplete` (`page.tsx:376`). | Files read at `a628b3a` |
| 0.17 | What reconnect resyncs on `main`. **New at Stage 4**, read from code, not observed | `onopen` sends `JOIN` and nothing else (`page.tsx:320-322`). `onclose` reconnects after 1000 ms (`:328-332`). The echoed `PLAYER_JOINED` refetches players (`:221-227`) and offers (`:217`), **but not properties**, because `PLAYER_JOINED` is not in the `:236` list. A client that was disconnected during a purchase keeps a stale Bank's Properties list until the next property event. See dial 3.7. | `page.tsx:214-241`, `:320-332` |
| 0.18 | Frontend test runner | **None.** `frontend/package.json:5-10` has `dev`, `build`, `start` and `lint`. A grep for `jest\|vitest\|playwright\|testing-library\|mocha\|"test"` finds nothing. | `grep -n -iE` over `frontend/package.json` |
| 0.19 | React StrictMode | **On in dev.** `frontend/next.config.ts` sets nothing. Next 16.3.5's bundled doc says Strict Mode has been `true` by default under the app router since 13.5.1. It is dev-only, so `scripts/emoney dev` (`next dev --port 3000`, `scripts/emoney:84-87`) runs with it and a production build does not. | `frontend/node_modules/next/dist/docs/01-app/03-api-reference/05-config/01-next-config-js/reactStrictMode.md`; `frontend/node_modules/next/package.json` (`16.3.5`) |
| 0.20 | Which origins can reach the real API | The CORS allowlist (`backend/main.go:24-26`) and the websocket origin check (`handler.go:18-20`) name exactly `http://localhost:3000`, `https://emoney.club` and `https://www.emoney.club`. | `grep -n` over both files |
| 0.21 | The walk fixture `TRDCHK` exists | **Yes.** `GET /v1/rooms/TRDCHK/players` returned 200, 747 B, 0.50 s. That is the same byte count `DESIGN.md:111-113` measured on 2026-09-17. | `curl -s -o /dev/null -w '%{http_code} %{size_download} %{time_total}' https://api.emoney.club/v1/rooms/TRDCHK/players`, 2026-09-23 |
| 0.22 | Row 25's `default:` arm | **Unmerged.** `worktree-handler-default` (`cc22512`) is not an ancestor of `main`. **Deployment is not re-verified here**, because no read-only probe exists. Board row 25's correction, dated 2026-09-23, says the silence is still live in production. That claim is carried, not checked. | `git merge-base --is-ancestor worktree-handler-default main` → exit 1; `PASSOFF.md:42` |
| 0.23 | Rows 11 and 23, the `websocketManager.go` rewrites `DESIGN.md:353-355` names | Both branches are now **committed**, not uncommitted. Row 11's `f7704d1` is superseded by the merge `2c4908c`, whose message reads "supersedes worktree-broadcast-deadline". Row 23's `d587a05` (2026-09-22) is unmerged. **Neither touches `frontend/`.** Separately, five agent worktrees hold uncommitted edits to `backend/websocket/types.go` or `websocketManager.go`; none touches `frontend/`. | `git log -1 2c4908c`; `git diff --stat main...worktree-mongo-v2 -- frontend` (empty); same for `worktree-broadcast-deadline` (empty); per-worktree `git status --short` |
| 0.24 | D1–D5 are still the frozen state | **Yes.** `DESIGN.md` has two commits, `89e470b` (2026-09-22) and `a628b3a` (2026-09-23). No date after 2026-09-23 appears in the file. There is no supersession and no `As built:` note after §6. The only "supersedes" hit is D1's conditional at `:387`. | `git log -- docs/incomplete/room-state-sync/`; `grep -nE '2026-09-2[4-9]\|2026-1[0-2]-'` (none); `grep -ni supersed` |
| 0.25 | D4's copy, byte for byte | `Couldn't refresh the room. Retrying…`. The final character is **U+2026** (bytes `342 200 246`), not three dots. Separately, D4 says the unused variants "are recorded in §4's dial table" (`DESIGN.md:414-415`), but §4's copy row points to §6 (`:337`) and §6 carries no variants. They are recorded nowhere in `DESIGN.md`. The ratified string is unaffected. This is noted, not amended: `DESIGN.md` changes only by amendment, and that is not this document's job. | `grep -o 'Retrying.' DESIGN.md \| od -c` |
| 0.26 | Board row 29's "Files it owns" | It names only `page.tsx` (`PASSOFF.md:46`). Under D2, B also owns `frontend/hooks/use-public-fetch.ts`. Raised for the board. | `PASSOFF.md:46`; `DESIGN.md:392-393` |

**`page.tsx` line remap, `a796509` → `a628b3a`.** These are `main`'s numbers. The file after the
merge will move again, so the build session re-derives them (Phase 1, step 0).

| `DESIGN.md` cites (`a796509`) | What | On `main` at `a628b3a` |
|---|---|---|
| `:31-58` | players and properties hooks | `:42-69`; offers hook (trades) `:71-83` |
| `:60-68` | straight projections | `:85-94` |
| `:129-138` | `ERROR` early return | `:196-205` |
| `:147` (board row 29) | `refetchPlayers()` | `:219-227`, behind the `OFFER_*` skip |
| — | `refetchOffers()` on every frame | `:214-217` |
| `:160` / the properties list | `refetchProperties()` | `:229-241` |
| `:240-242` | `onopen` → `JOIN` | `:320-322` |
| `:248-252` | `onclose` → reconnect after 1000 ms | `:328-332` |
| `:274-302` | `error` → `DataState` | `:354-381` (`error` `:355`, `DataState` `:373-381`) |
| `:285-291` | one-way latch | `:365-371` |
| `room.client.tsx:36` | the unused `loading` prop | `room.client.tsx:53` (only occurrence, `grep -n loading`) |
| — | `handleWebSocketNotification`, an effect event | `:187-243` |

### §0 addendum — Phase 1 step 0, re-run 2026-09-23 by the build session (wins over the table above)

Run in the build worktree, cut from `main` at `91a8f29` ("merge ui-facelift lane 1 onto main").
Line numbers are `main`'s at `91a8f29`, **before** Phase 1's edits.

| # | Claim | Verified state | Citation |
|---|---|---|---|
| A.1 | Both landing preconditions, plus facelift Lane 1 | **All three are ancestors of `main`.** Supersedes 0.2, 0.3 and 0.12's "unmerged". | `git merge-base --is-ancestor worktree-kick-phase4 main && echo …`, same for `worktree-frontend-sweep` and `worktree-ui-facelift`: all printed; `git diff --stat main...worktree-ui-facelift -- 'frontend/app/room/[code]/page.tsx'` empty |
| A.2 | The two owned files since `a628b3a` | `page.tsx` +116/−35 over `8848ec7`, `5c311e7`, `34be5ac`, `be5080e` and their merges. `use-public-fetch.ts` unchanged (0.6 holds: `applyResponse` `:18-26`, catch-nulling `:57-61`, `refetch` `:76-86`). | `git diff --stat a628b3a main -- <both>`; `git log --oneline a628b3a..main -- <both>` |
| A.3 | Importers (0.7) | Holds. The module has three importers (`page.tsx:16`, `create.tsx:10`, `join.tsx:12`); `usePublicFetch` has one caller, `page.tsx`, at three sites (`:52`, `:63`, `:76-78`). | `git grep -n use-public-fetch main -- frontend` |
| A.4 | The merged handler's shape | As §2 step 0 expected, so no step changes: `ERROR` return `:226-235`; `BID_PLACED` early return `:245-268` (`liveBid` overlay `:248-258`, unknown-lot `refetchPlayers()` `:266-267`); success toast `:270-275`; `refetchOffers()` on every frame `:284`; `OFFER_*` skip `:288-294`; `refetchProperties` list `:310-319`, four types with `AUCTION_LOT_CLOSED` at `:315`; `onopen` → `JOIN` `:398-400`; `onclose` → reconnect after 1000 ms `:406-410`; socket deps `[code, storedPlayerId]` `:430`; `storedPlayerId` from `useStoredPlayerId(code)` `:40` | `page.tsx` read at `91a8f29` |
| A.5 | Toast styling after facelift Lane 1 | Both landed toasts use `duration: 4000`, `position: "top-center"`, `className: \`font-semibold text-sm text-center\`` (`:229-233`, `:270-275`). The facelift's `text-xs` → `text-sm` is in. | same read |
| A.6 | Backend payloads (0.14) | Hold, lines moved: `PLAYER_LEFT` `handler.go:109-116`, `playerId` `:112`; `PLAYER_JOINED` `:188-195`, `playerId` `:191`. `types/events.ts` declares neither shape. | `grep -n` over `backend/websocket/handler.go`, `frontend/types/events.ts` |
| A.7 | **Correction (R5) to §2 step 3 ("so the initial load still ends in `Fallback`"), done-when (b)'s last bullet, and interleaving 7** | **On `main` a failed initial load does not show `Fallback`; it shows the dice loader until some later fetch succeeds.** `DataState` returns `LoadingComponent` whenever `loading` is true (`data-state.tsx:62`), before it reads `error` (`:64-65`), and the page passes `loading={!initialLoadComplete}` (`page.tsx:454`), which stays `true` until players and properties have both arrived (`:447-449`). So on `main`, `Fallback` was reachable only by a failure *after* the initial load, which is exactly the path D4 closes. Phase 1 keeps the pre-success path "exactly as it does on `main`" (step 3) and does **not** change the loader (no drive-by fix). After Phase 1, `Fallback` is unreachable from the room page. Whether a failed initial load should show `Fallback` is raised for Zach, not decided here. | `data-state.tsx:62-66`; `page.tsx:447-454` at `91a8f29` |
| A.8 | §0.13's all-worktrees loop | **Not re-run.** The build ran worktree-isolated, and its git guard refuses git against any other checkout. Carried, not checked: the dispatcher's own loop over 24 worktrees, recorded in the primary checkout's uncommitted Phase 1 header, found no uncommitted edit to either owned file. | the guard's refusal, this session |
| A.9 | Test runner (0.18) | Still none. | `grep -n -iE 'jest\|vitest\|playwright\|testing-library\|mocha\|"test"' frontend/package.json` → exit 1 |
| A.10 | D4's copy (0.25) | Holds: `Retrying` + bytes `342 200 246` (U+2026). | `grep -o 'Retrying.' DESIGN.md \| od -c` |

---

## 1. Decisions taken since ratification

Stage 4 wrote no code and took no build-level call. The calls this plan expects the
build to take are listed in §3 as *BD at build*. Each already carries its one-line reversal, so
the builder does not have to invent one. The build session records each here as `BD-1`, `BD-2`,
… in the order it takes them (R12).

**Phase 1's build, 2026-09-23 (uncommitted at the time of writing; line numbers are the final
built files': `use-public-fetch.ts` sha1 `8cecb0b…`, `page.tsx` sha1 `61f3ddc…`).** BD-1 to BD-5
take §3's recommended default each. BD-6 and BD-7 are not §3 dials. They are readings of step 3's
text and of "at most one in flight", recorded so they are visible and reversible, not so they
look like new scope. BD-8 is new scope that Zach approved after the review raised it.

- **BD-1 — dial 3.1, coalescing shape: trailing edge, no delay.** A `refetch()` mid-flight sets
  `run.dirty` (`use-public-fetch.ts:212-214`), and the settle path fires exactly one follow-up at
  once (`:195-197`). *Reversal:* put a debounce window in front of the trailing refetch.
- **BD-2 — dial 3.8: the `onopen` resync fires on every open, the first included.**
  `resyncRoom` (`page.tsx:379-383`) is called after the `JOIN` send (`:460-463`). On a first
  load it folds into whatever is in flight as a trailing follow-up. **Observed cost (walk,
  R1.0):** the resync (~352 ms) and this client's own `PLAYER_JOINED` echo (~414–454 ms) fell into
  different in-flight windows, so a first load made three sequential `GET /players` where `main`
  makes two (mount plus echo). It is bounded, correct and loud, and it is this dial's price.
  *Reversal:* skip the `onopen` refetch until the mount fetch has applied its first success.
- **BD-3 — dial 3.9: keep-last-good and the retry live in the hook, so all three hooks get
  them. D4's toast goes to the players and properties hooks only** (`page.tsx:78`, `:90`). The
  offers hook retries silently. One sonner `id`, `room-refresh-failed` (`page.tsx:45-55`), so two
  hooks failing together, or a second failure before a success, update one toast and do not
  stack. *Reversal:* pass `onRefetchError: toastRoomRefreshFailed` to the offers hook too.
- **BD-4 — dial 3.10: "known id" means the id is in `playersData.players`,** the last
  successfully applied list, which includes inactive players. With no list applied yet, it
  refetches (`page.tsx:323-343`). This client's own id always refetches. *Reversal:* compare
  against the rendered `otherPlayers` instead.
- **BD-5 — dial 3.11: `error` is set only before a run's first success** (`use-public-fetch.ts:165-170`).
  **Later failures surface through a new `onRefetchError` option** (`:55-60`), fired once per
  failure cycle (`:172-177`). A callback rather than a returned field, because the toast is an
  event, and a field would need an effect in the page to watch it. *Reversal:* restore
  `applyResponse`'s null-on-failure (`use-public-fetch.ts:22-24` at `91a8f29`).
- **BD-6 — reading of step 3's retry text (not a §3 dial).**
  - (i) Retries run only after a success: a pre-success failure behaves exactly as on `main`, so
    no retry and no toast.
  - (ii) Each external `refetch()` resets the retry count (`:203-210`). Each message whose own
    fetch fails therefore gets up to three retries, while the toast stays once per cycle, where a
    cycle ends on a success.
  - (iii) A failure with the dirty flag set schedules no retry, because the trailing follow-up
    *is* the next attempt (`:181-183`, interleaving 3).
  - (iv) Each delay counts from the previous failure, so fast failures retry at about 1 s, 3 s
    and 7 s after the first one (`RUNTIME-PASS.md` R1.2).

  *Reversal:* for (i), retry pre-success failures too; for (ii), reset `retriesUsed` only on a
  success.
- **BD-7 — "at most one request in flight" is per effect run (not a §3 dial).** A change of
  `enabled` or `dependencies`, and StrictMode's dev remount, dispose the run
  (`use-public-fetch.ts:223-236`). Its in-flight request is abandoned on the wire, its response
  is dropped (`:153`), and the new run fetches at once rather than waiting. The walk saw exactly
  this under `next dev` from the build worktree: two overlapping `GET /properties` at 208 ms and
  209 ms, then one trailing follow-up at 430 ms, after the second settled (`RUNTIME-PASS.md`
  R1.0). *(An earlier "smoke load" cited here ran the primary checkout's `main`, not this build.
  It is retracted; see R1.0.)*
  `refetch()` while disabled is now a no-op (`:243-245`); on `main` it fetched with whatever
  params it had. *Reversal:* make a new run wait for the abandoned request to settle before its
  first request.
- **BD-8 — a per-request timeout of 10 s (`REQUEST_TIMEOUT_MS`, `use-public-fetch.ts:22`).**
  - **Why.** The review found that single-flight lets one GET that never settles freeze that
    read, the `onopen` resync included. Zach approved adding a timeout on 2026-09-23; the
    coordinating session relayed the approval.
  - **How it works.** Each request races the timer (`:132-141`). A timeout settles as
    `{ success: false, error: { status: 408, timedOut: true } }` and takes the ordinary failure
    path: `inFlight` clears, then the retry or the dirty follow-up, and D4's toast once the room
    has loaded. The timer is cleared on settle (`:146-149`) and on dispose (`:229-232`).
  - **Why 10 s.** It is 20× the measured `/players` round trip (0.5 s, §0.21). It is also well
    clear of DevTools' Slow 3G profile (about 2 s of latency) for a sub-kilobyte payload, so walk
    (a) should never trip it.
  - **Why a race in the hook, and not axios's `timeout` or an `AbortSignal`.** Both would mean
    editing `api.ts` or `api.service.ts`, which sits outside Phase 1's two files, fails its
    scope check, and would change `usePublicAction` too. So the abandoned request is not
    cancelled, only ignored.
  - **Why the error has no `message`.** `Fallback` prints `error.message` (`fallback.tsx:44-46`),
    and a new string there would be new copy (R7).
  - **Review.** The same review subagent re-checked this delta: T1 prevented with a caveat, T2–T5
    prevented, and all earlier verdicts unchanged.
  - *Reversal:* change the constant, or delete the race and the two `clearTimeout` blocks.

**Raised by the Phase 1 review subagent, 2026-09-23. Each one is the owner's call.**
- **RESOLVED by BD-8: a GET that never settles stalled that hook's refreshes (medium
  confidence).** Before the timeout, single-flight turned every later trigger into
  `dirty = true` while `run.inFlight` stayed set, and `frontend/lib/utils/api.ts` sets no axios
  `timeout` (`grep -n timeout` → exit 1). Now each attempt is bounded at 10 s.
  - **Caveat from the re-review.** The abandoned request is not cancelled. A retry can ride the
    same dead connection and time out again, so recovery waits on the browser giving up on it.
    A dead-API cycle now spans about 47 s (four 10 s attempts plus 1, 2 and 4 s of backoff), not
    about 7 s.
  - **What walk (a) may see.** If any GET goes past 10 s under Slow 3G, the Network panel can show
    more than two `/players`. The extra one is the ignored, abandoned request, not a coalescing
    failure.
- **Once the retries run out, later failures are silent (low confidence; matches the stated
  contract).** `failing` resets only on a success (`use-public-fetch.ts:161`, `:174-177`), so a
  later message's failed refetch restarts the retries with no toast. The toast lasts 4 s, and
  the retries take about 7 s, or up to about 47 s when every attempt times out. Any new copy for
  this is an R7 ask.
- **The initial-load `Fallback` is unreachable, and the cause is older than this plan.** See
  step 3's correction and §0 A.7. This is a new board item.
- **Outside this diff (it came with `frontend-sweep` `5c311e7`; medium-low confidence; not
  observed).** On a hard reload, the hydrating commit's passive effects may run while
  `storedPlayerId` is still `null`, so the socket effect's "No player found for this room" toast
  (`page.tsx:446-448`) may fire on every direct load of a room URL. *Not reproduced on the one
  direct load checked for it:* a body-text read 1 s after load found "has joined the game" and no
  "No player found" (`RUNTIME-PASS.md` R1.0). One sample does not clear it.

---

## 2. Phases

| # | Phase | Driver | Subagents | Est. context | Why that shape |
|---|---|---|---|---|---|
| 1 | Coalesce, order and fail tolerantly in `usePublicFetch`; stop the no-op room refetches and make reconnect resync explicit in `page.tsx` | Opus 5 (Default) | One Opus 5 review subagent, in its own worktree, read-only | `comfortable` | Two files, one subsystem, no Go, no wire change (D2; `DESIGN.md:258-259`, "Small: two files, no Go"). The four changes cannot be split safely. Coalescing, ordering and keep-last-good all rewrite the same request path (`use-public-fetch.ts:41-86`). D3's skip list without its explicit `onopen` resync would remove the only reconnect catch-up (`DESIGN.md:127-132`), so the two ship in one commit. **Fixed reading:** about 123 KB, or roughly 31k tokens (`wc -c` of `AGENT-PRACTICES.md`, `CLAUDE.md`, `DESIGN.md`, `page.tsx`, `use-public-fetch.ts`, `data-state.tsx` and `fallback.tsx`, divided by 4, on 2026-09-23), plus this file. That is set against Default's ~400k ceiling (`docs/AGENT-PRACTICES.md` Part 5). |

### Phase 1 — Coalesced, ordered, failure-tolerant room refetch

**Status: `BUILT 2026-09-23, commit <owed: fill in the hash once Zach commits>`.**
- **Where.** Built in worktree `worktree-agent-a44b106d831121f4d`, cut from `main` at `91a8f29`.
  Both preconditions and GATE 2 were met (§0 addendum A.1).
- **Gates.** Green, and all three scope checks empty.
- **Review.** Eight interleavings, then a re-check of the timeout delta. The verdicts are in the
  close-out; #7 fails only on the pre-existing latch (step 3's correction).
- **Walk** (`RUNTIME-PASS.md`):
  - R1.0 and R1.3 (both halves) walked.
  - (a), (b) and (e) wait on Zach's own go-ahead in the build session, or on his own walk.
  - (d) waits on the deploy.
- *(Merge note: the primary checkout holds an uncommitted "IN FLIGHT" edit to this same
  paragraph. This BUILT text supersedes it.)*

Driver: Opus 5. **Files it owns:**
`frontend/app/room/[code]/page.tsx`, and `frontend/hooks/use-public-fetch.ts`, whose
`usePublicFetch` export only it may change. It owns no other file.

**Scope.** Each step is written so it can be executed without re-reading `DESIGN.md`.

0. **Re-run §0 against `main` before the first edit** (`docs/AGENT-PRACTICES.md` §2.4). Run
   `git log -1 --oneline main`, the two ancestor checks in *Done when*, and
   `git diff --stat a628b3a main -- 'frontend/app/room/[code]/page.tsx' frontend/hooks/use-public-fetch.ts`.
   Rerun §0.13's branch-and-worktree loop over both files, and check §0.12's facelift branch.
   Then read the **merged** `handleWebSocketNotification`, and write down each of these at the
   merged file's own line numbers:
   - the `ERROR` return;
   - the `BID_PLACED` early return (the `liveBid` overlay, and the `refetchPlayers()` fallback for an unknown lot);
   - the toast;
   - `refetchOffers()` on every frame;
   - the `OFFER_*` skip on `refetchPlayers()`;
   - the `refetchProperties` list (expect four types, `AUCTION_LOT_CLOSED` among them);
   - `onopen` → `JOIN`;
   - `onclose` → reconnect;
   - `storedPlayerId` coming from `useStoredPlayerId`.

   Record the result as a dated addendum under §0; the addendum wins over this plan's numbers.
   If the merged shape differs from that list in a way that changes a step below, stop and ask
   before editing (R12).
1. **Coalesce, in `usePublicFetch`.** Each hook instance, which means each resource, has at
   most one request in flight. A `refetch()` call that arrives while a request is in flight sets
   a dirty flag and returns. When the in-flight request settles with the flag set, exactly one
   follow-up request fires, immediately (trailing edge, no delay; dial 3.1). Keep the in-flight
   flag, the dirty flag and the request counter in `useRef`, because `refetch` is a new function
   on every render (`use-public-fetch.ts:76`). Route the mount effect's fetch (`:41-74`) and
   `refetch` (`:76-86`) through **one** request function, so those two paths cannot race each
   other.
2. **Order, in the same function.** Every request takes a rising number from a ref. A response
   is applied only if its number is higher than the last one applied; otherwise it is dropped.
   This covers the mount fetch, `refetch`, and a change of dependency (`:74`).
3. **Keep the last good room on a failed refetch (D4).** Once a hook has applied one success, a
   later failure no longer nulls `data` and no longer sets the `error` the page hands to
   `DataState`. A failure here means a response with `success: false` (`:22-24`) or a thrown
   error (`:57-61`, `:80-82`).

   Before the first success, a failure behaves exactly as it does on `main`, so the initial load
   still ends in `Fallback`.

   > **Correction (R5), 2026-09-23 — the second half of that sentence is disproven.** The
   > first half was built as written. The initial load does **not** end in `Fallback`, on `main`
   > or after Phase 1. `initialLoadComplete` latches only once `playersData` and
   > `propertiesData` are both non-null (`page.tsx:510-512` as built), so after a failed initial
   > load the page passes `loading={!initialLoadComplete}` = `true` (`:517`). `DataState` checks
   > `loading` (`data-state.tsx:62`) before `error` (`:64-65`), so the dice loader renders and
   > `Fallback` is never reached. The latch predates this plan (commit `3389b36`, 2025-05-21, per
   > the Phase 1 review), and Phase 1 did not touch it. It is a pre-existing bug, raised as a
   > new board item and not fixed here. See §0 addendum A.7.

   Retry with backoff: 3 attempts, at 1 s, 2 s and 4 s after the failure, then stop and wait
   for the next `refetch()` call. Each value is one named constant (dial 3.2). A `refetch()`
   that arrives while a retry is pending fetches at once and cancels the pending retry. Clear
   every timer on unmount and on a change of dependency.

   The page shows D4's toast, exactly `Couldn't refresh the room. Retrying…` with U+2026
   (§0.25), once per failure cycle, for the players or the properties hook (dial 3.9). Style it
   as the `ERROR` branch's `toast.error` is styled in the landed file (§5, the facelift hazard).
4. **Stop refetching the room where nothing changed (D3).** In the merged
   `handleWebSocketNotification`, add a skip to `refetchPlayers()` for:
   - `PLAYER_LEFT`, always;
   - `PLAYER_JOINED` whose `payload.playerId` is in the last applied players list **and** is not
     `storedPlayerId` (dial 3.10).

   Extend the existing `OFFER_*` skip idiom (`page.tsx:219-227` on `main`) rather than adding a
   second one: HANDOFF 35 already records two colliding idioms in this handler.

   **Nothing else changes.** Both messages still toast. `refetchOffers()` still fires on every
   non-`ERROR` frame, these two included (`DESIGN.md:425-428`, "trades' `refetchOffers` on every
   frame" survives unchanged). The properties list, the `BID_PLACED` branch, the `OFFER_*` skip
   and the `ERROR` branch are all untouched.
5. **Make reconnect resync explicit (D3).** In `socket.onopen`, after `sendMessage(... "JOIN" ...)`
   (`page.tsx:320-322` on `main`), refetch the room. Which resources it fetches is dial 3.7, a
   GATE 2 question. It fires on every open, the first included (dial 3.8).

   Route the call through a `useEffectEvent`. That is the house pattern stated at
   `page.tsx:187-188`, and it keeps the socket effect's dependencies at `[code, storedPlayerId]`
   (`:352`), so the socket does not reconnect on every render.
6. **Record it** (Part 7). The phase header becomes `**BUILT <date>, commit <hash>**`. Each call
   taken goes in §1 as a `BD-n` with its reversal. `DESIGN.md` gets an `As built:` note under D3
   and D4 for anything the build chose. `docs/incomplete/room-state-sync/RUNTIME-PASS.md` gets
   the entries for proofs (a)–(e) below, with fixtures. The session ends with the three
   hand-back blocks.

**Subagents.** **One, Opus 5 (Default), review only.** It runs in its own worktree, with the
model passed explicitly, and the diffstat is checked when it returns (`docs/AGENT-PRACTICES.md`
Part 4: "A subagent spawned into the shared worktree will edit source even when asked only to
review").

Its work-list is eight interleavings. For each one it says whether the code prevents the
failure, and how it knows, with a `file:line`:
1. three messages arrive while one `/players` GET is in flight: exactly two GETs, and the second
   starts only after the first settles;
2. the mount fetch is in flight when the `JOIN` echo's refetch fires: never more than one
   request in flight, and never an older response applied after a newer one;
3. a refetch fails while the dirty flag is set: the retry and the trailing follow-up do not
   both fire;
4. a message arrives while a retry timer is pending: the fetch happens at once and the timer is
   cancelled;
5. the page unmounts with a request in flight and a retry pending: no state update after
   unmount, and no timer left behind;
6. the dependency changes mid-flight (`storedPlayerId` going from `null` to an id under
   `useStoredPlayerId`): the response for the stale dependency is not applied;
7. the initial load fails: `Fallback` shows, and its Try Again (`page.tsx:378-381` on `main`)
   recovers; *(the review found this NOT PREVENTED, because of the pre-existing latch in step 3's
   correction, not because of Phase 1's diff)*
8. all three retries fail: no further request goes out until the next message, and the toast
   does not stack.

The verdict is pasted into the close-out, not summarised.

**Why Default and not Deep.** `DESIGN.md:256-259` classes B's failures as loud: a stale screen
on the first walk, never wrong money. D2 ratified B at the Default tier, and a Deep subagent is
for silent failures (Part 4). **Why a subagent at all:** this repo has no frontend test runner
(§0.18), so this concurrency logic has no gate. Part 4's Default subagent row covers "anything
where a wrong answer would be believed and the gates would not catch it".

No Mechanical subagent: step 0's re-check is about ten git commands, cheaper to run inline than
to delegate.

**Done when.**

*Precondition, checked before step 0.* The phase does not start until both of these print, and
the build session runs them itself at build time rather than trusting this plan (§2.4):

```bash
git merge-base --is-ancestor worktree-kick-phase4 main && echo "kick-phase4 landed"
```
```bash
git merge-base --is-ancestor worktree-frontend-sweep main && echo "frontend-sweep landed"
```

If either branch was landed by cherry-pick or rebase rather than by merge, the ancestor check
fails even though the content is on `main`. In that case, prove the content instead: `page.tsx`
on `main` holds the `BID_PLACED` early return with `setLiveBid`, `AUCTION_LOT_CLOSED` in the
`refetchProperties` list, and `useStoredPlayerId`. B's coalescing is written against that merged
handler (`DESIGN.md:343-349`), not against `main` at `a628b3a`.

*Gates* (`CLAUDE.md`, *Commands*), all green:

```bash
cd frontend && bun install && bun run lint
```
```bash
cd frontend && bunx tsc --noEmit
```
```bash
cd frontend && bun run build
```

*Scope checks*, all of which must be empty. They show no Go was touched (D2, "No wire contract
changes"), nothing outside the two owned files changed, and `usePublicAction` is untouched:

```bash
git diff --stat main -- backend/
```
```bash
git diff --stat main -- frontend/ ':!frontend/app/room/[code]/page.tsx' ':!frontend/hooks/use-public-fetch.ts'
```
```bash
git diff main -- frontend/hooks/use-public-fetch.ts | grep -n 'usePublicAction'
```

*The proof no gate can supply.* This is a live walk: two browser sessions (two profiles, or a
phone and a laptop) in one throwaway room, against the real API. `TRDCHK` exists as of
2026-09-23 (§0.21); re-check it with the §0.21 `curl`, or create a fresh room. Run locally
through `scripts/emoney dev`; port 3000 is required by `backend/main.go:24` and
`handler.go:18`. Keep DevTools' Network panel open on the observing client, filtered to
`/players`, `/properties` and `/offers`.

- **(a) Coalescing.** Throttle the observer to *Slow 3G*. From the other client, fire five banker
  transactions or transfers in quick succession. The observer shows **at most two**
  `GET /players` for the burst (the one in flight plus one trailing). Its final balances equal a
  direct `GET /v1/rooms/<code>/players`.
- **(b) Failure tolerance.** Use DevTools request blocking on `*/players` at the observer, and
  have the other client make a transfer.
  - The observer's room **stays on screen**, with no full-screen "Something went wrong".
  - A toast reading exactly `Couldn't refresh the room. Retrying…` shows **once**.
  - The Network panel shows three retries at about 1 s, 2 s and 4 s, and then nothing.
    **Corrected 2026-09-23:** each delay counts from the *previous* failure, so fast-failing
    blocked requests land at about 1 s, 3 s and 7 s after the first one (`RUNTIME-PASS.md`
    R1.2).
  - Unblock, and the next action brings one `GET /players` and a current room.
  - Then reload the page with the block still on: the full-screen `Fallback` **does** show,
    because it is an initial load (D4). **Corrected 2026-09-23 (R5):** it will not. The dice
    loader shows and stays, as on `main`. See the correction under step 3.
- **(c) D3's skip.** The other client closes its tab. The observer toasts "… has left the game"
  and makes **zero** `GET /players` and one `GET /offers`. The other client reopens the room: the
  observer toasts "… has joined the game" and again makes **zero** `GET /players` and one
  `GET /offers`.
- **(d) Explicit reconnect resync.** Walk this on a real phone against a production build: the
  deployed site after Zach pushes, because `next dev` double-mounts the socket effect under
  StrictMode (§0.19).
  - Take the observer offline for about 5 s.
  - While it is offline, have the other client make a transfer.
  - Bring the observer back online. Within about 1 s of the socket reopening, `onopen` fires a
    refetch of the set dial 3.7 settles, and the missed transfer is on screen.

  A desktop dev walk can approximate this with DevTools *Offline*, noting the StrictMode caveat.
- **(e) An unknown id still refetches.** A new player joins through the Join screen in a third
  session. The observer fires `GET /players` and the new card appears.
- **(f)** The review subagent's verdict on its eight interleavings is pasted into the close-out.
  **The ordering guard (step 2) cannot be forced reliably in a live walk.** Its proof is (f),
  and the close-out says so (R10).

The close-out marks each of (a)–(f) *walked* or *not walked* (R10). "Gates green, not seen
running" is a separate claim and is never merged with these.

**Watch for.**

- **The merged handler is the target, not `main` at `a628b3a`** (§0.9–0.11). `BID_PLACED`'s early
  return stays above the toast. Its unknown-lot `refetchPlayers()` must still fire; it coalesces
  like any other refetch. The skip list must not swallow it.
- **D3's skip applies to the room read only.** `refetchOffers()` stays on every frame, because it
  is trades' recovery for a dropped frame (`DESIGN.md:136-139`, `:425-428`). Do not fold it into
  the skip.
- **This client's own `PLAYER_JOINED` echo still refetches.** That is pinned by the phrase "not
  this client's own" (`DESIGN.md:249`, `:335`). Do not optimise it away: coalescing absorbs the
  overlap with the `onopen` refetch.
- **Under `frontend-sweep`, `storedPlayerId` is `null` on the hydrating render** (§0.11). The
  own-id check compares against the value after hydration. The socket effect already returns
  early while it is `null` (`page.tsx:306-309` on `main`).
- **No new user-facing copy beyond D4's one string.** A "retries exhausted" line, a "back online"
  line or any other string is an R7 ask to Zach, with variants, and never a `BD-n`.
- **Keep-last-good must not mask the initial load** (D4). A failure before the first success
  still produces `Fallback`.
- **A kicked player's refused-`JOIN` reconnect fires one `onopen` refetch.** That is expected and
  harmless, and the refetch succeeds (§0.15).

---

## 3. Dials

The first six are carried from `DESIGN.md` §4 (`:330-337`). The last five are new at Stage 4.
Each dial is one named constant or one named rule, never a literal typed twice
(`DESIGN.md:327-328`).

| # | Dial | Value | Pinned by | Latitude for the build |
|---|---|---|---|---|
| 3.1 | Coalescing shape | Trailing edge, no delay: fire at once, with one follow-up if anything arrived mid-flight | **No D-number.** It is §3-B item 1's own description of the ratified change (`DESIGN.md:241-243`) and §4's recommended default (`:332`) | **BD at build.** It implements D2's ratified B, so under R12 the builder takes the default and records it as a `BD-n`. It is **not** re-asked of Zach. *Reversal:* put a debounce window in front of the trailing refetch. |
| 3.2 | Failed-refetch retry | 3 attempts at 1 s, 2 s and 4 s, then wait for the next message | **D4** (`DESIGN.md:411-413`) | None. |
| 3.3 | Reconnect resync | An explicit refetch on `onopen`, after `JOIN` | **D3** (`DESIGN.md:403-406`) | None on *whether*. *What* it fetches is 3.7. |
| 3.4 | Messages that skip the room refetch | `PLAYER_LEFT`; `PLAYER_JOINED` for a known id that is not this client's | **D3**, with the qualifier from §3-B item 4 and §4 (`DESIGN.md:248-249`, `:335`) | None. What counts as "known" is 3.10. The list stays a **closed list of two named types**, never a "skip unless recognised" rule (§5, row 25). |
| 3.5 | Event-history pane | Stays in the `/players` payload; E caps it | *Rules that survive unchanged* (`DESIGN.md:432-433`), plus D2 (E is row 33) | Not B's to touch. |
| 3.6 | Failed-refetch toast copy | `Couldn't refresh the room. Retrying…` (U+2026, §0.25) | **D4** (`DESIGN.md:413-414`). R8: not reopened | None. |
| 3.7 | **Reconnect resync scope** (new) | Players **and** properties **and** offers | **D6** (`DESIGN.md`, ratified 2026-09-23, asked at GATE 2 alongside the facelift merge-order question in §5 below) | None. Ratified as asked; not a `BD-n`. |
| 3.8 | Resync on the first open (new) | Recommended: fire on every open, the first included. Coalescing absorbs the overlap with the mount fetch | Unpinned | **BD at build.** *Reversal:* skip the `onopen` refetch until the mount fetch has applied its first success. |
| 3.9 | Which failures toast (new) | Recommended: keep-last-good and the retry live in the hook, so all three hooks get them. D4's toast fires for the players or properties hook, which is "the room" (`page.tsx:355`). An offers failure keeps its last good inbox and retries without a toast: offers errors are already silent on `main` (§0.16), and the next frame's `refetchOffers()` is their backstop. One toast per failure cycle, with a stable sonner `id` so retries do not stack | Implements D4 | **BD at build.** *Reversal:* toast for offers too, with the same copy. |
| 3.10 | "Known id" (new) | Recommended: the id appears in the last **successfully applied** players list, which includes inactive players (`playerControllers.go:61-62`). If no list has been applied yet, refetch | Implements D3 | **BD at build.** *Reversal:* compare against the rendered `otherPlayers` instead of the raw list. |
| 3.11 | How the page learns of a refetch failure (new) | Recommended: the hook's existing `error` is set only until the first success, so `DataState` keeps its initial-load meaning. Later failures surface through a separate field or callback, which the page uses for the toast | Implements D4 | **BD at build.** *Reversal:* restore `applyResponse`'s null-on-failure (`use-public-fetch.ts:22-24`). |

---

## 4. Seams reserved, deliberately not built

- **Option C.** C is apply-from-payload with server-authored post-state and a per-room `seq`.
  **It is deferred into TRIAGE F4 by D2** (`DESIGN.md:394-398`), and none of it is built here,
  not partially either. There is no reducer, no `seq`, and no reading of payloads beyond what
  already exists: kick's `BID_PLACED` overlay (`DESIGN.md:168-176`) and D3's read of
  `payload.playerId`. F4 also decides §3-C's commit-order question (a per-room mutex,
  transaction-scoped `seq`, or per-document versions), not this row. B's coalesced refetch is the
  fallback C keeps (`DESIGN.md:255-257`), so nothing here is thrown away later.
- **Option E.** E makes `GetPlayersInRoom` cost 3 queries instead of `P + 3` and caps the
  `EventHistory` read. **It is board row 33** (`PASSOFF.md:50`: Sonnet 5, lane G), its own row
  and not folded in (D2, D5). It is `controllers/playerControllers.go`'s change, not this one.
- **A and D stay rejected**: A as `SETTLED AS NO` (`DESIGN.md:225`), D as "Rejected, and recorded
  so it is not proposed again" (`DESIGN.md:306-313`).
- **The ordering guarantee is a seam another row consumes.** Board row 32's Phase 5 (the
  cash count-up) names the missing ordering guard as its top hazard: "Motion can make a stale
  screen look live" (`docs/incomplete/ui-facelift/PLAN.md:913` on `worktree-ui-facelift`). B
  puts that guarantee in `usePublicFetch` and adds no new API for it. Phase 5 reads the same
  props `page.tsx` passes down.
- **No frontend test runner is added.** Raising one is a GATE 0 reopen (`DESIGN.md:359-361`).
- **A targeted "you were removed" notice** is kick D6's follow-on (kick `DESIGN.md:443-457`) and
  is not built here.
- **The stale comment at `handler.go:116-117`** (§0.15) is raised, not fixed. It is a Go file,
  outside B's two files, and fixing it would be a drive-by.

---

## 5. Repo hazards, with live numbers

- **The `page.tsx` merge is the gate.** Two branches are unmerged as of 2026-09-23:
  `kick-phase4` (`8848ec7`, +94/−3 against `dafd16d`) and `frontend-sweep` (`5c311e7`, +10/−17
  against `f999188`). Row 7 Phase 5 merges them by hand. HANDOFF 35 records a stated collision
  in `handleWebSocketNotification`, two idioms for "do not toast this one", and tells Phase 5 to
  keep the early return (`DESIGN.md:343-349`). B is written against the result, not against
  `main` at `a628b3a`.
- **`ui-facelift` is a third unmerged `page.tsx` diff, new since `DESIGN.md`** (§0.12,
  `a086e7f`, +2/−3). It restyles the two toast `className`s. Row 32's own plan says
  "`text-xs` → `text-sm` is two call sites" (`docs/incomplete/ui-facelift/PLAN.md:681` on its
  branch), and **B adds a third toast call site.**

  **Ratified 2026-09-23, at GATE 2, alongside D6 above:** facelift Lane 1 → B → facelift Lane 2.
  Land row 32's Lane 1 first (it is already built), then Phase 1 below, then row 32's Lane 2,
  which already waited on this row (`PASSOFF.md:49`). Not recorded as a `DESIGN.md` `D`-number —
  it is a build-sequencing call between two board rows, not a design decision about option B
  itself — but it is settled and not to be re-litigated (R8): a build session hand-back must not
  propose building B before row 32's Lane 1 lands.

  It was never a hard precondition like the other two: it collides on two lines of styling, not
  on the handler's logic. Separately, row 32's board status said the facelift "has deliberately
  never touched" `page.tsx` (`PASSOFF.md:49`) — §0.12 contradicted that, and both `PASSOFF.md`
  rows 29 and 32 are corrected as of 2026-09-23.
- **Correction (R5) to `DESIGN.md:353-355`.** That hazard, "Rows 11 and 23 hold uncommitted
  rewrites of `websocketManager.go` … Any server-side option collides with both and waits for
  both", **applies to the server-side options (C), not to B.** B touches no file under
  `backend/` (D2, `DESIGN.md:429`), and Phase 1's scope checks prove it with an empty
  `git diff --stat main -- backend/`.

  The facts under that hazard have also moved (§0.23):
  - row 11's branch is committed as `f7704d1` and superseded by `2c4908c`;
  - row 23's is committed as `d587a05` and unmerged;
  - neither touches `frontend/`;
  - five agent worktrees hold uncommitted backend websocket edits, and none touches `frontend/`.
- **`use-public-fetch.ts` has three importers, while `usePublicFetch` has one caller** (§0.7).
  This corrects `DESIGN.md:72-74` and `:350`. `usePublicAction` (`:99-144`), which
  `create.tsx:11` and `join.tsx:13` use, is outside B's scope, and a scope check enforces that.
- **React StrictMode double-mounts the socket effect in dev** (§0.19; `DESIGN.md:365-366`).
  Reconnect behaviour under `scripts/emoney dev` looks worse than it will in production, so
  proof (d) is walked on a real phone against a production build.
- **There is no frontend test runner** (§0.18; `DESIGN.md:359-361`). B's coalescing, ordering and
  retry logic is the first frontend code in this repo whose correctness is worth a unit test,
  and nothing runs one. The walk in (a)–(e) and the review verdict in (f) are its only proof.
- **Row 25's `default:` arm is unmerged** (§0.22), and board row 25 says it is also undeployed.
  A type the server does not know is dropped in silence. The refetch-on-unknown default path is
  how the backend has shipped ahead of the frontend (`DESIGN.md:143-146`), so B keeps that
  default. The skip list stays **exactly two named types the server does send**
  (`handler.go:74`, `:129`). B never stops refetching on a class of message; it only stops on
  those two.
- **Only port 3000 can walk against the real API** (§0.20). A dev server on any other port fails
  CORS preflight with no useful error (`scripts/emoney:82-83`). Port 3000 must be free.
- **A local `bun run build` bakes `localhost:8080` into the bundle.** That is not a defect to fix
  here (`CLAUDE.md`, *The gates that lie*). Walk through `scripts/emoney dev`, or against the
  deployed site.
- **The checkout is crowded.** `git worktree list` shows 23 worktrees on 2026-09-23, several with
  uncommitted backend edits. Build in a worktree cut from `main` after the merge. Never
  `git stash` or `git checkout --` (`CLAUDE.md`, *Never do this*).

---

## 6. Session protocol

- **Process:** [`docs/AGENT-PRACTICES.md`](../../AGENT-PRACTICES.md). Read it in full (R11); it
  is gitignored, so it exists on the owner's machine only. Stage 5 is one phase per session,
  closed out per Part 7.
- **Before any edit:** Phase 1's precondition, then scope step 0 (§2.4: a resumed effort re-runs
  Stage 4 §0 before it executes a phase).
- **Fresh worktree:** `cd frontend && bun install` is the whole recipe; `backend/` needs nothing
  (`docs/AGENT-PRACTICES.md` Part 6, verified 2026-09-16).
- **Gates**, from `frontend/` and not from the repo root (`CLAUDE.md`, *The rules that get
  broken*):

  ```bash
  cd frontend && bun install && bun run lint
  ```
  ```bash
  cd frontend && bunx tsc --noEmit
  ```
  ```bash
  cd frontend && bun run build
  ```

  **`bun run build` bakes in `localhost:8080` when the API variables are unset. That is not a
  defect, and it is not this phase's to fix** (`CLAUDE.md`, *The gates that lie*). A green build
  proves the code compiles and says nothing about which API the bundle points at.
- **The wire contract does not move.** If the build finds it needs a new event name or a new
  payload field, stop and ask (R12, D2, "No wire contract changes").
- **Commits are Zach's** (`CLAUDE.md`; Part 11). Run `git status --short`, then print one
  `git add` block naming the exact files and one `git commit` block naming the same files with a
  short all-lowercase message. `HANDOFF.md`, `PASSOFF.md` and `docs/AGENT-PRACTICES.md` are
  gitignored and never go in a `git add` block.
- **Close-out** (Part 7):
  - the phase header becomes `BUILT`, with the hash;
  - each `BD-n` goes in §1;
  - `DESIGN.md` gets `As built:` notes under D3 and D4;
  - `RUNTIME-PASS.md` gets its entries;
  - the session ends with the three hand-back blocks, and the last is
    `Next session: <tier>`.
