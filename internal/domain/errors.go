package domain

import "errors"

// Sentinel errors returned by validation, services, and stores. Handlers map
// these to HTTP status codes to match swagger.yaml.
var (
	ErrInvalidRepoFormat = errors.New("invalid repo format")
	ErrInvalidEmail      = errors.New("invalid email")
	ErrAlreadySubscribed = errors.New("already subscribed")
	ErrRepoNotFound      = errors.New("repo not found")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenNotFound     = errors.New("token not found")
)
