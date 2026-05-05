package onboarding

import "time"

type ConnectRequest struct {
	ConnectionToken string `json:"connection_token" validate:"required"       example:"abc123def456"`
	Email           string `json:"email"            validate:"required,email" example:"client@example.com"`
}

// RegisterClientRequest is sent by agency-hub when a client is set up.
type RegisterClientRequest struct {
	ClientID                     string `json:"client_id"                        validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ClientSupabaseURL            string `json:"client_supabase_url"              validate:"required,url" example:"https://abcdef.supabase.co"`
	ClientSupabaseAnonKey        string `json:"client_supabase_anon_key"         validate:"required"     example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ClientSupabaseServiceRoleKey string `json:"client_supabase_service_role_key" validate:"required"     example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ClientSupabaseDBURL          string `json:"client_supabase_db_url"           validate:"required"     example:"postgresql://postgres:password@db.abcdef.supabase.co:5432/postgres"`
	SiteURL                      string `json:"site_url"                         validate:"required,url" example:"https://client-site.com"`
}

// validateTokenResponse is the shape returned by agency-hub's validate-connection-token endpoint.
type validateTokenResponse struct {
	Valid     bool   `json:"valid"`
	Reason    string `json:"reason"`    // "expired" | "used" | "invalid"
	ClientID  string `json:"client_id"` // populated when valid=true
}

// EmailConfirmation maps to the email_confirmations table.
type EmailConfirmation struct {
	ID        string
	TenantID  string
	Email     string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
