package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

var ErrBudgetExceeded = errors.New("budget_exceeded")

type Repository struct {
	httpClient  *http.Client
	agencyURL   string
	agencyToken string
	clientID    string
}

func NewRepository(httpClient *http.Client, agencyURL, agencyToken, clientID string) *Repository {
	return &Repository{
		httpClient:  httpClient,
		agencyURL:   agencyURL,
		agencyToken: agencyToken,
		clientID:    clientID,
	}
}

// RecordUsage fires a POST to agency-hub in a goroutine and never blocks the caller.
func (r *Repository) RecordUsage(ctx context.Context, tenantID string, inputTokens, outputTokens int) {
	go func() {
		gctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		body, _ := json.Marshal(map[string]any{
			"tenant_id":     tenantID,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
		})

		req, err := http.NewRequestWithContext(gctx, http.MethodPost,
			r.agencyURL+"/api/claude/usage", bytes.NewReader(body))
		if err != nil {
			slog.Warn("claude: failed to build usage request", slog.String("error", err.Error()))
			return
		}
		r.setHeaders(req, tenantID)

		resp, err := r.httpClient.Do(req)
		if err != nil {
			slog.Warn("claude: usage recording failed", slog.String("error", err.Error()))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			slog.Warn("claude: usage recording returned unexpected status",
				slog.Int("status", resp.StatusCode))
		}
	}()
}

// CheckBudget returns ErrBudgetExceeded if the tenant has hit their monthly limit.
// On any HTTP or parse error it fails open (returns nil) and logs a warning —
// agency-hub being unavailable must not block client requests.
func (r *Repository) CheckBudget(ctx context.Context, tenantID string) error {
	url := fmt.Sprintf("%s/api/claude/budget/%s", r.agencyURL, tenantID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Warn("claude: failed to build budget request", slog.String("error", err.Error()))
		return nil
	}
	r.setHeaders(req, tenantID)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		slog.Warn("claude: budget check failed, failing open", slog.String("error", err.Error()))
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("claude: budget check returned unexpected status, failing open",
			slog.Int("status", resp.StatusCode))
		return nil
	}

	var result struct {
		Exceeded bool `json:"exceeded"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Warn("claude: failed to decode budget response, failing open",
			slog.String("error", err.Error()))
		return nil
	}

	if result.Exceeded {
		return ErrBudgetExceeded
	}
	return nil
}

func (r *Repository) setHeaders(req *http.Request, tenantID string) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+r.agencyToken)
	req.Header.Set("X-Client-ID", r.clientID)
	req.Header.Set("X-Tenant-ID", tenantID)
}
