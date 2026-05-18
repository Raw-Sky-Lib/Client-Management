package onboarding

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/database"
	"github.com/DagMT/client-portal/internal/mailer"
	"github.com/DagMT/client-portal/internal/utils"
)

var (
	ErrClientNotSetup = errors.New("This client is not set up in the portal yet. Contact your website team.")
	ErrLinkInvalid    = errors.New("Invalid or expired confirmation link.")
	ErrLinkUsed       = errors.New("This confirmation link has already been used.")
	ErrLinkExpired    = errors.New("Confirmation link expired. Request a new one.")
)

type Service struct {
	repo        *Repository
	httpClient  *http.Client
	mailer      mailer.Mailer
	agencyURL   string
	agencyToken string
	encKey      []byte
	publicURL   string
	frontendURL string
}

func NewService(
	repo *Repository,
	httpClient *http.Client,
	m mailer.Mailer,
	agencyURL, agencyToken string,
	encKey []byte,
	publicURL, frontendURL string,
) *Service {
	return &Service{
		repo:        repo,
		httpClient:  httpClient,
		mailer:      m,
		agencyURL:   agencyURL,
		agencyToken: agencyToken,
		encKey:      encKey,
		publicURL:   publicURL,
		frontendURL: frontendURL,
	}
}

// RegisterClient is called by agency-hub to register a project with the portal.
// It:
//  1. UPSERTs the tenant identity row
//  2. Validates + migrates the client's Supabase project
//  3. Encrypts and stores credentials in tenant_projects
//  4. Sends an invite email so the client can access their dashboard
func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) error {
	req.ClientSupabaseURL = strings.TrimRight(strings.TrimSuffix(
		strings.TrimRight(req.ClientSupabaseURL, "/"), "/rest/v1"), "/")

	if err := utils.ValidateSupabaseCredentials(req.ClientSupabaseURL, req.ClientSupabaseServiceRoleKey); err != nil {
		return fmt.Errorf("invalid supabase credentials: %w", err)
	}
	if err := database.MigrateClientDB(req.ClientSupabaseDBURL); err != nil {
		return fmt.Errorf("client db migration: %w", err)
	}

	bucketName := bucketNameFromSiteURL(req.SiteURL)
	if err := s.createDefaultBucket(ctx, req.ClientSupabaseURL, req.ClientSupabaseServiceRoleKey, bucketName); err != nil {
		_ = err // fire-and-forget
	}

	urlEnc, err := utils.EncryptString(req.ClientSupabaseURL, s.encKey)
	if err != nil {
		return err
	}
	anonEnc, err := utils.EncryptString(req.ClientSupabaseAnonKey, s.encKey)
	if err != nil {
		return err
	}
	srEnc, err := utils.EncryptString(req.ClientSupabaseServiceRoleKey, s.encKey)
	if err != nil {
		return err
	}
	dbEnc, err := utils.EncryptString(req.ClientSupabaseDBURL, s.encKey)
	if err != nil {
		return err
	}

	// 1. Tenant identity (idempotent — existing tenants are unchanged)
	if err := s.repo.UpsertTenant(ctx, req.ClientID); err != nil {
		return fmt.Errorf("upsert tenant: %w", err)
	}

	// 2. Project credentials
	if err := s.repo.UpsertTenantProject(ctx,
		req.ClientID, req.ProjectID, req.ProjectName, req.SiteURL,
		urlEnc, anonEnc, srEnc, dbEnc,
	); err != nil {
		return fmt.Errorf("upsert tenant project: %w", err)
	}

	// 3. Upsert tenant_users so magic-link login works immediately
	if err := s.repo.UpsertTenantUser(ctx, req.ClientID, req.Email); err != nil {
		return fmt.Errorf("upsert tenant user: %w", err)
	}

	// 4. Resolve the portal project UUID and send the invite
	portalProjectID, err := s.repo.GetProjectIDByAgencyID(ctx, req.ProjectID)
	if err != nil {
		return fmt.Errorf("resolve portal project id: %w", err)
	}
	return s.sendInvite(ctx, req.ClientID, req.Email, portalProjectID)
}

// ResendInvite re-sends an invite email for an existing tenant.
// It picks the first registered project for the tenant.
func (s *Service) ResendInvite(ctx context.Context, clientID, email string) error {
	exists, err := s.repo.TenantExists(ctx, clientID)
	if err != nil {
		return fmt.Errorf("check tenant: %w", err)
	}
	if !exists {
		return ErrClientNotSetup
	}
	// Find the first project for this tenant to scope the invite.
	projectID, err := s.getFirstProjectIDForTenant(ctx, clientID)
	if err != nil {
		return fmt.Errorf("find project for resend: %w", err)
	}
	return s.sendInvite(ctx, clientID, email, projectID)
}

// DeregisterClient removes all portal data for a client so they can no longer
// log in. Called by agency-hub when a client is permanently deleted.
func (s *Service) DeregisterClient(ctx context.Context, clientID string) error {
	if err := s.repo.DeleteTenant(ctx, clientID); err != nil {
		return fmt.Errorf("deregister client: %w", err)
	}
	return nil
}

func (s *Service) getFirstProjectIDForTenant(ctx context.Context, tenantID string) (string, error) {
	var id string
	err := s.repo.db.QueryRow(ctx, `
		SELECT id FROM tenant_projects WHERE tenant_id = $1 ORDER BY created_at ASC LIMIT 1
	`, tenantID).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("no project found for tenant %s", tenantID)
	}
	return id, nil
}

func (s *Service) sendInvite(ctx context.Context, tenantID, email, projectID string) error {
	plaintext, hash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate invite token: %w", err)
	}
	if err := s.repo.StoreEmailConfirmation(ctx, tenantID, email, hash, time.Now().Add(72*time.Hour), &projectID); err != nil {
		return fmt.Errorf("store invite: %w", err)
	}
	link := fmt.Sprintf("%s/api/onboarding/confirm?token=%s", s.publicURL, plaintext)
	return s.mailer.Send(ctx, email,
		"Your dashboard is ready",
		fmt.Sprintf(`
			<p>Your client portal is set up and ready to use.</p>
			<p><a href="%s">Access your dashboard →</a></p>
			<p>This link expires in 72 hours. After that, you can sign in with a magic link from the login page.</p>
		`, link),
	)
}

// Confirm verifies the emailed token, creates the Supabase auth user, and returns
// lean PortalClaims (no credentials embedded — frontend fetches those via GET /api/projects).
func (s *Service) Confirm(ctx context.Context, token string) (*auth.PortalClaims, error) {
	conf, err := s.repo.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, fmt.Errorf("lookup token: %w", err)
	}
	if conf == nil {
		return nil, ErrLinkInvalid
	}
	if conf.UsedAt != nil {
		return nil, ErrLinkUsed
	}
	if time.Now().After(conf.ExpiresAt) {
		return nil, ErrLinkExpired
	}
	if conf.ProjectID == nil {
		return nil, fmt.Errorf("confirmation has no project — cannot provision Supabase user")
	}

	urlEnc, anonEnc, srEnc, _, _, err := s.repo.GetProjectCredentials(ctx, *conf.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("fetch project credentials: %w", err)
	}
	supabaseURL, err := utils.DecryptString(urlEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(srEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt service role: %w", err)
	}
	_ = anonEnc // not needed here; frontend fetches via GET /api/projects

	userID, err := s.createSupabaseUser(ctx, supabaseURL, serviceRoleKey, conf.Email)
	if err != nil {
		return nil, fmt.Errorf("create supabase user: %w", err)
	}

	if err := s.repo.MarkConfirmationUsed(ctx, conf.ID); err != nil {
		return nil, fmt.Errorf("mark used: %w", err)
	}
	if err := s.repo.MarkTenantOnboarded(ctx, conf.TenantID); err != nil {
		return nil, fmt.Errorf("mark onboarded: %w", err)
	}
	if err := s.repo.UpsertTenantUser(ctx, conf.TenantID, conf.Email); err != nil {
		return nil, fmt.Errorf("upsert tenant user: %w", err)
	}

	return &auth.PortalClaims{
		UserID:   userID,
		TenantID: conf.TenantID,
		Email:    conf.Email,
	}, nil
}


func (s *Service) createSupabaseUser(ctx context.Context, supabaseURL, serviceRoleKey, email string) (string, error) {
	pw, err := randomHex(32)
	if err != nil {
		return "", err
	}
	body, _ := json.Marshal(map[string]any{
		"email":         email,
		"password":      pw,
		"email_confirm": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		supabaseURL+"/auth/v1/admin/users", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return s.getSupabaseUserByEmail(ctx, supabaseURL, serviceRoleKey, email)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode user: %w", err)
	}
	if result.ID == "" {
		return "", fmt.Errorf("supabase returned empty user id (status %d)", resp.StatusCode)
	}
	return result.ID, nil
}

func (s *Service) getSupabaseUserByEmail(ctx context.Context, supabaseURL, serviceRoleKey, email string) (string, error) {
	endpoint := supabaseURL + "/auth/v1/admin/users?email=" + url.QueryEscape(email) + "&per_page=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build lookup request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase user lookup: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Users []struct {
			ID string `json:"id"`
		} `json:"users"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode user list: %w", err)
	}
	if len(result.Users) == 0 || result.Users[0].ID == "" {
		return "", fmt.Errorf("supabase user not found for email")
	}
	return result.Users[0].ID, nil
}

func bucketNameFromSiteURL(siteURL string) string {
	u, err := url.Parse(siteURL)
	if err != nil || u.Hostname() == "" {
		return "media"
	}
	name := strings.ToLower(u.Hostname())
	name = strings.ReplaceAll(name, ".", "-")
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		}
	}
	name = strings.Trim(sb.String(), "-")
	if len(name) > 63 {
		name = name[:63]
	}
	if len(name) < 3 {
		return "media"
	}
	return name
}

func (s *Service) createDefaultBucket(ctx context.Context, supabaseURL, serviceRoleKey, bucketName string) error {
	body, _ := json.Marshal(map[string]any{
		"id":     bucketName,
		"name":   bucketName,
		"public": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		supabaseURL+"/storage/v1/bucket", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build bucket request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("create bucket returned %d", resp.StatusCode)
	}
	return nil
}


func generateToken() (plaintext, hash string, err error) {
	plaintext, err = randomHex(32)
	if err != nil {
		return
	}
	hash = hashToken(plaintext)
	return
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
