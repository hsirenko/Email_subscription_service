package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/store"
)

var _ store.ReleaseJobStore = (*Store)(nil)

func (s *Store) ListActiveSubscriptionsGroupedByRepo(ctx context.Context) (map[string][]domain.ActiveSubscription, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT id, email, repo_full_name, unsubscribe_token
		FROM subscriptions
		WHERE status = 'active'
		ORDER BY repo_full_name, id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string][]domain.ActiveSubscription)
	for rows.Next() {
		var r domain.ActiveSubscription
		if err := rows.Scan(&r.ID, &r.Email, &r.RepoFullName, &r.UnsubscribeTok); err != nil {
			return nil, err
		}
		out[r.RepoFullName] = append(out[r.RepoFullName], r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetRepoState(ctx context.Context, repoFullName string) (string, error) {
	var tag string
	err := s.DB.QueryRowContext(ctx, `
		SELECT last_seen_tag
		FROM repo_state
		WHERE repo_full_name = $1
	`, repoFullName).Scan(&tag)

	if errors.Is(err, sql.ErrNoRows) {
		return "", nil // treat missing as empty cursor
	}
	return tag, err
}

func (s *Store) UpsertRepoState(ctx context.Context, repoFullName string, lastSeenTag string) error {
	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO repo_state (repo_full_name, last_seen_tag, last_checked_at)
		VALUES ($1, $2, now())
		ON CONFLICT (repo_full_name)
		DO UPDATE SET last_seen_tag = EXCLUDED.last_seen_tag, last_checked_at = now()
	`, repoFullName, lastSeenTag)
	return err
}

func (s *Store) EnqueueNotification(ctx context.Context, subscriptionID int64, releaseTag string) (bool, error) {
	res, err := s.DB.ExecContext(ctx, `
		INSERT INTO notification_log (subscription_id, release_tag, status, updated_at)
		VALUES ($1, $2, 'pending', now())
		ON CONFLICT (subscription_id, release_tag) DO NOTHING
	`, subscriptionID, releaseTag)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (s *Store) MarkNotificationSent(ctx context.Context, subscriptionID int64, releaseTag string) error {
	_, err := s.DB.ExecContext(ctx, `
		UPDATE notification_log
		SET status = 'sent', last_error = NULL, updated_at = now()
		WHERE subscription_id = $1 AND release_tag = $2
	`, subscriptionID, releaseTag)
	return err
}

func (s *Store) MarkNotificationFailed(ctx context.Context, subscriptionID int64, releaseTag string, lastError string) error {
	lastError = strings.TrimSpace(lastError)
	if len(lastError) > 2000 {
		lastError = lastError[:2000]
	}
	_, err := s.DB.ExecContext(ctx, `
		UPDATE notification_log
		SET status = 'failed', last_error = $3, updated_at = now()
		WHERE subscription_id = $1 AND release_tag = $2
	`, subscriptionID, releaseTag, lastError)
	return err
}

func (s *Store) ListPendingNotifications(ctx context.Context, limit int) ([]domain.PendingNotification, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.DB.QueryContext(ctx, `
		SELECT
			n.subscription_id,
			s.email,
			s.repo_full_name,
			s.unsubscribe_token,
			n.release_tag
		FROM notification_log n
		JOIN subscriptions s ON s.id = n.subscription_id
		WHERE n.status = 'pending' AND s.status = 'active'
		ORDER BY n.updated_at ASC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.PendingNotification
	for rows.Next() {
		var pn domain.PendingNotification
		if err := rows.Scan(&pn.SubscriptionID, &pn.Email, &pn.RepoFullName, &pn.UnsubscribeTok, &pn.ReleaseTag); err != nil {
			return nil, err
		}
		out = append(out, pn)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
