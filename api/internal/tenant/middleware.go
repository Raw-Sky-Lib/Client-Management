package tenant

import (
	"log/slog"
	"net/http"

	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/utils"
)

// ResolveTenant reads the tenant_id from the JWT claims (set by Authenticate middleware),
// decrypts the full Supabase config from the portal DB, and injects it into the request context.
// Must be applied after Authenticate.
func ResolveTenant(svc *Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				utils.RespondError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			cfg, err := svc.Resolve(r.Context(), claims.TenantID)
			if err != nil {
				slog.Error("ResolveTenant failed",
					slog.String("tenant_id", claims.TenantID),
					slog.String("error", err.Error()),
				)
				utils.RespondError(w, http.StatusInternalServerError, "could not load tenant config")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithConfig(r.Context(), cfg)))
		})
	}
}
