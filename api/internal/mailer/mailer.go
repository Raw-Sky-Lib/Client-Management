package mailer

import (
	"context"
	"fmt"
)

// Mailer sends transactional emails.
type Mailer interface {
	Send(ctx context.Context, to, subject, html string) error
}

// New returns the mailer for the given provider.
// provider must be "resend" or "brevo".
func New(provider, from, resendAPIKey, brevoSMTPUser, brevoSMTPKey string) (Mailer, error) {
	switch provider {
	case "resend":
		return NewResendMailer(resendAPIKey, from), nil
	case "brevo":
		return NewBrevoMailer(brevoSMTPUser, brevoSMTPKey, from), nil
	default:
		return nil, fmt.Errorf("unknown mailer provider %q — must be resend or brevo", provider)
	}
}
