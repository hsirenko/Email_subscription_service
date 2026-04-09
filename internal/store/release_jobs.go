package store

import (
	"context"

	"email-subscription-service/internal/domain"
)

type ReleaseJobStore interface {
	ListActiveSubscriptionsGroupedByRepo(ctx context.Context) (map[string][]domain.ActiveSubscription, error)
	GetRepoState(ctx context.Context, repoFullName string) (lastSeenTag string, err error)
	UpsertRepoState(ctx context.Context, repoFullName string, lastSeenTag string) error

	EnqueueNotification(ctx context.Context, subscriptionID int64, releaseTag string) (enqueued bool, err error)
	MarkNotificationSent(ctx context.Context, subscriptionID int64, releaseTag string) error
	MarkNotificationFailed(ctx context.Context, subscriptionID int64, releaseTag string, lastError string) error

	ListPendingNotifications(ctx context.Context, limit int) ([]domain.PendingNotification, error)
}