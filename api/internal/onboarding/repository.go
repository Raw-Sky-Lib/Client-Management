package onboarding

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertTenant(ctx context.Context, clientID, urlEnc, anonEnc, srEnc, dbURLEnc, siteURL string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenants
			(id, supabase_url_encrypted, supabase_anon_encrypted,
			 supabase_service_role_encrypted, supabase_db_url_encrypted, site_url)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE SET
			supabase_url_encrypted          = EXCLUDED.supabase_url_encrypted,
			supabase_anon_encrypted         = EXCLUDED.supabase_anon_encrypted,
			supabase_service_role_encrypted = EXCLUDED.supabase_service_role_encrypted,
			supabase_db_url_encrypted       = EXCLUDED.supabase_db_url_encrypted,
			site_url                        = EXCLUDED.site_url
	`, clientID, urlEnc, anonEnc, srEnc, dbURLEnc, siteURL)
	return err
}

func (r *Repository) TenantExists(ctx context.Context, clientID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", clientID,
	).Scan(&exists)
	return exists, err
}

func (r *Repository) StoreEmailConfirmation(ctx context.Context, tenantID, email, hash string, expiresAt time.Time) error {
	// Invalidate any previous unused tokens for this tenant+email so only the latest link works.
	if _, err := r.db.Exec(ctx, `
		DELETE FROM email_confirmations WHERE tenant_id = $1 AND email = $2 AND used_at IS NULL
	`, tenantID, email); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO email_confirmations (tenant_id, email, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
	`, tenantID, email, hash, expiresAt)
	return err
}

func (r *Repository) GetByTokenHash(ctx context.Context, hash string) (*EmailConfirmation, error) {
	c := &EmailConfirmation{}
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, email, token_hash, expires_at, used_at, created_at
		FROM email_confirmations
		WHERE token_hash = $1
	`, hash).Scan(&c.ID, &c.TenantID, &c.Email, &c.TokenHash, &c.ExpiresAt, &c.UsedAt, &c.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get confirmation: %w", err)
	}
	return c, nil
}

func (r *Repository) GetTenantCredentials(ctx context.Context, tenantID string) (urlEnc, anonEnc, srEnc, siteURL string, err error) {
	err = r.db.QueryRow(ctx, `
		SELECT supabase_url_encrypted, supabase_anon_encrypted, supabase_service_role_encrypted,
		       COALESCE(site_url, '')
		FROM tenants WHERE id = $1
	`, tenantID).Scan(&urlEnc, &anonEnc, &srEnc, &siteURL)
	return
}

func (r *Repository) MarkConfirmationUsed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE email_confirmations SET used_at = NOW() WHERE id = $1", id)
	return err
}

func (r *Repository) MarkTenantOnboarded(ctx context.Context, tenantID string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE tenants SET onboarded_at = NOW() WHERE id = $1 AND onboarded_at IS NULL", tenantID)
	return err
}

func (r *Repository) UpsertTenantUser(ctx context.Context, tenantID, email string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenant_users (tenant_id, email)
		VALUES ($1, $2)
		ON CONFLICT (tenant_id, email) DO NOTHING
	`, tenantID, email)
	return err
}

func hashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return fmt.Sprintf("%x", h)
}
