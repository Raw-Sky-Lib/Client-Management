package mailer

import (
	"context"
	"fmt"
	"net/smtp"
)

const (
	brevoSMTPHost = "smtp-relay.brevo.com"
	brevoSMTPPort = "587"
	brevoSMTPAddr = brevoSMTPHost + ":" + brevoSMTPPort
)

type BrevoMailer struct {
	smtpUser string
	smtpKey  string
	from     string
}

func NewBrevoMailer(smtpUser, smtpKey, from string) *BrevoMailer {
	return &BrevoMailer{smtpUser: smtpUser, smtpKey: smtpKey, from: from}
}

func (m *BrevoMailer) Send(_ context.Context, to, subject, html string) error {
	auth := smtp.PlainAuth("", m.smtpUser, m.smtpKey, brevoSMTPHost)

	msg := fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		m.from, to, subject, html,
	)

	return smtp.SendMail(brevoSMTPAddr, auth, m.from, []string{to}, []byte(msg))
}
