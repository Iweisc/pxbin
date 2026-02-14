package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sertdev/pxbin/internal/api"
	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/billing"
	"github.com/sertdev/pxbin/internal/config"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/proxy"
	"github.com/sertdev/pxbin/internal/server"
	"github.com/sertdev/pxbin/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize database connection pool
	pool, err := store.NewPool(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Initialize store and run migrations
	st := store.New(pool)
	if err := st.Migrate(context.Background()); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// Initialize billing tracker
	billingTracker := billing.NewTracker(st)
	defer billingTracker.Close()

	// Initialize async logger
	asyncLogger := logging.NewAsyncLogger(st, cfg.LogBufferSize)
	defer asyncLogger.Close()

	// Initialize upstream client
	upstreamClient := proxy.NewUpstreamClient(cfg.UpstreamBaseURL, cfg.UpstreamAPIKey)

	// Initialize client cache for per-upstream connections
	clientCache := proxy.NewClientCache()

	// Initialize proxy handler
	proxyHandler := proxy.NewHandler(upstreamClient, clientCache, st, asyncLogger, billingTracker)

	// Initialize auth middleware functions
	llmAuth := auth.LLMAuthMiddleware(st)
	mgmtAuth := auth.ManagementAuthMiddleware(st)

	// Initialize management API router
	mgmtRouter := api.NewRouter(st, mgmtAuth, billingTracker)

	// Initialize bootstrap handler (nil if no bootstrap key configured)
	bootstrapHandler := api.NewBootstrapHandler(st, cfg.ManagementBootstrapKey)

	// Build the main server router
	router := server.New(cfg, proxyHandler, llmAuth, mgmtRouter, bootstrapHandler)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 0, // disabled — streaming responses (extended thinking) can run for 10+ minutes
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("pxbin listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	<-done
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown failed: %v", err)
	}
	log.Println("server stopped")
}
