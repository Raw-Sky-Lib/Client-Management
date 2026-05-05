package onboarding

import "time"

type ConnectRequest struct {
	ConnectionToken string `json:"connection_token" validate:"required"`
	Email           string `json:"email"            validate:"required,email"`
}

// RegisterClientRequest is sent by agency-hub when a client is set up.
type RegisterClientRequest struct {
	ClientID                     string `json:"client_id"                       validate:"required,uuid"`
	ClientSupabaseURL            string `json:"client_supabase_url"             validate:"required,url"`
	ClientSupabaseAnonKey        string `json:"client_supabase_anon_key"        validate:"required"`
	ClientSupabaseServiceRoleKey string `json:"client_supabase_service_role_key" validate:"required"`
	ClientSupabaseDBURL          string `json:"client_supabase_db_url"          validate:"required"`
	SiteURL                      string `json:"site_url"                        validate:"required,url"`
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
