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

// Resolve confirms the tenant exists then fetches its first project's encrypted
// credentials, decrypts them, and returns a Config ready for use by handlers.
func (s *Service) Resolve(ctx context.Context, tenantID string) (*Config, error) {
	raw, err := s.repo.GetByID(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch tenant: %w", err)
	}
	if raw == nil {
		return nil, ErrTenantNotFound
	}

	proj, err := s.repo.GetFirstProjectForTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch project: %w", err)
	}
	if proj == nil {
		return nil, fmt.Errorf("tenant has no projects")
	}

	supabaseURL, err := utils.DecryptString(proj.URLEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(proj.AnonEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt anon: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(proj.SREnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt service role: %w", err)
	}
	var revalidateSecret string
	if proj.RevalidateSecretEnc != "" {
		revalidateSecret, err = utils.DecryptString(proj.RevalidateSecretEnc, s.encKey)
		if err != nil {
			return nil, fmt.Errorf("decrypt revalidate secret: %w", err)
		}
	}

	return &Config{
		TenantID:         raw.ID,
		SupabaseURL:      supabaseURL,
		SupabaseAnonKey:  anonKey,
		ServiceRoleKey:   serviceRoleKey,
		SiteURL:          proj.SiteURL,
		RevalidateSecret: revalidateSecret,
	}, nil
}
