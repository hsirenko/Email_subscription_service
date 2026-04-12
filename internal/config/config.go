package config

import (
	"os"
	"strings"
	"time"
)

// Config holds process-wide settings read once at startup.
type Config struct {
	Port         string
	DatabaseURL  string
	PublicURL    string
	GitHubToken  string
	EmailDriver  string
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPFrom     string
	ScanInterval time.Duration

	// CORSAllowedOrigins is parsed from CORS_ALLOWED_ORIGINS (comma-separated).
	CORSAllowedOrigins []string
	// CORSAllowVercelSubdomains allows https origins whose host ends with .vercel.app.
	CORSAllowVercelSubdomains bool
	// WebUIURL is the static subscribe app (e.g. Vercel). Used for HTML confirm flow links.
	WebUIURL string
}

// FromEnv reads configuration from the process environment.
func FromEnv() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/releases?sslmode=disable"
	}
	publicURL := os.Getenv("PUBLIC_URL")
	if publicURL == "" {
		publicURL = "http://localhost:" + port
	}
	ghToken := os.Getenv("GITHUB_TOKEN")

	emailDriver := os.Getenv("EMAIL_DRIVER")
	if emailDriver == "" {
		emailDriver = "log"
	}
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "mailhog"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "1025"
	}
	smtpUser := os.Getenv("SMTP_USERNAME")
	smtpPass := os.Getenv("SMTP_PASSWORD")
	smtpFrom := os.Getenv("SMTP_FROM")
	if smtpFrom == "" {
		smtpFrom = "noreply@local"
	}

	scanIntervalRaw := os.Getenv("SCAN_INTERVAL")
	if scanIntervalRaw == "" {
		scanIntervalRaw = "2m"
	}
	scanInterval, err := time.ParseDuration(scanIntervalRaw)
	if err != nil {
		scanInterval = 2 * time.Minute
	}

	corsOrigins := parseCommaList(os.Getenv("CORS_ALLOWED_ORIGINS"))
	corsVercel := strings.EqualFold(strings.TrimSpace(os.Getenv("CORS_ALLOW_VERCEL_SUBDOMAINS")), "1") ||
		strings.EqualFold(strings.TrimSpace(os.Getenv("CORS_ALLOW_VERCEL_SUBDOMAINS")), "true")

	webUI := strings.TrimSpace(os.Getenv("WEB_UI_URL"))

	return Config{
		Port:                      port,
		DatabaseURL:               dbURL,
		PublicURL:                 publicURL,
		GitHubToken:               ghToken,
		EmailDriver:               emailDriver,
		SMTPHost:                  smtpHost,
		SMTPPort:                  smtpPort,
		SMTPUser:                  smtpUser,
		SMTPPass:                  smtpPass,
		SMTPFrom:                  smtpFrom,
		ScanInterval:              scanInterval,
		CORSAllowedOrigins:        corsOrigins,
		CORSAllowVercelSubdomains: corsVercel,
		WebUIURL:                  webUI,
	}
}

func parseCommaList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c Config) HTTPAddr() string {
	return ":" + c.Port
}
