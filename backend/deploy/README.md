# Deploying the backend to a free always-on VM

Target: a **GCP e2-micro** Always Free VM (`us-central1`/`us-west1`/`us-east1`, amd64)
or an **Oracle Always Free A1** (arm64 — pass `EMONEY_ARCH=arm64`). Both run a
long-lived process with no cold start and nothing to re-deploy on a timer.

**Run exactly one instance.** Room membership lives in process memory
(`websocket/websocketManager.go`, `clients map[string]map[*Client]bool`), so a
second replica silently splits rooms — players stop seeing each other's moves.
A single VM enforces this for free; any autoscaler breaks it.

## One-time server setup

```bash
# 1. packages
sudo apt update && sudo apt install -y ca-certificates curl debian-keyring debian-archive-keyring apt-transport-https
# Caddy (official repo)
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | sudo gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | sudo tee /etc/apt/sources.list.d/caddy-stable.list
sudo apt update && sudo apt install -y caddy

# 2. service account + directories
sudo useradd --system --no-create-home --shell /usr/sbin/nologin emoney
sudo install -d -o root -g root -m 0755 /opt/emoney
sudo install -d -o root -g emoney -m 0750 /etc/emoney
sudo install -d -o caddy -g caddy -m 0755 /var/log/caddy

# 3. secrets (see emoney.env.example)
sudo install -m 0640 -o root -g emoney /dev/null /etc/emoney/emoney.env
sudo nano /etc/emoney/emoney.env

# 4. units
sudo cp deploy/emoney.service /etc/systemd/system/
sudo cp deploy/Caddyfile /etc/caddy/Caddyfile
sudo systemctl daemon-reload
sudo systemctl reload caddy
```

Then deploy the binary for the first time, which also starts it:

```bash
EMONEY_HOST=you@VM_IP ./deploy/deploy.sh
sudo systemctl enable --now emoney     # on the VM, once, so it survives reboots
```

Firewall: open **80 and 443 only** (Caddy). On GCP that's a VPC firewall rule; on
Oracle it's both a security list *and* `iptables`/`firewalld` on the instance,
which trips people up — Oracle images ship with a local firewall already on.

Note that `main.go` calls `r.Run(":" + port)`, which listens on *all* interfaces,
so the firewall is the only thing keeping 8080 off the public internet. To close
that gap properly, make it `r.Run("127.0.0.1:" + port)` — Caddy reaches it either
way, and then a firewall mistake can't expose the API unencrypted.

## Frontend wiring (Vercel)

```
NEXT_PUBLIC_API_URL=https://api.emoney.club
NEXT_PUBLIC_API_URL_NO_PREFIX=api.emoney.club
```

`NEXT_PUBLIC_API_URL_NO_PREFIX` is host-only, no scheme — `wsHelpers.ts` prepends
`wss://` itself. Redeploy the frontend after changing these; `NEXT_PUBLIC_*` is
inlined at build time.

If you ever serve the app from a hostname other than `emoney.club`, add it to
`AllowOrigins` in `main.go` or the websocket handshake gets rejected by CORS.

## Day-to-day

```bash
EMONEY_HOST=you@VM_IP EMONEY_HEALTH_URL=https://api.emoney.club ./deploy/deploy.sh
ssh you@VM_IP 'journalctl -u emoney -f'        # live logs
ssh you@VM_IP 'systemctl status emoney caddy'  # state
```

## Expected, not a problem

`config/db.go` logs `No .env file at: ...` then `Proceeding with system
environment variables` on every start. That's correct on the VM — the values come
from systemd's `EnvironmentFile`, not a `.env`. The line to actually watch for is
`Successfully connected to MongoDB!`.
