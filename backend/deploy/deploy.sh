#!/usr/bin/env bash
#
# Build the backend, ship the binary to the VM, restart the service.
#
#   EMONEY_HOST=zach@203.0.113.10 ./deploy/deploy.sh
#
# Optional env:
#   EMONEY_ARCH        amd64 (GCP e2-micro, default) | arm64 (Oracle A1)
#   EMONEY_HEALTH_URL  e.g. https://api.emoney.club -- verifies the deploy end to end
#   EMONEY_REMOTE_DIR  default /opt/emoney
#   EMONEY_SERVICE     default emoney
#
# Rollback:  ssh $EMONEY_HOST 'sudo mv /opt/emoney/emoney.prev /opt/emoney/emoney && sudo systemctl restart emoney'

set -euo pipefail

HOST="${EMONEY_HOST:-}"
if [[ -z "$HOST" ]]; then
	echo "error: set EMONEY_HOST, e.g. EMONEY_HOST=zach@203.0.113.10 $0" >&2
	exit 1
fi

ARCH="${EMONEY_ARCH:-amd64}"
REMOTE_DIR="${EMONEY_REMOTE_DIR:-/opt/emoney}"
SERVICE="${EMONEY_SERVICE:-emoney}"
HEALTH_URL="${EMONEY_HEALTH_URL:-}"

# repo root (backend/), regardless of where this was invoked from
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

echo "==> build   linux/$ARCH"
CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" \
	go build -trimpath -ldflags="-s -w" -o "$STAGE/emoney" .

echo "==> upload  $(du -h "$STAGE/emoney" | cut -f1) -> $HOST"
scp -q "$STAGE/emoney" "$HOST:/tmp/emoney.new"

echo "==> install + restart"
ssh "$HOST" "set -e
	# keep the last good binary for a one-line rollback
	if [ -f '$REMOTE_DIR/emoney' ]; then
		sudo cp -p '$REMOTE_DIR/emoney' '$REMOTE_DIR/emoney.prev'
	fi
	sudo install -o root -g root -m 0755 /tmp/emoney.new '$REMOTE_DIR/emoney.staged'
	# rename, not copy: you cannot overwrite a running executable in place (ETXTBSY),
	# but rename is atomic and the old inode stays alive until the process exits.
	sudo mv '$REMOTE_DIR/emoney.staged' '$REMOTE_DIR/emoney'
	rm -f /tmp/emoney.new
	sudo systemctl restart '$SERVICE'
	sleep 1
	if ! systemctl is-active --quiet '$SERVICE'; then
		echo '--- service failed to come up ---' >&2
		sudo journalctl -u '$SERVICE' -n 40 --no-pager >&2
		exit 1
	fi
"

if [[ -n "$HEALTH_URL" ]]; then
	echo "==> health  $HEALTH_URL"
	# /v1/rooms/:code/exists hits Mongo, so a 200 proves both the process and the
	# database connection are live -- not just that something is listening.
	for attempt in 1 2 3 4 5; do
		code="$(curl -fsS -o /dev/null -w '%{http_code}' --max-time 10 \
			"$HEALTH_URL/v1/rooms/HEALTH/exists" 2>/dev/null || true)"
		if [[ "$code" == "200" ]]; then
			echo "==> ok      deployed and serving"
			exit 0
		fi
		echo "    attempt $attempt: got '${code:-no response}', retrying..."
		sleep 2
	done
	echo "error: health check never returned 200" >&2
	echo "       ssh $HOST 'journalctl -u $SERVICE -n 50'" >&2
	exit 1
fi

echo "==> ok      deployed (set EMONEY_HEALTH_URL to verify over the network)"
