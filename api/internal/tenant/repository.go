package tenant

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rawTenant struct {
	ID string
}

type rawTenantProject struct {
	ID      string
	URLEnc  string
	AnonEnc string
	SREnc   string
	SiteURL string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByID(ctx context.Context, tenantID string) (*rawTenant, error) {
	t := &rawTenant{}
	err := r.db.QueryRow(ctx,
		"SELECT id FROM tenants WHERE id = $1", tenantID,
	).Scan(&t.ID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return t, nil
}

func (r *Repository) GetFirstProjectForTenant(ctx context.Context, tenantID string) (*rawTenantProject, error) {
	p := &rawTenantProject{}
	err := r.db.QueryRow(ctx, `
		SELECT id, supabase_url_encrypted, supabase_anon_encrypted,
		       supabase_service_role_encrypted, COALESCE(site_url, '')
		FROM tenant_projects
		WHERE tenant_id = $1
		ORDER BY created_at ASC
		LIMIT 1
	`, tenantID).Scan(&p.ID, &p.URLEnc, &p.AnonEnc, &p.SREnc, &p.SiteURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get first project: %w", err)
	}
	return p, nil
}
