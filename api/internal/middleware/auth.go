package middleware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("access_token")
			if err != nil || cookie.Value == "" {
				utils.RespondError(w, http.StatusUnauthorized, "authentication required")
				return
			}

			claims := &auth.PortalClaims{}
			_, err = jwt.ParseWithClaims(cookie.Value, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if err != nil {
				utils.RespondError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			next.ServeHTTP(w, r.WithContext(auth.WithClaims(r.Context(), claims)))
		})
	}
}

// ClaimsFromContext extracts PortalClaims injected by Authenticate middleware.
func ClaimsFromContext(ctx context.Context) (*auth.PortalClaims, bool) {
	return auth.ClaimsFromContext(ctx)
}
