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

// TenantLookup carries just the identity fields needed to build a PortalClaims.
// Credentials now live in tenant_projects — fetched separately when needed.
type TenantLookup struct {
	TenantID string
}

// rawTenantProject holds the encrypted credential columns from tenant_projects.
type rawTenantProject struct {
	ID       string
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
		SELECT tu.tenant_id
		FROM tenant_users tu
		JOIN tenants t ON t.id = tu.tenant_id
		WHERE tu.email = $1
		ORDER BY t.created_at DESC
		LIMIT 1
	`, email).Scan(&t.TenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by email: %w", err)
	}
	return t, nil
}

// StoreLoginToken clears unused login tokens for this tenant+email and stores a fresh one.
// Only login-type tokens are deleted — invite tokens are left untouched so that a client
// requesting a magic link or password reset while their invite is still pending doesn't
// lose their original onboarding link.
func (r *Repository) StoreLoginToken(ctx context.Context, tenantID, email, hash string, expiresAt time.Time) error {
	if _, err := r.db.Exec(ctx, `
		DELETE FROM email_confirmations
		WHERE tenant_id = $1 AND email = $2 AND token_type = 'login' AND used_at IS NULL
	`, tenantID, email); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO email_confirmations (tenant_id, email, token_hash, expires_at, token_type)
		VALUES ($1, $2, $3, $4, 'login')
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
	err := r.db.QueryRow(ctx,
		"SELECT id FROM tenants WHERE id = $1", tenantID,
	).Scan(&t.TenantID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return t, nil
}

// GetFirstProjectForTenant returns the first tenant_projects row for the given tenant.
// Used by auth flows that still need Supabase credentials (password auth, token exchange).
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
		return nil, fmt.Errorf("get first project for tenant: %w", err)
	}
	return p, nil
}

// GetTenantUserID returns the tenant_users.id for a given tenant + email pair.
// Used to populate UserID in PortalClaims during magic link login.
func (r *Repository) GetTenantUserID(ctx context.Context, tenantID, email string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		SELECT id FROM tenant_users WHERE tenant_id = $1 AND email = $2 LIMIT 1
	`, tenantID, email).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("tenant user not found")
		}
		return "", fmt.Errorf("get tenant user id: %w", err)
	}
	return id, nil
}
