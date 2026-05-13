package revalidate

import (
	"encoding/json"
	"net/http"

	"github.com/DagMT/client-portal/internal/tenant"
	"github.com/DagMT/client-portal/internal/utils"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Trigger handles POST /api/revalidate
//
// @Summary     Trigger ISR revalidation
// @Description Fires a non-blocking revalidation request to the client's live site.
// @Tags        revalidate
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token"
// @Param       body body object true "Paths to revalidate"
// @Success     200 {object} map[string]bool
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Router      /api/revalidate [post]
// @Security    CookieAuth
func (h *Handler) Trigger(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Paths) == 0 {
		utils.RespondError(w, http.StatusBadRequest, "paths required")
		return
	}

	h.svc.TriggerISR(cfg, req.Paths)
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"triggered": true})
}
