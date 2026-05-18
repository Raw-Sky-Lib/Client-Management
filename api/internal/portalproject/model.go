package portalproject

// ProjectEntry is the shape returned by GET /api/projects.
// The anon key is safe to include — it's the public client key.
// The service role key is never returned.
type ProjectEntry struct {
	ID              string `json:"id"`               // tenant_projects.id (portal UUID)
	AgencyProjectID string `json:"agency_project_id"` // project ID from agency-hub
	Name            string `json:"name"`
	SiteURL         string `json:"site_url,omitempty"`
	SupabaseURL     string `json:"supabase_url"`
	SupabaseAnonKey string `json:"supabase_anon_key"`
}

// ListProjectsResponse wraps the project list.
type ListProjectsResponse struct {
	Projects []ProjectEntry `json:"projects"`
}
