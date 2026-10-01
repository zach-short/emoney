# E-Money frontend

This folder holds the web app for E-Money, a digital companion for playing Monopoly. It keeps every player's money and property in a shared room on each player's phone, so a game needs no paper cash. The app is a pure client. All game state lives in the Go backend in `../backend`, which stores it in MongoDB, and this folder has no Next.js route handlers or server actions.

## Why it exists

Monopoly shows what free-market capitalism does when no central bank or government steps in. While playing it with friends I thought about why fiat currency matters: without the colored bills we would be bargaining with cans of beer, sweaters or chickens. About five hours into the game my friend passed Go and held out his hand for his $200. As the banker I reached for two $100 bills and found the bill holders empty. We had put so much money into the game that we had caused a bank run. That session inspired E-Money, a tool built around liquidity and resource management.

## What it does

A player creates a room by choosing a room name, a room code, a player name and color, and the game rules: starting cash (default 1500), houses (default 32) and hotels (default 12). Other players join with the code, a name and a color. There are no accounts. The browser keeps the player id in `localStorage` under `room_<code>_playerId`, and the My rooms page lists the rooms it finds there.

Inside a room, players can send and request money, pay rent, buy properties from the bank, manage houses and mortgages, and add to or take from Free Parking. A player can make another player an offer of cash, properties or both, and the other player can accept, decline or counter it. The banker can add or remove money and can remove a player. The removed player's properties either return to the bank, stay frozen or go to auction, and players then place bids until the banker closes the lot.

The app has a web app manifest (`app/manifest.ts`) and an `/install` page, so it can be installed on a phone. It registers no service worker. The interface is dark only.

## How it talks to the backend

The app uses two transports to the Go API, both under the `/v1` prefix.

- REST through axios. `lib/utils/api.ts` creates the client and `lib/utils/api.service.ts` holds the typed calls (`roomApi` and `playerApi`).
- One websocket per room at `/v1/ws/room/:code`, opened by `app/room/[code]/page.tsx` with the URL from `lib/utils/wsHelpers.ts`. The page sends `JOIN` when the socket opens and reconnects one second after any close.

REST is the source of truth for state. The socket carries commands from the client (`TRANSFER`, `JOIN`, `PURCHASE_PROPERTY`, `BANKER_TRANSACTION`, `FREE_PARKING`, `MANAGE_PROPERTIES`, `KICK_PLAYER`, `CREATE_OFFER`, `RESPOND_OFFER`, `PLACE_BID` and `CLOSE_AUCTION`) and broadcasts from the server. On almost every broadcast the page shows a toast and refetches from REST, and it refetches the room, the properties and the offers every time the socket opens. A dropped frame therefore does not lose state. The one exception is `BID_PLACED`, which updates the auction panel directly from its payload.

The frontend declares the shape of each message in `types/events.ts`, `types/payloads.ts` and `types/schema.ts`. The Go side builds its payloads as inline maps with bare string event names, and nothing checks that the two sides agree. Change both sides in the same commit, and search the other side for the event name before renaming it.

### Environment variables

Two variables set where the app finds the API. Both are read at build time, because Next.js inlines `NEXT_PUBLIC_` values into the bundle.

| Variable | Meaning | Default when unset |
| --- | --- | --- |
| `NEXT_PUBLIC_API_URL` | Base URL of the API, with scheme and without `/v1`. The socket uses `wss` when this is `https` and `ws` otherwise. | `http://localhost:8080` |
| `NEXT_PUBLIC_API_URL_NO_PREFIX` | Host of the socket, with no scheme. | The host of `NEXT_PUBLIC_API_URL` |

A build made with `NEXT_PUBLIC_API_URL` unset succeeds and points at `localhost:8080`, and nothing in the build output says so. After any build, confirm which API it targets.

The backend also limits which origins may call it. The allowlist in `../backend/main.go` names `http://localhost:3000`, `https://emoney.club` and `https://www.emoney.club`. Run the dev server on port 3000, and expect a new frontend origin to need a backend code change and a manual backend deploy.

## Requirements

- Node.js 20.9 or later, which is the minimum for the installed Next.js 16.3.5.
- bun. The lockfile is `bun.lockb`, and the commands in this file were run with bun 1.2.9.

The app uses React 19.3, TypeScript 5 in strict mode, Tailwind CSS 3.4, and shadcn/ui components on Radix (`components.json`). Fonts are Josefin Sans, Manrope and JetBrains Mono from `next/font/google`.

## Install

```bash
cd frontend
bun install
```

## Run locally

With no environment variables the app targets a backend on `localhost:8080`. Start the backend first. It needs `DATABASE_URL` and `DATABASE_NAME` for a MongoDB database, set in the environment or in `backend/.env`, and it exits at startup without them. See `../backend/README.md` for the backend.

```bash
cd backend
go run .
```

Then start the frontend and open `http://localhost:3000`.

```bash
cd frontend
bun run dev
```

To run the frontend without a local backend, use the repository script from the repository root. It sets both variables to the production API, `https://api.emoney.club`, and starts `next dev` on port 3000. Any room you create this way is stored in the production database.

```bash
./scripts/emoney dev
```

## Checks

The frontend has no test runner and `package.json` has no test script. The checks are lint, the type check and a build.

```bash
bun run lint
bunx tsc --noEmit
bun run build
```

`bun run lint` runs ESLint 9 with the `eslint-config-next` core-web-vitals and TypeScript rules, and `@typescript-eslint/no-explicit-any` is turned off in `eslint.config.mjs`. The build downloads the Google fonts, so it needs network access. After a build, `bun run start` serves the production bundle.

## Deploy

The frontend deploys to Vercel. The project is named `emoney` (see `.vercel/project.json`) and, according to `../scripts/emoney`, it is connected to `github.com/zach-short/emoney` with `frontend` as its Root Directory. A push to `main` builds it. The `ignoreCommand` in `vercel.json` cancels the build when the commit changed nothing under `frontend/`.

Set the two environment variables in the Vercel project settings before the first build, and again after any change, since they are inlined at build time. The production values are `https://api.emoney.club` for `NEXT_PUBLIC_API_URL` and `api.emoney.club` for `NEXT_PUBLIC_API_URL_NO_PREFIX` (see `../backend/deploy/README.md`).

To deploy without a push, run `vercel --prod` from the repository root, or `./scripts/emoney deploy`, which does the same. Do not run it from `frontend/`. Vercel applies the Root Directory to the uploaded folder, so it looks for `frontend/frontend` and the deploy fails.

The backend is a separate deploy. It runs as one process on a VM and is shipped by hand, so merging a backend change does not make it live. See `../backend/deploy/README.md`. After either deploy, `./scripts/emoney health` checks the API, the site and a direct room URL.

## Project layout

- `app/` holds the routes: `/` (home), `/create`, `/join`, `/my-rooms`, `/help`, `/install` and `/room/[code]`, plus `layout.tsx`, `manifest.ts` and `globals.css`.
- `components/` holds the UI, grouped by feature: `room/`, `players/` (including `make-offer/`), `property/` (deed cards and popover), `navbar/`, `containers/`, `loaders/` and `not-found/`. `ui/` holds the shadcn primitives and the hand-written house components.
- `hooks/` holds the React hooks: `use-public-fetch`, `use-stored-player-id`, `use-count-up`, `use-deed-change` and `use-browser-env`.
- `lib/utils/` holds the axios client, the typed API calls, the websocket URL and send helpers, and the money, player and property helpers. `lib/properties.ts` holds the default property list.
- `types/` holds the shared wire shapes: `events.ts`, `payloads.ts` and `schema.ts`.
- `public/` holds the app icons and the property images.

## Where to read next

- `../backend/README.md` for the API and the Go server.
- `../backend/deploy/README.md` for the VM, the reverse proxy and the backend deploy.
- `../scripts/emoney` for the repository commands (`deploy`, `api`, `dev`, `gates`, `health`, `room`, `logs`).
- `../CLAUDE.md` for the working rules of the repository.
- `node_modules/next/dist/docs/` for the Next.js documentation that matches the installed version, available after `bun install`.

The code is under the MIT license (`LICENSE`).
