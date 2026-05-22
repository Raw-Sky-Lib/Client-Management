package cms

import "github.com/go-chi/chi/v5"

func Routes(h *Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Put("/pages/{slug}/sections", h.UpdateSections)
		r.Put("/pages/{slug}/visibility", h.UpdateVisibility)
	}
}
