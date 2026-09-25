# emoney

> **There is no per-language code standard in this repo yet** — `docs/` holds only
> `AGENT-PRACTICES.md`. Until one is written, the configs are the rule: `frontend/eslint.config.mjs`,
> `frontend/tsconfig.json`, and `gofmt` for Go. Follow them, and follow the house patterns in the
> file you are editing.

> **Before scoping, planning or building a feature, read `docs/AGENT-PRACTICES.md`.** It is the
> process standard, adapted to this repo on 2026-09-16. Do not ask how the flow works; it is
> written down. `HANDOFF.md` is what is true, `PASSOFF.md` is what is next — read `HANDOFF.md`
> first, every session.

## The rules that get broken

- **Never run `git commit`, `git push`, `git add -A` or `git add .`.** Several sessions run in
  this one checkout and only Zach knows which uncommitted file belongs to which. When work is
  ready: run `git status --short`, then print exactly two copyable `bash` blocks — `git add`
  naming the exact files this session touched, then `git commit` naming those same files with a
  short all-lowercase message. Never add a `Co-Authored-By:` trailer or a "Generated with" line,
  on a commit or a pull request.
- **Run the frontend gates from `frontend/`, not the repo root.** There is no root
  `package.json` (`ls` of the repo root, 2026-09-16), so `bun run lint` at the root fails with a
  missing-script error that reads like a broken config.
- **Read `gofmt`'s output, not its exit code.** `gofmt -l .` exits 0 whether or not it lists
  files. As of 2026-09-16 it lists `backend/models/roomModel.go`. The gate is
  `test -z "$(gofmt -l .)"`.
- **A green local `bun run build` can still be wrong, and it no longer says so.** See *Commands*
  below — it bakes `localhost:8080` into the bundle when the API env vars are unset, which is the
  normal local state. There used to be a `console.log` in the build output that named the chosen
  URL; it is gone, so nothing in a green build distinguishes a correct bundle from a localhost
  one.
- **The websocket contract is typed on the frontend only.** Go sends
  `Message{Type string, Payload interface{}}` (`backend/websocket/types.go:12-15`) and builds every
  payload as an inline `map[string]interface{}` at the send site; the event names are bare string
  literals in `backend/websocket/handler.go` and `backend/websocket/websocketManager.go`. The
  frontend declares the real shapes in `frontend/types/events.ts` and `frontend/types/payloads.ts`.
  Nothing checks the two agree, in either direction. Change both sides in the same commit, and
  grep the other side for the literal before renaming any event.
- **No drive-by fixes.** Fix what the task is. Note anything else you find and raise it; do not
  fold it into an unrelated change, where it hides in the diff and in its review.
- **`HANDOFF.md`, `PASSOFF.md` and `docs/AGENT-PRACTICES.md` are gitignored on purpose**
  (`~/.config/git/ignore:4-6`, dated 2026-09-14). Never try to commit them, never list them in a
  `git add` block, and never assume a clone has them.

## Stack

Verified 2026-09-16 from `frontend/package.json`, `backend/go.mod` and the tool versions on this
machine.

- **frontend/** — Next.js 16.3.5 (App Router, **Turbopack is the default builder**), React
  19.3.0, TypeScript 5 with `"strict": true` (`frontend/tsconfig.json:12` — turned on 2026-09-16;
  it was `false` before that), Tailwind 3.4, shadcn/ui on Radix, axios, sonner, vaul,
  next-themes. Package manager is **bun 1.2.9**
  (`frontend/bun.lockb`) — not npm, not pnpm.
- **backend/** — Go (declared 1.23.2 in `backend/go.mod:3`; the installed toolchain is 1.24.1),
  Gin 1.10.0, gorilla/websocket 1.5.3, mongo-driver 1.17.1, godotenv.
- **Data** — MongoDB Atlas. The backend requires `DATABASE_URL` and `DATABASE_NAME` and calls
  `log.Fatal` without them (`backend/config/db.go:62-85`).
- **No CI** (no `.github/`, 2026-09-16). **Go has tests as of 2026-09-16** (HANDOFF 4-5) — two
  files, 35 cases, in `backend/websocket/` and `backend/manager/`; **the claim that there were
  none anywhere, which stood here until 2026-09-16, is wrong.** The frontend still has no runner.
  See *Commands*. **No migrations** (grepped 2026-09-16). **No auth** — see *Architecture*.

## Architecture

- **Two deployables in one repo.** `frontend/` → Vercel (auto-deploys on push to `main`);
  `backend/` → one GCP e2-micro VM behind Caddy, deployed **by hand**. Merging is not shipping
  for the backend.
- **Exactly one backend instance, ever.** Room membership lives in process memory —
  `clients map[string]map[*Client]bool` at `backend/websocket/websocketManager.go:23`. A second
  replica silently splits rooms: players stop seeing each other's moves, and nothing errors.
- **The Next app is a pure client of the Go API.** There are no Next route handlers and no
  server actions touching data (`find frontend/app -name route.ts` returns nothing, 2026-09-16).
  Two transports: REST through axios (`frontend/lib/utils/api.ts`) and one websocket per room at
  `/v1/ws/room/:code` (`frontend/lib/utils/wsHelpers.ts:3-8`). Do not add a Next-side data layer
  without asking — it would be a second way to reach Mongo.
- **Every backend route is unauthenticated.** `backend/routes/routes.go` has no middleware, and
  every auth line in `frontend/lib/utils/api.ts` is commented out (lines 31-41, 72-77). Anyone
  with a room code can move money or `DELETE` the room. This is the current design, not an
  oversight to fix in passing.
- **CORS is a hardcoded allowlist** at `backend/main.go:22-26`:
  `http://localhost:3000`, `https://emoney.club`, `https://www.emoney.club`. A new frontend
  origin needs a backend code change *and* a manual redeploy.

## Directory map

| Path | Belongs here | Does not |
|---|---|---|
| `frontend/app/` | Routed pages only — one `page.tsx` per route, plus `layout.tsx` and `manifest.ts` | Components, data helpers, types, API route handlers |
| `frontend/components/` | Reusable UI, grouped by feature (`room/`, `players/`, `property/`, `containers/`, `loaders/`, `navbar/`) | `ui/` is generated shadcn primitives — regenerate, do not hand-edit |
| `frontend/lib/utils/` | The axios client, the websocket URL/send helpers, player and property helpers | React hooks (those go in `hooks/`), component markup |
| `frontend/types/` | The shared wire shapes: `events.ts`, `payloads.ts`, `schema.ts` | Component prop types local to one file |
| `backend/controllers/` | Gin handlers — request in, response out | Mongo queries expressed inline; put reusable reads in `services/` or `projections/` |
| `backend/models/` | Mongo document structs and their bson tags | Business logic |
| `backend/websocket/` | The hub (`websocketManager.go`), the upgrade handler, and the event structs | Anything that assumes more than one process |
| `backend/deploy/` | systemd unit, Caddyfile, `deploy.sh`, env example | Secrets — the real env lives at `/etc/emoney/emoney.env` on the VM |

> **Exception to the `components/ui/` rule, ratified 2026-09-22** (UI-facelift GATE 1, decision
> `D8` in `DESIGN.md`, archived 2026-09-24 to `~/Projects/archive/emoney/ui-facelift/DESIGN.md`
> after the effort closed). The rule narrows to: **regenerate the
> primitives; the theming layer over them is ours.** Token and `className` changes to
> `drawer.tsx`, `dialog.tsx`, `sonner.tsx` and `button.tsx` are sanctioned — they are what makes
> the theme reachable, and wrapping each primitive to avoid the rule would add a file per
> component to work around a rule whose purpose is already served. Structural edits to generated
> markup are still forbidden. Note also that only 7 of the 17 files in `frontend/components/ui/`
> are generated shadcn (verified 2026-09-22); the rest — `button-custom.tsx`, `link.tsx`,
> `cusotm-link.tsx`, `reason-select.tsx`, `toasts.tsx`, `fonts.ts`, `helper-funcs.ts`,
> `install-app-button.tsx`, `return-to-menu.tsx`, `loader.tsx` — are hand-written house files the
> rule never meant to cover.

## Commands

Package manager: **bun**, in `frontend/` only. Go needs no install step.

Dev servers:

```bash
cd frontend && bun run dev
```

```bash
cd backend && go run .
```

**The backend does not run on this machine.** There is no `backend/.env` (checked 2026-09-16)
and `backend/config/db.go:83-85` calls `log.Fatal` when `DATABASE_URL` or `DATABASE_NAME` is
missing, so `go run .` exits immediately. Running it locally means creating `backend/.env` with
an Atlas URI, and that Atlas cluster allowlists the VM's IP.

The gates. Every one below was run in this repo on 2026-09-16 and its result recorded in
`HANDOFF.md`:

```bash
cd frontend && bun install && bun run lint
```

```bash
cd frontend && bunx tsc --noEmit
```

```bash
cd frontend && bun run build
```

```bash
cd backend && go build ./... && go vet ./...
```

```bash
cd backend && test -z "$(gofmt -l .)" && echo "gofmt clean"
```

```bash
cd backend && go test ./...
```

### The gates that lie

- **`bun run build` is green while producing the wrong artifact, and there is no tell —
  CORRECTED 2026-09-16, HANDOFF 9.** `frontend/lib/utils/api.ts:3` and
  `frontend/lib/utils/wsHelpers.ts:6` fall back to `localhost:8080` when `NEXT_PUBLIC_API_URL` /
  `NEXT_PUBLIC_API_URL_NO_PREFIX` are unset. Those live only on Vercel; `frontend/.env.local`
  holds a `VERCEL_OIDC_TOKEN` and nothing else. The values are inlined at build time, so a local
  build bakes in localhost. **The claim that stood here until 2026-09-16 — that the tell is a
  `console.log` at `api.ts:4` printing the chosen URL — is wrong: that line no longer exists.**
  It was removed in `bec2350`; `api.ts` is now ten lines with no logging, and a captured build
  log of the merged tree contains no `apiUrl` line and no `localhost:8080` line at all (grepped
  2026-09-16). **The hazard is unchanged and the signal is gone.** A local build proves the code
  compiles; nothing in its output tells you which API the bundle points at. To check that, read
  `api.ts:3` against the environment you built in, or check the deployed site rather than the
  build — `scripts/emoney dev` passes both variables explicitly and is the way to run locally
  against the real API.
- **`go test ./...` runs something now, but over two packages out of ten — CORRECTED
  2026-09-16, HANDOFF 4.** The claim that stood here until then: there were no `_test.go` files
  at all, so the command printed `[no test files]` for all ten packages and succeeded having run
  nothing. As of HANDOFF 4-5 it reports `ok` for `manager` and `websocket` (35 cases) and
  `[no test files]` for the other eight — `controllers`, where the money is actually written, is
  one of the eight. **What the 35 cover is rejection only**: the branches of
  `handleBankTransaction`, `handleManageProperties`, `handleTransfer` and
  `ExtractPropertyDetails` that return an error *before* any call into `controllers/` or
  `manager/`. **No test in this repo asserts that a valid transaction moves the right money**,
  because that needs Mongo and there is none on this machine. A green `go test` says bad input
  is refused; it says nothing about good input. There is still no frontend test runner at all
  (no test script and no runner dependency in `frontend/package.json`, 2026-09-16).
- **`config.DB` is a nil `*mongo.Database` in a test binary**, so a test that reaches it panics
  with a nil-pointer dereference rather than failing with an error — the panic site is
  `controllers/playerControllers.go:154`, verified 2026-09-16. This is the boundary on what any
  test here can reach: **if a handler calls into Mongo before the branch you want to pin, that
  branch is untestable** without moving the call or extracting a seam. That is why HANDOFF 5
  hoisted the `transactionType` switch above the `controllers.GetPlayer` call in
  `handleBankTransaction`. Two tests in `websocketManager_test.go` deliberately *assert* this
  panic, as the only available signal that a value was accepted rather than rejected; if a seam
  is ever added in front of `GetPlayer` they will start failing, and the comment above
  `bankTransactionOutcome` says what to change.
- **`gofmt -l .` exits 0 while listing unformatted files.** Use the `test -z` form above.
- **`go build ./...` and `go vet ./...` pass over dead packages.** `backend/middleware/rateLimit.go`
  defines `RateLimitMiddleware` and nothing imports it — the only `middleware` hit in the Go tree
  is its own `package` line (grepped 2026-09-16). Go does not flag an unused package the way it
  flags an unused import.
- **A green `bunx tsc --noEmit` still permits explicit `any`.** `strict` was turned on
  2026-09-16, so implicit `any` and unchecked null no longer pass — **the claim that they do,
  which stood here until 2026-09-16, is wrong.** What remains permitted: `"skipLibCheck": true`
  (`frontend/tsconfig.json:9`) and `@typescript-eslint/no-explicit-any` turned off
  (`frontend/eslint.config.mjs:12`), so an explicit `any` is still invisible to both gates.
- **`bun run lint` genuinely lints** — verified 2026-09-16 by counting `eslint . -f json`
  output: 73 files, 0 errors, 0 warnings. A silent pass here is a real pass.

### Fresh checkout

`bun install` in `frontend/` is the whole recipe; `backend/` needs nothing. Verified 2026-09-16
by extracting `git archive HEAD` into an empty directory — `tsc --noEmit` and `bun run build`
both passed after `bun install` alone, **without** the gitignored `frontend/next-env.d.ts`,
which `frontend/tsconfig.json` names under `include` but does not actually need.

## Never do this

- Never run a second backend instance, or put the backend behind an autoscaler — it splits rooms
  silently (`backend/websocket/websocketManager.go:23`).
- Never trust a local `bun run build` as evidence the deployed frontend is right. There is no
  longer an `apiUrl` line in the build output to check — see *The gates that lie*; verify against
  the deployed site, or run locally through `scripts/emoney dev`, which passes both API variables.
- Never reintroduce `next-pwa` or `@serwist/next`. Both register a webpack config, which fails
  `next build` now that Turbopack is the default. Installability rests on `frontend/app/manifest.ts`
  alone; Chrome no longer requires a service worker.
- Never move ESLint off 9.x. 9.17 is too old for `eslint/config`; 10.x breaks
  `eslint-plugin-react` through `eslint-config-next`.
- Never run `vercel deploy --prod` from `frontend/`. Vercel applies the project's Root Directory
  (`frontend`) to whatever is uploaded, so it would look for `frontend/frontend`. Deploy from the
  repo root.
- Never assume a merged backend change is live. Ship it with
  `EMONEY_HOST=emoney ./backend/deploy/deploy.sh` and set `EMONEY_HEALTH_URL=https://api.emoney.club`
  so the script proves the process *and* Mongo are up.
- Never `git checkout --` or `git stash` to undo an experiment — both reach files belonging to
  other sessions. Copy the file aside and restore it with `cp`.
