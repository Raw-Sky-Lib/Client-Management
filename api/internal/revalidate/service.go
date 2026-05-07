package revalidate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DagMT/client-portal/internal/tenant"
)

type Service struct {
	httpClient *http.Client
}

func NewService(httpClient *http.Client) *Service {
	return &Service{httpClient: httpClient}
}

// TriggerISR fires a non-blocking POST to the client site's revalidation endpoint.
// It returns immediately — failures are logged but never surfaced to the caller.
func (s *Service) TriggerISR(cfg *tenant.Config, paths []string) {
	if cfg.SiteURL == "" {
		slog.Warn("TriggerISR skipped: tenant has no site_url", slog.String("tenant_id", cfg.TenantID))
		return
	}

	// Capture by value before the goroutine runs.
	siteURL := cfg.SiteURL
	tenantID := cfg.TenantID
	secret := cfg.ServiceRoleKey

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.trigger(ctx, siteURL, tenantID, secret, paths); err != nil {
			slog.Error("ISR revalidation failed",
				slog.String("tenant_id", tenantID),
				slog.String("site_url", siteURL),
				slog.String("error", err.Error()),
			)
		}
	}()
}

func (s *Service) trigger(ctx context.Context, siteURL, tenantID, secret string, paths []string) error {
	body, err := json.Marshal(map[string]any{"paths": paths})
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		siteURL+"/api/revalidate", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Revalidate-Secret", secret)
	req.Header.Set("X-Client-ID", tenantID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
