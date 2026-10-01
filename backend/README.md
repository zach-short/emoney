# E-Money backend

This is the server for E-Money, a digital Monopoly companion. It stores each room's players, balances, property deeds, event history, trade offers and auctions in MongoDB, and it pushes every change to the players in a room over WebSockets. It is written in Go, with [Gin](https://github.com/gin-gonic/gin) for HTTP routing and [gorilla/websocket](https://github.com/gorilla/websocket) for the real-time connection. The Next.js app in `../frontend` is its only client.

Room membership (which sockets belong to which room) lives in process memory. Run exactly one instance. A second replica would split rooms and nothing would report an error.

## Requirements

- Go 1.26. `go.mod` declares `go 1.26.0` and `toolchain go1.26.8`, and a Go installation from 1.21 onward downloads that toolchain on its own.
- A MongoDB deployment. The code uses multi-document transactions, which need a replica set or a sharded cluster. The project runs on MongoDB Atlas.

## Configuration

The server reads its settings from environment variables. At startup it loads a `.env` file from `backend/` or from the working directory when one exists, and otherwise it uses the process environment. `.env` is gitignored.

| Variable | Required | Meaning |
|---|---|---|
| `DATABASE_URL` | yes | MongoDB connection URI |
| `DATABASE_NAME` | yes | Name of the database |
| `PORT` | no | Listen port, default `8080` |
| `GIN_MODE` | no | Set to `release` to run Gin in release mode |

If either required variable is missing, the process logs which one and exits. `deploy/emoney.env.example` shows the file format used on the production VM.

## Run

```bash
cd backend
go run .
```

On start the server connects to MongoDB and exits if the ping fails. It then creates two indexes on the `Offer` collection (an error there is logged and does not stop startup) and listens on `PORT`.

Two lists of allowed origins are hard-coded: `http://localhost:3000`, `https://emoney.club` and `https://www.emoney.club`. One is the CORS list in `main.go`. The other is the WebSocket `Origin` check in `websocket/handler.go`. A new frontend origin needs an edit in both places.

## Test

```bash
go build ./... && go vet ./...
test -z "$(gofmt -l .)" && echo "gofmt clean"
go test ./...
```

The `controllers`, `manager`, `models` and `websocket` packages have tests, and the other packages report `[no test files]`. No test needs a database. They cover the rules that refuse bad input before any Mongo call, the BSON and JSON shape of the models, and the hub's fan-out, write deadlines and keepalive. Because nothing runs against Mongo, no test asserts that a valid transaction moves the right money.

The `gofmt -l` command exits 0 even when it lists unformatted files, so the gate is the empty-output check in the second line.

## HTTP API

All routes start with `/v1`, and `:code` is the room code. No route has authentication or middleware. Anyone who holds a room code can call every route below and open the WebSocket for that room.

- `POST /v1/rooms` creates a room, its banker player and the default properties from `config/properties.go`. The JSON body needs `playerName`, `roomName`, `roomCode` and `playerColor`, and accepts `startingCash`, `houses` and `hotels`. The reply holds `roomId`, `roomCode` and `playerId`.
- `DELETE /v1/rooms/:code` deletes the `Room` document. It removes nothing else.
- `GET /v1/rooms/:code/exists` replies `{"exists": true}` or `{"exists": false}`. It reads Mongo, so the deploy script uses it as its health check.
- `GET /v1/rooms/:code/players` returns the room, its players with the properties each one owns, and the newest 100 event history entries. The optional `playerId` query parameter adds an `existingPlayer` field after checking that the player belongs to the room.
- `GET /v1/rooms/:code/properties` returns the properties that have no owner as `availableProperties`, with the `roomId`.
- `GET /v1/rooms/:code/offers?playerId=<id>` returns the pending trade offers that the player sent or received, newest first. The `playerId` parameter is required.
- `POST /v1/rooms/:code/players` joins the room as a new player. The JSON body needs `roomCode`, `playerName` and `playerColor`, and the handler reads the room from the body, not from the path. It replies 409 when an active player already uses the name or the color.
- `GET /v1/rooms/:code/players/:playerId` returns one player and that player's properties. It replies 404 for a player who was removed from the game.

Three more routes are registered under `/v1/rooms/:code/players/:playerId/properties/:propertyId`: `POST`, `DELETE`, and `POST .../mortgage`. Their handlers (`AddProperty`, `RemoveProperty` and `MortgageProperty` in `controllers/propertyControllers.go`) are empty and do nothing. Purchases, houses, mortgages and sales all go through the WebSocket.

## WebSocket protocol

Connect to `/v1/ws/room/:code` from an allowed origin. Every frame in both directions is JSON with a `type` and a `payload`. A new connection belongs to no player until the client sends `JOIN` with the id that room creation or joining returned:

```json
{ "type": "JOIN", "payload": { "playerId": "<player id>" } }
```

The server refuses a `JOIN` for a player who was removed from the game and sends nothing back. It pings every 54 seconds and drops a connection that has been silent for 60 seconds. When a connection that joined a player closes, for any reason, the room receives `PLAYER_LEFT`. Browsers answer pings on their own.

The server accepts these message types: `JOIN`, `TRANSFER`, `FREE_PARKING`, `PURCHASE_PROPERTY`, `BANKER_TRANSACTION`, `MANAGE_PROPERTIES`, `KICK_PLAYER`, `PLACE_BID`, `CLOSE_AUCTION`, `CREATE_OFFER` and `RESPOND_OFFER`. `MANAGE_PROPERTIES` takes a `managementType` of `HOUSES`, `MORTGAGE`, `UNMORTGAGE` or `SELL`. `KICK_PLAYER` takes a `disposition` of `BANK`, `FREEZE` or `AUCTION` for the removed player's properties. An unknown type, or an action the server rejects, gets an `ERROR` frame on the sender's connection only, with a message string as the payload.

An accepted `TRANSFER`, `FREE_PARKING`, `PURCHASE_PROPERTY`, `BANKER_TRANSACTION` or `MANAGE_PROPERTIES` is broadcast to the whole room under the same type name. The room also receives `PLAYER_JOINED`, `PLAYER_LEFT`, `PLAYER_KICKED`, `AUCTION_STARTED`, `BID_PLACED`, `AUCTION_LOT_CLOSED` and `OFFER_ACCEPTED`. Three trade events go only to the players involved: `OFFER_RECEIVED`, `OFFER_SENT` and `OFFER_RESOLVED`.

The Go side builds every payload inline as a `map[string]interface{}` and declares no shared type for it. The frontend declares the shapes in `../frontend/types/events.ts` and `../frontend/types/payloads.ts`, and nothing checks that the two sides agree. Change an event name or a payload field on both sides in the same commit.

## Code layout

- `main.go` sets up Gin and CORS, connects to Mongo, registers the routes and starts the server.
- `routes/` maps each URL to a handler.
- `controllers/` holds the Gin handlers and the transactional money writes (`PurchaseProperty`, `PlayerTransfer`).
- `websocket/` holds the connection handler, the room hub (`RoomManager`), the keepalive and write-deadline code, and the handlers for auctions and trade offers.
- `manager/` holds the balance and property helpers that the WebSocket handlers call.
- `models/` holds the Mongo document structs and their BSON and JSON tags.
- `config/` holds the database connection and the default properties.
- `deploy/` holds the systemd unit, the Caddyfile, `deploy.sh` and the environment example.

The `middleware/` package (a rate limiter), the `services/` package and the `projections/` package are not imported anywhere, so the running server does not use them.

The data lives in five collections: `Room`, `Player`, `Property`, `EventHistory` and `Offer`. A live auction is stored on its `Room` document.

## Deploy

The backend runs on one VM behind Caddy, which terminates TLS for `api.emoney.club` and proxies to the Go process on `127.0.0.1:8080`. Nothing deploys it automatically, so merging a change does not ship it. From `backend/`, run:

```bash
EMONEY_HOST=you@VM_IP EMONEY_HEALTH_URL=https://api.emoney.club ./deploy/deploy.sh
```

The script builds a linux/amd64 binary, copies it to the VM, restarts the `emoney` systemd service and then calls the `exists` route to confirm that the process and the database are both up. The one-time server setup, the rollback command and the firewall notes are in [`deploy/README.md`](deploy/README.md).
