package domain

// Subscription is a subscriber-facing view for GET /api/subscriptions.
type Subscription struct {
	Email       string `json:"email"`
	Repo        string `json:"repo"`
	Confirmed   bool   `json:"confirmed"`
	LastSeenTag string `json:"last_seen_tag,omitempty"`
}

// ActiveSubscription is one row used by the release scanner, grouped by repo.
type ActiveSubscription struct {
	ID             int64
	Email          string
	RepoFullName   string
	UnsubscribeTok string
}

// PendingNotification is a queued email the scanner retries until sent/failed.
type PendingNotification struct {
	SubscriptionID int64
	Email          string
	RepoFullName   string
	UnsubscribeTok string
	ReleaseTag     string
}

// Repo is a GitHub repository in owner/name form (e.g. cli/cli).
type Repo struct {
	Owner string
	Name  string
}

// FullName returns owner/name for API and storage keys.
func (r Repo) FullName() string { return r.Owner + "/" + r.Name }
