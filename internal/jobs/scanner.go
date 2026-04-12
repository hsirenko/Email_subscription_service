package jobs

import (
	"context"
	"log"
	"strings"
	"time"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/integrations/email"
	"email-subscription-service/internal/integrations/github"
	"email-subscription-service/internal/store"
)

// Scanner periodically checks GitHub for new release tags and notifies subscribers.
type Scanner struct {
	Store   store.ReleaseJobStore
	GitHub  github.Client
	Email   email.Sender
	Every   time.Duration
	BaseURL string // API public base URL; used to build unsubscribe links in emails
}

func (s *Scanner) Run(ctx context.Context) {
	t := time.NewTicker(s.Every)
	defer t.Stop()

	s.scanOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.scanOnce(ctx)
		}
	}
}

func (s *Scanner) scanOnce(ctx context.Context) {
	cycleStart := time.Now()
	log.Printf("scanner.cycle start interval=%s", s.Every)

	// First, retry any pending notifications from previous cycles (or restarts).
	s.retryPending(ctx)

	grouped, err := s.Store.ListActiveSubscriptionsGroupedByRepo(ctx)
	if err != nil {
		log.Printf("scanner.cycle error=list_active_subs err=%q", err.Error())
		return
	}
	log.Printf("scanner.cycle active_repos=%d", len(grouped))

	for repoFull, subs := range grouped {
		owner, name, ok := splitRepo(repoFull)
		if !ok {
			log.Printf("scanner.repo skip reason=invalid_repo_full repo=%q", repoFull)
			continue
		}

		tag, okTag, err := s.GitHub.LatestReleaseTag(ctx, domain.Repo{Owner: owner, Name: name})
		if err != nil {
			log.Printf("scanner.repo error=github_latest_release repo=%s err=%q", repoFull, err.Error())
			continue
		}
		if !okTag {
			log.Printf("scanner.repo skip reason=no_releases repo=%s", repoFull)
			continue
		}

		lastSeen, err := s.Store.GetRepoState(ctx, repoFull)
		if err != nil {
			log.Printf("scanner.repo error=get_repo_state repo=%s err=%q", repoFull, err.Error())
			continue
		}
		if tag == lastSeen {
			continue
		}

		if err := s.Store.UpsertRepoState(ctx, repoFull, tag); err != nil {
			log.Printf("scanner.repo error=upsert_repo_state repo=%s tag=%s err=%q", repoFull, tag, err.Error())
			continue
		}
		log.Printf("scanner.repo new_release repo=%s tag=%s last_seen=%s subs=%d", repoFull, tag, lastSeen, len(subs))

		releaseURL := githubReleaseURL(repoFull, tag)

		for _, sub := range subs {
			enq, err := s.Store.EnqueueNotification(ctx, sub.ID, tag)
			if err != nil {
				log.Printf("scanner.notify error=enqueue sub_id=%d repo=%s tag=%s err=%q", sub.ID, repoFull, tag, err.Error())
				continue
			}
			if !enq {
				continue
			}

			unsubURL := unsubscribeURL(s.BaseURL, sub.UnsubscribeTok)
			if err := s.Email.SendRelease(sub.Email, repoFull, tag, releaseURL, unsubURL); err != nil {
				log.Printf("scanner.notify failed sub_id=%d to=%s repo=%s tag=%s err=%q", sub.ID, sub.Email, repoFull, tag, err.Error())
				_ = s.Store.MarkNotificationFailed(ctx, sub.ID, tag, err.Error())
				continue
			}
			log.Printf("scanner.notify sent sub_id=%d to=%s repo=%s tag=%s", sub.ID, sub.Email, repoFull, tag)
			_ = s.Store.MarkNotificationSent(ctx, sub.ID, tag)
		}
	}

	log.Printf("scanner.cycle done duration_ms=%d", time.Since(cycleStart).Milliseconds())
}

func (s *Scanner) retryPending(ctx context.Context) {
	pending, err := s.Store.ListPendingNotifications(ctx, 100)
	if err != nil {
		log.Printf("scanner.retry error=list_pending err=%q", err.Error())
		return
	}
	if len(pending) > 0 {
		log.Printf("scanner.retry pending=%d", len(pending))
	}
	for _, pn := range pending {
		releaseURL := githubReleaseURL(pn.RepoFullName, pn.ReleaseTag)
		unsubURL := unsubscribeURL(s.BaseURL, pn.UnsubscribeTok)
		if err := s.Email.SendRelease(pn.Email, pn.RepoFullName, pn.ReleaseTag, releaseURL, unsubURL); err != nil {
			log.Printf("scanner.retry failed sub_id=%d to=%s repo=%s tag=%s err=%q", pn.SubscriptionID, pn.Email, pn.RepoFullName, pn.ReleaseTag, err.Error())
			_ = s.Store.MarkNotificationFailed(ctx, pn.SubscriptionID, pn.ReleaseTag, err.Error())
			continue
		}
		log.Printf("scanner.retry sent sub_id=%d to=%s repo=%s tag=%s", pn.SubscriptionID, pn.Email, pn.RepoFullName, pn.ReleaseTag)
		_ = s.Store.MarkNotificationSent(ctx, pn.SubscriptionID, pn.ReleaseTag)
	}
}

func splitRepo(full string) (owner, name string, ok bool) {
	parts := strings.Split(full, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func githubReleaseURL(repoFullName, tag string) string {
	return "https://github.com/" + repoFullName + "/releases/tag/" + tag
}

func unsubscribeURL(apiBaseURL, unsubscribeToken string) string {
	return strings.TrimRight(apiBaseURL, "/") + "/api/unsubscribe/" + unsubscribeToken
}

// ScanOnce runs a single scanner cycle (for tests and manual triggers).
func (s *Scanner) ScanOnce(ctx context.Context) { s.scanOnce(ctx) }
