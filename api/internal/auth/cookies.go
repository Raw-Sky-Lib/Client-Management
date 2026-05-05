package auth

import (
	"net/http"
	"time"
)

func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string, secure bool, refreshExp time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   900, // 15 minutes
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refreshToken,
		Path:     "/api/auth/refresh",
		MaxAge:   int(refreshExp.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteStrictMode,
	})
}

func ClearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/api/auth/refresh",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
