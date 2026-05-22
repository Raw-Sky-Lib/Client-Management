package onboarding

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc         *Service
	validate    *validator.Validate
	agencyToken string
	frontendURL string
	jwtIssuer   auth.JWTIssuer
}

func NewHandler(svc *Service, agencyToken, frontendURL string, jwtIssuer auth.JWTIssuer) *Handler {
	return &Handler{
		svc:         svc,
		validate:    validator.New(),
		agencyToken: agencyToken,
		frontendURL: frontendURL,
		jwtIssuer:   jwtIssuer,
	}
}

// Confirm handles GET /api/onboarding/confirm
//
// @Summary     Confirm email and complete onboarding
// @Description Verifies the emailed token, creates the Supabase user, sets portal JWT cookies, and redirects to the dashboard.
// @Tags        onboarding
// @Produce     json
// @Param       token query string true "Email confirmation token"
// @Success     307 "Redirect to /dashboard"
// @Failure     400 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/onboarding/confirm [get]
func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=invalid", http.StatusTemporaryRedirect)
		return
	}
	claims, err := h.svc.Confirm(r.Context(), token)
	if err != nil {
		reason := "error"
		switch {
		case errors.Is(err, ErrLinkUsed):
			reason = "used"
		case errors.Is(err, ErrLinkExpired):
			reason = "expired"
		case errors.Is(err, ErrLinkInvalid):
			reason = "invalid"
		}
		http.Redirect(w, r, h.frontendURL+"/link-error?reason="+reason, http.StatusTemporaryRedirect)
		return
	}
	if err := h.jwtIssuer.IssueTokenPair(w, claims); err != nil {
		http.Redirect(w, r, h.frontendURL+"/link-error?reason=error", http.StatusTemporaryRedirect)
		return
	}
	http.Redirect(w, r, h.frontendURL+"/welcome", http.StatusTemporaryRedirect)
}

// SendInvite handles POST /api/admin/send-invite
//
// @Summary     Send portal invite to a client
// @Description Creates the tenant record if needed and sends a magic-link invite. No Supabase credentials required. Called by agency-hub from the client page.
// @Tags        admin
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "Bearer {AGENCY_MANAGEMENT_TOKEN}"
// @Param       body          body   ResendInviteRequest true "Client ID and email"
// @Success     200 {object} map[string]bool
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/admin/send-invite [post]
func (h *Handler) SendInvite(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || authHeader[7:] != h.agencyToken {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req ResendInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.SendClientInvite(r.Context(), req.ClientID, req.Email); err != nil {
		slog.Error("send invite failed", "error", err)
		utils.RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

// ResendInvite handles POST /api/admin/resend-invite
//
// @Summary     Resend onboarding invite email
// @Description Generates a fresh invite token and resends the onboarding email. Called by agency-hub when a client reports a broken or expired link.
// @Tags        admin
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "Bearer {AGENCY_MANAGEMENT_TOKEN}"
// @Param       body          body   ResendInviteRequest true "Client ID and email"
// @Success     200 {object} map[string]bool
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Failure     404 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/admin/resend-invite [post]
func (h *Handler) ResendInvite(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || authHeader[7:] != h.agencyToken {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req ResendInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.ResendInvite(r.Context(), req.ClientID, req.Email); err != nil {
		if errors.Is(err, ErrClientNotSetup) {
			utils.RespondError(w, http.StatusNotFound, "client not registered in portal")
		} else {
			slog.Error("resend invite failed", "error", err)
			utils.RespondError(w, http.StatusInternalServerError, "failed to resend invite")
		}
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"sent": true})
}

// RegisterClient handles POST /api/admin/register-client
//
// @Summary     Register a client tenant
// @Description Called by agency-hub to register a client's Supabase project. Validates credentials, runs CMS migrations, and encrypts secrets. No CSRF required — machine-to-machine.
// @Tags        admin
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "Bearer {AGENCY_MANAGEMENT_TOKEN}"
// @Param       body          body   RegisterClientRequest true "Client registration payload"
// @Success     201 {object} RegisteredResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/admin/register-client [post]
func (h *Handler) RegisterClient(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") || auth[7:] != h.agencyToken {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	var req RegisterClientRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.svc.RegisterClient(r.Context(), req); err != nil {
		slog.Error("register client failed", "error", err)
		switch {
		case errors.Is(err, ErrBadSupabaseCredentials):
			utils.RespondError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrBadDBURL):
			utils.RespondError(w, http.StatusBadRequest, err.Error())
		default:
			utils.RespondError(w, http.StatusInternalServerError, "registration failed")
		}
		return
	}
	utils.RespondJSON(w, http.StatusCreated, map[string]bool{"registered": true})
}

// DeregisterClient handles DELETE /api/admin/deregister-client/{client_id}
//
// @Summary     Deregister a client tenant
// @Description Deletes all portal data for a client (tenant, users, projects, tokens). Called by agency-hub when a client is permanently deleted.
// @Tags        admin
// @Produce     json
// @Param       Authorization header string true "Bearer {AGENCY_MANAGEMENT_TOKEN}"
// @Param       client_id     path   string true "Client ID"
// @Success     200 {object} map[string]bool
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/admin/deregister-client/{client_id} [delete]
func (h *Handler) DeregisterClient(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || authHeader[7:] != h.agencyToken {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	clientID := chi.URLParam(r, "client_id")
	if clientID == "" {
		utils.RespondError(w, http.StatusBadRequest, "client_id required")
		return
	}

	if err := h.svc.DeregisterClient(r.Context(), clientID); err != nil {
		slog.Error("deregister client failed", "error", err, "client_id", clientID)
		// 200 even if not found — idempotent; portal may never have had this client
		utils.RespondJSON(w, http.StatusOK, map[string]bool{"deregistered": true})
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"deregistered": true})
}

// UpdateClientEmail handles PATCH /api/admin/update-client/{client_id}/email
//
// @Summary     Update a client's email in the portal
// @Description Replaces the email in tenant_users so magic-link and password-reset emails go to the correct address. Called by agency-hub when a client's email is changed.
// @Tags        admin
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "Bearer {PORTAL_ADMIN_SECRET}"
// @Param       client_id     path   string true "Client ID (tenant_id)"
// @Param       body          body   object true "{\"email\": \"new@example.com\"}"
// @Success     200 {object} map[string]bool
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Router      /api/admin/update-client/{client_id}/email [patch]
func (h *Handler) UpdateClientEmail(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || authHeader[7:] != h.agencyToken {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	clientID := chi.URLParam(r, "client_id")
	if clientID == "" {
		utils.RespondError(w, http.StatusBadRequest, "client_id required")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" {
		utils.RespondError(w, http.StatusBadRequest, "email required")
		return
	}

	if err := h.svc.UpdateClientEmail(r.Context(), clientID, body.Email); err != nil {
		slog.Error("update client email failed", "error", err, "client_id", clientID)
		utils.RespondError(w, http.StatusInternalServerError, "could not update email")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"updated": true})
}

// DeregisterProject handles DELETE /api/admin/deregister-project/{project_id}
//
// @Summary     Deregister a single client project
// @Description Removes the tenant_projects row and its stored Supabase credentials. Called by agency-hub when a project is deleted. Idempotent — returns 200 even if the project was never registered.
// @Tags        admin
// @Produce     json
// @Param       Authorization header string true "Bearer {PORTAL_ADMIN_SECRET}"
// @Param       project_id    path   string true "Agency project ID"
// @Success     200 {object} map[string]bool
// @Failure     401 {object} utils.ErrorResponse
// @Router      /api/admin/deregister-project/{project_id} [delete]
func (h *Handler) DeregisterProject(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || authHeader[7:] != h.agencyToken {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	projectID := chi.URLParam(r, "project_id")
	if projectID == "" {
		utils.RespondError(w, http.StatusBadRequest, "project_id required")
		return
	}

	if err := h.svc.DeregisterProject(r.Context(), projectID); err != nil {
		slog.Error("deregister project failed", "error", err, "project_id", projectID)
	}
	// Always 200 — idempotent; portal may never have had this project
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"deregistered": true})
}

// SyncProjectCredentials handles PATCH /api/admin/projects/{agency_project_id}/credentials
//
// @Summary     Sync credentials for an already-registered project
// @Description Updates stored Supabase credentials without running migrations or sending emails.
// @Description Only non-empty fields overwrite existing values. Returns 404 if the project
// @Description has never been registered — use POST /api/admin/register-client for first-time setup.
// @Tags        admin
// @Accept      json
// @Produce     json
// @Param       Authorization      header string true  "Bearer {PORTAL_ADMIN_SECRET}"
// @Param       agency_project_id  path   string true  "Agency project ID"
// @Param       body               body   SyncCredentialsRequest false "Fields to update (all optional)"
// @Success     200 {object} map[string]bool
// @Failure     401 {object} utils.ErrorResponse
// @Failure     404 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/admin/projects/{agency_project_id}/credentials [patch]
func (h *Handler) SyncProjectCredentials(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") || authHeader[7:] != h.agencyToken {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	agencyProjectID := chi.URLParam(r, "agency_project_id")
	if agencyProjectID == "" {
		utils.RespondError(w, http.StatusBadRequest, "agency_project_id required")
		return
	}
	var req SyncCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.AgencyProjectID = agencyProjectID
	if err := h.svc.SyncCredentials(r.Context(), req); err != nil {
		if errors.Is(err, ErrProjectNotFound) {
			utils.RespondError(w, http.StatusNotFound, "project not registered in portal")
			return
		}
		slog.Error("sync project credentials failed", "error", err, "agency_project_id", agencyProjectID)
		utils.RespondError(w, http.StatusInternalServerError, "credential sync failed")
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]bool{"synced": true})
}

var userFacingErrs = []error{
	ErrClientNotSetup,
	ErrLinkInvalid, ErrLinkUsed, ErrLinkExpired,
}

func isUserFacingErr(err error) bool {
	for _, e := range userFacingErrs {
		if errors.Is(err, e) {
			return true
		}
	}
	return false
}
