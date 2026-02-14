package server

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/config"
)

// ProxyHandler defines the interface for the LLM proxy handler.
type ProxyHandler interface {
	HandleAnthropic(w http.ResponseWriter, r *http.Request)
	HandleOpenAI(w http.ResponseWriter, r *http.Request)
}

// New creates and configures the chi router with all routes mounted.
func New(cfg *config.Config, proxy ProxyHandler, llmAuth func(http.Handler) http.Handler, mgmtRouter chi.Router, bootstrapHandler http.HandlerFunc) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(requestID)
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
		r.Post("/messages", proxy.HandleAnthropic)
		r.Post("/chat/completions", proxy.HandleOpenAI)
	})

	// Management API routes (already handled by the management router's middleware)
	r.Mount("/api/v1", mgmtRouter)

	// Bootstrap endpoint (only active when bootstrap key is configured)
	if bootstrapHandler != nil {
		r.Post("/api/v1/bootstrap", bootstrapHandler)
	}

	// Health check (no auth)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

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
