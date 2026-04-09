package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/store"
)

type Store struct {
	DB *sql.DB
}

var _ store.SubscriptionStore = (*Store)(nil)

func (s *Store) CreatePending(ctx context.Context, email string, repo domain.Repo, confirmToken string, unsubscribeToken string) error {
	email = strings.TrimSpace(email)
	repoFull := repo.FullName()

	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO subscriptions (
			email,
			repo_full_name,
			repo_owner,
			repo_name,
			confirm_token,
			unsubscribe_token,
			status
		)
		VALUES ($1, $2, $3, $4, $5, $6, 'pending')
	`, email, repoFull, repo.Owner, repo.Name, confirmToken, unsubscribeToken)

	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrAlreadySubscribed
		}
		return err
	}

	return nil
}

func (s *Store) ConfirmByToken(ctx context.Context, token string) error {
	var status string
	err := s.DB.QueryRowContext(ctx, `
		SELECT status
		FROM subscriptions
		WHERE confirm_token = $1
	`, token).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrTokenNotFound
		}
		return err
	}
	switch status {
	case "active":
		return nil // idempotent
	case "unsubscribed":
		return nil // token exists; we choose no-op
	case "pending":
		_, err := s.DB.ExecContext(ctx, `
			UPDATE subscriptions
			SET status = 'active', updated_at = now()
			WHERE confirm_token = $1
		`, token)
		return err
	default:
		return nil // unknown status shouldn't break confirm
	}
}

func (s *Store) UnsubscribeByToken(ctx context.Context, token string) error {
	var status string
	err := s.DB.QueryRowContext(ctx, `
		SELECT status
		FROM subscriptions
		WHERE unsubscribe_token = $1
	`, token).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrTokenNotFound
		}
		return err
	}
	if status == "unsubscribed" {
		return nil // idempotent
	}
	_, err = s.DB.ExecContext(ctx, `
		UPDATE subscriptions
		SET status = 'unsubscribed', updated_at = now()
		WHERE unsubscribe_token = $1
	`, token)
	return err
}

func (s *Store) ListActiveByEmail(ctx context.Context, email string) ([]domain.Subscription, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT
			s.email,
			s.repo_full_name,
			(s.status = 'active') AS confirmed,
			COALESCE(rs.last_seen_tag, '') AS last_seen_tag
		FROM subscriptions s
		LEFT JOIN repo_state rs ON rs.repo_full_name = s.repo_full_name
		WHERE s.email = $1 AND s.status = 'active'
		ORDER BY s.repo_full_name ASC
	`, strings.TrimSpace(email))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Subscription
	for rows.Next() {
		var sub domain.Subscription
		if err := rows.Scan(&sub.Email, &sub.Repo, &sub.Confirmed, &sub.LastSeenTag); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.UniqueViolation // "23505"
	}
	return false
}
