package claude

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/DagMT/client-portal/internal/tenant"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/go-playground/validator/v10"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

// Generate handles POST /api/assistant/generate.
//
// @Summary     Generate content suggestions
// @Description Uses Claude to suggest field changes for a page section. Returns a diff preview — changes are not applied automatically.
// @Tags        assistant
// @Accept      json
// @Produce     json
// @Param       X-CSRF-Token header string true "CSRF token from GET /api/auth/csrf (double-submit cookie pattern)"
// @Param       body         body   GenerateRequest true "Generate request"
// @Success     200  {object} GenerateResponse
// @Failure     400  {object} utils.ErrorResponse
// @Failure     403  {object} utils.ErrorResponse "Missing or invalid CSRF token"
// @Failure     429  {object} utils.ErrorResponse
// @Failure     500  {object} utils.ErrorResponse
// @Router      /api/assistant/generate [post]
// @Security    CookieAuth
func (h *Handler) Generate(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		utils.RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.svc.Generate(r.Context(), cfg, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrMinuteLimitExceeded):
			utils.RespondError(w, http.StatusTooManyRequests, "You're making requests too quickly. Please wait a moment.")
		case errors.Is(err, ErrHourLimitExceeded):
			utils.RespondError(w, http.StatusTooManyRequests, "Hourly limit reached. The assistant will be available again soon.")
		case errors.Is(err, ErrBudgetExceeded):
			utils.RespondError(w, http.StatusTooManyRequests, "Your monthly content assistant limit has been reached. Your website team will be in touch.")
		case errors.Is(err, ErrPageNotFound), errors.Is(err, ErrSectionNotFound):
			utils.RespondError(w, http.StatusBadRequest, err.Error())
		default:
			utils.RespondError(w, http.StatusInternalServerError, "The assistant is temporarily unavailable. Please try again.")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, resp)
}
