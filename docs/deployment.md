# DarkVoid - Deployment Guide

## Overview

Two deployment options:

1. **Quick start** — one command, works with defaults
2. **CI/CD** — GitHub Actions auto-deploys on push to `main`

```
[Push to main] -> [Test] -> [Build & Push to ghcr.io] -> [SCP compose] -> [SSH Deploy to VPS]
```

---

## 1. Quick Start (Self-hosting)

Only requirement: Docker installed on the machine.

### One command

```bash
curl -sO https://raw.githubusercontent.com/jarviisha/darkvoid/main/docker-compose.prod.yml && docker compose -f docker-compose.prod.yml up -d
```

This pulls and starts Postgres, Redis, and the app with sensible defaults. API available at `http://localhost:8080`.

### Custom configuration

Create a `.env` file in the same directory to override any default:

```env
# Security — strongly recommended for production
JWT_SECRET=<random-secret-at-least-32-chars>
DB_PASSWORD=<strong-password>
REDIS_PASSWORD=<redis-password>

# Domain
CORS_ALLOWED_ORIGINS=https://your-domain.com
STORAGE_BASE_URL=https://your-domain.com/static

# Root admin user (created on first boot)
ROOT_EMAIL=admin@your-domain.com
ROOT_PASSWORD=<strong-password>
ROOT_USERNAME=admin

# Optional: real email delivery (default: log only)
# MAILER_PROVIDER=smtp
# MAILER_HOST=smtp.example.com
# MAILER_USERNAME=...
# MAILER_PASSWORD=...
```

Then restart:

```bash
docker compose -f docker-compose.prod.yml up -d
```

### Default values

| Variable | Default |
|----------|---------|
| `DB_USER` | `postgres` |
| `DB_PASSWORD` | `postgres` |
| `DB_NAME` | `darkvoid` |
| `JWT_SECRET` | `darkvoid-default-secret-change-me` |
| `REDIS_ENABLED` | `true` |
| `REDIS_PASSWORD` | _(empty)_ |
| `SERVER_PORT` | `8080` |
| `STORAGE_PROVIDER` | `local` |
| `MAILER_PROVIDER` | `nop` |

---

## 2. CI/CD with GitHub Actions

### 2.1. VPS preparation

Install Docker (one time only):

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
# Log out and back in to apply group
```

Optionally create `~/darkvoid/.env` to override defaults (see section above).

### 2.2. GitHub Secrets

Go to **Settings > Secrets and variables > Actions** and add:

| Secret | Description | Example |
|--------|-------------|---------|
| `VPS_HOST` | VPS IP or domain | `123.456.789.0` |
| `VPS_USER` | SSH user | `deploy` |
| `VPS_SSH_KEY` | SSH private key (full content) | Content of `~/.ssh/id_ed25519` |
| `VPS_PORT` | SSH port | `22` |
| `GH_PAT` | Personal Access Token with `read:packages` scope | `ghp_xxx...` |

### 2.3. Create SSH deploy key

```bash
# On local machine
ssh-keygen -t ed25519 -C "github-actions-deploy" -f ~/.ssh/darkvoid_deploy

# Copy public key to VPS
ssh-copy-id -i ~/.ssh/darkvoid_deploy.pub user@vps-ip

# Copy private key content into GitHub Secret VPS_SSH_KEY
cat ~/.ssh/darkvoid_deploy
```

> Copy the **entire** private key including `-----BEGIN` and `-----END` lines.

### 2.4. Create GH_PAT

1. GitHub > Settings > Developer settings > Personal access tokens > Fine-grained tokens
2. Create token with **Read** permission for **Packages** on the `darkvoid` repo
3. Copy token into GitHub Secret `GH_PAT`

### 2.5. How it works

Push to `main` triggers 3 jobs:

| Job | What it does |
|-----|--------------|
| **test** | `go vet ./...` and `go test -race ./...` |
| **build-and-push** | Build Docker image, push to `ghcr.io/jarviisha/darkvoid` with tags `latest` + commit SHA |
| **deploy** | SCP `docker-compose.prod.yml` to VPS, SSH in, pull image, `docker compose up -d`, prune old images |

GitHub Actions copies `docker-compose.prod.yml` to the VPS automatically — no need to clone the repo on the VPS.

---

## 3. Verify

```bash
# Container status
docker compose -f docker-compose.prod.yml ps

# App logs
docker compose -f docker-compose.prod.yml logs -f app

# Health check
curl http://localhost:8080/health
```

---

## 4. Reverse Proxy (Nginx + HTTPS)

```bash
sudo apt install nginx certbot python3-certbot-nginx -y
```

Create `/etc/nginx/sites-available/darkvoid`:

```nginx
server {
    server_name api.your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # WebSocket support
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }

    client_max_body_size 50M;
}
```

Enable and install SSL:

```bash
sudo ln -s /etc/nginx/sites-available/darkvoid /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
sudo certbot --nginx -d api.your-domain.com
```

---

## 5. Maintenance

### Logs

```bash
docker compose -f docker-compose.prod.yml logs -f app
docker compose -f docker-compose.prod.yml logs --tail 100 app
```

### Restart

```bash
docker compose -f docker-compose.prod.yml restart app
```

### Manual update

```bash
cd ~/darkvoid
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
docker image prune -f
```

### Database backup

```bash
# Backup
docker compose -f docker-compose.prod.yml exec postgres \
  pg_dump -U postgres darkvoid > backup_$(date +%Y%m%d).sql

# Restore
docker compose -f docker-compose.prod.yml exec -T postgres \
  psql -U postgres darkvoid < backup_20260403.sql
```

---

## 6. Troubleshooting

| Problem | Check |
|---------|-------|
| Actions fail at test | View logs in Actions tab, fix and push again |
| Actions fail at build | Check `Dockerfile`, ensure local build works |
| Actions fail at deploy | Verify GitHub Secrets (SSH key, host, port, user) |
| Container crash loop | `docker logs darkvoid-app-1` |
| DB connection error | Check `.env` (DB_PASSWORD), `docker compose ps` (postgres healthy?) |
| Port in use | `sudo lsof -i :8080` |
