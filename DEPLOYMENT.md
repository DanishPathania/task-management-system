# Deployment Guide — TaskMgr Production

> [!NOTE]
> This guide covers deploying to a Linux VPS (Ubuntu 22.04 / Debian 12) using Docker Compose and Caddy as a reverse proxy. No domain name is required initially — you can access the application via your server's IP address.

---

## Prerequisites

Your VPS must have:
- **Docker** (v24+) and **Docker Compose** (v2.20+) installed
- Port **80** and **443** open in your firewall / security group
- At least 1 GB RAM and 10 GB disk
- SSH access as a non-root user with `sudo`

Install Docker on Ubuntu:
```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in for group change to take effect
```

---

## Step 1 — Upload the Code to Your Server

From your local machine:
```bash
# Option A: Git clone (if repo is on GitHub)
ssh user@YOUR_SERVER_IP
git clone https://github.com/YOUR_USERNAME/YOUR_REPO.git TaskMgr
cd TaskMgr

# Option B: rsync from local machine
rsync -az --exclude 'node_modules' --exclude '.next' --exclude 'uploads' \
  ./Task_management/ user@YOUR_SERVER_IP:~/TaskMgr/
```

---

## Step 2 — Create the Production Environment File

```bash
cd ~/TaskMgr
cp .env.production.example .env.production
```

Generate strong secrets first:
```bash
# JWT secret (64 hex chars / 32 bytes)
openssl rand -hex 32

# Redis password
openssl rand -hex 24

# Postgres password
openssl rand -hex 24
```

Edit `.env.production` with your real values:
```bash
nano .env.production
```

**Required changes:**

| Variable | What to set |
|----------|------------|
| `SERVER_HOST` | `http://YOUR_SERVER_IP` (for IP-only HTTP access) or `app.example.com` (for domain + HTTPS) |
| `POSTGRES_USER` | e.g. `taskuser` |
| `POSTGRES_PASSWORD` | Strong random password |
| `POSTGRES_DB` | e.g. `taskdb` |
| `DATABASE_URL` | `postgres://taskuser:STRONG_PASS@postgres:5432/taskdb?sslmode=disable` (must match above) |
| `REDIS_PASSWORD` | Strong random password |
| `JWT_SECRET` | 64-char hex string from `openssl rand -hex 32` |
| `CORS_ALLOWED_ORIGINS` | `http://YOUR_SERVER_IP` (or `https://app.example.com`) |
| `NEXT_PUBLIC_API_URL` | `http://YOUR_SERVER_IP/api/v1` (or `https://app.example.com/api/v1`) |
| `NEXT_PUBLIC_WS_URL` | `ws://YOUR_SERVER_IP/api/v1/ws` (or `wss://app.example.com/api/v1/ws`) |
| `COOKIE_SECURE` | `false` for HTTP, `true` after adding HTTPS |

> [!CAUTION]
> Never commit `.env.production` to source control. It is gitignored.

---

## Step 3 — Validate Compose & Build and Start the Application

All production commands pass `--env-file .env.production` so Docker Compose injects `SERVER_HOST`, database credentials, Redis password, and frontend `NEXT_PUBLIC_*` build arguments consistently without needing shell `export` hacks.

```bash
cd ~/TaskMgr

# Validate the resolved production Compose configuration
docker compose --env-file .env.production -f docker-compose.prod.yml config

# Build all images (this takes several minutes on first run)
docker compose --env-file .env.production -f docker-compose.prod.yml build

# Start all services in the background
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

Watch the startup logs:
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f
```

---

## Step 4 — Verify the Deployment

```bash
# Check all containers are healthy
docker compose --env-file .env.production -f docker-compose.prod.yml ps

# Test the backend health endpoint
curl http://YOUR_SERVER_IP/api/v1/health

# Expected response:
# {"service":"taskmgr-api","status":"healthy","timestamp":"..."}
```

Open your browser and navigate to:
```
http://YOUR_SERVER_IP
```

---

## Step 5 — Adding a Custom Domain (Later)

When you purchase a domain and point it to your server:

1. Update your DNS: add an **A record** for `app.example.com` → `YOUR_SERVER_IP`.

2. Update `.env.production`:
   ```env
   SERVER_HOST=app.example.com
   CORS_ALLOWED_ORIGINS=https://app.example.com
   NEXT_PUBLIC_API_URL=https://app.example.com/api/v1
   NEXT_PUBLIC_WS_URL=wss://app.example.com/api/v1/ws
   COOKIE_SECURE=true
   COOKIE_SAME_SITE=strict
   ```
   (Omitting `http://` in `SERVER_HOST` tells Caddy to auto-provision Let's Encrypt TLS.)

3. Rebuild the frontend (because `NEXT_PUBLIC_*` vars are baked at build time) and recreate services:
   ```bash
   docker compose --env-file .env.production -f docker-compose.prod.yml build frontend
   docker compose --env-file .env.production -f docker-compose.prod.yml up -d
   ```

4. Caddy auto-provisions and renews Let's Encrypt certificates. Ensure ports 80 and 443 are open.

---

## Managing the Application

### View logs
```bash
# All services
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f

# Single service
docker compose --env-file .env.production -f docker-compose.prod.yml logs -f backend
```

### Restart a service
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml restart backend
```

### Stop everything (WITHOUT deleting volumes/data)
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml stop
```

### Pull new code and redeploy
```bash
git pull
docker compose --env-file .env.production -f docker-compose.prod.yml build
docker compose --env-file .env.production -f docker-compose.prod.yml up -d
```

> [!CAUTION]
> **NEVER run `docker compose down -v`** — the `-v` flag deletes all named volumes including your PostgreSQL database and uploaded files.

### View database directly
```bash
docker compose --env-file .env.production -f docker-compose.prod.yml exec postgres psql -U taskuser -d taskdb
```

---

## Firewall Configuration (UFW)

```bash
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
sudo ufw status
```

---

## Troubleshooting

### Backend fails to start — "JWT_SECRET is set to the default insecure value"
Set a real `JWT_SECRET` (at least 32 characters) in `.env.production` and recreate the container.

### Frontend shows API errors
Verify `NEXT_PUBLIC_API_URL` matches where your server is accessible from the browser. This is a **build-time** variable; changing it requires rebuilding frontend with `--env-file .env.production`.

### WebSocket connection fails
Verify `NEXT_PUBLIC_WS_URL` starts with `ws://` (HTTP) or `wss://` (HTTPS) and matches your server address. Also requires a frontend rebuild if changed.

### Caddy fails to start or "no server name" error
Make sure `SERVER_HOST` is set in `.env.production` and you are passing `--env-file .env.production` to `docker compose`.

### Database migrations fail
Check backend logs: `docker compose --env-file .env.production -f docker-compose.prod.yml logs backend`. The backend applies migrations automatically on startup.
