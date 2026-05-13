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
	ErrTokenExpired   = errors.New("Your access code has expired. Ask your website team for a new one.")
	ErrTokenUsed      = errors.New("This access code has already been used. Contact your website team.")
	ErrTokenInvalid   = errors.New("Invalid access code. Check for typos and try again.")
	ErrClientNotSetup = errors.New("This client is not set up in the portal yet. Contact your website team.")
	ErrLinkInvalid    = errors.New("Invalid or expired confirmation link.")
	ErrLinkUsed       = errors.New("This confirmation link has already been used.")
	ErrLinkExpired    = errors.New("Confirmation link expired. Request a new one.")
)

type Service struct {
	repo           *Repository
	httpClient     *http.Client
	mailer         mailer.Mailer
	agencyURL      string
	agencyToken    string
	agencyClientID string
	encKey         []byte
	publicURL      string // backend's own URL — confirmation email links point here
	frontendURL    string // frontend URL — post-confirm redirect target
}

func NewService(
	repo *Repository,
	httpClient *http.Client,
	m mailer.Mailer,
	agencyURL, agencyToken, agencyClientID string,
	encKey []byte,
	publicURL, frontendURL string,
) *Service {
	return &Service{
		repo:           repo,
		httpClient:     httpClient,
		mailer:         m,
		agencyURL:      agencyURL,
		agencyToken:    agencyToken,
		agencyClientID: agencyClientID,
		encKey:         encKey,
		publicURL:      publicURL,
		frontendURL:    frontendURL,
	}
}

func (s *Service) ResendInvite(ctx context.Context, clientID, email string) error {
	exists, err := s.repo.TenantExists(ctx, clientID)
	if err != nil {
		return fmt.Errorf("check tenant: %w", err)
	}
	if !exists {
		return ErrClientNotSetup
	}
	return s.sendInvite(ctx, clientID, email)
}

func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) error {
	// Strip trailing /rest/v1 or /rest/v1/ that callers sometimes include.
	req.ClientSupabaseURL = strings.TrimRight(strings.TrimSuffix(
		strings.TrimRight(req.ClientSupabaseURL, "/"), "/rest/v1"), "/")

	if err := utils.ValidateSupabaseCredentials(req.ClientSupabaseURL, req.ClientSupabaseServiceRoleKey); err != nil {
		return fmt.Errorf("invalid supabase credentials: %w", err)
	}
	if err := database.MigrateClientDB(req.ClientSupabaseDBURL); err != nil {
		return fmt.Errorf("client db migration: %w", err)
	}

	// Create the client's default storage bucket (named from their site URL).
	// Fire-and-forget — a failed bucket creation shouldn't block onboarding.
	bucketName := bucketNameFromSiteURL(req.SiteURL)
	if err := s.createDefaultBucket(ctx, req.ClientSupabaseURL, req.ClientSupabaseServiceRoleKey, bucketName); err != nil {
		// Log but don't fail — bucket may already exist, or Storage not yet enabled.
		_ = err
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
	if err := s.repo.UpsertTenant(ctx, req.ClientID, urlEnc, anonEnc, srEnc, dbEnc, req.SiteURL); err != nil {
		return err
	}
	// Auto-send invite email so the client can access their dashboard without a connection token.
	return s.sendInvite(ctx, req.ClientID, req.Email)
}

func (s *Service) sendInvite(ctx context.Context, tenantID, email string) error {
	plaintext, hash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate invite token: %w", err)
	}
	if err := s.repo.StoreEmailConfirmation(ctx, tenantID, email, hash, time.Now().Add(72*time.Hour)); err != nil {
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

func (s *Service) Connect(ctx context.Context, req ConnectRequest) error {
	// Validate without consuming — token stays usable if a downstream step fails.
	clientID, err := s.validateConnectionToken(ctx, req.ConnectionToken, req.Email)
	if err != nil {
		return err
	}
	exists, err := s.repo.TenantExists(ctx, clientID)
	if err != nil {
		return fmt.Errorf("check tenant: %w", err)
	}
	if !exists {
		return ErrClientNotSetup
	}
	plaintext, hash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	if err := s.repo.StoreEmailConfirmation(ctx, clientID, req.Email, hash, time.Now().Add(24*time.Hour)); err != nil {
		return fmt.Errorf("store confirmation: %w", err)
	}
	// Send email before consuming — only burn the token after the email is out.
	// If send fails, the stored confirmation expires naturally and the user can retry.
	if err := s.sendConfirmationEmail(req.Email, plaintext); err != nil {
		return fmt.Errorf("send confirmation email: %w", err)
	}
	return s.consumeConnectionToken(ctx, req.ConnectionToken, req.Email)
}

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

	urlEnc, anonEnc, srEnc, siteURL, err := s.repo.GetTenantCredentials(ctx, conf.TenantID)
	if err != nil {
		return nil, fmt.Errorf("fetch credentials: %w", err)
	}
	supabaseURL, err := utils.DecryptString(urlEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(anonEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt anon: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(srEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt service role: %w", err)
	}

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
		UserID:                userID,
		TenantID:              conf.TenantID,
		Email:                 conf.Email,
		ClientSupabaseURL:     supabaseURL,
		ClientSupabaseAnonKey: anonKey,
		SiteURL:               siteURL,
	}, nil
}

func (s *Service) consumeConnectionToken(ctx context.Context, token, email string) error {
	body, _ := json.Marshal(map[string]string{"token": token, "email": email})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.agencyURL+"/api/consume-connection-token", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.agencyToken)
	req.Header.Set("X-Client-ID", s.agencyClientID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call agency-hub: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("consume token returned %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) validateConnectionToken(ctx context.Context, token, email string) (string, error) {
	body, _ := json.Marshal(map[string]string{"token": token, "email": email})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.agencyURL+"/api/validate-connection-token", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.agencyToken)
	req.Header.Set("X-Client-ID", s.agencyClientID)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call agency-hub: %w", err)
	}
	defer resp.Body.Close()

	var result validateTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response (status %d): %w", resp.StatusCode, err)
	}
	if !result.Valid {
		switch result.Reason {
		case "expired":
			return "", ErrTokenExpired
		case "used":
			return "", ErrTokenUsed
		default:
			return "", ErrTokenInvalid
		}
	}
	return result.ClientID, nil
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

	// User already exists — look them up rather than failing.
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

// bucketNameFromSiteURL derives a valid Supabase bucket name from the client's site URL.
// e.g. "https://acmecorp.com" → "acmecorp-com"
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

// createDefaultBucket creates a public storage bucket in the client's Supabase project.
// A 409 conflict means the bucket already exists — treated as success.
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
		return nil // already exists
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("create bucket returned %d", resp.StatusCode)
	}
	return nil
}

func (s *Service) sendConfirmationEmail(to, token string) error {
	link := fmt.Sprintf("%s/api/onboarding/confirm?token=%s", s.publicURL, token)
	return s.mailer.Send(context.Background(), to,
		"Confirm your email to access your dashboard",
		fmt.Sprintf(`
			<p>Click the link below to access your dashboard:</p>
			<p><a href="%s">Confirm your email →</a></p>
			<p>This link expires in 24 hours.</p>
		`, link),
	)
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
