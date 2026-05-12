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
)

type Service struct {
	repo        *Repository
	httpClient  *http.Client
	mailer      mailer.Mailer
	frontendURL string
	publicURL   string // backend's own URL — used in magic link emails
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

// LoginWithPassword verifies email+password against Supabase Auth and issues a portal JWT pair.
func (s *Service) LoginWithPassword(ctx context.Context, email, password string) (*PortalClaims, error) {
	tenant, err := s.repo.GetTenantByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidCredentials
	}

	supabaseURL, err := utils.DecryptString(tenant.URLEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(tenant.AnonEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt anon: %w", err)
	}

	userID, err := s.verifyPassword(ctx, supabaseURL, anonKey, email, password)
	if err != nil {
		return nil, err
	}

	return &PortalClaims{
		UserID:                userID,
		TenantID:              tenant.TenantID,
		Email:                 email,
		ClientSupabaseURL:     supabaseURL,
		ClientSupabaseAnonKey: anonKey,
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

// SetUserPassword updates the user's password in Supabase Auth via the Admin API.
func (s *Service) SetUserPassword(ctx context.Context, tenantID, userID, password string) error {
	tenant, err := s.repo.GetTenantByID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		return fmt.Errorf("tenant not found")
	}

	supabaseURL, err := utils.DecryptString(tenant.URLEnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt url: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(tenant.SREnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt service role: %w", err)
	}

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
		return fmt.Errorf("supabase update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("supabase update password status %d", resp.StatusCode)
	}
	return nil
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
// The link points to the backend directly — no Supabase redirect URL configuration required.
func (s *Service) RequestMagicLink(ctx context.Context, email string) error {
	tenant, err := s.repo.GetTenantByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		// Silently succeed — don't reveal whether the email is registered.
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

// VerifyLoginToken validates a portal magic-link token, issues a session, and returns the claims.
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
		return nil, fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		return nil, ErrInvalidToken
	}

	supabaseURL, err := utils.DecryptString(tenant.URLEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(tenant.AnonEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt anon: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(tenant.SREnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt service role: %w", err)
	}

	userID, err := s.getSupabaseUserByEmail(ctx, supabaseURL, serviceRoleKey, rec.Email)
	if err != nil {
		return nil, fmt.Errorf("get supabase user: %w", err)
	}

	if err := s.repo.MarkLoginTokenUsed(ctx, rec.ID); err != nil {
		return nil, fmt.Errorf("mark used: %w", err)
	}

	return &PortalClaims{
		UserID:                userID,
		TenantID:              rec.TenantID,
		Email:                 rec.Email,
		ClientSupabaseURL:     supabaseURL,
		ClientSupabaseAnonKey: anonKey,
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

// ExchangeToken verifies a Supabase access token and issues a portal JWT pair.
func (s *Service) ExchangeToken(ctx context.Context, supabaseToken string) (*PortalClaims, error) {
	// Decode (without verifying) to extract email for the tenant lookup.
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

	supabaseURL, err := utils.DecryptString(tenant.URLEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(tenant.AnonEnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt anon: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(tenant.SREnc, s.encKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt service role: %w", err)
	}

	// Verify the Supabase token is legitimate by calling /auth/v1/user.
	userID, err := s.verifySupabaseToken(ctx, supabaseURL, serviceRoleKey, supabaseToken)
	if err != nil {
		return nil, err
	}

	return &PortalClaims{
		UserID:                userID,
		TenantID:              tenant.TenantID,
		Email:                 email,
		ClientSupabaseURL:     supabaseURL,
		ClientSupabaseAnonKey: anonKey,
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

// RefreshAccessToken parses a refresh token and re-issues an access token.
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
	if err != nil {
		return "", fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		return "", ErrInvalidToken
	}

	supabaseURL, err := utils.DecryptString(tenant.URLEnc, s.encKey)
	if err != nil {
		return "", fmt.Errorf("decrypt url: %w", err)
	}
	anonKey, err := utils.DecryptString(tenant.AnonEnc, s.encKey)
	if err != nil {
		return "", fmt.Errorf("decrypt anon: %w", err)
	}

	return s.issueAccessToken(&PortalClaims{
		UserID:                claims.UserID,
		TenantID:              claims.TenantID,
		Email:                 claims.Email,
		ClientSupabaseURL:     supabaseURL,
		ClientSupabaseAnonKey: anonKey,
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
