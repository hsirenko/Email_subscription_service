package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/integrations/email"
	"email-subscription-service/internal/integrations/github"
	"email-subscription-service/internal/store"
)

// SubscriptionService coordinates subscribe / confirm / unsubscribe flows.
type SubscriptionService struct {
	Store     store.SubscriptionStore
	GitHub    github.Client
	Email     email.Sender
	PublicURL string // base URL for confirm and unsubscribe links in outbound mail
}

func (s SubscriptionService) Subscribe(ctx context.Context, emailAddr string, repoRaw string) error {
	if err := domain.ValidateEmail(emailAddr); err != nil {
		return err
	}

	repo, err := domain.ParseRepo(repoRaw)
	if err != nil {
		return err
	}

	ok, err := s.GitHub.RepoExists(ctx, repo)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrRepoNotFound
	}

	confirmToken := newToken(16)
	unsubToken := newToken(16)

	if err := s.Store.CreatePending(ctx, strings.TrimSpace(emailAddr), repo, confirmToken, unsubToken); err != nil {
		return err
	}

	confirmURL := buildConfirmURL(s.PublicURL, confirmToken)
	unsubscribeURL := buildUnsubscribeURL(s.PublicURL, unsubToken)
	return s.Email.SendConfirm(emailAddr, confirmURL, unsubscribeURL)
}

func (s SubscriptionService) Confirm(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.ErrInvalidToken
	}
	return s.Store.ConfirmByToken(ctx, token)
}

func (s SubscriptionService) Unsubscribe(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return domain.ErrInvalidToken
	}
	return s.Store.UnsubscribeByToken(ctx, token)
}

func (s SubscriptionService) ListByEmail(ctx context.Context, emailAddr string) ([]domain.Subscription, error) {
	if err := domain.ValidateEmail(emailAddr); err != nil {
		return nil, err
	}
	return s.Store.ListActiveByEmail(ctx, strings.TrimSpace(emailAddr))
}

func newToken(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func buildConfirmURL(publicURL string, token string) string {
	base := strings.TrimRight(publicURL, "/")
	u := fmt.Sprintf("%s/api/confirm/%s", base, url.PathEscape(token))
	return u
}

func buildUnsubscribeURL(publicURL string, token string) string {
	base := strings.TrimRight(publicURL, "/")
	return fmt.Sprintf("%s/api/unsubscribe/%s", base, url.PathEscape(token))
}
