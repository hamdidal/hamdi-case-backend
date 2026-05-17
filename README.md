# DPP Backend

A production-grade REST API built with Go and Gin, backed by PostgreSQL, and shipped with a fully integrated observability stack, automated CI, and a resilient backup strategy.

---

## Architecture Overview

```
┌─────────────┐     ┌──────────────┐     ┌────────────────────────────┐
│   Client    │────▶│  dpp-backend │────▶│       dpp-postgres         │
└─────────────┘     │  :8080       │     └────────────────────────────┘
                    └──────┬───────┘
                           │ /metrics
               ┌───────────▼───────────┐
               │    dpp-prometheus     │
               │    dpp-node-exporter  │
               └───────────┬───────────┘
                           │
               ┌───────────▼───────────┐
               │     dpp-grafana       │◀──── dpp-loki ◀──── dpp-promtail
               │     :3000             │          ▲
               └───────────────────────┘          │
                                            Docker socket
```

---

## Quick Start

### Prerequisites

- Docker + Docker Compose
- Go 1.25+
- `make`
- Nginx (for host-level reverse proxy)

### 1. Configure environment

```bash
cp .env.example .env
# Edit .env — set DB_PASSWORD, JWT_SECRET, GRAFANA_ADMIN_PASSWORD at minimum
```

### 2. Start the stack

```bash
make up
```

All services start in dependency order with health-checked readiness gates. The backend waits for Postgres to be ready; Grafana waits for both the backend and Prometheus.

### 3. Verify

| Service    | URL                          |
|------------|------------------------------|
| API        | http://localhost:8080/health |
| Grafana    | http://localhost:3000        |
| Prometheus | internal only (no host port) |

---

## Nginx Reverse Proxy

Pre-built server block configs are provided in the `nginx/` directory for both the API and the frontend.

| File | Domain | Upstream |
|---|---|---|
| `nginx/hamdi-case-backend.conf` | `hamdi-case-backend.mindmons.com` | `127.0.0.1:8080` |
| `nginx/hamdi-case-frontend.conf` | `hamdi-case-frontend.mindmons.com` | `127.0.0.1:3001` |

### Install

```bash
sudo cp nginx/*.conf /etc/nginx/sites-available/

sudo ln -s /etc/nginx/sites-available/hamdi-case-backend.conf /etc/nginx/sites-enabled/
sudo ln -s /etc/nginx/sites-available/hamdi-case-frontend.conf /etc/nginx/sites-enabled/

sudo nginx -t
sudo systemctl reload nginx
```

Both configs proxy `X-Real-IP`, `X-Forwarded-For`, `X-Forwarded-Proto`, and `Host` headers, enforce a 10 MB request body limit, and write separate access and error logs under `/var/log/nginx/`. HTTPS termination can be added with Certbot: `sudo certbot --nginx -d hamdi-case-backend.mindmons.com -d hamdi-case-frontend.mindmons.com`.

---

## Makefile Reference

| Command       | Action                                              |
|---------------|-----------------------------------------------------|
| `make up`     | Start all services in detached mode                 |
| `make down`   | Stop and remove all containers                      |
| `make test`   | Run the full test suite with verbose output         |
| `make backup` | Execute a manual database backup                    |
| `make logs`   | Stream live logs from the backend container         |

---

## API

All protected endpoints require a `Bearer` token in the `Authorization` header, obtained from `/api/v1/auth/login`.

### Authentication

| Method | Path                    | Auth     | Description        |
|--------|-------------------------|----------|--------------------|
| POST   | `/api/v1/auth/register` | Public   | Register a new user |
| POST   | `/api/v1/auth/login`    | Public   | Obtain a JWT token  |

### Products

| Method | Path                            | Role            | Description                        |
|--------|---------------------------------|-----------------|------------------------------------|
| GET    | `/api/v1/products`              | admin, auditor  | List all products                  |
| GET    | `/api/v1/products/:id`          | admin, auditor  | Get product with `qr_code_url`     |
| GET    | `/api/v1/products/:id/qrcode`            | admin, auditor  | Generate QR code PNG (256×256)          |
| GET    | `/api/v1/products/:id/pdf`               | admin, auditor  | Export product passport as PDF          |
| GET    | `/api/v1/products/:id/versions`          | admin, auditor  | Version history list                    |
| GET    | `/api/v1/products/:id/versions/:version_number` | admin, auditor | Specific version snapshot          |
| POST   | `/api/v1/products`              | admin           | Create product                     |
| PUT    | `/api/v1/products/:id`          | admin           | Update product                     |
| DELETE | `/api/v1/products/:id`          | admin           | Delete product                     |

The QR code endpoint returns `Content-Type: image/png`. The encoded URL points to the public passport page: `{PUBLIC_BASE_URL}/p/:uuid`.

### Public

| Method | Path       | Auth   | Description                          |
|--------|------------|--------|--------------------------------------|
| GET    | `/p/:uuid` | Public | Public product passport (no auth)    |

### Users

| Method | Path                     | Role  | Description        |
|--------|--------------------------|-------|--------------------|
| GET    | `/api/v1/users`          | admin | List users         |
| PATCH  | `/api/v1/users/:id/role` | admin | Change user role   |
| DELETE | `/api/v1/users/:id`      | admin | Delete user        |

### Audit Logs

| Method | Path                  | Role  | Description                              |
|--------|-----------------------|-------|------------------------------------------|
| GET    | `/api/v1/audit-logs`  | admin | Paginated audit trail for product changes |

### System

| Method | Path       | Auth   | Description                         |
|--------|------------|--------|-------------------------------------|
| GET    | `/health`  | Public | Liveness probe                      |
| GET    | `/metrics` | Public | Prometheus exposition format        |

---

## Audit Log System

Every create, update, and delete operation on products is recorded in the `audit_logs` table. The log captures who made the change, when, and exactly what changed.

### What is recorded

| Field        | Description                                            |
|--------------|--------------------------------------------------------|
| `user_id`    | UUID of the authenticated user who triggered the action |
| `username`   | Username at the time of the action                     |
| `action`     | `create`, `update`, or `delete`                        |
| `entity_type`| Always `product` for product mutations                 |
| `entity_id`  | UUID of the affected product                           |
| `entity_name`| Product name at the time of the action                 |
| `changes`    | JSON diff: `{before, after}` for updates; `{after}` for creates; `{before}` for deletes |
| `created_at` | Timestamp of the event                                 |

### Querying audit logs

```
GET /api/v1/audit-logs
```

| Query param  | Description                        | Example                                |
|--------------|------------------------------------|----------------------------------------|
| `entity_id`  | Filter by product UUID             | `?entity_id=abc-123`                   |
| `user_id`    | Filter by user UUID                | `?user_id=def-456`                     |
| `action`     | Filter by action type              | `?action=delete`                       |
| `limit`      | Page size (default 50, max 200)    | `?limit=20`                            |
| `offset`     | Pagination offset (default 0)      | `?offset=40`                           |

Response envelope:

```json
{
  "total": 120,
  "limit": 50,
  "offset": 0,
  "data": [...]
}
```

---

## Observability

### Structured Logging

Every request produces a single JSON log line on stdout:

```json
{
  "time": "2026-05-15T22:37:28Z",
  "level": "INFO",
  "msg": "request",
  "request_id": "3a244b30-0dfe-42c7-92cf-d127291d25f2",
  "method": "GET",
  "path": "/health",
  "status": 200,
  "duration_ms": 0,
  "client_ip": "172.18.0.1"
}
```

Every request is assigned a `request_id` (propagated via `X-Request-ID` header) that threads through all log entries, making distributed tracing straightforward.

### Log Aggregation — Loki + Grafana

Promtail tails the Docker socket and ships all container logs to Loki. Grafana is pre-provisioned with the Loki datasource.

**To query backend logs:**

1. Open Grafana at `http://localhost:3000`
2. Navigate to **Explore → Loki**
3. Run:

```logql
{container="dpp-backend"}
```

Filter by level:

```logql
{container="dpp-backend", level="ERROR"}
```

Parse and filter by path:

```logql
{container="dpp-backend"} | json | path="/health"
```

### Metrics — Prometheus + Grafana

Prometheus scrapes `/metrics` every 15 seconds. HTTP request counts, latencies, and Go runtime metrics are available out of the box via `go-gin-prometheus`. Alerting rules are provisioned under `prometheus/alerts.yml`.

---

## CI/CD

Two CI pipelines run in parallel depending on the hosting platform.

### GitHub Actions — `.github/workflows/ci.yml`

Triggers on every push and pull request to `main`.

| Step | Command |
|---|---|
| Checkout | — |
| Set up Go | reads version from `go.mod` |
| Verify dependencies | `go mod verify` |
| Vet | `go vet ./...` |
| Test | `go test ./...` |

### GitLab CI — `.gitlab-ci.yml`

Three sequential stages using the `golang:1.22-alpine` image. Go modules are cached between jobs via `cache: paths: [/go/pkg/mod]`.

| Stage | Commands |
|---|---|
| `lint` | `go mod verify` + `go vet ./...` |
| `test` | `go test -v ./...` |
| `build` | `go build ./...` |

Both pipelines are intentionally minimal: no external services, no Docker builds, no deploy steps. They enforce code correctness on every commit with sub-10-second feedback.

---

## Backup Strategy

Database backups are managed by `scripts/db_backup.sh`.

### How it works

1. `pg_dump` streams a compressed (gzip) SQL dump from the running Postgres container
2. The file is written to `backups/` with a `<db>_YYYYMMDD_HHMMSS.sql.gz` naming scheme
3. The seven most recent backups are retained; older files are pruned automatically
4. If `RCLONE_REMOTE_NAME` is set in `.env` and `rclone` is installed, the backup is copied to `<remote>:backups` via Rclone

### Fail-safe design

Cloud sync is **optional and non-blocking**. If Rclone is not installed or `RCLONE_REMOTE_NAME` is empty, the script prints a warning and exits with code `0`. The local dump is always preserved. A missing cloud sync never causes a backup job failure or a cron alert.

### Enabling cloud sync

```bash
# Install rclone: https://rclone.org/install/
# Configure a remote: rclone config

# In .env:
RCLONE_REMOTE_NAME=myremote   # e.g. an S3, GCS, or B2 remote
```

### Running a manual backup

```bash
make backup
```

### Scheduling with cron

```cron
0 2 * * * /path/to/hamdi-case-backend/scripts/db_backup.sh >> /var/log/db_backup.log 2>&1
```

---

## Environment Variables

| Variable               | Required | Default          | Description                              |
|------------------------|----------|------------------|------------------------------------------|
| `PORT`                 | No       | `8080`           | HTTP listen port                         |
| `DB_HOST`              | Yes      | —                | Postgres hostname                        |
| `DB_PORT`              | Yes      | `5432`           | Postgres port                            |
| `DB_USER`              | Yes      | —                | Postgres username                        |
| `DB_PASSWORD`          | Yes      | —                | Postgres password                        |
| `DB_NAME`              | Yes      | —                | Postgres database name                   |
| `JWT_SECRET`           | Yes      | —                | HS256 signing key (min 256-bit)          |
| `JWT_EXPIRY_HOURS`     | No       | `24`             | Token lifetime in hours                  |
| `GRAFANA_ADMIN_USER`   | No       | `admin`          | Grafana admin username                   |
| `GRAFANA_ADMIN_PASSWORD` | Yes    | —                | Grafana admin password                   |
| `GRAFANA_PORT`         | No       | `3000`           | Host port for Grafana                    |
| `BACKUP_CONTAINER`     | No       | `dpp-postgres`   | Docker container name for `pg_dump`      |
| `RCLONE_REMOTE_NAME`   | No       | —                | Rclone remote for off-site backup sync   |
| `PUBLIC_BASE_URL`      | No       | `http://localhost:8080` | Base URL embedded in QR codes (e.g. `http://51.102.69.153:3001`) |
| `HC_INTERVAL`          | No       | `15s`            | Healthcheck probe interval               |
| `HC_TIMEOUT`           | No       | `5s`             | Healthcheck timeout                      |
| `HC_RETRIES`           | No       | `3`              | Healthcheck retry count                  |

---

## Project Structure

```
.
├── cmd/server/          # Application entrypoint and router setup
├── internal/
│   ├── auth/            # JWT authentication handlers and routes
│   ├── auditlog/        # Audit log helper, handler, and routes
│   ├── health/          # Health check handler
│   ├── metrics/         # Metrics endpoint
│   ├── middleware/       # JWT verification middleware
│   ├── models/          # GORM model definitions (Product, User, AuditLog)
│   ├── product/         # Product CRUD, QR code, and public passport handlers
│   └── user/            # User management handlers and routes
├── pkg/database/        # Database connection and migration
├── grafana/             # Grafana provisioning (datasources, alerting)
├── prometheus/          # Prometheus config and alert rules
├── loki/                # Loki configuration
├── promtail/            # Promtail pipeline config
├── nginx/               # Host-level Nginx reverse proxy configs
├── scripts/             # Operational scripts (backup)
├── .github/workflows/   # GitHub Actions CI pipeline
├── .gitlab-ci.yml       # GitLab CI pipeline
├── docker-compose.yml   # Full stack service definitions
├── Makefile             # Developer convenience targets
└── .env.example         # Environment variable template
```
