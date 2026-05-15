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

| Method | Path                     | Auth     |
|--------|--------------------------|----------|
| GET    | `/api/v1/products`       | Required |
| POST   | `/api/v1/products`       | Required |
| GET    | `/api/v1/products/:id`   | Required |
| PUT    | `/api/v1/products/:id`   | Required |
| DELETE | `/api/v1/products/:id`   | Required |

### Users

| Method | Path                  | Auth     |
|--------|-----------------------|----------|
| GET    | `/api/v1/users`       | Required |
| GET    | `/api/v1/users/:id`   | Required |
| PUT    | `/api/v1/users/:id`   | Required |
| DELETE | `/api/v1/users/:id`   | Required |

### System

| Method | Path          | Auth   | Description                  |
|--------|---------------|--------|------------------------------|
| GET    | `/health`     | Public | Liveness probe               |
| GET    | `/metrics`    | Public | Prometheus exposition format |

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

The GitHub Actions workflow at `.github/workflows/ci.yml` runs on every push and pull request targeting `main`.

**Pipeline steps:**

1. **Checkout** — fetch the full commit history
2. **Set up Go** — install the version declared in `go.mod`, restore the module cache
3. **Verify dependencies** — `go mod verify` confirms all modules match their checksums
4. **Vet** — `go vet ./...` catches suspicious constructs before they reach review
5. **Test** — `go test ./...` executes all tests, including the `/health` integration test

The pipeline is intentionally minimal: no external services, no Docker builds, no deploy steps. It enforces code correctness on every commit with sub-10-second feedback.

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
│   ├── metrics/         # Metrics endpoint
│   ├── middleware/       # JWT verification middleware
│   ├── models/          # GORM model definitions
│   ├── product/         # Product CRUD handlers and routes
│   └── user/            # User management handlers and routes
├── pkg/database/        # Database connection and migration
├── grafana/             # Grafana provisioning (datasources, alerting)
├── prometheus/          # Prometheus config and alert rules
├── loki/                # Loki configuration
├── promtail/            # Promtail pipeline config
├── nginx/               # Host-level Nginx reverse proxy configs
├── scripts/             # Operational scripts (backup)
├── .github/workflows/   # GitHub Actions CI pipeline
├── docker-compose.yml   # Full stack service definitions
├── Makefile             # Developer convenience targets
└── .env.example         # Environment variable template
```
