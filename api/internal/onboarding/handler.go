package onboarding

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/DagMT/client-portal/internal/utils"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc          *Service
	validate     *validator.Validate
	agencyToken  string
	frontendURL  string
}

func NewHandler(svc *Service, agencyToken, frontendURL string) *Handler {
	return &Handler{
		svc:         svc,
		validate:    validator.New(),
		agencyToken: agencyToken,
		frontendURL: frontendURL,
	}
}

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
	// JWT issuance wired in CLI-10 — claims returned and ready
	_ = claims
	http.Redirect(w, r, h.frontendURL+"/dashboard", http.StatusTemporaryRedirect)
}

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
