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

// UpsertTenant creates an identity-only row in tenants (no credentials).
// Credentials live in tenant_projects — see UpsertTenantProject.
func (r *Repository) UpsertTenant(ctx context.Context, clientID string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO tenants (id) VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`, clientID)
	return err
}

// UpsertTenantProject creates or updates a project credential row in tenant_projects.
// Conflict target is agency_project_id — re-inviting an existing project updates its creds.
//
// Before inserting, any placeholder rows for this tenant are removed. A placeholder
// is a row whose agency_project_id differs from the incoming one AND whose site_url
// either matches the incoming site_url or is empty — these are rows created by the
// migration 004 data-copy rather than a real agency-hub registration.
func (r *Repository) UpsertTenantProject(ctx context.Context,
	tenantID, projectID, name, siteURL,
	urlEnc, anonEnc, srEnc, dbURLEnc string,
) error {
	// Remove stale migration-created placeholder rows so re-registration doesn't
	// leave a duplicate "Default"/"My Project" entry alongside the real one.
	if _, err := r.db.Exec(ctx, `
		DELETE FROM tenant_projects
		WHERE tenant_id = $1
		  AND agency_project_id != $2
		  AND (site_url IS NULL OR site_url = '' OR site_url = $3)
	`, tenantID, projectID, siteURL); err != nil {
		return fmt.Errorf("cleanup placeholder projects: %w", err)
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO tenant_projects
			(tenant_id, agency_project_id, name, site_url,
			 supabase_url_encrypted, supabase_anon_encrypted,
			 supabase_service_role_encrypted, supabase_db_url_encrypted)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (agency_project_id) DO UPDATE SET
			name                            = EXCLUDED.name,
			site_url                        = EXCLUDED.site_url,
			supabase_url_encrypted          = EXCLUDED.supabase_url_encrypted,
			supabase_anon_encrypted         = EXCLUDED.supabase_anon_encrypted,
			supabase_service_role_encrypted = EXCLUDED.supabase_service_role_encrypted,
			supabase_db_url_encrypted       = EXCLUDED.supabase_db_url_encrypted
	`, tenantID, projectID, name, siteURL, urlEnc, anonEnc, srEnc, dbURLEnc)
	return err
}

// GetProjectIDByAgencyID returns the tenant_projects.id (portal UUID) for a given
// agency_project_id. Used when building the email confirmation record.
func (r *Repository) GetProjectIDByAgencyID(ctx context.Context, agencyProjectID string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx,
		"SELECT id FROM tenant_projects WHERE agency_project_id = $1 LIMIT 1",
		agencyProjectID,
	).Scan(&id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("project not found for agency_project_id %s", agencyProjectID)
		}
		return "", fmt.Errorf("get project id: %w", err)
	}
	return id, nil
}

// GetProjectCredentials returns encrypted credential columns for a tenant_projects row.
func (r *Repository) GetProjectCredentials(ctx context.Context, projectID string) (
	urlEnc, anonEnc, srEnc, dbURLEnc, siteURL string, err error,
) {
	err = r.db.QueryRow(ctx, `
		SELECT supabase_url_encrypted, supabase_anon_encrypted,
		       supabase_service_role_encrypted,
		       COALESCE(supabase_db_url_encrypted, ''),
		       COALESCE(site_url, '')
		FROM tenant_projects WHERE id = $1
	`, projectID).Scan(&urlEnc, &anonEnc, &srEnc, &dbURLEnc, &siteURL)
	return
}

func (r *Repository) TenantExists(ctx context.Context, clientID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)", clientID,
	).Scan(&exists)
	return exists, err
}

// IsTenantOnboarded returns true when the client has already confirmed their invite
// (tenants.onboarded_at is set). Used by RegisterClient to skip re-sending invites.
func (r *Repository) IsTenantOnboarded(ctx context.Context, clientID string) (bool, error) {
	var onboarded bool
	err := r.db.QueryRow(ctx,
		"SELECT onboarded_at IS NOT NULL FROM tenants WHERE id = $1", clientID,
	).Scan(&onboarded)
	if err != nil {
		if err == pgx.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return onboarded, nil
}

// StoreEmailConfirmation stores an invite confirmation token scoped to a tenant_projects row.
// projectID is the tenant_projects.id (portal UUID) — nullable for pre-project invites.
//
// The DELETE sweeps all unused invite tokens for this email across ALL tenants so that
// orphaned tokens from a manually-deleted-and-recreated client cannot cause a
// "account already verified" error on the new invite link. Login tokens (magic link,
// password reset) are left untouched — they use token_type='login'.
func (r *Repository) StoreEmailConfirmation(ctx context.Context, tenantID, email, hash string, expiresAt time.Time, projectID *string) error {
	if _, err := r.db.Exec(ctx, `
		DELETE FROM email_confirmations WHERE email = $1 AND token_type = 'invite' AND used_at IS NULL
	`, email); err != nil {
		return err
	}
	_, err := r.db.Exec(ctx, `
		INSERT INTO email_confirmations (tenant_id, email, token_hash, expires_at, project_id, token_type)
		VALUES ($1, $2, $3, $4, $5, 'invite')
	`, tenantID, email, hash, expiresAt, projectID)
	return err
}

func (r *Repository) GetByTokenHash(ctx context.Context, hash string) (*EmailConfirmation, error) {
	c := &EmailConfirmation{}
	err := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, email, token_hash, expires_at, used_at, created_at, project_id
		FROM email_confirmations
		WHERE token_hash = $1
	`, hash).Scan(&c.ID, &c.TenantID, &c.Email, &c.TokenHash, &c.ExpiresAt, &c.UsedAt, &c.CreatedAt, &c.ProjectID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get confirmation: %w", err)
	}
	return c, nil
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

// DeleteTenantProject removes a single tenant_projects row by agency_project_id.
// No-ops silently if the row doesn't exist (portal may never have received the project).
func (r *Repository) DeleteTenantProject(ctx context.Context, agencyProjectID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM tenant_projects WHERE agency_project_id = $1`, agencyProjectID)
	return err
}

// DeleteTenant removes the tenant and all related rows (cascades to tenant_users,
// tenant_projects, email_confirmations).
func (r *Repository) DeleteTenant(ctx context.Context, clientID string) error {
	res, err := r.db.Exec(ctx, `DELETE FROM tenants WHERE id = $1`, clientID)
	if err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}
	n := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tenant not found: %s", clientID)
	}
	return nil
}

// UpdateTenantUserEmail replaces the email address for all users under a tenant.
// Called when a client's email is changed in agency-hub so portal auth keeps working.
func (r *Repository) UpdateTenantUserEmail(ctx context.Context, tenantID, newEmail string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE tenant_users SET email = $2 WHERE tenant_id = $1`,
		tenantID, newEmail,
	)
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

// EvictEmailFromOtherTenants removes tenant_users rows for the given email that belong
// to any tenant other than tenantID. Called during RegisterClient so that a manually
// deleted-and-recreated client doesn't leave orphaned rows that would cause GetTenantByEmail
// to return the wrong (now-dead) tenant, silently breaking password reset and magic link flows.
func (r *Repository) EvictEmailFromOtherTenants(ctx context.Context, tenantID, email string) error {
	_, err := r.db.Exec(ctx, `
		DELETE FROM tenant_users WHERE email = $1 AND tenant_id != $2
	`, email, tenantID)
	return err
}

// GetTenantUserID returns the tenant_users.id for a given tenant + email.
// Used when confirming an invite that has no project yet.
func (r *Repository) GetTenantUserID(ctx context.Context, tenantID, email string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx,
		"SELECT id FROM tenant_users WHERE tenant_id = $1 AND email = $2 LIMIT 1",
		tenantID, email,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("get tenant user id: %w", err)
	}
	return id, nil
}

func hashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return fmt.Sprintf("%x", h)
}
