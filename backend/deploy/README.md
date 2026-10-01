# Deploying the backend to a free always-on VM

This folder holds everything needed to run the Go backend on one small VM: a systemd unit, a Caddy config, an env file template and a deploy script. The API routes are listed in the [backend README](../README.md).

| File | Purpose |
|---|---|
| `emoney.service` | systemd unit. Runs `/opt/emoney/emoney` as the `emoney` user, restarts it on any exit and loads `/etc/emoney/emoney.env`. |
| `Caddyfile` | Caddy config for `api.emoney.club`. Gets the TLS certificate and proxies to `127.0.0.1:8080`. |
| `emoney.env.example` | Template for `/etc/emoney/emoney.env`. Lists the variable names the server reads. |
| `deploy.sh` | Cross-compiles the binary on your machine, copies it to the VM and restarts the service. |

The target is a GCP e2-micro Always Free VM (`us-central1`, `us-west1` or `us-east1`, amd64) or an Oracle Always Free A1 (arm64, so pass `EMONEY_ARCH=arm64`). Both run a long-lived process with no cold start and nothing to re-deploy on a timer.

**Run exactly one instance.** Room membership lives in process memory (`clients map[string]map[*Client]bool` in `websocket/websocketManager.go`), so a second replica silently splits rooms and players stop seeing each other's moves. A single VM enforces this for free. Any autoscaler breaks it.

Nothing deploys the backend automatically. Merging a backend change does not ship it, so run the deploy script (see "Day-to-day").

## One-time server setup

The apt commands assume a Debian or Ubuntu image. Run this block on the VM.

```bash
# 1. packages
sudo apt update && sudo apt install -y ca-certificates curl debian-keyring debian-archive-keyring apt-transport-https
# Caddy (official repo)
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo chmod o+r /usr/share/keyrings/caddy-stable-archive-keyring.gpg
sudo chmod o+r /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy

# 2. service account + directories
sudo useradd --system --no-create-home --shell /usr/sbin/nologin emoney
sudo install -d -o root -g root -m 0755 /opt/emoney
sudo install -d -o root -g emoney -m 0750 /etc/emoney
sudo install -d -o caddy -g caddy -m 0755 /var/log/caddy

# 3. secrets (see emoney.env.example)
sudo install -m 0640 -o root -g emoney /dev/null /etc/emoney/emoney.env
sudo nano /etc/emoney/emoney.env
```

The env file needs these names. `DATABASE_URL` and `DATABASE_NAME` are required: `config/db.go` exits at startup if either is empty. `PORT` defaults to `8080` in `main.go`, and the Caddyfile proxies to that port, so change both together. `GIN_MODE=release` puts Gin in release mode. systemd reads the file as literal `KEY=VALUE` lines with no shell quoting or expansion, so do not wrap the Mongo URI in quotes. The Atlas cluster must also accept connections from the VM's IP address.

Next, copy the unit and the Caddyfile to the VM. From `backend/` on your machine:

```bash
scp deploy/emoney.service deploy/Caddyfile you@VM_IP:/tmp/
```

Then on the VM:

```bash
sudo cp /tmp/emoney.service /etc/systemd/system/
sudo cp /tmp/Caddyfile /etc/caddy/Caddyfile
sudo systemctl daemon-reload
sudo systemctl reload caddy
```

### DNS and TLS

Point an A record for `api.emoney.club` at the VM's static IP. Caddy requests and renews the TLS certificate on its own (an ACME HTTP-01 challenge), which is what makes `wss://` work. If the domain is on Cloudflare, leave the record grey-clouded (DNS only). An orange-clouded record breaks that challenge and adds a proxy hop in front of the websockets.

### Firewall

Open ports 80 and 443 only (Caddy). On GCP that is a VPC firewall rule. On Oracle it is both a security list and `iptables` or `firewalld` on the instance, which trips people up because Oracle images ship with a local firewall already on.

`main.go` calls `r.Run(":" + port)`, which listens on all interfaces, so the firewall is the only thing keeping port 8080 off the public internet. To close that gap, change it to `r.Run("127.0.0.1:" + port)`. Caddy reaches the process either way, and a firewall mistake can no longer expose the API unencrypted.

## First deploy

The service cannot start until the binary exists, so deploy first and enable second. From `backend/` on your machine:

```bash
EMONEY_HOST=you@VM_IP ./deploy/deploy.sh
```

Then on the VM, once, so the service survives reboots:

```bash
sudo systemctl enable --now emoney
```

The SSH user needs sudo without a password prompt, because the script runs `sudo` over a non-interactive `ssh`.

## Day-to-day

Deploy from `backend/`, with the health URL so the script proves the process and Mongo are both up:

```bash
EMONEY_HOST=you@VM_IP EMONEY_HEALTH_URL=https://api.emoney.club ./deploy/deploy.sh
```

From the repo root, `scripts/emoney api` runs the same script with the health URL already set. `scripts/emoney logs` prints the last 60 lines of the service journal, and `scripts/emoney health` checks the API and the site. Both `api` and `logs` need `EMONEY_HOST` set.

```bash
ssh you@VM_IP 'journalctl -u emoney -f'        # live logs
ssh you@VM_IP 'systemctl status emoney caddy'  # state
```

Caddy writes its access log to `/var/log/caddy/emoney.log` (rolled at 10 MiB, five files kept).

The script reads these variables:

- `EMONEY_HOST`: required. The SSH target, for example `you@VM_IP`.
- `EMONEY_ARCH`: `amd64` (default) or `arm64`.
- `EMONEY_HEALTH_URL`: if set, the script polls `/v1/rooms/HEALTH/exists` up to five times and expects a 200. That route queries Mongo, so a 200 means the database connection works.
- `EMONEY_REMOTE_DIR`: default `/opt/emoney`.
- `EMONEY_SERVICE`: default `emoney`.

The build runs on your machine with `CGO_ENABLED=0`, so the VM needs no Go toolchain. `go.mod` pins `go 1.26.0` with `toolchain go1.26.8`. A local Go of version 1.21 or later downloads that toolchain on its own.

Before each install the script copies the current binary to `/opt/emoney/emoney.prev`. It then swaps the new binary in with a rename, because a running executable cannot be overwritten in place. If the service is not active a second after the restart, the script prints the last 40 journal lines and exits with an error.

### Rollback

```bash
ssh you@VM_IP 'sudo mv /opt/emoney/emoney.prev /opt/emoney/emoney && sudo systemctl restart emoney'
```

Every deploy overwrites `emoney.prev`, so this goes back one deploy only.

## Frontend wiring (Vercel)

Set these in the Vercel project:

```
NEXT_PUBLIC_API_URL=https://api.emoney.club
NEXT_PUBLIC_API_URL_NO_PREFIX=api.emoney.club
```

`NEXT_PUBLIC_API_URL` is the REST base URL. `lib/utils/wsHelpers.ts` also reads its scheme to pick the socket scheme: `https` gives `wss` and `http` gives `ws`. `NEXT_PUBLIC_API_URL_NO_PREFIX` is the socket host with no scheme. If it is unset, the host comes from `NEXT_PUBLIC_API_URL`. If `NEXT_PUBLIC_API_URL` itself is unset, both the REST client and the socket fall back to `http://localhost:8080`.

`NEXT_PUBLIC_*` values are inlined at build time, so redeploy the frontend after changing them.

## Allowed origins

The origin list is written twice in the backend: `AllowOrigins` in `main.go` (the REST CORS middleware) and `CheckOrigin` in `websocket/handler.go` (the websocket upgrade). Both currently allow `http://localhost:3000`, `https://emoney.club` and `https://www.emoney.club`. To serve the app from another hostname, add it to both lists and deploy the backend again. A request from an unlisted origin is rejected.

## Expected log lines

On every start, `config/db.go` logs `No .env file at: ...` once for each of its two search paths, then `Warning: no .env file found in search paths` and `Proceeding with system environment variables`. That is correct on the VM, because the values come from systemd's `EnvironmentFile` and not from a `.env` file.

A healthy start then logs `Successfully connected to MongoDB!` followed by `Server starting on port 8080`. If you see `Missing required environment variable`, the env file lacks `DATABASE_URL` or `DATABASE_NAME`. If you see `Failed to ping MongoDB`, check the connection string and the Atlas network access list. A `Failed to create the Offer indexes` line is logged and does not stop the server.

## Unit file behavior

`emoney.service` sets `Restart=always` with a two-second delay and `StartLimitIntervalSec=0`, so the process comes back after a crash or reboot and never enters a permanent failed state, even after an Atlas outage at boot. `main.go` has no graceful-shutdown hook, so SIGTERM ends the process and websocket clients reconnect. The unit enables several hardening options (`ProtectSystem=strict`, `NoNewPrivileges` and others). It keeps `AF_NETLINK` in `RestrictAddressFamilies` because Go's resolver needs it and `mongodb+srv://` connections need DNS SRV lookups. Removing it stops Mongo from connecting.
