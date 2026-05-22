package onboarding

import (
	"errors"
	"time"
)

var (
	ErrBadSupabaseCredentials = errors.New("invalid supabase credentials")
	ErrBadDBURL               = errors.New("database migration failed")
	ErrProjectNotFound        = errors.New("project not registered in portal")
)

// SyncCredentialsRequest is the body for PATCH /api/admin/projects/{agency_project_id}/credentials.
// All fields are optional — only non-empty fields overwrite the existing stored values.
// Used for already-registered projects where only credentials need updating (no migrations, no email).
type SyncCredentialsRequest struct {
	AgencyProjectID              string `json:"agency_project_id"`
	Name                         string `json:"name"`
	SiteURL                      string `json:"site_url"`
	ClientSupabaseURL            string `json:"client_supabase_url"`
	ClientSupabaseAnonKey        string `json:"client_supabase_anon_key"`
	ClientSupabaseServiceRoleKey string `json:"client_supabase_service_role_key"`
	ClientSupabaseDBURL          string `json:"client_supabase_db_url"`
	RevalidateSecret             string `json:"revalidate_secret"`
}

// RegisterClientRequest is sent by agency-hub when a project is registered with the portal.
type RegisterClientRequest struct {
	ClientID                     string `json:"client_id"                        validate:"required,uuid"   example:"550e8400-e29b-41d4-a716-446655440000"`
	ProjectID                    string `json:"project_id"                       validate:"required,uuid"   example:"660e8400-e29b-41d4-a716-446655440111"`
	ProjectName                  string `json:"project_name"                     validate:"required"        example:"Website Redesign"`
	Email                        string `json:"email"                            validate:"required,email"  example:"client@example.com"`
	ClientSupabaseURL            string `json:"client_supabase_url"              validate:"required,url"    example:"https://abcdef.supabase.co"`
	ClientSupabaseAnonKey        string `json:"client_supabase_anon_key"         validate:"required"        example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ClientSupabaseServiceRoleKey string `json:"client_supabase_service_role_key" validate:"required"        example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	ClientSupabaseDBURL          string `json:"client_supabase_db_url"           validate:"required"        example:"postgresql://postgres:password@db.abcdef.supabase.co:5432/postgres"`
	SiteURL                      string `json:"site_url"                         validate:"omitempty,url"   example:"https://client-site.com"`
	RevalidateSecret             string `json:"revalidate_secret"                validate:"omitempty"       example:"abc123..."`
}

// RegisteredResponse is returned by POST /api/admin/register-client.
type RegisteredResponse struct {
	Registered bool `json:"registered" example:"true"`
}

// ResendInviteRequest is the body for POST /api/admin/resend-invite.
type ResendInviteRequest struct {
	ClientID string `json:"client_id" validate:"required,uuid"  example:"550e8400-e29b-41d4-a716-446655440000"`
	Email    string `json:"email"     validate:"required,email" example:"client@example.com"`
}

// EmailConfirmation maps to the email_confirmations table.
type EmailConfirmation struct {
	ID        string
	TenantID  string
	ProjectID *string // which tenant_project this confirmation is for
	Email     string
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}
