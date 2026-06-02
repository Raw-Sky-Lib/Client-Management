package portalproject

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/utils"
)

type repoIface interface {
	GetProjectsForTenant(ctx context.Context, tenantID string) ([]rawProject, error)
}

type Handler struct {
	repo   repoIface
	encKey []byte
}

func NewHandler(repo *Repository, encKey []byte) *Handler {
	return &Handler{repo: repo, encKey: encKey}
}

// ListProjects handles GET /api/projects
func (h *Handler) ListProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	raw, err := h.repo.GetProjectsForTenant(r.Context(), claims.TenantID)
	if err != nil {
		slog.Error("portalproject: list-projects: db query failed",
			slog.String("tenant_id", claims.TenantID),
			slog.String("error", err.Error()),
		)
		utils.RespondError(w, http.StatusInternalServerError, "failed to load projects")
		return
	}

	entries := make([]ProjectEntry, 0, len(raw))
	for _, p := range raw {
		supabaseURL, err := utils.DecryptString(p.URLEnc, h.encKey)
		if err != nil {
			slog.Error("portalproject: list-projects: decrypt url failed",
				slog.String("tenant_id", claims.TenantID),
				slog.String("project_id", p.ID),
				slog.String("error", err.Error()),
			)
			continue
		}
		anonKey, err := utils.DecryptString(p.AnonEnc, h.encKey)
		if err != nil {
			slog.Error("portalproject: list-projects: decrypt anon key failed",
				slog.String("tenant_id", claims.TenantID),
				slog.String("project_id", p.ID),
				slog.String("error", err.Error()),
			)
			continue
		}
		entries = append(entries, ProjectEntry{
			ID:              p.ID,
			AgencyProjectID: p.AgencyProjectID,
			Name:            p.Name,
			SiteURL:         p.SiteURL,
			SupabaseURL:     supabaseURL,
			SupabaseAnonKey: anonKey,
		})
	}

	utils.RespondJSON(w, http.StatusOK, ListProjectsResponse{Projects: entries})
}
