package cms

import (
	"encoding/json"
	"net/http"

	"github.com/DagMT/client-portal/internal/revalidate"
	"github.com/DagMT/client-portal/internal/tenant"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc        *Service
	revalidate *revalidate.Service
}

func NewHandler(svc *Service, revalidateSvc *revalidate.Service) *Handler {
	return &Handler{svc: svc, revalidate: revalidateSvc}
}

// UpdateSections handles PUT /api/cms/pages/{slug}/sections
func (h *Handler) UpdateSections(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if !isValidSlug(slug) {
		utils.RespondError(w, http.StatusBadRequest, "invalid slug")
		return
	}

	var req UpdateSectionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !json.Valid(req.Sections) {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.UpdatePageSections(r.Context(), cfg, slug, req.Sections); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "failed to update page sections")
		return
	}

	pagePath := "/"
	if slug != "home" {
		pagePath = "/" + slug
	}
	h.revalidate.TriggerISR(cfg, []string{"/", pagePath})

	utils.RespondJSON(w, http.StatusOK, utils.OKResponse{OK: true})
}

// UpdateVisibility handles PUT /api/cms/pages/{slug}/visibility
func (h *Handler) UpdateVisibility(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	slug := chi.URLParam(r, "slug")
	if !isValidSlug(slug) {
		utils.RespondError(w, http.StatusBadRequest, "invalid slug")
		return
	}

	var req UpdateVisibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.UpdatePageVisibility(r.Context(), cfg, slug, req.IsPublished); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "failed to update page visibility")
		return
	}

	pagePath := "/"
	if slug != "home" {
		pagePath = "/" + slug
	}
	h.revalidate.TriggerISR(cfg, []string{"/", pagePath})

	utils.RespondJSON(w, http.StatusOK, utils.OKResponse{OK: true})
}

// isValidSlug allows lowercase letters, digits, hyphens, and underscores only.
func isValidSlug(s string) bool {
	if s == "" || len(s) > 100 {
		return false
	}
	for _, c := range s {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}
