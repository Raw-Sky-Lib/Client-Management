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
	"time"

	"github.com/DagMT/client-portal/internal/auth"
	"github.com/DagMT/client-portal/internal/database"
	"github.com/DagMT/client-portal/internal/utils"
	"github.com/resend/resend-go/v2"
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
	resend         *resend.Client
	resendFrom     string
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
	resendClient *resend.Client,
	resendFrom string,
	agencyURL, agencyToken, agencyClientID string,
	encKey []byte,
	publicURL, frontendURL string,
) *Service {
	return &Service{
		repo:           repo,
		httpClient:     httpClient,
		resend:         resendClient,
		resendFrom:     resendFrom,
		agencyURL:      agencyURL,
		agencyToken:    agencyToken,
		agencyClientID: agencyClientID,
		encKey:         encKey,
		publicURL:      publicURL,
		frontendURL:    frontendURL,
	}
}

func (s *Service) RegisterClient(ctx context.Context, req RegisterClientRequest) error {
	if err := utils.ValidateSupabaseCredentials(req.ClientSupabaseURL, req.ClientSupabaseServiceRoleKey); err != nil {
		return fmt.Errorf("invalid supabase credentials: %w", err)
	}
	if err := database.MigrateClientDB(req.ClientSupabaseDBURL); err != nil {
		return fmt.Errorf("client db migration: %w", err)
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
	return s.repo.UpsertTenant(ctx, req.ClientID, urlEnc, anonEnc, srEnc, dbEnc, req.SiteURL)
}

func (s *Service) Connect(ctx context.Context, req ConnectRequest) error {
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
	return s.sendConfirmationEmail(req.Email, plaintext)
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

	urlEnc, anonEnc, srEnc, err := s.repo.GetTenantCredentials(ctx, conf.TenantID)
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
	}, nil
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
		return "", fmt.Errorf("decode response: %w", err)
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

func (s *Service) sendConfirmationEmail(to, token string) error {
	link := fmt.Sprintf("%s/api/onboarding/confirm?token=%s", s.publicURL, token)
	_, err := s.resend.Emails.Send(&resend.SendEmailRequest{
		From:    s.resendFrom,
		To:      []string{to},
		Subject: "Confirm your email to access your dashboard",
		Html: fmt.Sprintf(`
			<p>Click the link below to access your dashboard:</p>
			<p><a href="%s">Confirm your email →</a></p>
			<p>This link expires in 24 hours.</p>
		`, link),
	})
	return err
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
