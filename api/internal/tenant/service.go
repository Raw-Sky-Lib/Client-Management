package tenant

import (
	"context"
	"errors"
	"fmt"

	"github.com/DagMT/client-portal/internal/utils"
)

var ErrTenantNotFound = errors.New("tenant not found")

type Service struct {
	repo   *Repository
	encKey []byte
}

func NewService(repo *Repository, encKey []byte) *Service {
	return &Service{repo: repo, encKey: encKey}
}

// Resolve fetches a tenant's encrypted credentials from the portal DB,
// decrypts them, and returns a Config ready for use by handlers.
func (s *Service) Resolve(ctx context.Context, tenantID string) (*Config, error) {
	raw, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch tenant: %w", err)
	}
	if raw == nil {
		return nil, ErrTenantNotFound
	}

	supabaseURL, err := utils.DecryptString(raw.URLEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(raw.AnonEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt anon: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(raw.SREnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt service role: %w", err)
	}

	return &Config{
		TenantID:        raw.ID,
		SupabaseURL:     supabaseURL,
		SupabaseAnonKey: anonKey,
		ServiceRoleKey:  serviceRoleKey,
		SiteURL:         raw.SiteURL,
	}, nil
}
