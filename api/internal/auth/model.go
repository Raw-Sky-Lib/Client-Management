package auth

import (
	"context"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

type PortalClaims struct {
	UserID                string `json:"user_id"`
	TenantID              string `json:"tenant_id"`
	Email                 string `json:"email"`
	ClientSupabaseURL     string `json:"supabase_url"`
	ClientSupabaseAnonKey string `json:"supabase_anon_key"`
	jwt.RegisteredClaims
}

type RefreshClaims struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}

// JWTIssuer is implemented by the auth Service and injected into the onboarding handler.
type JWTIssuer interface {
	IssueTokenPair(w http.ResponseWriter, claims *PortalClaims) error
}

// MagicLinkRequest is the body for POST /api/auth/magic-link.
type MagicLinkRequest struct {
	Email string `json:"email" validate:"required,email" example:"client@example.com"`
}

// ExchangeRequest is the body for POST /api/auth/exchange.
type ExchangeRequest struct {
	AccessToken string `json:"access_token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// ProfileResponse is the body for GET /api/auth/profile.
type ProfileResponse struct {
	UserID           string `json:"user_id"           example:"550e8400-e29b-41d4-a716-446655440000"`
	TenantID         string `json:"tenant_id"         example:"550e8400-e29b-41d4-a716-446655440001"`
	Email            string `json:"email"             example:"client@example.com"`
	SupabaseURL      string `json:"supabase_url"      example:"https://abcdef.supabase.co"`
	SupabaseAnonKey  string `json:"supabase_anon_key" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// CSRFResponse is the body for GET /api/auth/csrf.
type CSRFResponse struct {
	CSRFToken string `json:"csrf_token" example:"a1b2c3d4e5f6..."`
}

type contextKey struct{}

func WithClaims(ctx context.Context, claims *PortalClaims) context.Context {
	return context.WithValue(ctx, contextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) (*PortalClaims, bool) {
	claims, ok := ctx.Value(contextKey{}).(*PortalClaims)
	return claims, ok
}
