package portalproject

import "github.com/go-chi/chi/v5"

func Routes(h *Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Get("/", h.ListProjects)
	}
}
