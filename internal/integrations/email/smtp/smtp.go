package smtp

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Host     string // e.g. smtp.gmail.com or mailhog
	Port     int    // e.g. 587 or 1025
	Username string // optional
	Password string // optional
	From     string // e.g. "noreply@example.com"
}

type Sender struct {
	cfg Config
}

func New(cfg Config) Sender {
	return Sender{cfg: cfg}
}

func (s Sender) SendConfirm(toEmail string, confirmURL string, unsubscribeURL string) error {
	subject := "Confirm your subscription"
	body := fmt.Sprintf(
		"Please confirm your subscription:\n\n%s\n\nUnsubscribe link:\n%s\n",
		confirmURL,
		unsubscribeURL,
	)
	return s.send(toEmail, subject, body)
}

func (s Sender) SendRelease(toEmail string, repoFullName string, tag string, releaseURL string, unsubscribeURL string) error {
	subject := fmt.Sprintf("New release: %s %s", repoFullName, tag)
	body := fmt.Sprintf(
		"New release detected for %s:\n\nTag: %s\nURL: %s\n\nUnsubscribe:\n%s\n",
		repoFullName,
		tag,
		releaseURL,
		unsubscribeURL,
	)
	return s.send(toEmail, subject, body)
}

func (s Sender) send(toEmail, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)

	msg := strings.Builder{}
	msg.WriteString("From: " + s.cfg.From + "\r\n")
	msg.WriteString("To: " + toEmail + "\r\n")
	msg.WriteString("Subject: " + subject + "\r\n")
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	msg.WriteString("\r\n")
	msg.WriteString(body)

	var auth smtp.Auth
	// MailHog: no auth. Many SMTP providers require auth.
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	return smtp.SendMail(addr, auth, s.cfg.From, []string{toEmail}, []byte(msg.String()))
}