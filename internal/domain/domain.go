package domain

import "errors"

var (
	ErrInvalidRepoFormat   = errors.New("invalid repo format")
	ErrInvalidEmail        = errors.New("invalid email")
	ErrAlreadySubscribed   = errors.New("already subscribed")
	ErrRepoNotFound        = errors.New("repo not found")
	ErrInvalidToken        = errors.New("invalid token")
	ErrTokenNotFound       = errors.New("token not found")
)

type Subscription struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag,omitempty"`
}

type ActiveSubscription struct {
	ID             int64
	Email          string
	RepoFullName   string
	UnsubscribeTok string
}

type PendingNotification struct {
	SubscriptionID int64
	Email          string
	RepoFullName   string
	UnsubscribeTok string
	ReleaseTag     string
}

type Repo struct {
	Owner string
	Name  string
}

func (r Repo) FullName() string { return r.Owner + "/" + r.Name }