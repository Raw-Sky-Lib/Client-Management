package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DagMT/client-portal/internal/utils"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc         *Service
	validate    *validator.Validate
	secure      bool
	frontendURL string
}

func NewHandler(svc *Service, secure bool, frontendURL string) *Handler {
	return &Handler{
		svc:         svc,
		validate:    validator.New(),
		secure:      secure,
		frontendURL: frontendURL,
	}
}

// Login handles POST /api/auth/login
//
// @Summary     Password login
// @Description Verifies email and password against the client's Supabase project, then sets portal JWT cookies.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token"
// @Param       body         body   PasswordLoginRequest true "Credentials"
// @Success     200 {object} utils.OKResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse "Wrong email or password"
// @Failure     403 {object} utils.ErrorResponse "Missing or invalid CSRF token"
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/auth/login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req PasswordLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	claims, err := h.svc.LoginWithPassword(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			utils.RespondError(w, http.StatusUnauthorized, "incorrect email or password")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong, please try again")
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// SetPassword handles POST /api/auth/set-password (authenticated)
//
// @Summary     Set user password
// @Description Sets the authenticated user's password in their Supabase project. Called from the welcome page after first onboarding.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token"
// @Param       body         body   SetPasswordRequest true "New password (min 8 chars)"
// @Success     200 {object} utils.OKResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/auth/set-password [post]
func (h *Handler) SetPassword(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req SetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if err := h.svc.SetUserPassword(r.Context(), claims.TenantID, claims.UserID, req.Password); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "could not set password, please try again")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
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

// ResetPasswordRequest handles POST /api/auth/reset-password/request
//
// @Summary     Request a password reset link
// @Description Generates a reset token and emails a link. Always returns 200 to prevent email enumeration.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token"
// @Param       body         body   MagicLinkRequest true "Email address"
// @Success     200 {object} utils.MessageResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     403 {object} utils.ErrorResponse
// @Router      /api/auth/reset-password/request [post]
func (h *Handler) ResetPasswordRequest(w http.ResponseWriter, r *http.Request) {
	var req MagicLinkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "a valid email is required")
		return
	}
	if err := h.svc.RequestPasswordReset(r.Context(), req.Email); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong, please try again")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "If that email is registered, you'll receive a reset link shortly.",
	})
}

// ResetPasswordVerify handles GET /api/auth/reset-password/verify
//
// @Summary     Validate a reset token and redirect to the reset page
// @Description Called when the user clicks the password reset email link. Validates the token and redirects to the frontend reset page with the token as a query param.
// @Tags        auth
// @Param       token query string true "Reset token"
// @Success     307 "Redirect to /reset-password?token=..."
// @Failure     307 "Redirect to /link-error on invalid/expired token"
// @Router      /api/auth/reset-password/verify [get]
func (h *Handler) ResetPasswordVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	valid, err := h.svc.ValidateResetToken(r.Context(), token)
	if err != nil || !valid {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, h.frontendURL+"/reset-password?token="+token, http.StatusTemporaryRedirect)
}

// ResetPasswordConfirm handles POST /api/auth/reset-password/confirm
//
// @Summary     Confirm password reset
// @Description Validates the reset token, sets the new password in Supabase, and issues portal JWT cookies so the user is immediately signed in.
// @Tags        auth
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token"
// @Param       body         body   ResetPasswordConfirmRequest true "Token and new password"
// @Success     200 {object} utils.OKResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse "Invalid or expired token"
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/auth/reset-password/confirm [post]
func (h *Handler) ResetPasswordConfirm(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "token and a password of at least 8 characters are required")
		return
	}
	claims, err := h.svc.ConfirmPasswordReset(r.Context(), req.Token, req.Password)
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			utils.RespondError(w, http.StatusUnauthorized, "this reset link is invalid or has expired")
			return
		}
		utils.RespondError(w, http.StatusInternalServerError, "could not reset password, please try again")
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// LoginVerify handles GET /api/auth/login/verify
//
// @Summary     Verify portal magic-link token
// @Description Called when the user clicks the sign-in link from their email. Validates the token, sets portal JWT cookies, and redirects to the dashboard.
// @Tags        auth
// @Param       token query string true "Portal magic-link token"
// @Success     307 "Redirect to /dashboard"
// @Failure     307 "Redirect to /link-error on invalid/expired token"
// @Router      /api/auth/login/verify [get]
func (h *Handler) LoginVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	claims, err := h.svc.VerifyLoginToken(r.Context(), token)
	if err != nil {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=error", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, h.frontendURL+"/dashboard", http.StatusTemporaryRedirect)
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
