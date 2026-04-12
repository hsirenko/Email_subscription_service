package jobs_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/jobs"
)

type fakeGitHub struct {
	Tag string
	Ok  bool
	Err error
}

func (f fakeGitHub) RepoExists(ctx context.Context, repo domain.Repo) (bool, error) { return true, nil }
func (f fakeGitHub) LatestReleaseTag(ctx context.Context, repo domain.Repo) (string, bool, error) {
	return f.Tag, f.Ok, f.Err
}

type fakeJobStore struct {
	Grouped map[string][]domain.ActiveSubscription

	RepoState map[string]string // repo_full -> lastSeenTag

	Enqueued map[[2]string]bool // key: [subID+tag] simplified below
	Sent     map[[2]string]bool

	Pending []domain.PendingNotification
}

func (f *fakeJobStore) ListActiveSubscriptionsGroupedByRepo(ctx context.Context) (map[string][]domain.ActiveSubscription, error) {
	return f.Grouped, nil
}
func (f *fakeJobStore) GetRepoState(ctx context.Context, repoFullName string) (string, error) {
	return f.RepoState[repoFullName], nil
}
func (f *fakeJobStore) UpsertRepoState(ctx context.Context, repoFullName string, lastSeenTag string) error {
	f.RepoState[repoFullName] = lastSeenTag
	return nil
}
func (f *fakeJobStore) EnqueueNotification(ctx context.Context, subscriptionID int64, releaseTag string) (bool, error) {
	key := [2]string{itoa(subscriptionID), releaseTag}
	if f.Enqueued[key] {
		return false, nil
	}
	f.Enqueued[key] = true
	// also add to pending list so retryPending could pick it up (optional)
	return true, nil
}
func (f *fakeJobStore) MarkNotificationSent(ctx context.Context, subscriptionID int64, releaseTag string) error {
	key := [2]string{itoa(subscriptionID), releaseTag}
	f.Sent[key] = true
	return nil
}
func (f *fakeJobStore) MarkNotificationFailed(ctx context.Context, subscriptionID int64, releaseTag string, lastError string) error {
	return nil
}
func (f *fakeJobStore) ListPendingNotifications(ctx context.Context, limit int) ([]domain.PendingNotification, error) {
	return f.Pending, nil
}

func itoa(id int64) string { return fmt.Sprintf("%d", id) }

type captureEmail struct {
	ReleaseCalls int
}

func (c *captureEmail) SendConfirm(toEmail, confirmURL, unsubscribeURL string) error { return nil }
func (c *captureEmail) SendRelease(toEmail, repoFullName, tag, releaseURL, unsubscribeURL string) error {
	c.ReleaseCalls++
	return nil
}

func TestScanner_NewTag_SendsOnceAndUpdatesRepoState(t *testing.T) {
	st := &fakeJobStore{
		Grouped: map[string][]domain.ActiveSubscription{
			"cli/cli": {
				{ID: 1, Email: "a@b.com", RepoFullName: "cli/cli", UnsubscribeTok: "u1"},
			},
		},
		RepoState: map[string]string{"cli/cli": "old"},
		Enqueued:  map[[2]string]bool{},
		Sent:      map[[2]string]bool{},
	}
	gh := fakeGitHub{Tag: "v1.0.0", Ok: true}
	m := &captureEmail{}
	sc := jobs.Scanner{
		Store:   st,
		GitHub:  gh,
		Email:   m,
		Every:   time.Hour,
		BaseURL: "http://localhost:8080",
	}

	// First scan should send once and update cursor.
	sc.ScanOnce(context.Background())
	if st.RepoState["cli/cli"] != "v1.0.0" {
		t.Fatalf("expected repo_state tag to be updated, got %q", st.RepoState["cli/cli"])
	}
	if m.ReleaseCalls != 1 {
		t.Fatalf("expected 1 release email, got %d", m.ReleaseCalls)
	}

	// Second scan should NOT send again (dedupe + last_seen_tag).
	sc.ScanOnce(context.Background())
	if m.ReleaseCalls != 1 {
		t.Fatalf("expected still 1 release email after second scan, got %d", m.ReleaseCalls)
	}
}
