package server

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sertdev/pxbin/internal/config"
	"github.com/sertdev/pxbin/internal/ratelimit"
)

// ProxyHandler defines the interface for the LLM proxy handler.
type ProxyHandler interface {
	HandleAnthropic(w http.ResponseWriter, r *http.Request)
	HandleOpenAI(w http.ResponseWriter, r *http.Request)
	HandleOpenAIResponses(w http.ResponseWriter, r *http.Request)
}

// Opts holds optional middleware and dependencies for server construction.
type Opts struct {
	RateLimiter       *ratelimit.Limiter      // nil = disabled
	MetricsMiddleware func(http.Handler) http.Handler // nil = disabled
	MetricsHandler    http.Handler                     // nil = no /metrics endpoint
	Pool              *pgxpool.Pool                    // for readiness probe
}

// New creates and configures the chi router with all routes mounted.
func New(cfg *config.Config, proxy ProxyHandler, llmAuth func(http.Handler) http.Handler, mgmtRouter chi.Router, bootstrapHandler http.HandlerFunc, frontendFS fs.FS, opts *Opts) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(requestID)
	r.Use(SecurityHeaders)

	if opts != nil && opts.MetricsMiddleware != nil {
		r.Use(opts.MetricsMiddleware)
	}

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID", "x-api-key"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// LLM proxy routes (require LLM API key auth)
	r.Route("/v1", func(r chi.Router) {
		r.Use(llmAuth)
		if opts != nil && opts.RateLimiter != nil {
			r.Use(rateLimitMiddleware(opts.RateLimiter))
		}
		r.Post("/messages", proxy.HandleAnthropic)
		r.Post("/chat/completions", proxy.HandleOpenAI)
		r.Post("/responses", proxy.HandleOpenAIResponses)
		r.Post("/responses/compact", proxy.HandleOpenAIResponses)
	})

	// Management API routes (already handled by the management router's middleware)
	r.Mount("/api/v1", mgmtRouter)

	// Bootstrap endpoint (only active when bootstrap key is configured)
	if bootstrapHandler != nil {
		r.Post("/api/v1/bootstrap", bootstrapHandler)
	}

	// Health and readiness probes (no auth)
	r.Get("/health", HealthHandler())
	if opts != nil && opts.Pool != nil {
		r.Get("/ready", ReadinessHandler(opts.Pool))
	}

	// Prometheus metrics endpoint
	if opts != nil && opts.MetricsHandler != nil {
		r.Handle("/metrics", opts.MetricsHandler)
	}

	// Serve embedded frontend (SPA with index.html fallback)
	if frontendFS != nil {
		fileServer := http.FileServer(http.FS(frontendFS))
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			// Try serving the exact file first
			if _, err := fs.Stat(frontendFS, r.URL.Path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
			// Fall back to index.html for SPA client-side routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
		})
	}

	return r
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// rateLimitMiddleware creates a chi middleware that rate-limits by auth key ID.
func rateLimitMiddleware(limiter *ratelimit.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Use the request's X-Request-ID header as a fallback key, but
			// prefer the authenticated key ID from context if available.
			key := r.Header.Get("X-Api-Key")
			if key == "" {
				key = r.Header.Get("Authorization")
			}
			if key == "" {
				key = r.RemoteAddr
			}

			if !limiter.Allow(key) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
