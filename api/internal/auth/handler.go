package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/DagMT/client-portal/internal/utils"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
	secure   bool
}

func NewHandler(svc *Service, secure bool) *Handler {
	return &Handler{
		svc:      svc,
		validate: validator.New(),
		secure:   secure,
	}
}

// MagicLink handles POST /api/auth/magic-link
//
// @Summary     Request a magic link
// @Description Generates a Supabase magic link and delivers it via email. Always returns 200 to prevent email enumeration.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token from GET /api/auth/csrf (double-submit cookie pattern)"
// @Param       body         body   MagicLinkRequest true "Email address"
// @Success     200 {object} utils.MessageResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     403 {object} utils.ErrorResponse "Missing or invalid CSRF token"
// @Failure     429 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/auth/magic-link [post]
func (h *Handler) MagicLink(w http.ResponseWriter, r *http.Request) {
	var req MagicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	// RequestMagicLink swallows "email not registered" to prevent enumeration.
	if err := h.svc.RequestMagicLink(r.Context(), req.Email); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong, please try again")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "If that email is registered, you'll receive a sign-in link shortly.",
	})
}

// Exchange handles POST /api/auth/exchange
//
// @Summary     Exchange Supabase token for portal JWT
// @Description Verifies the Supabase access token from the magic link callback, sets portal JWT cookies (access_token + refresh_token), and returns ok.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token from GET /api/auth/csrf (double-submit cookie pattern)"
// @Param       body         body   ExchangeRequest true "Supabase access token from /auth/callback fragment"
// @Success     200 {object} utils.OKResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Failure     403 {object} utils.ErrorResponse "Missing or invalid CSRF token"
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/auth/exchange [post]
func (h *Handler) Exchange(w http.ResponseWriter, r *http.Request) {
	var req ExchangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "access_token is required")
		return
	}
	claims, err := h.svc.ExchangeToken(r.Context(), req.AccessToken)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// Refresh handles POST /api/auth/refresh
//
// @Summary     Refresh access token
// @Description Uses the refresh_token HTTP-only cookie to issue a new access_token cookie. Requires the refresh_token cookie to be present.
// @Tags        auth
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token from GET /api/auth/csrf (double-submit cookie pattern)"
// @Success     200 {object} utils.OKResponse
// @Failure     401 {object} utils.ErrorResponse "Missing or expired refresh_token cookie"
// @Failure     403 {object} utils.ErrorResponse "Missing or invalid CSRF token"
// @Router      /api/auth/refresh [post]
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		utils.RespondError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}
	accessToken, err := h.svc.RefreshAccessToken(r.Context(), cookie.Value)
	if err != nil {
		utils.RespondError(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "access_token",
		Value:    accessToken,
		Path:     "/",
		MaxAge:   900,
		HttpOnly: true,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// Logout handles POST /api/auth/logout
//
// @Summary     Logout
// @Description Clears the portal JWT cookies (access_token and refresh_token).
// @Tags        auth
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token from GET /api/auth/csrf (double-submit cookie pattern)"
// @Success     200 {object} utils.OKResponse
// @Failure     403 {object} utils.ErrorResponse "Missing or invalid CSRF token"
// @Router      /api/auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.svc.Logout(w)
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// CSRF handles GET /api/auth/csrf
//
// @Summary     Get CSRF token
// @Description Generates a CSRF token, sets it as a readable cookie (double-submit pattern), and returns it. Call this before any state-changing browser request.
// @Tags        auth
// @Produce     json
// @Success     200 {object} CSRFResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/auth/csrf [get]
func (h *Handler) CSRF(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not generate CSRF token")
		return
	}
	token := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false, // must be readable by JS for the double-submit pattern
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
	utils.RespondJSON(w, http.StatusOK, CSRFResponse{CSRFToken: token})
}

// Profile handles GET /api/auth/profile
//
// @Summary     Get current user profile
// @Description Returns the authenticated user's non-sensitive claims, including the tenant Supabase config needed to initialize the frontend Supabase client.
// @Tags        auth
// @Produce     json
// @Success     200 {object} ProfileResponse
// @Failure     401 {object} utils.ErrorResponse
// @Router      /api/auth/profile [get]
// @Security    CookieAuth
func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	utils.RespondJSON(w, http.StatusOK, ProfileResponse{
		UserID:          claims.UserID,
		TenantID:        claims.TenantID,
		Email:           claims.Email,
		SupabaseURL:     claims.ClientSupabaseURL,
		SupabaseAnonKey: claims.ClientSupabaseAnonKey,
	})
}
