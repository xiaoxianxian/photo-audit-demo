# Schema Migrations

Photo Audit Platform uses simple SQL migration files managed by a custom `migrate` CLI tool.

## Directory Structure

```
backend/migrations/
├── 001_initial_schema.up.sql    # Full initial schema (all 20 tables)
├── 001_initial_schema.down.sql  # Rollback: DROP TABLE for all tables
└── ...                           # Future migrations: 002_*.up.sql / 002_*.down.sql
```

## Usage

### CLI Tool

```bash
# From backend/ directory
cd backend

# View migration status
go run cmd/migrate/main.go status

# Apply all pending migrations
go run cmd/migrate/main.go up

# Rollback last migration
go run cmd/migrate/main.go down

# Rollback last N migrations
go run cmd/migrate/main.go down 3

# Rollback all migrations
go run cmd/migrate/main.go reset
```

### Environment Variables

- `DATABASE_URL` — PostgreSQL connection string (default: `postgresql://postgres:postgres@localhost:5432/photo_audit?sslmode=disable`)

### Auto-Migration on Startup

Set `MIGRATE_AUTO_UP=true` in `.env` to run pending migrations automatically when the server starts. Disabled by default.

## Adding a New Migration

1. Create `002_your_change.up.sql` — forward migration SQL
2. Create `002_your_change.down.sql` — rollback SQL
3. Run `go run cmd/migrate/main.go up` to apply
4. Verify with `go run cmd/migrate/main.go status`

## Migration Tracking

Applied versions are tracked in a `schema_migrations` table (auto-created on first run):

| Column   | Type        | Description              |
|----------|-------------|--------------------------|
| version  | INT (PK)    | Migration version number |
| applied_at | TIMESTAMPTZ | When it was applied      |
