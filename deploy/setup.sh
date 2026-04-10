#!/usr/bin/env bash
# Initial VPS setup — run once as root on a fresh Ubuntu 24.04 instance (Vultr).
# Usage: ssh root@YOUR_VPS 'bash -s' < deploy/setup.sh
set -euo pipefail

DEPLOY_USER="deploy"
APP_DIR="/opt/openplays"
NODE_MAJOR=25

echo "==> Updating system packages"
apt-get update && apt-get upgrade -y

echo "==> Installing essentials"
apt-get install -y curl sqlite3 rsync

# --- Node.js ---
echo "==> Installing Node.js ${NODE_MAJOR}.x"
curl -fsSL https://deb.nodesource.com/setup_${NODE_MAJOR}.x | bash -
apt-get install -y nodejs

# --- Caddy ---
echo "==> Installing Caddy"
apt-get install -y debian-keyring debian-archive-keyring apt-transport-https
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
apt-get update && apt-get install -y caddy

# --- Goose (migrations) ---
echo "==> Installing goose"
curl -fsSL https://raw.githubusercontent.com/pressly/goose/master/install.sh | sh

# --- Deploy user ---
echo "==> Creating deploy user"
if ! id "$DEPLOY_USER" &>/dev/null; then
    useradd --system --create-home --shell /bin/bash "$DEPLOY_USER"
fi

# Allow deploy user to restart services without password
cat > /etc/sudoers.d/openplays <<'SUDOERS'
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl restart openplays-*
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl reload caddy
deploy ALL=(ALL) NOPASSWD: /usr/bin/systemctl daemon-reload
deploy ALL=(ALL) NOPASSWD: /usr/bin/cp * /etc/systemd/system/*
deploy ALL=(ALL) NOPASSWD: /usr/bin/cp * /etc/caddy/*
SUDOERS

# --- App directory (mirrors repo structure) ---
echo "==> Creating app directory structure"
mkdir -p "${APP_DIR}/server/bin" \
         "${APP_DIR}/server/db/migrations" \
         "${APP_DIR}/data" \
         "${APP_DIR}/web" \
         "${APP_DIR}/deploy"
chown -R "${DEPLOY_USER}:${DEPLOY_USER}" "$APP_DIR"

# --- Env files (create stubs if missing) ---
if [ ! -f "${APP_DIR}/server/.env" ]; then
    cat > "${APP_DIR}/server/.env" <<'ENV'
TELEGRAM_API_ID=
TELEGRAM_API_HASH=
TELEGRAM_USER_PHONE=
TELEGRAM_SESSION_DB=/opt/openplays/data/tele_session.db
TELEGRAM_GROUP_USERNAME=
TELEGRAM_GROUP_TIMEZONE=Asia/Singapore

DB_URL=/opt/openplays/data/openplays.db

LLM_BASE_URL=
LLM_MODEL=
LLM_API_KEY=

ONEMAP_EMAIL=
ONEMAP_PASSWORD=

GOOGLE_PLACES_API_KEY=

API_PORT=8080
ENV
    chown "${DEPLOY_USER}:${DEPLOY_USER}" "${APP_DIR}/server/.env"
    chmod 600 "${APP_DIR}/server/.env"
fi

if [ ! -f "${APP_DIR}/web/.env" ]; then
    cat > "${APP_DIR}/web/.env" <<'ENV'
API_BASE_URL=http://localhost:8080
ORIGIN=https://openplays.456123789.xyz
ENV
    chown "${DEPLOY_USER}:${DEPLOY_USER}" "${APP_DIR}/web/.env"
    chmod 600 "${APP_DIR}/web/.env"
fi

# --- Swap (safety net for 512MB instances) ---
echo "==> Setting up 1GB swap"
if [ ! -f /swapfile ]; then
    fallocate -l 1G /swapfile
    chmod 600 /swapfile
    mkswap /swapfile
    swapon /swapfile
    echo '/swapfile none swap sw 0 0' >> /etc/fstab
    # Low swappiness — only swap under pressure
    echo 'vm.swappiness=10' >> /etc/sysctl.conf
    sysctl vm.swappiness=10
fi

# --- Systemd services ---
echo "==> Installing systemd services"
for svc in openplays-api openplays-listener openplays-web; do
    if [ -f "${APP_DIR}/deploy/${svc}.service" ]; then
        cp "${APP_DIR}/deploy/${svc}.service" "/etc/systemd/system/${svc}.service"
    fi
done
systemctl daemon-reload

# --- Firewall (ufw) ---
echo "==> Configuring firewall"
ufw allow OpenSSH
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

echo ""
echo "==> Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Edit ${APP_DIR}/server/.env with your real secrets"
echo "  2. Edit ${APP_DIR}/web/.env if needed"
echo "  3. Set up SSH key for the deploy user:"
echo "     mkdir -p /home/${DEPLOY_USER}/.ssh"
echo "     cp ~/.ssh/authorized_keys /home/${DEPLOY_USER}/.ssh/"
echo "     chown -R ${DEPLOY_USER}:${DEPLOY_USER} /home/${DEPLOY_USER}/.ssh"
echo "  4. Run the deploy script or GitHub Actions to push the first build"
