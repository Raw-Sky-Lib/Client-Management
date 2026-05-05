package utils

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ValidateSupabaseCredentials confirms the URL + service role key can reach the Supabase project.
func ValidateSupabaseCredentials(supabaseURL, serviceRoleKey string) error {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequestWithContext(
		context.Background(), http.MethodGet,
		supabaseURL+"/rest/v1/",
		nil,
	)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("cannot reach supabase project: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid service role key")
	}

	return nil
}
