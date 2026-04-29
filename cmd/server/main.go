// Package main is the entrypoint for the LJ-List API server. It wires together
// configuration, database, middleware, and routes, then starts an HTTP server
// with graceful shutdown. No business logic lives here, this file only
// orchestrates dependencies. If you're looking for how a request is handled,
// start in internal/routes/routes.go and follow the chain down.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GordenArcher/godenv"
	"github.com/GordenArcher/lj-list-api/internal/config"
	"github.com/GordenArcher/lj-list-api/internal/database"
	"github.com/GordenArcher/lj-list-api/internal/middleware"
	"github.com/GordenArcher/lj-list-api/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	// Environment
	//
	// Load .env for local development, but don't require it in containerized
	// or managed environments where variables are injected externally.
	// Parse errors still abort startup because a malformed .env is a real
	// configuration bug, not an acceptable fallback condition.
	if err := godenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("failed to load .env: %v", err)
	}

	cfg := config.Load()

	//  Database
	//
	// pgxpool manages a pool of PostgreSQL connections. We pass the pool to
	// every repository; no package creates its own connection. This ensures
	// connection limits are respected globally and tests can inject a
	// separate pool or a transaction.
	pool, err := database.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Run pending migrations on startup. The server will not start against
	// a database that hasn't been migrated. This prevents the dreaded
	// "column does not exist" error at 2 AM because someone forgot to run
	// migrations during deployment.
	if err := database.RunMigrations(cfg); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	log.Println("database connected and migrations applied")

	// Router
	//
	// gin.New() instead of gin.Default() because we attach our own middleware
	// explicitly. This gives us control over the order and avoids the default
	// logger (which writes to stdout and doesn't include request IDs, our
	// middleware does both).
	router := gin.New()

	// Middleware runs in the order it's added. RequestID must be first so
	// every subsequent middleware and handler can access it via the context.
	// CORS runs before auth so OPTIONS preflight requests don't get rejected
	// as unauthenticated. RateLimit runs globally (after CORS) so every API
	// route shares the same per-IP cap.
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS(cfg))
	router.Use(middleware.GlobalRateLimit(cfg))
	router.Use(gin.Recovery())

	// Routes receives everything via explicit parameters. No package-level
	// globals, no init() side effects. If a handler needs the database, it
	// gets it through its service, which gets it through its repository,
	// which received the pool here.
	routes.Register(router, pool, cfg)

	// HTTP Server
	//
	// Timeouts are critical for production. ReadTimeout prevents a slow
	// client from holding a connection open while sending a request body.
	// WriteTimeout caps how long we wait while writing the response.
	// IdleTimeout releases keep-alive connections that sit unused.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful Shutdown
	//
	// We listen for SIGINT (Ctrl+C) and SIGTERM (Docker stop, systemd
	// restart) in a goroutine. When the signal arrives, we stop accepting
	// new connections and give in-flight requests a deadline to complete
	// before forcefully closing.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("server starting on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-quit
	log.Println("shutting down server...")

	// The context with timeout gives in-flight requests time to finish.
	// After the deadline, Shutdown returns and we close the pool.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("server stopped gracefully")
}
