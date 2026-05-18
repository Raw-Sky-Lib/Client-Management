package portalproject

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type rawProject struct {
	ID              string
	AgencyProjectID string
	Name            string
	SiteURL         string
	URLEnc          string
	AnonEnc         string
}

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// GetProjectsForTenant returns all tenant_projects rows for the given tenant,
// ordered oldest-first so the primary project is always index 0.
func (r *Repository) GetProjectsForTenant(ctx context.Context, tenantID string) ([]rawProject, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, agency_project_id, name,
		       COALESCE(site_url, ''),
		       supabase_url_encrypted,
		       supabase_anon_encrypted
		FROM tenant_projects
		WHERE tenant_id = $1
		ORDER BY created_at ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	var out []rawProject
	for rows.Next() {
		var p rawProject
		if err := rows.Scan(&p.ID, &p.AgencyProjectID, &p.Name, &p.SiteURL, &p.URLEnc, &p.AnonEnc); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
