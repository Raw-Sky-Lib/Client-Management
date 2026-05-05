package onboarding

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/utils"
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

// Connect handles POST /api/onboarding/connect
//
// @Summary     Request email confirmation
// @Description Validates the connection token against agency-hub and sends a confirmation link to the client's email.
// @Tags        onboarding
// @Accept      json
// @Produce     json
// @Param       body body ConnectRequest true "Connection request"
// @Success     200 {object} utils.MessageResponse
// @Failure     400 {object} utils.ErrorResponse
// @Failure     429 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/onboarding/connect [post]
func (h *Handler) Connect(w http.ResponseWriter, r *http.Request) {
	var req ConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "connection_token and a valid email are required")
		return
	}
	if err := h.svc.Connect(r.Context(), req); err != nil {
		if isUserFacingErr(err) {
			utils.RespondError(w, http.StatusBadRequest, err.Error())
		} else {
			utils.RespondError(w, http.StatusInternalServerError, "something went wrong, please try again")
		}
		return
	}
	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"message": "Check your email for a confirmation link.",
	})
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
		utils.RespondError(w, http.StatusBadRequest, "missing token")
		return
	}
	claims, err := h.svc.Confirm(r.Context(), token)
	if err != nil {
		if isUserFacingErr(err) {
			utils.RespondError(w, http.StatusBadRequest, err.Error())
		} else {
			utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		}
		return
	}
	if err := h.jwtIssuer.IssueTokenPair(w, claims); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "something went wrong")
		return
	}
	http.Redirect(w, r, h.frontendURL+"/dashboard", http.StatusTemporaryRedirect)
}

// RegisterClient handles POST /api/admin/register-client
//
// @Summary     Register a client tenant
// @Description Called by agency-hub to register a client's Supabase project. Validates credentials, runs CMS migrations, and encrypts secrets.
// @Tags        admin
// @Accept      json
// @Produce     json
// @Param       Authorization header string true "Bearer {AGENCY_MANAGEMENT_TOKEN}"
// @Param       body body RegisterClientRequest true "Client registration payload"
// @Success     201 {object} utils.OKResponse
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
		utils.RespondError(w, http.StatusInternalServerError, "registration failed")
		return
	}
	utils.RespondJSON(w, http.StatusCreated, map[string]bool{"registered": true})
}

var userFacingErrs = []error{
	ErrTokenExpired, ErrTokenUsed, ErrTokenInvalid, ErrClientNotSetup,
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
