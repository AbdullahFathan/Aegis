# Runtime schema is applied with GORM AutoMigrate (`database.AutoMigrate`).
# PostgreSQL-oriented snapshot for Phase 4 (work hours + generated reports).

CREATE TABLE IF NOT EXISTS work_hours (
    id UUID PRIMARY KEY,
    location_id UUID REFERENCES locations(id) ON DELETE CASCADE,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    hours DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS generated_reports (
    id UUID PRIMARY KEY,
    type VARCHAR(32) NOT NULL,
    format VARCHAR(8) NOT NULL,
    status VARCHAR(16) NOT NULL,
    params JSONB,
    stored_key VARCHAR(512),
    error_message TEXT,
    requested_by_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_work_hours_location_id ON work_hours (location_id);
CREATE INDEX IF NOT EXISTS idx_work_hours_period_start ON work_hours (period_start);
CREATE INDEX IF NOT EXISTS idx_generated_reports_status ON generated_reports (status);
CREATE INDEX IF NOT EXISTS idx_generated_reports_requested_by_id ON generated_reports (requested_by_id);
