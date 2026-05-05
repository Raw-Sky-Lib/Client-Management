package auth

import "github.com/golang-jwt/jwt/v5"

type PortalClaims struct {
	UserID                string `json:"user_id"`
	TenantID              string `json:"tenant_id"`
	Email                 string `json:"email"`
	ClientSupabaseURL     string `json:"supabase_url"`
	ClientSupabaseAnonKey string `json:"supabase_anon_key"`
	jwt.RegisteredClaims
}
