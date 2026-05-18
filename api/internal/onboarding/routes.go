package onboarding

import (
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

// Routes mounts browser-facing onboarding endpoints (CSRF-protected via parent router).
func Routes(h *Handler, rdb *redis.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/confirm", h.Confirm)
	}
}

// AdminRoutes mounts machine-to-machine admin endpoints (no CSRF, management token auth).
func AdminRoutes(h *Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/register-client", h.RegisterClient)
		r.Post("/resend-invite", h.ResendInvite)
		r.Delete("/deregister-client/{client_id}", h.DeregisterClient)
	}
}
