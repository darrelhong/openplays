#!/usr/bin/env bash
# Deploy script — run from the repo root on your local machine.
# Builds everything locally, rsyncs to VPS, runs migrations, restarts services.
#
# Usage: ./deploy/deploy.sh
#
# Env vars (or set in .env.deploy):
#   VPS_HOST  — e.g. deploy@123.45.67.89
#   VPS_ARCH  — amd64 or arm64 (default: amd64)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Load deploy config if present
if [ -f "${REPO_DIR}/.env.deploy" ]; then
    # shellcheck source=/dev/null
    source "${REPO_DIR}/.env.deploy"
fi

VPS_HOST="${VPS_HOST:?Set VPS_HOST (e.g. deploy@your-vps-ip)}"
VPS_ARCH="${VPS_ARCH:-amd64}"
APP_DIR="/opt/openplays"

echo "==> Building Go binaries (linux/${VPS_ARCH})"
cd "${REPO_DIR}/server"
GOOS=linux GOARCH="${VPS_ARCH}" CGO_ENABLED=0 go build -o bin/api ./cmd/api
GOOS=linux GOARCH="${VPS_ARCH}" CGO_ENABLED=0 go build -o bin/listener ./cmd/listener
echo "    Built: server/bin/api, server/bin/listener"

echo "==> Building SvelteKit"
cd "${REPO_DIR}/web"
pnpm install --frozen-lockfile
pnpm build
echo "    Built: web/build/"

echo "==> Syncing to ${VPS_HOST}"

# Go binaries
rsync -avz --progress \
    "${REPO_DIR}/server/bin/" \
    "${VPS_HOST}:${APP_DIR}/server/bin/"

# Migrations
rsync -avz --delete \
    "${REPO_DIR}/server/db/migrations/" \
    "${VPS_HOST}:${APP_DIR}/server/db/migrations/"

# SvelteKit build output
rsync -avz --delete \
    "${REPO_DIR}/web/build/" \
    "${VPS_HOST}:${APP_DIR}/web/build/"

rsync -avz \
    "${REPO_DIR}/web/package.json" \
    "${VPS_HOST}:${APP_DIR}/web/package.json"

# Deploy config (systemd units, Caddyfile)
rsync -avz \
    "${REPO_DIR}/deploy/openplays-api.service" \
    "${REPO_DIR}/deploy/openplays-listener.service" \
    "${REPO_DIR}/deploy/openplays-web.service" \
    "${REPO_DIR}/deploy/Caddyfile" \
    "${VPS_HOST}:${APP_DIR}/deploy/"

echo "==> Running migrations + restarting services"
ssh "${VPS_HOST}" bash <<'REMOTE'
set -euo pipefail
APP_DIR="/opt/openplays"

# Install systemd units if changed
for svc in openplays-api openplays-listener openplays-web; do
    sudo cp "${APP_DIR}/deploy/${svc}.service" "/etc/systemd/system/${svc}.service"
done
sudo cp "${APP_DIR}/deploy/Caddyfile" /etc/caddy/Caddyfile
sudo systemctl daemon-reload

# Run migrations (backup first if DB exists)
source "${APP_DIR}/server/.env"
[ -f "${DB_URL}" ] && cp "${DB_URL}" "${DB_URL}.bak.$(date +%Y%m%d%H%M%S)"
goose -dir "${APP_DIR}/server/db/migrations" sqlite3 "${DB_URL}" up

# Restart services
sudo systemctl restart openplays-api openplays-listener openplays-web
sudo systemctl reload caddy

# Enable on boot
sudo systemctl enable openplays-api openplays-listener openplays-web caddy

echo "==> Deploy complete"
systemctl --no-pager status openplays-api openplays-listener openplays-web | head -30
REMOTE
