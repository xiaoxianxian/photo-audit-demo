package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var versionRe = regexp.MustCompile(`^(\d+)_.*\.sql$`)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	migrationsDir := "migrations"
	if len(args) > 0 && args[0] != "" {
		migrationsDir = args[0]
	}

	pool, err := connectDB()
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ctx := context.Background()

	switch cmd {
	case "status":
		status(ctx, pool, migrationsDir)

	case "up":
		if err := migrateUp(ctx, pool, migrationsDir); err != nil {
			log.Fatalf("migration up: %v", err)
		}
		fmt.Println("All migrations applied successfully.")

	case "down":
		n := 1
		if len(args) > 0 {
			if v, err := strconv.Atoi(args[0]); err == nil && v > 0 {
				n = v
			}
		}
		if err := migrateDown(ctx, pool, migrationsDir, n); err != nil {
			log.Fatalf("migration down: %v", err)
		}

	case "reset":
		if err := migrateDown(ctx, pool, migrationsDir, 999999); err != nil {
			log.Fatalf("migration reset: %v", err)
		}

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintln(os.Stderr, `Usage:
  migrate status [dir]     Show migration status
  migrate up [dir]         Apply all pending migrations
  migrate down [dir] [n]   Rollback last N migrations (default: 1)
  migrate reset [dir]      Rollback all migrations

If [dir] is omitted, defaults to "migrations/" in current directory.

Environment: DATABASE_URL (defaults to postgresql://postgres:postgres@localhost:5432/photo_audit?sslmode=disable)`)
}

func connectDB() (*pgxpool.Pool, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgresql://postgres:postgres@localhost:5432/photo_audit?sslmode=disable"
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}
	return pool, nil
}

// schemaMigrations ensures the tracking table exists.
func schemaMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version   INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	return err
}

func listMigrations(dir string) ([]struct {
	version uint
	file    string
	isDown  bool
}, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var migs []struct {
		version uint
		file    string
		isDown  bool
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		isDown := strings.HasSuffix(name, ".down.sql")
		if !strings.HasSuffix(name, ".up.sql") && !isDown {
			continue
		}
		base := filepath.Base(name)
		matches := versionRe.FindStringSubmatch(base)
		if len(matches) < 2 {
			continue
		}
		var v uint
		fmt.Sscanf(matches[1], "%d", &v)
		migs = append(migs, struct {
			version uint
			file    string
			isDown  bool
		}{version: v, file: filepath.Join(dir, base), isDown: isDown})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })
	return migs, nil
}

func status(ctx context.Context, pool *pgxpool.Pool, dir string) {
	if err := schemaMigrations(ctx, pool); err != nil {
		log.Fatalf("create schema_migrations: %v", err)
	}

	migs, err := listMigrations(dir)
	if err != nil {
		log.Fatalf("list migrations: %v", err)
	}

	var applied []uint
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err == nil {
		for rows.Next() {
			var v int
			rows.Scan(&v)
			applied = append(applied, uint(v))
		}
	}

	// Build set for lookup
	appliedSet := make(map[uint]bool)
	for _, v := range applied {
		appliedSet[v] = true
	}

	fmt.Println("Migrations:")
	fmt.Printf("  %-10s %-50s %-10s\n", "VERSION", "FILE", "STATUS")
	fmt.Println("  " + strings.Repeat("-", 75))

	for _, m := range migs {
		status := "pending"
		if appliedSet[m.version] {
			status = "applied"
		}
		if m.isDown {
			continue // skip down files in status
		}
		fmt.Printf("  %-10d %-50s %-10s\n", m.version, m.file, status)
	}

	fmt.Printf("\nApplied: %d/%d\n", len(applied), len(migs))
}

func migrateUp(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if err := schemaMigrations(ctx, pool); err != nil {
		return err
	}

	migs, err := listMigrations(dir)
	if err != nil {
		return err
	}

	var applied map[uint]bool
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version")
	if err == nil {
		applied = make(map[uint]bool)
		for rows.Next() {
			var v int
			rows.Scan(&v)
			applied[uint(v)] = true
		}
	}

	pending := make([]struct{ version uint; file string }, 0)
	for _, m := range migs {
		if m.isDown || applied[m.version] {
			continue
		}
		pending = append(pending, struct {
			version uint
			file    string
		}{m.version, m.file})
	}

	if len(pending) == 0 {
		fmt.Println("No pending migrations.")
		return nil
	}

	for _, p := range pending {
		data, err := os.ReadFile(p.file)
		if err != nil {
			return fmt.Errorf("read %s: %w", p.file, err)
		}

		_, err = pool.Exec(ctx, string(data))
		if err != nil {
			return fmt.Errorf("exec %s: %w", p.file, err)
		}

		_, err = pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", p.version)
		if err != nil {
			return fmt.Errorf("track %s: %w", p.file, err)
		}

		fmt.Printf("  migrated up %d (%s)\n", p.version, filepath.Base(p.file))
	}

	return nil
}

func migrateDown(ctx context.Context, pool *pgxpool.Pool, dir string, n int) error {
	if err := schemaMigrations(ctx, pool); err != nil {
		return err
	}

	migs, err := listMigrations(dir)
	if err != nil {
		return err
	}

	// Get applied versions in reverse order
	var applied []uint
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations ORDER BY version DESC")
	if err != nil {
		return err
	}
	for rows.Next() {
		var v int
		rows.Scan(&v)
		applied = append(applied, uint(v))
	}

	if len(applied) == 0 {
		fmt.Println("No migrations to rollback.")
		return nil
	}

	rollbackCount := n
	if rollbackCount > len(applied) {
		rollbackCount = len(applied)
	}

	for i := 0; i < rollbackCount; i++ {
		v := applied[i]
		// Find matching down file
		downFile := ""
		for _, m := range migs {
			if m.version == v && m.isDown {
				downFile = m.file
				break
			}
		}

		if downFile == "" {
			fmt.Printf("  WARNING: no down migration for version %d, skipping\n", v)
			continue
		}

		data, err := os.ReadFile(downFile)
		if err != nil {
			return fmt.Errorf("read down %s: %w", downFile, err)
		}

		_, err = pool.Exec(ctx, string(data))
		if err != nil {
			return fmt.Errorf("exec down %s: %w", downFile, err)
		}

		_, err = pool.Exec(ctx, "DELETE FROM schema_migrations WHERE version = $1", v)
		if err != nil {
			return fmt.Errorf("untrack %d: %w", v, err)
		}

		fmt.Printf("  migrated down %d (%s)\n", v, filepath.Base(downFile))
	}

	return nil
}
