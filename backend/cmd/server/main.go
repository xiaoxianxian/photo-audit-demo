package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"audit-platform/internal/api"
	"audit-platform/internal/config"
	altlogger "audit-platform/internal/logger"
	"audit-platform/internal/middleware"
	"audit-platform/internal/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool *pgxpool.Pool
	dbOnce sync.Once
)

// getDB returns a singleton database connection pool.
var dbInitErr error // preserved across sync.Once so callers see the real failure

func getDB(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	dbOnce.Do(func() {
		cfg, err := pgxpool.ParseConfig(dsn)
		if err != nil {
			dbInitErr = fmt.Errorf("parse database config: %w", err)
			return
		}
		cfg.MaxConns = 100
		cfg.MinConns = 10
		cfg.MaxConnIdleTime = 5 * time.Minute
		cfg.MaxConnLifetime = 30 * time.Minute
		cfg.HealthCheckPeriod = 30 * time.Second

		pool, err := pgxpool.NewWithConfig(ctx, cfg)
		if err != nil {
			dbInitErr = fmt.Errorf("connect database: %w", err)
			return
		}

		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			dbInitErr = fmt.Errorf("ping database: %w", err)
			return
		}
		dbPool = pool
	})
	return dbPool, dbInitErr
}

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger with file rotation support
	logFile := os.Getenv("LOG_FILE")
	if logFile == "" {
		logFile = filepath.Join("logs", "audit-platform.log")
	}
	mainLogger := altlogger.NewWithConfig(altlogger.LogConfig{
		Component: "main",
		Level:     altlogger.LevelInfo,
		LogFile:   logFile,
	})
	defer mainLogger.Close()

	// Validate JWT secret meets minimum length requirement.
	// This prevents deploying with a predictable default secret.
	if len(cfg.JWTSecret) < cfg.JWTMinLength {
		log.Fatalf("JWT_SECRET must be at least %d characters long (current: %d). Set a strong secret via environment variable.", cfg.JWTMinLength, len(cfg.JWTSecret))
	}

	// Connect to PostgreSQL
	log.Printf("connecting to database at %s", redactURL(cfg.DatabaseURL))
	db, err := getDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()
	log.Println("database connected successfully")

	// Auto-run migrations if MIGRATE_AUTO_UP=true
	if os.Getenv("MIGRATE_AUTO_UP") == "true" {
		log.Println("running schema migrations...")
		if err := runMigrations(ctx, db); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		log.Println("migrations completed")
	}

	// Wire dependencies
	svc := service.NewServices(db, cfg.JWTSecret, cfg)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: defaultErrorHandler,
		BodyLimit:    int(cfg.MaxUploadBytes),
	})

	// Global middleware
	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.AllowedOrigins,
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization,X-Tenant-ID",
	}))

	app.Use(logger.New(logger.Config{
		Format:     "[${ip}] ${method} ${path} ${status} ${latency}\n",
		TimeZone:   "Asia/Shanghai",
	}))
	app.Use(recover.New())

	// Rate limiting: 100 requests per 15 minutes per IP
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"code":    429,
				"message": "请求过于频繁，请稍后重试",
			})
		},
	}))

	// Build middleware instances
	authMW := middleware.Auth(cfg)
	tenantMW := middleware.Tenant(db)

	// Build handlers
	handlers := api.NewHandlers(svc)

	// Build health checker
	healthChecker := api.NewHealthChecker(db)

	// Start WebSocket hub in background
	go svc.WsHub.Run()

	// Start stream snapshot scheduler in background
	go svc.StreamScheduler.Start(ctx)

	// Register routes
	api.SetupRoutes(app, handlers, authMW, tenantMW)
	
	// Register health check routes
	api.SetupHealthRoutes(app, healthChecker)

	// Start server
	banner := "=========================\n" +
		"  Photo Audit Platform  \n" +
		"  Port: " + cfg.ServerPort + "\n" +
		"========================="
	log.Println(banner)

	// Start server in background
	go func() {
		if err := app.Listen(":" + cfg.ServerPort); err != nil {
			log.Fatalf("server failed: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server...")

	// Stop WebSocket hub
	svc.WsHub.Stop()

	// Stop stream scheduler
	svc.StreamScheduler.Stop()

	// Shutdown Fiber server gracefully with 30s timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Printf("server forced to shutdown: %v", err)
	}

	log.Println("server exited properly")
}

// defaultErrorHandler is the global Fiber error handler.
func defaultErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}
	return c.Status(code).JSON(fiber.Map{
		"code":    code,
		"message": err.Error(),
	})
}

// migrationRe matches versioned migration files: 001_name.up.sql
var migrationRe = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

// runMigrations applies pending migrations from the migrations/ directory.
func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Ensure tracking table
	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version   INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`); err != nil {
		return err
	}

	// Read migration files (relative to project root)
	cwd, _ := os.Getwd()
	// cwd is backend/cmd/server, go up 3 levels to project root
	migrationsDir := filepath.Join(cwd, "..", "..", "..", "backend", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	type mig struct {
		version uint
		file    string
	}
	var migs []mig

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		matches := migrationRe.FindStringSubmatch(e.Name())
		if len(matches) < 2 {
			continue
		}
		var v uint
		fmt.Sscanf(matches[1], "%d", &v)
		migs = append(migs, mig{v, filepath.Join(migrationsDir, e.Name())})
	}

	sort.Slice(migs, func(i, j int) bool { return migs[i].version < migs[j].version })

	// Get applied versions
	var applied map[uint]bool
	rows, err := pool.Query(ctx, "SELECT version FROM schema_migrations")
	if err == nil {
		applied = make(map[uint]bool)
		for rows.Next() {
			var v int
			rows.Scan(&v)
			applied[uint(v)] = true
		}
	}

	// Apply pending
	for _, m := range migs {
		if applied[m.version] {
			continue
		}
		data, err := os.ReadFile(m.file)
		if err != nil {
			return fmt.Errorf("read %s: %w", m.file, err)
		}
		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("exec %s: %w", m.file, err)
		}
		if _, err := pool.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", m.version); err != nil {
			return fmt.Errorf("track %s: %w", m.file, err)
		}
		log.Printf("  migrated up %d (%s)", m.version, filepath.Base(m.file))
	}

	return nil
}

// redactURL strips credentials from a DSN for safe logging.
func redactURL(url string) string {
	if url == "" {
		return ""
	}
	return "postgresql://***:***@..."
}
