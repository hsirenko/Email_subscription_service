package memstore

import (
	"context"
	"sync"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/store"
)

const (
	statusPending      = "pending"
	statusActive       = "active"
	statusUnsubscribed = "unsubscribed"
)

type MemStore struct {
	mu sync.Mutex

	byEmailRepo map[string]*record // key: email + "|" + repoFull
	byConfirm   map[string]*record // key: confirmToken
	byUnsub     map[string]*record // key: unsubToken
}

type record struct {
	sub domain.Subscription

	confirmToken string
	unsubToken   string
	status       string
	lastSeenTag  string
}

func New() *MemStore {
	return &MemStore{
		byEmailRepo: make(map[string]*record),
		byConfirm:   make(map[string]*record),
		byUnsub:     make(map[string]*record),
	}
}

func key(email string, repo domain.Repo) string { return email + "|" + repo.FullName() }

var _ store.SubscriptionStore = (*MemStore)(nil)

func (m *MemStore) CreatePending(ctx context.Context, email string, repo domain.Repo, confirmToken string, unsubscribeToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	k := key(email, repo)
	if rec, exists := m.byEmailRepo[k]; exists {
		if rec.status == statusPending || rec.status == statusActive {
			return domain.ErrAlreadySubscribed
		}
		// Resubscribe after unsubscribe: drop old token indexes
		delete(m.byConfirm, rec.confirmToken)
		delete(m.byUnsub, rec.unsubToken)
	}

	rec := &record{
		sub: domain.Subscription{
			Email:       email,
			Repo:        repo.FullName(),
			Confirmed:   false,
			LastSeenTag: "",
		},
		confirmToken: confirmToken,
		unsubToken:   unsubscribeToken,
		status:       statusPending,
		lastSeenTag:  "",
	}

	m.byEmailRepo[k] = rec
	m.byConfirm[confirmToken] = rec
	m.byUnsub[unsubscribeToken] = rec
	return nil
}

func (m *MemStore) ConfirmByToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.byConfirm[token]
	if !ok || rec.status != statusPending {
		return domain.ErrTokenNotFound
	}
	if rec.status == statusActive || rec.status == statusUnsubscribed {
		return nil
	}
	// pending -> active
	rec.status = statusActive
	rec.sub.Confirmed = true
	return nil
}

func (m *MemStore) UnsubscribeByToken(ctx context.Context, token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rec, ok := m.byUnsub[token]
	if !ok {
		return domain.ErrTokenNotFound
	}
	if rec.status == statusUnsubscribed {
		return nil
	}
	rec.status = statusUnsubscribed
	return nil
}

func (m *MemStore) ListActiveByEmail(ctx context.Context, email string) ([]domain.Subscription, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	out := []domain.Subscription{}
	for _, rec := range m.byEmailRepo {
		if rec.sub.Email == email && rec.status == statusActive {
			s := rec.sub
			s.LastSeenTag = rec.lastSeenTag
			out = append(out, s)
		}
	}
	return out, nil
}

// TokensFor returns confirm/unsubscribe tokens for an email+repo row (for tests).
func (m *MemStore) TokensFor(email string, repo domain.Repo) (confirmToken, unsubscribeToken string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.byEmailRepo[key(email, repo)]
	if !ok {
		return "", ""
	}
	return rec.confirmToken, rec.unsubToken
}
