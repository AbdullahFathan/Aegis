# Aegis backend

Go API on port **8080** (JWT auth, RBAC, users, locations). Background worker is a no-op heartbeat until Phase 3.

This folder lives under the `aegis/` monorepo (backend + frontend).

## Tests (phase gate)

```bash
go test ./internal/... ./pkg/... -count=1
```

## Run with Docker Compose

Copy `.env.example` to `.env`. Values there are placeholders, not production secrets.

All services (`db`, `redis`, `rustfs`, `backend`, `worker`) run in Compose. The stack overrides `DATABASE_DSN`, `REDIS_ADDR`, and `RUSTFS_ENDPOINT` to Docker DNS names (`db`, `redis`, `rustfs`).

```bash
docker compose up -d --build
```

- Health: http://localhost:8080/health
- Swagger UI: http://localhost:8080/swagger/index.html
- Postgres: 5432
- Redis: 6379
- RustFS: 9000 (API), 9001 (console)

Optional: run the API on the host against published ports (`DATABASE_DSN` / `REDIS_ADDR` pointing at `localhost`) while Compose still runs Postgres, Redis, and RustFS.

Regenerate OpenAPI after changing handler annotations:

```bash
go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o docs --parseInternal
```

## Worker

`cmd/worker` will run overdue CA, SLA reminders, and PDF jobs in later phases. Right now it only logs a heartbeat. `Dockerfile.worker` builds a separate image so the API and jobs can scale independently.
