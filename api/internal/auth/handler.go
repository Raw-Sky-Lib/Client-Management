package auth

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
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
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			utils.RespondError(w, http.StatusUnauthorized, "incorrect email or password")
		case errors.Is(err, ErrSupabaseDNSFailure):
			slog.Warn("auth: login: supabase project not found (DNS)", slog.String("email", req.Email))
			utils.RespondErrorWithCode(w, http.StatusServiceUnavailable, "supabase_dns_failure",
				"We can't reach your workspace. This usually means your portal is paused or hasn't finished setup — your administrator needs to look at it.")
		case errors.Is(err, ErrSupabaseUnreachable):
			slog.Warn("auth: login: supabase unreachable", slog.String("email", req.Email), slog.String("error", err.Error()))
			utils.RespondErrorWithCode(w, http.StatusServiceUnavailable, "supabase_unreachable",
				"Password sign-in is temporarily unavailable. Try the magic-link option below — it works even when your workspace data is briefly offline.")
		default:
			slog.Error("auth: login: unexpected error", slog.String("email", req.Email), slog.String("error", err.Error()))
			utils.RespondError(w, http.StatusInternalServerError, "something went wrong, please try again")
		}
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		slog.Error("auth: login: issue token pair", slog.String("tenant_id", claims.TenantID), slog.String("error", err.Error()))
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	slog.Info("auth: login: ok", slog.String("tenant_id", claims.TenantID), slog.String("email", claims.Email))
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// SetPassword handles POST /api/auth/set-password (authenticated)
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
	if err := h.svc.SetUserPassword(r.Context(), claims.TenantID, claims.Email, req.Password); err != nil {
		switch {
		case errors.Is(err, ErrPortalNotReady):
			utils.RespondError(w, http.StatusUnprocessableEntity, "no project configured yet")
		case errors.Is(err, ErrSupabaseDNSFailure):
			slog.Warn("auth: set-password: supabase project not found (DNS)", slog.String("tenant_id", claims.TenantID))
			utils.RespondErrorWithCode(w, http.StatusServiceUnavailable, "supabase_dns_failure",
				"We can't reach your workspace right now. Your administrator needs to check the portal configuration.")
		case errors.Is(err, ErrSupabaseUnreachable):
			slog.Warn("auth: set-password: supabase unreachable", slog.String("tenant_id", claims.TenantID))
			utils.RespondErrorWithCode(w, http.StatusServiceUnavailable, "supabase_unreachable",
				"Couldn't set your password right now — your workspace data is briefly offline. Try again in a moment.")
		default:
			slog.Error("auth: set-password: unexpected error", slog.String("tenant_id", claims.TenantID), slog.String("error", err.Error()))
			utils.RespondError(w, http.StatusInternalServerError, "could not set password, please try again")
		}
		return
	}
	slog.Info("auth: set-password: ok", slog.String("tenant_id", claims.TenantID))
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// MagicLink handles POST /api/auth/magic-link
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
	if err := h.svc.RequestMagicLink(r.Context(), req.Email); err != nil {
		slog.Error("auth: magic-link: send failed", slog.String("email", req.Email), slog.String("error", err.Error()))
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong, please try again")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "If that email is registered, you'll receive a sign-in link shortly.",
	})
}

// Exchange handles POST /api/auth/exchange
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
		switch {
		case errors.Is(err, ErrInvalidToken):
			utils.RespondError(w, http.StatusUnauthorized, "invalid or expired token")
		default:
			slog.Error("auth: exchange: unexpected error", slog.String("error", err.Error()))
			utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		slog.Error("auth: exchange: issue token pair", slog.String("tenant_id", claims.TenantID), slog.String("error", err.Error()))
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	slog.Info("auth: exchange: ok", slog.String("tenant_id", claims.TenantID))
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// Refresh handles POST /api/auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		utils.RespondError(w, http.StatusUnauthorized, "missing refresh token")
		return
	}
	accessToken, err := h.svc.RefreshAccessToken(r.Context(), cookie.Value)
	if err != nil {
		// Expired refresh token is normal (not an error worth logging).
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
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	h.svc.Logout(w)
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// CSRF handles GET /api/auth/csrf
func (h *Handler) CSRF(w http.ResponseWriter, r *http.Request) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		slog.Error("auth: csrf: rand.Read failed", slog.String("error", err.Error()))
		utils.RespondError(w, http.StatusInternalServerError, "could not generate CSRF token")
		return
	}
	token := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   h.secure,
		SameSite: http.SameSiteStrictMode,
	})
	utils.RespondJSON(w, http.StatusOK, CSRFResponse{CSRFToken: token})
}

// ResetPasswordRequest handles POST /api/auth/reset-password/request
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
		slog.Error("auth: reset-password-request: send failed", slog.String("email", req.Email), slog.String("error", err.Error()))
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong, please try again")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "If that email is registered, you'll receive a reset link shortly.",
	})
}

// ResetPasswordVerify handles GET /api/auth/reset-password/verify
func (h *Handler) ResetPasswordVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	valid, err := h.svc.ValidateResetToken(r.Context(), token)
	if err != nil {
		slog.Error("auth: reset-password-verify: validate failed", slog.String("error", err.Error()))
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	if !valid {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, h.frontendURL+"/reset-password?token="+token, http.StatusTemporaryRedirect)
}

// ResetPasswordConfirm handles POST /api/auth/reset-password/confirm
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
		switch {
		case errors.Is(err, ErrInvalidToken):
			utils.RespondError(w, http.StatusUnauthorized, "this reset link is invalid or has expired")
		case errors.Is(err, ErrPortalNotReady):
			utils.RespondError(w, http.StatusUnprocessableEntity, "your portal isn't fully set up yet — please use the sign-in link from your invitation email")
		case errors.Is(err, ErrSupabaseDNSFailure):
			slog.Warn("auth: reset-password-confirm: supabase project not found (DNS)")
			utils.RespondErrorWithCode(w, http.StatusServiceUnavailable, "supabase_dns_failure",
				"We can't reach your workspace. Your administrator needs to check the portal configuration before you can reset your password.")
		case errors.Is(err, ErrSupabaseUnreachable):
			slog.Warn("auth: reset-password-confirm: supabase unreachable")
			utils.RespondErrorWithCode(w, http.StatusServiceUnavailable, "supabase_unreachable",
				"Password reset is temporarily unavailable. Try the magic-link sign-in from the login page instead.")
		default:
			slog.Error("auth: reset-password-confirm: unexpected error", slog.String("error", err.Error()))
			utils.RespondError(w, http.StatusInternalServerError, "could not reset password, please try again")
		}
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		slog.Error("auth: reset-password-confirm: issue token pair", slog.String("tenant_id", claims.TenantID), slog.String("error", err.Error()))
		utils.RespondError(w, http.StatusInternalServerError, "password updated but could not sign you in, please sign in manually")
		return
	}
	slog.Info("auth: reset-password-confirm: ok", slog.String("tenant_id", claims.TenantID))
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// LoginVerify handles GET /api/auth/login/verify
func (h *Handler) LoginVerify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	claims, err := h.svc.VerifyLoginToken(r.Context(), token)
	if err != nil {
		if !errors.Is(err, ErrInvalidToken) {
			slog.Error("auth: login-verify: unexpected error", slog.String("error", err.Error()))
		}
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	if err := h.svc.IssueTokenPair(w, claims); err != nil {
		slog.Error("auth: login-verify: issue token pair", slog.String("tenant_id", claims.TenantID), slog.String("error", err.Error()))
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=error", http.StatusTemporaryRedirect)
		return
	}
	slog.Info("auth: login-verify: ok", slog.String("tenant_id", claims.TenantID))
	http.Redirect(w, r, h.frontendURL+"/dashboard", http.StatusTemporaryRedirect)
}

// Profile handles GET /api/auth/profile
func (h *Handler) Profile(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	utils.RespondJSON(w, http.StatusOK, ProfileResponse{
		UserID:   claims.UserID,
		TenantID: claims.TenantID,
		Email:    claims.Email,
	})
}
