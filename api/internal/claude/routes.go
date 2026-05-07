package claude

import "github.com/go-chi/chi/v5"

func Routes(h *Handler) func(r chi.Router) {
	return func(r chi.Router) {
		r.Post("/generate", h.Generate)
	}
}
