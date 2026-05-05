package tenant

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type rawTenant struct {
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
	err := r.db.QueryRow(ctx, `
		SELECT id, supabase_url_encrypted, supabase_anon_encrypted,
		       supabase_service_role_encrypted, COALESCE(site_url, '')
		FROM tenants WHERE id = $1
	`, tenantID).Scan(&t.ID, &t.URLEnc, &t.AnonEnc, &t.SREnc, &t.SiteURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}
	return t, nil
}
