package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/DagMT/client-portal/internal/mailer"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmailNotRegistered = errors.New("email not registered")
	ErrInvalidToken       = errors.New("invalid or expired token")
	ErrInvalidCredentials = errors.New("invalid email or password")
	// ErrPortalNotReady is returned when a password reset is attempted before the
	// client's Supabase project credentials have been configured in the portal.
	// Password auth is Supabase-backed, so it cannot work without a project.
	ErrPortalNotReady = errors.New("portal not ready")
)

type Service struct {
	repo        *Repository
	httpClient  *http.Client
	mailer      mailer.Mailer
	frontendURL string
	publicURL   string
	encKey      []byte
	jwtSecret   string
	accessExp   time.Duration
	refreshExp  time.Duration
	secure      bool
}

func NewService(
	repo *Repository,
	httpClient *http.Client,
	m mailer.Mailer,
	frontendURL string,
	publicURL string,
	encKey []byte,
	jwtSecret string,
	accessExp, refreshExp time.Duration,
	secure bool,
) *Service {
	return &Service{
		repo:        repo,
		httpClient:  httpClient,
		mailer:      m,
		frontendURL: frontendURL,
		publicURL:   publicURL,
		encKey:      encKey,
		jwtSecret:   jwtSecret,
		accessExp:   accessExp,
		refreshExp:  refreshExp,
		secure:      secure,
	}
}

// LoginWithPassword verifies email+password against the client's first Supabase project
// and issues a portal JWT pair.
func (s *Service) LoginWithPassword(ctx context.Context, email, password string) (*PortalClaims, error) {
	tenant, err := s.repo.GetTenantByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidCredentials
	}

	proj, err := s.repo.GetFirstProjectForTenant(ctx, tenant.TenantID)
	if err != nil {
		return nil, fmt.Errorf("lookup project: %w", err)
	}
	if proj == nil {
		return nil, ErrInvalidCredentials
	}

	supabaseURL, err := utils.DecryptString(proj.URLEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(proj.AnonEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt anon: %w", err)
	}

	userID, err := s.verifyPassword(ctx, supabaseURL, anonKey, email, password)
	if err != nil {
		return nil, err
	}

	return &PortalClaims{
		UserID:   userID,
		TenantID: tenant.TenantID,
		Email:    email,
	}, nil
}

func (s *Service) verifyPassword(ctx context.Context, supabaseURL, anonKey, email, password string) (string, error) {
	body, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		supabaseURL+"/auth/v1/token?grant_type=password", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", anonKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase auth: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
		return "", ErrInvalidCredentials
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("supabase password auth status %d", resp.StatusCode)
	}

	var result struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if result.User.ID == "" {
		return "", fmt.Errorf("supabase returned empty user id")
	}
	return result.User.ID, nil
}

// SetUserPassword updates the user's password in their first Supabase project.
func (s *Service) SetUserPassword(ctx context.Context, tenantID, email, password string) error {
	proj, err := s.repo.GetFirstProjectForTenant(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("lookup project: %w", err)
	}
	if proj == nil {
		return ErrPortalNotReady
	}
	supabaseURL, err := utils.DecryptString(proj.URLEnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt url: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(proj.SREnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt service role: %w", err)
	}
	userID, err := s.getSupabaseUserByEmail(ctx, supabaseURL, serviceRoleKey, email)
	if err != nil {
		return fmt.Errorf("get supabase user: %w", err)
	}
	return s.updateSupabasePassword(ctx, supabaseURL, serviceRoleKey, userID, password)
}

func (s *Service) updateSupabasePassword(ctx context.Context, supabaseURL, serviceRoleKey, userID, password string) error {
	body, _ := json.Marshal(map[string]string{"password": password})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut,
		supabaseURL+"/auth/v1/admin/users/"+userID, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("supabase update password: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("supabase update password status %d", resp.StatusCode)
	}
	return nil
}

// RequestPasswordReset generates a reset token and emails a password reset link.
func (s *Service) RequestPasswordReset(ctx context.Context, email string) error {
	tenant, err := s.repo.GetTenantByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		slog.Info("password reset requested for unregistered email", slog.String("email", email))
		return nil
	}
	plaintext, hash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	if err := s.repo.StoreLoginToken(ctx, tenant.TenantID, email, hash, time.Now().Add(time.Hour)); err != nil {
		return fmt.Errorf("store reset token: %w", err)
	}
	link := fmt.Sprintf("%s/api/auth/reset-password/verify?token=%s", s.publicURL, plaintext)
	return s.mailer.Send(context.Background(), email,
		"Reset your password",
		fmt.Sprintf(`
			<p>You requested a password reset for your client dashboard.</p>
			<p><a href="%s">Reset password →</a></p>
			<p>This link expires in 1 hour. If you didn't request this, you can safely ignore it.</p>
		`, link),
	)
}

// ValidateResetToken checks a reset token is valid without consuming it.
func (s *Service) ValidateResetToken(ctx context.Context, token string) (bool, error) {
	rec, err := s.repo.GetLoginToken(ctx, hashToken(token))
	if err != nil {
		return false, fmt.Errorf("lookup token: %w", err)
	}
	if rec == nil || rec.UsedAt != nil || time.Now().After(rec.ExpiresAt) {
		return false, nil
	}
	return true, nil
}

// ConfirmPasswordReset validates the token and sets the new password.
// It does not issue a session — the user must sign in after resetting.
func (s *Service) ConfirmPasswordReset(ctx context.Context, token, password string) error {
	rec, err := s.repo.GetLoginToken(ctx, hashToken(token))
	if err != nil {
		return fmt.Errorf("lookup token: %w", err)
	}
	if rec == nil || rec.UsedAt != nil || time.Now().After(rec.ExpiresAt) {
		return ErrInvalidToken
	}

	proj, err := s.repo.GetFirstProjectForTenant(ctx, rec.TenantID)
	if err != nil {
		return fmt.Errorf("lookup project: %w", err)
	}
	if proj == nil {
		// No Supabase project configured yet — password auth is unavailable.
		// Client should use the magic-link sign-in or wait for the agency to push credentials.
		return ErrPortalNotReady
	}

	supabaseURL, err := utils.DecryptString(proj.URLEnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt url: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(proj.SREnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt service role: %w", err)
	}

	// Ensure the Supabase user exists — createSupabaseUser during RegisterClient is
	// non-fatal, so the user may not have been created if it failed silently.
	userID, err := s.ensureSupabaseUser(ctx, supabaseURL, serviceRoleKey, rec.Email)
	if err != nil {
		return fmt.Errorf("ensure supabase user: %w", err)
	}
	if err := s.updateSupabasePassword(ctx, supabaseURL, serviceRoleKey, userID, password); err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	return s.repo.MarkLoginTokenUsed(ctx, rec.ID)
}

// ensureSupabaseUser returns the Supabase user ID for email, creating the user if absent.
// This guards against the case where createSupabaseUser failed silently during RegisterClient.
func (s *Service) ensureSupabaseUser(ctx context.Context, supabaseURL, serviceRoleKey, email string) (string, error) {
	userID, err := s.getSupabaseUserByEmail(ctx, supabaseURL, serviceRoleKey, email)
	if err == nil {
		return userID, nil
	}
	// User not found — create with a random password (email_confirm bypasses verification).
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
		return "", fmt.Errorf("build create-user request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase create user: %w", err)
	}
	defer resp.Body.Close()

	// 422 = already exists (race condition) — re-fetch.
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

// IssueTokenPair implements JWTIssuer — called by the onboarding handler after Confirm.
func (s *Service) IssueTokenPair(w http.ResponseWriter, claims *PortalClaims) error {
	accessToken, err := s.issueAccessToken(claims)
	if err != nil {
		return fmt.Errorf("issue access token: %w", err)
	}
	refreshToken, err := s.issueRefreshToken(claims.UserID, claims.TenantID, claims.Email)
	if err != nil {
		return fmt.Errorf("issue refresh token: %w", err)
	}
	SetAuthCookies(w, accessToken, refreshToken, s.secure, s.refreshExp)
	return nil
}

// RequestMagicLink generates a portal-native login token and emails a sign-in link.
func (s *Service) RequestMagicLink(ctx context.Context, email string) error {
	tenant, err := s.repo.GetTenantByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		slog.Info("magic link requested for unregistered email", slog.String("email", email))
		return nil
	}

	plaintext, hash, err := generateToken()
	if err != nil {
		return fmt.Errorf("generate token: %w", err)
	}
	if err := s.repo.StoreLoginToken(ctx, tenant.TenantID, email, hash, time.Now().Add(time.Hour)); err != nil {
		return fmt.Errorf("store login token: %w", err)
	}
	return s.sendLoginLinkEmail(email, plaintext)
}

func (s *Service) sendLoginLinkEmail(to, token string) error {
	link := fmt.Sprintf("%s/api/auth/login/verify?token=%s", s.publicURL, token)
	return s.mailer.Send(context.Background(), to,
		"Your sign-in link",
		fmt.Sprintf(`
			<p>Click the link below to sign in to your dashboard:</p>
			<p><a href="%s">Sign in →</a></p>
			<p>This link expires in 1 hour and can only be used once.</p>
		`, link),
	)
}

// VerifyLoginToken validates a portal magic-link token and issues a session.
// UserID is set to the tenant_users.id (portal UUID) — no Supabase roundtrip needed.
func (s *Service) VerifyLoginToken(ctx context.Context, token string) (*PortalClaims, error) {
	rec, err := s.repo.GetLoginToken(ctx, hashToken(token))
	if err != nil {
		return nil, fmt.Errorf("lookup token: %w", err)
	}
	if rec == nil || rec.UsedAt != nil || time.Now().After(rec.ExpiresAt) {
		return nil, ErrInvalidToken
	}

	tenant, err := s.repo.GetTenantByID(ctx, rec.TenantID)
	if err != nil {
		return nil, fmt.Errorf("verify tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidToken
	}

	userID, err := s.repo.GetTenantUserID(ctx, rec.TenantID, rec.Email)
	if err != nil {
		return nil, fmt.Errorf("get user id: %w", err)
	}

	if err := s.repo.MarkLoginTokenUsed(ctx, rec.ID); err != nil {
		return nil, fmt.Errorf("mark used: %w", err)
	}

	return &PortalClaims{
		UserID:   userID,
		TenantID: rec.TenantID,
		Email:    rec.Email,
	}, nil
}

func (s *Service) getSupabaseUserByEmail(ctx context.Context, supabaseURL, serviceRoleKey, email string) (string, error) {
	endpoint := supabaseURL + "/auth/v1/admin/users?email=" + url.QueryEscape(email) + "&per_page=1"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
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
		return "", fmt.Errorf("supabase user not found for email %s", email)
	}
	return result.Users[0].ID, nil
}

// ExchangeToken verifies a Supabase access token and issues a portal JWT pair.
func (s *Service) ExchangeToken(ctx context.Context, supabaseToken string) (*PortalClaims, error) {
	rawClaims := jwt.MapClaims{}
	if _, _, err := jwt.NewParser().ParseUnverified(supabaseToken, rawClaims); err != nil {
		return nil, ErrInvalidToken
	}
	email, _ := rawClaims["email"].(string)
	if email == "" {
		return nil, ErrInvalidToken
	}

	tenant, err := s.repo.GetTenantByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidToken
	}

	proj, err := s.repo.GetFirstProjectForTenant(ctx, tenant.TenantID)
	if err != nil {
		return nil, fmt.Errorf("lookup project: %w", err)
	}
	if proj == nil {
		return nil, ErrInvalidToken
	}

	supabaseURL, err := utils.DecryptString(proj.URLEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(proj.SREnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt service role: %w", err)
	}

	userID, err := s.verifySupabaseToken(ctx, supabaseURL, serviceRoleKey, supabaseToken)
	if err != nil {
		return nil, err
	}

	return &PortalClaims{
		UserID:   userID,
		TenantID: tenant.TenantID,
		Email:    email,
	}, nil
}

func (s *Service) verifySupabaseToken(ctx context.Context, supabaseURL, serviceRoleKey, supabaseToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		supabaseURL+"/auth/v1/user", nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+supabaseToken)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase user lookup: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", ErrInvalidToken
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("supabase user lookup status %d", resp.StatusCode)
	}

	var user struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return "", fmt.Errorf("decode user: %w", err)
	}
	if user.ID == "" {
		return "", fmt.Errorf("supabase returned empty user id")
	}
	return user.ID, nil
}

// RefreshAccessToken parses a refresh token, verifies the tenant still exists,
// and re-issues a lean access token. The tenant check ensures deleted clients
// cannot continue refreshing after deregistration.
func (s *Service) RefreshAccessToken(ctx context.Context, refreshToken string) (string, error) {
	claims := &RefreshClaims{}
	_, err := jwt.ParseWithClaims(refreshToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil {
		return "", ErrInvalidToken
	}

	tenant, err := s.repo.GetTenantByID(ctx, claims.TenantID)
	if err != nil || tenant == nil {
		return "", ErrInvalidToken
	}

	return s.issueAccessToken(&PortalClaims{
		UserID:   claims.UserID,
		TenantID: claims.TenantID,
		Email:    claims.Email,
	})
}

func (s *Service) Logout(w http.ResponseWriter) {
	ClearAuthCookies(w)
}

func (s *Service) issueAccessToken(claims *PortalClaims) (string, error) {
	now := time.Now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExp)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
}

func (s *Service) issueRefreshToken(userID, tenantID, email string) (string, error) {
	now := time.Now()
	claims := &RefreshClaims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshExp)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.jwtSecret))
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

func hashToken(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return fmt.Sprintf("%x", h)
}
