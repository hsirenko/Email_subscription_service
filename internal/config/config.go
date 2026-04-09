package config

import (
	"os"
	"time"
)

type Config struct {
	Port string
	DatabaseURL string
	PublicURL  string
	GitHubToken string // os.Getenv("GITHUB_TOKEN")
	EmailDriver string
	SMTPHost    string
	SMTPPort    string
	SMTPUser    string
	SMTPPass    string
	SMTPFrom    string
	ScanInterval time.Duration
}




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

	return Config{
		Port:        port,
		DatabaseURL: dbURL,
		PublicURL:   publicURL,
		GitHubToken: ghToken,
		EmailDriver: emailDriver,
		SMTPHost:    smtpHost,
		SMTPPort:    smtpPort,
		SMTPUser:    smtpUser,
		SMTPPass:    smtpPass,
		SMTPFrom:    smtpFrom,
		ScanInterval: scanInterval,
	}
}

func (c Config) HTTPAddr() string {
	return ":" + c.Port
}