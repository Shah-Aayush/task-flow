package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Shah-Aayush/task-flow/backend/internal/config"
	"github.com/Shah-Aayush/task-flow/backend/internal/handler"
	"github.com/Shah-Aayush/task-flow/backend/internal/handler/middleware"
	embedMigrations "github.com/Shah-Aayush/task-flow/backend/internal/migrations"
	repoPostgres "github.com/Shah-Aayush/task-flow/backend/internal/repository/postgres"
	"github.com/Shah-Aayush/task-flow/backend/internal/service"
	"github.com/Shah-Aayush/task-flow/backend/internal/validator"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	migratePostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// --- Logger (structured JSON, stdout) ---
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// --- Config (fail-fast validation) ---
	cfg := config.Load(logger)
	logger.Info("configuration loaded", "port", cfg.ServerPort, "db_host", cfg.DBHost)

	// --- Database Connection Pool ---
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL())
	if err != nil {
		logger.Error("failed to parse database URL", "error", err)
		os.Exit(1)
	}
	poolCfg.MaxConns = 25
	poolCfg.MinConns = 5
	poolCfg.MaxConnLifetime = 5 * time.Minute
	poolCfg.MaxConnIdleTime = 2 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		logger.Error("failed to create connection pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Health check ping with retry logic (wait for Postgres to be ready)
	if err := waitForDB(pool, logger, 30*time.Second); err != nil {
		logger.Error("database is not reachable", "error", err)
		os.Exit(1)
	}
	logger.Info("database connection established")

	// --- Run Migrations ---
	if err := runMigrations(cfg.DatabaseURL(), logger); err != nil {
		logger.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations applied successfully")

	// --- Run Seed Data ---
	if err := runSeed(pool, logger); err != nil {
		logger.Error("failed to run seed data", "error", err)
		// Non-fatal — app can still run without seed data
	}

	// --- Wire Repositories ---
	userRepo := repoPostgres.NewUserRepository(pool)
	projectRepo := repoPostgres.NewProjectRepository(pool)
	taskRepo := repoPostgres.NewTaskRepository(pool)

	// --- Wire Services ---
	authService := service.NewAuthService(userRepo, cfg.JWTSecret, cfg.BcryptCost)
	projectService := service.NewProjectService(projectRepo)
	taskService := service.NewTaskService(taskRepo, projectRepo, userRepo)

	// --- Wire Handlers ---
	v := validator.New()
	authHandler := handler.NewAuthHandler(authService, v)
	projectHandler := handler.NewProjectHandler(projectService, v)
	taskHandler := handler.NewTaskHandler(taskService, v)

	// --- Router Setup ---
	r := chi.NewRouter()

	// Global middleware stack (applied to all routes)
	r.Use(chiMiddleware.Recoverer)   // recover from panics, return 500
	r.Use(chiMiddleware.RequestID)   // inject X-Request-ID for tracing
	r.Use(chiMiddleware.RealIP)      // use X-Forwarded-For if behind proxy
	r.Use(middleware.Logger(logger)) // structured request logging

	// Health check (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		handler.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// --- Auth routes (no auth middleware) ---
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// --- Protected routes (JWT required) ---
	// The auth middleware is scoped to this route group only.
	// Placing it here in the router (not per-handler) ensures we never forget to
	// protect a new endpoint added to this group.
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth(cfg.JWTSecret))

		// Projects
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projectHandler.List)
			r.Post("/", projectHandler.Create)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", projectHandler.GetByID)
				r.Patch("/", projectHandler.Update)
				r.Delete("/", projectHandler.Delete)

				// Task sub-resources (nested under project)
				r.Get("/tasks", taskHandler.ListByProject)
				r.Post("/tasks", taskHandler.Create)

				// Bonus: stats endpoint
				r.Get("/stats", taskHandler.GetStats)
			})
		})

		// Tasks (top-level routes for update/delete by task ID)
		r.Route("/tasks/{id}", func(r chi.Router) {
			r.Patch("/", taskHandler.Update)
			r.Delete("/", taskHandler.Delete)
		})
	})

	// --- HTTP Server ---
	server := &http.Server{
		Addr:         cfg.ServerAddr(),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so we can listen for shutdown signals
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	// --- Graceful Shutdown ---
	// Wait for SIGTERM or SIGINT (Ctrl+C in dev, Kubernetes pod termination in prod)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	select {
	case err := <-serverErrors:
		logger.Error("server error", "error", err)
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig.String())
		// Give in-flight requests up to 10 seconds to complete
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			logger.Error("graceful shutdown failed", "error", err)
		}
		logger.Info("server shut down cleanly")
	}
}

// waitForDB pings the database with retries until it succeeds or the deadline passes.
// Necessary because docker-compose's `depends_on: condition: service_healthy` can
// still have a brief window where the DB is not fully accepting connections.
func waitForDB(pool *pgxpool.Pool, logger *slog.Logger, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := pool.Ping(context.Background()); err == nil {
			return nil
		}
		logger.Info("waiting for database...", "retry_in", "1s")
		time.Sleep(1 * time.Second)
	}
	return pool.Ping(context.Background())
}

// runMigrations runs all pending UP migrations using golang-migrate as a library.
// Running migrations in Go code (not a shell script) allows us to use a minimal
// distroless Docker image without a shell, and ensures migrations are always run
// before the server starts.
func runMigrations(dbURL string, logger *slog.Logger) error {
	// Use embed.FS from the migrations package (baked into the binary)
	sourceDriver, err := iofs.New(embedMigrations.FS, "sql/migrations")
	if err != nil {
		return err
	}

	// Open a *sql.DB for migrate (pgxpool is not *sql.DB compatible)
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		return err
	}
	defer db.Close()

	dbDriver, err := migratePostgres.WithInstance(db, &migratePostgres.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "postgres", dbDriver)
	if err != nil {
		return err
	}

	upErr := m.Up()
	if upErr != nil && upErr != migrate.ErrNoChange {
		return upErr
	}

	if upErr == migrate.ErrNoChange {
		logger.Info("migrations: no changes to apply")
	}

	return nil
}

// runSeed executes the seed SQL file idempotently.
// Uses ON CONFLICT DO NOTHING, so running this multiple times is safe.
func runSeed(pool *pgxpool.Pool, logger *slog.Logger) error {
	if embedMigrations.SeedSQL == "" {
		return nil
	}
	_, err := pool.Exec(context.Background(), embedMigrations.SeedSQL)
	if err != nil {
		return err
	}
	logger.Info("seed data applied")
	return nil
}

// stdlib is imported for side-effects only (pgx sql driver registration for golang-migrate).
