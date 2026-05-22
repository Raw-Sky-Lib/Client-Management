package cms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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

// UpdatePageSections writes sections to the client's Supabase using the service role key,
// which bypasses RLS. The anon key (held by the browser) is read-only.
func (s *Service) UpdatePageSections(ctx context.Context, cfg *tenant.Config, slug string, sections json.RawMessage) error {
	body, err := json.Marshal(map[string]any{
		"sections":   json.RawMessage(sections),
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	return s.patch(ctx, cfg, "/rest/v1/pages?slug=eq."+slug, body)
}

// UpdatePageVisibility sets is_published on the page.
func (s *Service) UpdatePageVisibility(ctx context.Context, cfg *tenant.Config, slug string, isPublished bool) error {
	body, err := json.Marshal(map[string]any{"is_published": isPublished})
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	return s.patch(ctx, cfg, "/rest/v1/pages?slug=eq."+slug, body)
}

func (s *Service) patch(ctx context.Context, cfg *tenant.Config, path string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch,
		cfg.SupabaseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceRoleKey)
	req.Header.Set("apikey", cfg.ServiceRoleKey)
	req.Header.Set("Prefer", "return=minimal")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("supabase patch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("supabase returned status %d", resp.StatusCode)
	}
	return nil
}
