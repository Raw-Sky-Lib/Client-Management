package media

import "github.com/go-chi/chi/v5"

func Routes(h *Handler) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/init-bucket", h.InitBucket)
		r.Get("/files", h.ListFiles)
		r.Post("/upload", h.Upload)
		r.Delete("/file", h.DeleteFile)
	}
}
