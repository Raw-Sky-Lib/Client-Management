package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TokenRecord is a row from email_confirmations used for portal-native login tokens.
type TokenRecord struct {
	ID        string
	TenantID  string
	Email     string
	ExpiresAt time.Time
	UsedAt    *time.Time
}

// TenantLookup carries the encrypted credentials needed to build a PortalClaims.
type TenantLookup struct {
	TenantID string
	URLEnc   string
	AnonEnc  string
	SREnc    string
	SiteURL  string
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
		SELECT tu.tenant_id, t.supabase_url_encrypted, t.supabase_anon_encrypted, t.supabase_service_role_encrypted, COALESCE(t.site_url, '')
		FROM tenant_users tu
		JOIN tenants t ON t.id = tu.tenant_id
		WHERE tu.email = $1
	`, email).Scan(&t.TenantID, &t.URLEnc, &t.AnonEnc, &t.SREnc, &t.SiteURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by email: %w", err)
	}
	return t, nil
}

// StoreLoginToken clears unused tokens for the same email and stores a fresh one.
func (r *Repository) StoreLoginToken(ctx context.Context, tenantID, email, hash string, expiresAt time.Time) error {
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

func (r *Repository) GetLoginToken(ctx context.Context, hash string) (*TokenRecord, error) {
	rec := &TokenRecord{}
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, email, expires_at, used_at
		FROM email_confirmations WHERE token_hash = $1
	`, hash).Scan(&rec.ID, &rec.TenantID, &rec.Email, &rec.ExpiresAt, &rec.UsedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get login token: %w", err)
	}
	return rec, nil
}

func (r *Repository) MarkLoginTokenUsed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx,
		"UPDATE email_confirmations SET used_at = NOW() WHERE id = $1", id)
	return err
}

func (r *Repository) GetTenantByID(ctx context.Context, tenantID string) (*TenantLookup, error) {
	t := &TenantLookup{}
	err := r.db.QueryRow(ctx, `
		SELECT id, supabase_url_encrypted, supabase_anon_encrypted, supabase_service_role_encrypted, COALESCE(site_url, '')
		FROM tenants WHERE id = $1
	`, tenantID).Scan(&t.TenantID, &t.URLEnc, &t.AnonEnc, &t.SREnc, &t.SiteURL)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}
