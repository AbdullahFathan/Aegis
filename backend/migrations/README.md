# Migrations

Production and local Postgres apply schema via GORM `AutoMigrate` at API startup.

`0001_init.sql` is a human-readable PostgreSQL snapshot of the same entities (PRD §10 plus Region/Area). Do not treat it as the runtime migrator for this phase.
