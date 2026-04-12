package domain

import (
	"net/mail"
	"strings"
)

// ParseRepo splits "owner/name". Extra slashes or empty segments return
// ErrInvalidRepoFormat.
func ParseRepo(s string) (Repo, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Repo{}, ErrInvalidRepoFormat
	}
	// Optional: reject extra slashes/whitespace etc; keep minimal now.
	return Repo{Owner: parts[0], Name: parts[1]}, nil
}

// ValidateEmail accepts a single RFC 5322 address via net/mail.
func ValidateEmail(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return ErrInvalidEmail
	}
	_, err := mail.ParseAddress(s)
	if err != nil {
		return ErrInvalidEmail
	}
	return nil
}
