package claude

import (
	"encoding/json"
	"errors"
	"log/slog"
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

// Generate handles POST /api/assistant/generate
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
			slog.Error("claude: generate: unexpected error",
				slog.String("tenant_id", cfg.TenantID),
				slog.String("page_slug", req.PageSlug),
				slog.String("section_type", req.SectionType),
				slog.String("error", err.Error()),
			)
			utils.RespondError(w, http.StatusInternalServerError, "The assistant is temporarily unavailable. Please try again.")
		}
		return
	}

	utils.RespondJSON(w, http.StatusOK, resp)
}
