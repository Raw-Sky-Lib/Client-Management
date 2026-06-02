package revalidate

import (
	"encoding/json"
	"log/slog"
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

	if cfg.SiteURL == "" {
		slog.Warn("revalidate: trigger: no site_url configured", slog.String("tenant_id", cfg.TenantID))
		utils.RespondError(w, http.StatusUnprocessableEntity, "no site URL configured for this tenant")
		return
	}

	slog.Info("revalidate: trigger", slog.String("tenant_id", cfg.TenantID), slog.Any("paths", req.Paths))
	h.svc.TriggerISR(cfg, req.Paths)
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"triggered": true})
}
