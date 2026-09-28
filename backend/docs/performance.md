# Dashboard query notes

Target from the product requirements: dashboard load with 1,000+ incidents under 3 seconds. k6 is not part of CI.

## Indexes

`incidents` already has single-column indexes on `status`, `location_id`, and `incident_datetime`. Summary, trends, and heatmap filter those columns together (and optionally `category`) in `internal/dashboard/service.go` (`incidentQ`).

A composite index matches that filter:

`idx_incident_status_location_time (status, location_id, incident_datetime)`

GORM creates it on AutoMigrate from the model tags. The SQL snapshot is in `migrations/0001_init.sql`. `category` stays on its own index.

## How to measure on Postgres

This note does not include a timed run against a loaded Postgres instance. To measure locally:

1. Start Postgres from Compose and point `DATABASE_DSN` at it.
2. Insert at least 1,000 non-draft incidents spread across sites and months.
3. Explain the summary filter:

```sql
EXPLAIN ANALYZE
SELECT count(*) FROM incidents
WHERE deleted_at IS NULL
  AND status <> 'DRAFT'
  AND location_id = '<site-uuid>'
  AND incident_datetime >= '2026-01-01'
  AND incident_datetime <= '2026-09-28';
```

4. Time `GET /dashboard/summary`, `/dashboard/trends`, and `/dashboard/heatmap` with the same filters. The NFR target is under 3 seconds for the dashboard load.

`BenchmarkDashboardSummary` in `internal/dashboard` exercises the summary query on a small sqlite fixture. It is not a substitute for the 1,000-row Postgres check.
