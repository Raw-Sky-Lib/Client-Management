package tenant

import "context"

// Config holds the fully decrypted Supabase credentials for a tenant.
// Injected into request context by ResolveTenant middleware.
type Config struct {
	TenantID        string
	SupabaseURL     string
	SupabaseAnonKey string
	ServiceRoleKey  string
	SiteURL         string
}

type contextKey struct{}

func WithConfig(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, contextKey{}, cfg)
}

func ConfigFromContext(ctx context.Context) (*Config, bool) {
	cfg, ok := ctx.Value(contextKey{}).(*Config)
	return cfg, ok
}
