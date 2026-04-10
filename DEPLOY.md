# Deployment

## Architecture

```
Cloudflare → Caddy (:443) → SvelteKit (:3000)
                           → /docs, /openapi.json → Go API (:8080) (basic auth)
                           Go Listener (background)
                           SQLite (file on disk)
```

## First-Time VPS Setup

```bash
# 1. Provision server
ssh root@VPS_IP 'bash -s' < deploy/setup.sh

# 2. Edit secrets
nano /opt/openplays/server/.env
nano /opt/openplays/web/.env

# 3. SSH key for deploy user
mkdir -p /home/deploy/.ssh
cp ~/.ssh/authorized_keys /home/deploy/.ssh/
chown -R deploy:deploy /home/deploy/.ssh
chmod 700 /home/deploy/.ssh
chmod 600 /home/deploy/.ssh/authorized_keys

# 4. Copy Telegram session from local machine
scp server/tele_session.db deploy@VPS_IP:/opt/openplays/data/tele_session.db

# 5. (Optional) Copy local DB
scp server/openplays_local.db deploy@VPS_IP:/opt/openplays/data/openplays.db

# 6. Set up Caddy password for /docs
caddy hash-password --plaintext YOUR_PASSWORD
nano /etc/caddy/Caddyfile  # replace REPLACE_WITH_HASHED_PASSWORD
sudo systemctl restart caddy
```

## GitHub Actions Secrets

| Secret | Value |
|--------|-------|
| `VPS_HOST` | VPS IP address |
| `VPS_USER` | `deploy` |
| `VPS_SSH_KEY` | Contents of `~/.ssh/openplays_deploy` (private key) |

| Variable | Value |
|----------|-------|
| `VPS_ARCH` | `amd64` |

## Manual Deploy

```bash
./deploy/deploy.sh
```

Requires `.env.deploy` at repo root:

```
VPS_HOST=deploy@VPS_IP
```

## Debugging

### Service Status

```bash
# All services at a glance
sudo systemctl status openplays-api openplays-listener openplays-web caddy --no-pager

# Individual service
sudo systemctl status openplays-api --no-pager
sudo systemctl status openplays-listener --no-pager
sudo systemctl status openplays-web --no-pager
sudo systemctl status caddy --no-pager
```

### Logs

```bash
# Follow live logs
sudo journalctl -u openplays-api -f
sudo journalctl -u openplays-listener -f
sudo journalctl -u openplays-web -f
sudo journalctl -u caddy -f

# Recent logs (last 50 lines)
sudo journalctl -u openplays-api -n 50 --no-pager
sudo journalctl -u openplays-listener -n 50 --no-pager
sudo journalctl -u openplays-web -n 50 --no-pager
sudo journalctl -u caddy -n 50 --no-pager

# Logs since last boot
sudo journalctl -u openplays-api -b --no-pager

# Logs in a time range
sudo journalctl -u openplays-listener --since "10 min ago" --no-pager
```

### Service Control

```bash
# Stop all
sudo systemctl stop openplays-api openplays-listener openplays-web

# Start all
sudo systemctl start openplays-api openplays-listener openplays-web

# Restart individual
sudo systemctl restart openplays-api

# Reload Caddy config without downtime
sudo systemctl reload caddy

# If Caddy is stuck, restart instead of reload
sudo systemctl restart caddy
```

### Caddy

```bash
# Validate Caddyfile syntax
caddy validate --config /etc/caddy/Caddyfile

# Check which ports Caddy is listening on
sudo ss -tlnp | grep caddy

# Test if Caddy is proxying correctly
curl -s http://localhost:3000  # SvelteKit directly
curl -s http://localhost:8080/api/plays  # Go API directly
```

### Firewall

```bash
sudo ufw status
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

### Database

```bash
# Check DB exists
ls -la /opt/openplays/data/

# Open DB
sqlite3 /opt/openplays/data/openplays.db

# Check tables
sqlite3 /opt/openplays/data/openplays.db ".tables"

# Check migration version
sqlite3 /opt/openplays/data/openplays.db "SELECT * FROM goose_db_version ORDER BY id DESC LIMIT 5;"

# Run migrations manually
source /opt/openplays/server/.env
goose -dir /opt/openplays/server/db/migrations sqlite3 "${DB_URL}" status
goose -dir /opt/openplays/server/db/migrations sqlite3 "${DB_URL}" up

# Rollback last migration
goose -dir /opt/openplays/server/db/migrations sqlite3 "${DB_URL}" down
```

### Restore from Backup

```bash
# List backups
ls -la /opt/openplays/data/openplays.db.bak.*

# Restore
sudo systemctl stop openplays-api openplays-listener
cp /opt/openplays/data/openplays.db.bak.TIMESTAMP /opt/openplays/data/openplays.db
sudo systemctl start openplays-api openplays-listener
```

### Disk & Memory

```bash
# Memory usage
free -h

# Swap usage
swapon --show

# Disk usage
df -h
du -sh /opt/openplays/*
du -sh /opt/openplays/data/*

# Which process is using the most memory
ps aux --sort=-%mem | head -10
```

### Network

```bash
# Check what's listening
sudo ss -tlnp

# Test API from server
curl -s localhost:8080/api/plays | head -c 200

# Test SvelteKit from server
curl -s localhost:3000 | head -c 200

# Test external HTTPS (run from local machine)
curl -s https://openplays.456123789.xyz
```

### Env Files

```bash
# Check server env (redacted)
cat /opt/openplays/server/.env | grep -v '=' | head; cat /opt/openplays/server/.env | sed 's/=.*/=***/'

# Check web env
cat /opt/openplays/web/.env
```
