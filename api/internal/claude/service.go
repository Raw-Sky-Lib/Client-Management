package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/DagMT/client-portal/internal/tenant"
)

type Service struct {
	rl     *RateLimiter
	repo   *Repository
	prompt *PromptBuilder
	client anthropic.Client
	model  string
}

func NewService(rl *RateLimiter, repo *Repository, prompt *PromptBuilder, apiKey, model string) *Service {
	return &Service{
		rl:     rl,
		repo:   repo,
		prompt: prompt,
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  model,
	}
}

func (s *Service) Generate(ctx context.Context, cfg *tenant.Config, req GenerateRequest) (*GenerateResponse, error) {
	if err := s.rl.Check(ctx, cfg.TenantID); err != nil {
		return nil, err
	}

	if err := s.repo.CheckBudget(ctx, cfg.TenantID); err != nil {
		return nil, err
	}

	systemPrompt, _, err := s.prompt.Build(ctx, cfg, req)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     s.model,
		MaxTokens: 1024,
		System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(req.Instruction)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("claude api: %w", err)
	}

	var text string
	for _, block := range resp.Content {
		if block.Type == "text" {
			text = block.Text
			break
		}
	}

	var changes []FieldChange
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &changes); err != nil {
		return nil, ErrInvalidClaudeResponse
	}

	s.repo.RecordUsage(ctx, cfg.TenantID, int(resp.Usage.InputTokens), int(resp.Usage.OutputTokens))

	return &GenerateResponse{Changes: changes}, nil
}
