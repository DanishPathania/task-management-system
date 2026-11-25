# Backup and Restore Guide — TaskMgr Production

> [!IMPORTANT]
> Run all commands on your production server via SSH. Never delete named Docker volumes.

---

## What Needs to Be Backed Up

| Data | Storage | Criticality |
|------|---------|-------------|
| PostgreSQL database | `postgres_data` Docker volume | 🔴 Critical |
| Uploaded files | `uploads_data` Docker volume | 🟠 High |
| `.env.production` file | Server filesystem | 🔴 Critical |
| `Caddyfile` | Server filesystem | 🟡 Medium |
| `docker-compose.prod.yml` | Server filesystem (or git) | 🟡 Medium |

---

## PostgreSQL Backup

### Manual backup (pg_dump)

```bash
# Create a timestamped backup
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T postgres \
  pg_dump -U taskuser -d taskdb --no-password -Fc \
  > backup_$(date +%Y%m%d_%H%M%S).dump
```

This creates a custom-format dump file (compressed, supports parallel restore).

### Automated daily backup script

Create `/home/user/backup_taskmgr.sh`:
```bash
#!/bin/bash
set -e

BACKUP_DIR="/home/user/backups/db"
COMPOSE_FILE="/home/user/TaskMgr/docker-compose.prod.yml"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
FILENAME="$BACKUP_DIR/taskmgr_$TIMESTAMP.dump"

mkdir -p "$BACKUP_DIR"

docker compose -f "$COMPOSE_FILE" exec -T postgres \
  pg_dump -U taskuser -d taskdb --no-password -Fc \
  > "$FILENAME"

echo "Backup saved: $FILENAME"

# Keep only the last 14 backups
ls -t "$BACKUP_DIR"/*.dump | tail -n +15 | xargs -r rm
```

Make executable and schedule:
```bash
chmod +x /home/user/backup_taskmgr.sh

# Add to crontab — run daily at 3:00 AM
crontab -e
# Add this line:
0 3 * * * /home/user/backup_taskmgr.sh >> /home/user/backups/backup.log 2>&1
```

---

## Restore PostgreSQL from Backup

> [!WARNING]
> Restoring overwrites existing data. The application should be stopped first to prevent concurrent writes.

```bash
# 1. Stop the application (DO NOT use -v)
docker compose --env-file .env.production -f docker-compose.prod.yml stop backend frontend caddy

# 2. Restore from dump file
docker compose --env-file .env.production -f docker-compose.prod.yml exec -T postgres \
  pg_restore -U taskuser -d taskdb --clean --no-owner \
  < /path/to/backup.dump

# 3. Restart
docker compose --env-file .env.production -f docker-compose.prod.yml start backend frontend caddy
```

---

## File Uploads Backup

Uploaded files are stored in the `uploads_data` Docker volume, mounted at `/app/uploads` in the backend container.

### Backup uploads
```bash
docker run --rm \
  -v taskmgr_prod_uploads_data:/uploads:ro \
  -v /home/user/backups/uploads:/backup \
  alpine \
  tar czf /backup/uploads_$(date +%Y%m%d_%H%M%S).tar.gz -C / uploads
```

### Restore uploads
```bash
# Stop the backend first
docker compose -f docker-compose.prod.yml stop backend

docker run --rm \
  -v taskmgr_prod_uploads_data:/uploads \
  -v /home/user/backups/uploads:/backup:ro \
  alpine \
  tar xzf /backup/uploads_TIMESTAMP.tar.gz -C /

docker compose -f docker-compose.prod.yml start backend
```

---

## Backing Up the Environment File

Your `.env.production` file contains all secrets and must be stored securely outside the server:

```bash
# Copy to your local machine
scp user@YOUR_SERVER_IP:~/TaskMgr/.env.production ~/secure_backups/taskmgr.env.production

# Or encrypt and store
gpg --symmetric --cipher-algo AES256 ~/TaskMgr/.env.production
```

> [!CAUTION]
> Do NOT commit `.env.production` to any git repository, even a private one. Store it in a password manager (Bitwarden, 1Password) or encrypted secret storage.

---

## Disaster Recovery — Full Server Restore

If your server is completely lost:

1. Provision a new VPS with Docker installed.
2. Clone the application code.
3. Restore `.env.production` from your secure backup.
4. Start the services: `docker compose -f docker-compose.prod.yml up -d`
5. Wait for the database to be healthy: `docker compose -f docker-compose.prod.yml ps`
6. Restore the PostgreSQL dump:
   ```bash
   docker compose -f docker-compose.prod.yml exec -T postgres \
     pg_restore -U taskuser -d taskdb --clean --no-owner \
     < /path/to/backup.dump
   ```
7. Restore uploaded files from your uploads backup.
8. Verify at `http://NEW_SERVER_IP`.

**Expected recovery time:** ~15-30 minutes depending on database size and network speed.

---

## Volume Naming Reference

The `docker-compose.prod.yml` creates these named volumes:

| Volume Name | Contents |
|-------------|---------|
| `taskmgr_prod_postgres_data` | PostgreSQL database files |
| `taskmgr_prod_redis_data` | Redis AOF persistence |
| `taskmgr_prod_uploads_data` | User-uploaded file attachments |
| `taskmgr_prod_caddy_data` | Caddy TLS certificates and ACME state |
| `taskmgr_prod_caddy_config` | Caddy runtime configuration |

> [!CAUTION]
> **Never run `docker compose down -v`** — it deletes ALL named volumes, including your PostgreSQL database and uploaded files. Use `docker compose stop` or `docker compose down` (without `-v`) instead.
