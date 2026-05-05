package auth

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TenantLookup carries the encrypted credentials needed to build a PortalClaims.
type TenantLookup struct {
	TenantID string
	URLEnc   string
	AnonEnc  string
	SREnc    string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetTenantByEmail(ctx context.Context, email string) (*TenantLookup, error) {
	t := &TenantLookup{}
	err := r.db.QueryRow(ctx, `
		SELECT tu.tenant_id, t.supabase_url_encrypted, t.supabase_anon_encrypted, t.supabase_service_role_encrypted
		FROM tenant_users tu
		JOIN tenants t ON t.id = tu.tenant_id
		WHERE tu.email = $1
	`, email).Scan(&t.TenantID, &t.URLEnc, &t.AnonEnc, &t.SREnc)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by email: %w", err)
	}
	return t, nil
}

func (r *Repository) GetTenantByID(ctx context.Context, tenantID string) (*TenantLookup, error) {
	t := &TenantLookup{}
	err := r.db.QueryRow(ctx, `
		SELECT id, supabase_url_encrypted, supabase_anon_encrypted, supabase_service_role_encrypted
		FROM tenants WHERE id = $1
	`, tenantID).Scan(&t.TenantID, &t.URLEnc, &t.AnonEnc, &t.SREnc)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}
