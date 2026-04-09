package domain

import (
	"net/mail"
	"strings"
)

func ParseRepo(s string) (Repo, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return Repo{}, ErrInvalidRepoFormat
	}
	// Optional: reject extra slashes/whitespace etc; keep minimal now.
	return Repo{Owner: parts[0], Name: parts[1]}, nil
}

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