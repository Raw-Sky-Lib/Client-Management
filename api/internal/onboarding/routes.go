package onboarding

import (
	"time"

	"github.com/DagMT/client-portal/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// Routes mounts browser-facing onboarding endpoints (rate-limited, CSRF-protected via parent router).
func Routes(h *Handler, rdb *redis.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.With(middleware.RateLimit(rdb, "onboard", 5, time.Minute)).
			Post("/connect", h.Connect)
		r.Get("/confirm", h.Confirm)
	}
}

// AdminRoutes mounts the machine-to-machine register-client endpoint (no CSRF, management token auth).
func AdminRoutes(h *Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/register-client", h.RegisterClient)
	}
}
