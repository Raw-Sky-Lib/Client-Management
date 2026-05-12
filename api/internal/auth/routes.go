package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

// Routes mounts all portal auth endpoints. Must be mounted inside the CSRF middleware group.
// magicLinkRL and authenticate are passed in from main.go to avoid an import cycle
// (auth → middleware → auth). Their types are the standard Chi/stdlib middleware signature.
func Routes(h *Handler, magicLinkRL func(http.Handler) http.Handler, authenticate func(http.Handler) http.Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/login", h.Login)
		r.With(magicLinkRL).Post("/magic-link", h.MagicLink)
		r.With(magicLinkRL).Get("/login/verify", h.LoginVerify)
		r.Post("/exchange", h.Exchange)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)
		r.Get("/csrf", h.CSRF)
		r.With(authenticate).Get("/profile", h.Profile)
		r.With(authenticate).Post("/set-password", h.SetPassword)
	}
}
