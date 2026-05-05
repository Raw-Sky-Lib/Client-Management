package startup

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ValidateManagementToken calls agency-hub at startup to confirm this portal instance
// is registered and its management token is valid. Retries 3× with 2s backoff to
// handle Railway cold-start delays. Returns an error — caller is responsible for os.Exit(1).
func ValidateManagementToken(agencyURL, managementToken, clientID string, client *http.Client) error {
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		if err := doValidate(agencyURL, managementToken, clientID, client); err != nil {
			lastErr = err
			if attempt < 3 {
				time.Sleep(2 * time.Second)
			}
			continue
		}
		return nil
	}
	return fmt.Errorf("validation failed after 3 attempts: %w", lastErr)
}

func doValidate(agencyURL, managementToken, clientID string, client *http.Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, agencyURL+"/api/validate-management-token", nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+managementToken)
	req.Header.Set("X-Client-ID", clientID)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("invalid management token (401)")
	case http.StatusForbidden:
		return fmt.Errorf("client is not active (403)")
	default:
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
}
