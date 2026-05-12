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
		r.With(middleware.RateLimit(rdb, "onboard", 1, 2*time.Minute,
			"Please wait 2 minutes before requesting another confirmation link.")).
			Post("/connect", h.Connect)
		r.Get("/confirm", h.Confirm)
	}
}

// AdminRoutes mounts machine-to-machine admin endpoints (no CSRF, management token auth).
func AdminRoutes(h *Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/register-client", h.RegisterClient)
		r.Post("/resend-invite", h.ResendInvite)
	}
}
