package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"

	"github.com/DagMT/client-portal/internal/utils"
)

const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
)

func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}

		cookie, err := r.Cookie(CSRFCookieName)
		if err != nil || cookie.Value == "" {
			utils.RespondError(w, http.StatusForbidden, "missing CSRF token")
			return
		}

		if r.Header.Get(CSRFHeaderName) != cookie.Value {
			utils.RespondError(w, http.StatusForbidden, "invalid CSRF token")
			return
		}

		next.ServeHTTP(w, r)
	})
}
