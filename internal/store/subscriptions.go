package store

import (
	"context"

	"email-subscription-service/internal/domain"
)

type SubscriptionStore interface {
	// CreatePending creates a new pending subscription for (email, repoFullName).
	// Must return domain.ErrAlreadySubscribed if it already exists (pending or confirmed),
	// to satisfy swagger 409.
	CreatePending(ctx context.Context, email string, repo domain.Repo, confirmToken string, unsubscribeToken string) error

	ConfirmByToken(ctx context.Context, token string) error
	UnsubscribeByToken(ctx context.Context, token string) error

	ListActiveByEmail(ctx context.Context, email string) ([]domain.Subscription, error)
}

