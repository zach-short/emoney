# RUNTIME-PASS — room state sync (option B)

Stage 7 of `docs/AGENT-PRACTICES.md` §2.2. **Zach walks this; it does not block a phase or its
close-out.** What it blocks is a session claiming something was seen working when it was not
(R10). Each entry is three lines: the goal in product terms, where to look, and what the right
answer is.

Entries are numbered `R<phase>.<n>` and appended by the phase that created them. `R1.1`–`R1.6`
are `PLAN.md` §2 Phase 1's done-when (a)–(f). `R1.0` is the build's own first-load check.

---

## Phase 1 — Coalesced, ordered, failure-tolerant room refetch. Entries written 2026-09-23.

**How the phase was walked: partly, presence-only, with no money moved and no document
written.**
- **What was not walked, and why.** (a), (b) and (e) need form submissions and money-moving game
  actions against the production API: transfers, banker adds, and a new player joining. The
  build session's rules require the owner's own go-ahead **in the build session's chat** for
  those. A go-ahead relayed by a coordinating agent ("Zach authorized it") does not count, however
  it is framed. One such relay arrived on 2026-09-23, and the build session declined to act on it
  and asked for Zach's own word instead. (d) also needs the deployed build, which exists only
  after Zach pushes.
- **A retraction first.** The session's first two "smoke loads" went through
  `.claude/launch.json` → `emoney-frontend`, whose relative `cd frontend` resolved against the
  **primary checkout**. `lsof -a -p <pid> -d cwd` showed `/Users/zachshort/Projects/emoney/frontend`.
  So they ran `main`, not Phase 1, and nothing they showed is evidence for this phase. The only
  thing they left behind was a regenerated, gitignored `frontend/next-env.d.ts` in the primary
  checkout.
- **Where the real walk ran.** `next dev --port 3000` from **the build worktree's** `frontend/`
  (cwd verified with the same `lsof`), with `scripts/emoney dev`'s two variables pointing at the
  real API, in the Claude desktop browser pane.
- **The observer.** Seated as TRDCHK's existing player **Bob** (`6aac56ddd541af10f22e8818`, from a
  read-only `GET /players`), by writing `localStorage["room_TRDCHK_playerId"]`, the app's own
  identity key (`playerHelpers.ts:15`). The key was removed afterwards.
- **The second identity, for the rejoin half of (c).** TRDCHK's other player, **Alice**
  (`6aac56c9d541af10f22e87fa`). She ran in a **separate headless Chrome with a throwaway profile**
  in the session scratchpad (`--user-data-dir`, `--remote-debugging-port=9333`), driven over the
  DevTools protocol by a small `bun` script: seat, join, and close the target. No download, and
  nothing in Zach's own browser profile was touched. The instance was quit afterwards.
- **Why presence only is safe here.** `JOIN` and leave write no document: D3's premise, re-checked
  by the review in `handler.go`. `SeatClient` closes no other connection
  (`websocketManager.go:72-77`).
- **How the numbers were read.** Every count below comes from
  `performance.getEntriesByType('resource')` and is exact.

### R1.0 — The build runs, and a first load coalesces. **Verified 2026-09-23 (dev, StrictMode on).**
- **Where.** A direct load of `/room/TRDCHK` as Bob, from the build worktree.
- **Right answer.** No runtime error, and per resource at most one request in flight within a
  live run, with every follow-up starting after the one before settles. Seen, start → end in ms:
  - `/properties`: 208→316 and 209→427 (the StrictMode pair; the first run is disposed), then
    430→534 (the `onopen` resync's follow-up).
  - `/players`: 211→414, then 414→673, then 674→860 (strictly sequential; the resync and the own
    echo fell into different windows, see `PLAN.md` BD-2).
  - `/offers`: 212→315, then 352→454 (the resync, fired at once because the resource was idle),
    then 455→556 (the echo's follow-up).

  The "has joined the game" toast was on screen at 1 s, and "No player found" was not. No console
  errors other than Next's own HMR socket.
- **Re-checked on the final code (with BD-8's timeout), same result.**
  - `/properties`: 523→844 and 523→950 (the StrictMode pair), then 952→1101.
  - `/players`: 526→943, then 944→1140.
  - `/offers`: 526→834, then 836→950, then 951→1101.

  The follow-up count per resource varies with where the resync and the echo land; the rule
  never breaks.

**Walk set-up for (a)–(e).**
- **Where.** `scripts/emoney dev` (port 3000 only, since CORS names it), two browser profiles
  (or a phone and a laptop), in one throwaway room.
- **Fixture.** `TRDCHK` existed on 2026-09-23 (Alice the banker, Bob). Re-check it with
  `curl -s -o /dev/null -w '%{http_code}\n' https://api.emoney.club/v1/rooms/TRDCHK/players`,
  or create a fresh room and delete it after.
- **Observer.** Keep DevTools' Network panel open on the observing client, filtered to
  `/players`, `/properties` and `/offers`.

### R1.1 — (a) A burst of actions costs two room reads, not five. **Not walked.**
- **Where.** Observer throttled to *Slow 3G*; the other client fires five banker transactions or
  transfers in quick succession.
- **Right answer.** At most two `GET /players` on the observer for the burst: the one in flight,
  then exactly one trailing, which starts only after the first settles. At most two
  `GET /offers` likewise. The final balances equal a direct
  `GET https://api.emoney.club/v1/rooms/<code>/players`. If any one GET runs past 10 s, BD-8's
  timeout abandons it without cancelling it, and a third `/players` can appear. That one is
  ignored and is not a coalescing failure (`PLAN.md` §1 BD-8).

### R1.2 — (b) A failed refresh keeps the room on screen, toasts once, retries three times. **Not walked.**
- **Where.** Observer: DevTools request blocking on `*/players`. The other client makes a
  transfer. Then unblock, and make one more action.
- **Right answer, while blocked.**
  - The room stays on screen, with no full-screen "Something went wrong".
  - One toast reads exactly `Couldn't refresh the room. Retrying…`, styled like the red
    rejection toast (top-centre, bold, small).
  - The Network panel shows three more blocked `/players`, then nothing until the next message.
    Each retry waits 1 s, 2 s and then 4 s **after the previous failure**, not after the first
    (`REFETCH_RETRY_DELAYS_MS`, `use-public-fetch.ts:14`). Blocked requests fail at once, so they
    land at about **1 s, 3 s and 7 s after the first failure**. *(Corrected 2026-09-23; this
    line first said "1 s, 2 s and 4 s after the first failure".)*
  - A second message during the outage refetches at once (plus its own three retries) and does
    **not** add a second toast.
- **Right answer, after unblocking.** The next action brings one `GET /players` and a current
  room.
- **Changed from the plan's text (`PLAN.md` §0 addendum A.7).** Reload with the block still on,
  and the **dice loader** shows and stays, exactly as on `main`, **not** `Fallback`. `DataState`
  renders the loader before it reads the error (`data-state.tsx:62`). Whether this should become
  `Fallback` is an open question for Zach, not a defect in this phase.

### R1.3 — (c) A player leaving or rejoining costs no room read. **Walked 2026-09-23, both halves.**
- **Walked 2026-09-23 on the final code, with another player as the other client.** Alice, in the
  headless Chrome, against the observer Bob. Each step was measured from a `performance.now()`
  mark, with a 100 ms poll for the toast text.

  | Step | Toast, and when | `GET /offers` | `GET /players` | `GET /properties` |
  |---|---|---|---|---|
  | Alice joins (a **known** id that is not the observer's) | "Alice has joined the game" +902 ms | exactly one | **zero** | zero |
  | Alice leaves (target closed) | "Alice has left the game" +2958 ms | exactly one | **zero** | zero |
  | **Alice rejoins** (fresh target) | "Alice has joined the game" +2503 ms | exactly one | **zero** | zero |

  **A walk hazard.** Navigating the other client to `about:blank` did **not** produce a
  `PLAYER_LEFT`, even after 29 s. Chrome kept the navigated-away page's socket alive, so the
  browser went on answering pings, and the server only drops a silent peer after `pongWait` =
  60 s (`types.go:97`). Close the tab outright when walking a leave.
- **Earlier, leave only, with the same identity, twice (before BD-8's timeout was added).**
  - **How.** A second pane tab opened `/room/TRDCHK`, so it was seated as Bob as well, since the
    tabs share localStorage. After a `performance.now()` mark on the observer, that tab was
    closed.
  - **Run 1.** Exactly one `GET /offers` (+4562 ms, 200), zero `GET /players`, zero
    `GET /properties`.
  - **Run 2.** The "Bob has left the game" toast rendered 103 ms after the close (polled every
    100 ms). Exactly one `GET /offers` (+55 → +189 ms, 200), zero `GET /players`, zero
    `GET /properties`.
  - **The join on the way in.** Each time the second tab joined, the observer's own-id
    `PLAYER_JOINED` produced one `/players` and one `/offers`. That is the "this client's own
    echo still refetches" pin.
- **The earlier same-identity runs** (the two runs above, before BD-8) could not test the
  other-player rejoin, because pane tabs share localStorage. The headless-Chrome run above is
  what covers it.
- **Where.** The other client closes its tab, then reopens the room.
- **Right answer.**
  - On leave, the observer toasts "… has left the game" and makes **zero** `GET /players` and
    **one** `GET /offers`.
  - On reopen, it toasts "… has joined the game" and again makes **zero** `GET /players` and
    **one** `GET /offers`.
  - The rejoining client itself does refetch: its own `onopen` resync plus its own echo, at most
    two `GET /players`.

### R1.4 — (d) A dropped connection catches up on its own when it comes back. **Not walked.**
- **Where.** **A real phone against the deployed site, after Zach pushes.** `next dev`
  double-mounts the socket effect under StrictMode (`PLAN.md` §0.19). A desktop dev walk with
  DevTools *Offline* approximates it, caveat noted.
- **Steps.** Take the observer offline for about 5 s. While it is offline, have the other client
  make a transfer, and ideally buy a property too. Bring the observer back online.
- **Right answer.** Within about 1 s of the socket reopening, the observer fires `GET /players`,
  `GET /properties` and `GET /offers` (D6), at most two of each with its own `PLAYER_JOINED`
  echo coalesced in. The missed transfer shows in the balances, and a missed purchase is gone
  from Bank's Properties.

### R1.5 — (e) A genuinely new player still appears. **Not walked.**
- **Where.** A third session joins the room through the Join screen.
- **Right answer.** The observer toasts "… has joined the game", fires one `GET /players`, and
  the new player's card appears.

### R1.6 — (f) The concurrency review. **Done 2026-09-23. 1–6 and 8 PREVENTED (2 with a dev-only StrictMode caveat); 7 NOT PREVENTED, cause pre-existing.**
- **Where.** The Phase 1 close-out. It holds the Opus review subagent's verdict on `PLAN.md` §2's
  eight interleavings, verbatim, as relayed to the build session by the coordinating session.
- **Right answer.**
  - Each interleaving is prevented, with a `file:line` trace.
  - #7 fails because `DataState` shows the loader before the error (`data-state.tsx:62`, `:64-65`;
    `page.tsx:510-512`, `:517`). That latch dates to `3389b36` (2025-05-21), not to this diff; see
    `PLAN.md` step 3's correction.
  - The ordering guard (step 2) cannot be forced reliably in a live walk, so this review is its
    only proof (R10).
- **Timeout re-review (BD-8), 2026-09-23.** The same subagent received this one directly, not
  through a relay. T1 (a stuck GET is now bounded) is PREVENTED WITH CAVEAT: the abandoned
  request is not cancelled. T2–T5 are PREVENTED: the late response is ignored, the timer race
  resolves once, dispose leaves no timer, and #3, #4 and #8 are unchanged. Nothing new above low
  confidence.
