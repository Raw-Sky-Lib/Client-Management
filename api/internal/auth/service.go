package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/DagMT/client-portal/internal/utils"
	"github.com/golang-jwt/jwt/v5"
	"github.com/resend/resend-go/v2"
)

var (
	ErrEmailNotRegistered = errors.New("email not registered")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type Service struct {
	repo        *Repository
	httpClient  *http.Client
	resend      *resend.Client
	resendFrom  string
	frontendURL string
	encKey      []byte
	jwtSecret   string
	accessExp   time.Duration
	refreshExp  time.Duration
	secure      bool
}

func NewService(
	repo *Repository,
	httpClient *http.Client,
	resendClient *resend.Client,
	resendFrom string,
	frontendURL string,
	encKey []byte,
	jwtSecret string,
	accessExp, refreshExp time.Duration,
	secure bool,
) *Service {
	return &Service{
		repo:        repo,
		httpClient:  httpClient,
		resend:      resendClient,
		resendFrom:  resendFrom,
		frontendURL: frontendURL,
		encKey:      encKey,
		jwtSecret:   jwtSecret,
		accessExp:   accessExp,
		refreshExp:  refreshExp,
		secure:      secure,
	}
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

// RequestMagicLink generates a Supabase magic link via the Admin API and sends it via Resend.
func (s *Service) RequestMagicLink(ctx context.Context, email string) error {
	tenant, err := s.repo.GetTenantByEmail(ctx, email)
	if err != nil {
		return fmt.Errorf("lookup tenant: %w", err)
	}
	if tenant == nil {
		// Don't reveal that the email isn't registered — log and return nil so the
		// handler can send a generic "check your email" response.
		slog.Info("magic link requested for unregistered email", slog.String("email", email))
		return nil
	}

	supabaseURL, err := utils.DecryptString(tenant.URLEnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt url: %w", err)
	}
	serviceRoleKey, err := utils.DecryptString(tenant.SREnc, s.encKey)
	if err != nil {
		return fmt.Errorf("decrypt service role: %w", err)
	}

	actionLink, err := s.generateMagicLink(ctx, supabaseURL, serviceRoleKey, email)
	if err != nil {
		return fmt.Errorf("generate magic link: %w", err)
	}
	return s.sendMagicLinkEmail(email, actionLink)
}

func (s *Service) generateMagicLink(ctx context.Context, supabaseURL, serviceRoleKey, email string) (string, error) {
	body, _ := json.Marshal(map[string]any{
		"type":  "magiclink",
		"email": email,
		"options": map[string]string{
			"redirect_to": s.frontendURL + "/auth/callback",
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		supabaseURL+"/auth/v1/admin/generate_link", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("supabase generate_link: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("supabase generate_link status %d", resp.StatusCode)
	}

	var result struct {
		ActionLink string `json:"action_link"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}
	if result.ActionLink == "" {
		return "", fmt.Errorf("supabase returned empty action_link")
	}
	return result.ActionLink, nil
}

func (s *Service) sendMagicLinkEmail(to, actionLink string) error {
	_, err := s.resend.Emails.Send(&resend.SendEmailRequest{
		From:    s.resendFrom,
		To:      []string{to},
		Subject: "Your sign-in link",
		Html: fmt.Sprintf(`
			<p>Click the link below to sign in to your dashboard:</p>
			<p><a href="%s">Sign in →</a></p>
			<p>This link expires in 1 hour and can only be used once.</p>
		`, actionLink),
	})
	return err
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
