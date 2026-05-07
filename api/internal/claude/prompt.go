package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/DagMT/client-portal/internal/tenant"
)

type PromptBuilder struct {
	httpClient *http.Client
}

func NewPromptBuilder(httpClient *http.Client) *PromptBuilder {
	return &PromptBuilder{httpClient: httpClient}
}

// Build fetches the current section from the client's Supabase and assembles
// the system prompt. Uses the service role key — never called from the frontend.
func (p *PromptBuilder) Build(ctx context.Context, cfg *tenant.Config, req GenerateRequest) (systemPrompt string, currentContent map[string]any, err error) {
	businessName, err := p.fetchBusinessName(ctx, cfg)
	if err != nil {
		return "", nil, fmt.Errorf("fetch business name: %w", err)
	}

	pageTitle, sections, err := p.fetchPageSections(ctx, cfg, req.PageSlug)
	if err != nil {
		return "", nil, err
	}

	section, ok := sections[req.SectionType]
	if !ok {
		return "", nil, ErrSectionNotFound
	}

	sectionMap, ok := section.(map[string]any)
	if !ok {
		return "", nil, fmt.Errorf("section %q has unexpected format", req.SectionType)
	}

	contentJSON, err := json.MarshalIndent(sectionMap, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("marshal section: %w", err)
	}

	systemPrompt = fmt.Sprintf(`You are a content assistant for %s.
Page: %s
Section: %s
Current content:
%s

The user will give you an instruction. Respond ONLY with a JSON array of field changes:
[{"field":"...","current":"...","proposed":"...","notes":"..."}]
No explanation. No markdown. Only the JSON array.`,
		businessName, pageTitle, req.SectionType, string(contentJSON),
	)

	return systemPrompt, sectionMap, nil
}

func (p *PromptBuilder) fetchBusinessName(ctx context.Context, cfg *tenant.Config) (string, error) {
	u := cfg.SupabaseURL + "/rest/v1/site_settings?key=eq.business_name&select=value&limit=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	p.setHeaders(req, cfg.ServiceRoleKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var rows []struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return "", fmt.Errorf("decode site_settings: %w", err)
	}
	if len(rows) == 0 || rows[0].Value == "" {
		return "this business", nil // graceful fallback
	}
	return rows[0].Value, nil
}

func (p *PromptBuilder) fetchPageSections(ctx context.Context, cfg *tenant.Config, slug string) (pageTitle string, sections map[string]any, err error) {
	u := cfg.SupabaseURL + "/rest/v1/pages?slug=eq." + url.QueryEscape(slug) + "&select=title,sections&limit=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", nil, err
	}
	p.setHeaders(req, cfg.ServiceRoleKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var rows []struct {
		Title    string         `json:"title"`
		Sections map[string]any `json:"sections"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return "", nil, fmt.Errorf("decode pages: %w", err)
	}
	if len(rows) == 0 {
		return "", nil, ErrPageNotFound
	}
	return rows[0].Title, rows[0].Sections, nil
}

func (p *PromptBuilder) setHeaders(req *http.Request, serviceRoleKey string) {
	req.Header.Set("apikey", serviceRoleKey)
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")
}
