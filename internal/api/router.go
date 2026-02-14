package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/sertdev/pxbin/internal/billing"
	"github.com/sertdev/pxbin/internal/store"
)

func NewRouter(s *store.Store, authMw func(http.Handler) http.Handler, bt *billing.Tracker) chi.Router {
	r := chi.NewRouter()

	r.Group(func(r chi.Router) {
		r.Use(authMw)

		r.Route("/keys", func(r chi.Router) {
			h := &keysHandler{store: s}
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Patch("/{id}", h.Update)
			r.Delete("/{id}", h.Delete)
		})

		r.Route("/logs", func(r chi.Router) {
			h := &logsHandler{store: s}
			r.Get("/", h.List)
			r.Get("/{id}", h.Get)
		})

		r.Route("/models", func(r chi.Router) {
			h := &modelsHandler{store: s, billing: bt}
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Post("/discover", h.Discover)
			r.Post("/import", h.Import)
			r.Post("/sync-pricing", h.SyncPricing)
			r.Post("/bulk-delete", h.BulkDelete)
			r.Patch("/{id}", h.Update)
			r.Delete("/{id}", h.Delete)
		})

		r.Route("/upstreams", func(r chi.Router) {
			h := &upstreamsHandler{store: s}
			r.Get("/", h.List)
			r.Post("/", h.Create)
			r.Post("/bulk-delete", h.BulkDelete)
			r.Patch("/{id}", h.Update)
			r.Delete("/{id}", h.Delete)
		})

		r.Route("/stats", func(r chi.Router) {
			h := &statsHandler{store: s}
			r.Get("/overview", h.Overview)
			r.Get("/by-key", h.ByKey)
			r.Get("/by-model", h.ByModel)
			r.Get("/timeseries", h.TimeSeries)
			r.Get("/latency", h.Latency)
		})
	})

	return r
}
