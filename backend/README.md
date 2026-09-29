# Aegis backend

Go API for K3 incident reporting. Listens on port **8080**. The background worker handles overdue corrective actions, supervisor SLA reminders, async PDF reports, and the monthly recap on the 1st (`REPORT_TZ`, default `Asia/Jakarta`).

This folder lives under the `aegis/` monorepo (backend + frontend).

## Prerequisites

- Go 1.25
- Docker Compose, for the full stack

## Configuration

Copy `.env.example` to `.env` and fill values on the machine that runs the stack. `.env` is gitignored. Placeholders in `.env.example` are empty; do not commit real passwords, JWT secrets, or RustFS keys.

Production must set `JWT_ACCESS_SECRET`, `POSTGRES_PASSWORD`, `RUSTFS_ACCESS_KEY`, and `RUSTFS_SECRET_KEY`. The process falls back to a development JWT secret only when `JWT_ACCESS_SECRET` is unset.

When `SUPERADMIN_USERNAME` and `SUPERADMIN_PASSWORD` are both set, the API inserts that user once at startup if the email is missing. Log in with `POST /auth/login` using the username as `email`. An existing account is left unchanged. Leave either value empty to skip seeding.

## Run

Development (publishes Postgres `5432` and Redis `6379`):

```bash
docker compose up -d --build
```


You can also run the API on the host (`go run ./cmd/server`) against published ports, with `DATABASE_DSN` and `REDIS_ADDR` pointing at `localhost`.

## Migrations

Schema is applied with GORM AutoMigrate when the API or worker starts. SQL files in `migrations/` are PostgreSQL snapshots for review. They are not a separate migrate CLI.

## Tests

```bash
go test ./internal/... ./pkg/... -count=1
go test ./... -count=1
```

Coverage for `internal/` (Phase 5 target is at least 70%):

```bash
go test -count=1 -coverpkg=./internal/... -coverprofile=coverage.out ./internal/...
go tool cover -func=coverage.out
```

## URLs

- Health: http://localhost:8080/health
- Swagger UI: http://localhost:8080/swagger/index.html
- Postgres: `5432` (development compose only)
- Redis: `6379` (development compose only)
- RustFS API: http://localhost:9000
- RustFS console: http://localhost:9001

Regenerate OpenAPI after handler comment changes:

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs --parseInternal
```

## Worker

`cmd/worker` (`Dockerfile.worker`) runs independently of the API:

- daily overdue corrective actions and due-soon reminders
- supervisor SLA reminders (T-4 hours and past 24 hours)
- PDF jobs queued by the API, stored in RustFS
- monthly recap enqueue on the 1st in `REPORT_TZ`

Dashboard query notes: [docs/performance.md](docs/performance.md).
