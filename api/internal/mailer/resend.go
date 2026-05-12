package mailer

import (
	"context"

	"github.com/resend/resend-go/v2"
)

type ResendMailer struct {
	client *resend.Client
	from   string
}

func NewResendMailer(apiKey, from string) *ResendMailer {
	return &ResendMailer{client: resend.NewClient(apiKey), from: from}
}

func (m *ResendMailer) Send(_ context.Context, to, subject, html string) error {
	_, err := m.client.Emails.Send(&resend.SendEmailRequest{
		From:    m.from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	})
	return err
}
