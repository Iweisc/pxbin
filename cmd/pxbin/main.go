package main

import (
	"context"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pxbin "github.com/sertdev/pxbin"
	"github.com/sertdev/pxbin/internal/api"
	"github.com/sertdev/pxbin/internal/auth"
	"github.com/sertdev/pxbin/internal/billing"
	"github.com/sertdev/pxbin/internal/config"
	"github.com/sertdev/pxbin/internal/crypto"
	"github.com/sertdev/pxbin/internal/logging"
	"github.com/sertdev/pxbin/internal/metrics"
	"github.com/sertdev/pxbin/internal/proxy"
	"github.com/sertdev/pxbin/internal/ratelimit"
	"github.com/sertdev/pxbin/internal/resilience"
	"github.com/sertdev/pxbin/internal/server"
	"github.com/sertdev/pxbin/internal/slogger"
	"github.com/sertdev/pxbin/internal/store"
)

func main() {
	// 1. Load config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 2. Validate config
	if err := config.Validate(cfg); err != nil {
		log.Fatalf("config validation failed: %v", err)
	}

	// 3. Setup structured logging
	slogger.Setup(cfg.LogFormat)

	// 4. Derive encryption key (if set)
	var encryptionKey []byte
	if cfg.EncryptionKey != "" {
		encryptionKey = crypto.DeriveKey(cfg.EncryptionKey)
	}

	// 5. Initialize database connection pool with configurable sizes
	pool, err := store.NewPool(context.Background(), cfg.DatabaseURL, cfg.DatabaseSchema, cfg.MaxDBConns, cfg.MinDBConns)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	// 6. Initialize store (with encryption if key is set)
	var st *store.Store
	if encryptionKey != nil {
		st = store.NewWithEncryption(pool, encryptionKey)
	} else {
		st = store.New(pool)
	}

	// 7. Run migrations
	if err := st.Migrate(context.Background()); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	// 8. Initialize billing tracker
	billingTracker := billing.NewTracker(st)
	defer billingTracker.Close()

	// 9. Initialize async logger
	asyncLogger := logging.NewAsyncLogger(st, cfg.LogBufferSize)
	defer asyncLogger.Close()

	// 10. Initialize log retention cleaner
	logCleaner := logging.NewLogCleaner(st, cfg.LogRetentionDays)
	defer logCleaner.Close()

	// 11. Initialize metrics (if enabled)
	var m *metrics.Metrics
	var metricsMiddleware func(http.Handler) http.Handler
	var metricsHandler http.Handler
	if cfg.MetricsEnabled {
		m = metrics.New()
		metricsMiddleware = metrics.Middleware(m)
		metricsHandler = m.Handler()
		asyncLogger.SetDroppedCounter(m.DroppedLogsTotal)
	}

	// 12. Initialize rate limiter (if configured)
	var rateLimiter *ratelimit.Limiter
	if cfg.RateLimitRPS > 0 {
		burst := cfg.RateLimitBurst
		if burst <= 0 {
			burst = int(cfg.RateLimitRPS * 2) // default burst = 2x RPS
		}
		rateLimiter = ratelimit.NewLimiter(cfg.RateLimitRPS, burst)
		defer rateLimiter.Close()
	}

	// 13. Initialize upstream options (circuit breaker + retry)
	var upstreamOpts *proxy.UpstreamOpts
	if cfg.CBFailureThreshold > 0 || cfg.RetryMaxAttempts > 1 {
		upstreamOpts = &proxy.UpstreamOpts{
			CBOpts: resilience.CircuitBreakerOpts{
				Threshold: cfg.CBFailureThreshold,
				Timeout:   time.Duration(cfg.CBTimeoutSeconds) * time.Second,
			},
			RetryOpts: resilience.RetryOpts{
				MaxAttempts: cfg.RetryMaxAttempts,
				BaseDelay:   time.Duration(cfg.RetryBaseDelayMS) * time.Millisecond,
			},
		}
	}

	// 14. Initialize client cache with resilience options
	clientCache := proxy.NewClientCache(upstreamOpts)

	// 15. Initialize model resolution cache (60s TTL)
	modelCache := proxy.NewModelCache(st, 60*time.Second)
	if err := modelCache.Warm(context.Background()); err != nil {
		log.Printf("model cache warmup failed: %v", err)
	}

	// 16. Initialize proxy handler
	proxyHandler := proxy.NewHandler(clientCache, modelCache, st, asyncLogger, billingTracker)

	// 17. Initialize auth key cache and last-used tracker
	keyCache := auth.NewKeyCache(st, 60*time.Second)
	lastUsedTracker := auth.NewLastUsedTracker(st)
	defer lastUsedTracker.Close()

	// 18. Initialize auth middleware functions
	llmAuth := auth.LLMAuthMiddleware(keyCache, lastUsedTracker)
	mgmtAuth := auth.ManagementAuthMiddleware(st)

	// 19. Initialize management API router
	mgmtRouter := api.NewRouter(st, mgmtAuth, billingTracker)

	// 20. Initialize bootstrap handler (nil if no bootstrap key configured)
	bootstrapHandler := api.NewBootstrapHandler(st, cfg.ManagementBootstrapKey)

	// 21. Strip "frontend/dist" prefix from embedded FS
	frontendFS, err := fs.Sub(pxbin.FrontendDist, "frontend/dist")
	if err != nil {
		log.Fatalf("failed to load embedded frontend: %v", err)
	}

	// 22. Build the main server router with middleware
	serverOpts := &server.Opts{
		RateLimiter:       rateLimiter,
		MetricsMiddleware: metricsMiddleware,
		MetricsHandler:    metricsHandler,
		Pool:              pool,
	}
	router := server.New(cfg, proxyHandler, llmAuth, mgmtRouter, bootstrapHandler, frontendFS, serverOpts)

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
