package service_test

import (
	"context"
	"testing"
	"strings"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/integrations/email/log"
	ghfake "email-subscription-service/internal/integrations/github/fake"
	"email-subscription-service/internal/service"
	"email-subscription-service/internal/store/memstore"
)

type captureSender struct {
	LastConfirmURL string
	LastUnsubscribeURL string

	LastReleaseToEmail string
	LastReleaseRepo    string
	LastReleaseTag     string
	LastReleaseURL     string
	LastReleaseUnsub   string
}

func (c *captureSender) SendConfirm(toEmail string, confirmURL string, unsubscribeURL string) error {
	c.LastConfirmURL = confirmURL
	c.LastUnsubscribeURL = unsubscribeURL
	return nil
}

func (c *captureSender) SendRelease(toEmail string, repoFullName string, tag string, releaseURL string, unsubscribeURL string) error {
	c.LastReleaseToEmail = toEmail
	c.LastReleaseRepo = repoFullName
	c.LastReleaseTag = tag
	c.LastReleaseURL = releaseURL
	c.LastReleaseUnsub = unsubscribeURL
	return nil
}

func confirmTokenFromURL(confirmURL string) string {
	const marker = "/api/confirm/"
	i := strings.LastIndex(confirmURL, marker)
	if i < 0 {
		return ""
	}
	return confirmURL[i+len(marker):]
}

func TestSubscribe_InvalidRepoFormat(t *testing.T) {
	svc := service.SubscriptionService{
		Store:     memstore.New(),
		GitHub:    ghfake.Client{Exists: true},
		Email:     log.Sender{},
		PublicURL: "http://localhost:8080",
	}

	err := svc.Subscribe(context.Background(), "a@b.com", "badformat")
	if err == nil || err != domain.ErrInvalidRepoFormat {
		t.Fatalf("expected ErrInvalidRepoFormat, got %v", err)
	}
}

func TestSubscribe_RepoNotFound(t *testing.T) {
	svc := service.SubscriptionService{
		Store:     memstore.New(),
		GitHub:    ghfake.Client{Exists: false},
		Email:     log.Sender{},
		PublicURL: "http://localhost:8080",
	}

	err := svc.Subscribe(context.Background(), "a@b.com", "golang/go")
	if err == nil || err != domain.ErrRepoNotFound {
		t.Fatalf("expected ErrRepoNotFound, got %v", err)
	}
}

func TestSubscribe_AlreadySubscribed(t *testing.T) {
	svc := service.SubscriptionService{
		Store:     memstore.New(),
		GitHub:    ghfake.Client{Exists: true},
		Email:     log.Sender{},
		PublicURL: "http://localhost:8080",
	}

	err := svc.Subscribe(context.Background(), "a@b.com", "golang/go")
	if err != nil {
		t.Fatalf("expected first subscribe to succeed, got %v", err)
	}

	err = svc.Subscribe(context.Background(), "a@b.com", "golang/go")
	if err == nil || err != domain.ErrAlreadySubscribed {
		t.Fatalf("expected ErrAlreadySubscribed, got %v", err)
	}
}

func TestConfirm_EmptyToken(t *testing.T) {
	svc := service.SubscriptionService{
		Store:     memstore.New(),
		GitHub:    ghfake.Client{Exists: true},
		Email:     log.Sender{},
		PublicURL: "http://localhost:8080",
	}

	err := svc.Confirm(context.Background(), "")
	if err == nil || err != domain.ErrInvalidToken {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestConfirm_UnknownToken(t *testing.T) {
	svc := service.SubscriptionService{
		Store:     memstore.New(),
		GitHub:    ghfake.Client{Exists: true},
		Email:     log.Sender{},
		PublicURL: "http://localhost:8080",
	}

	err := svc.Confirm(context.Background(), "nonexistent-token")
	if err == nil || err != domain.ErrTokenNotFound {
		t.Fatalf("expected ErrTokenNotFound, got %v", err)
	}
}

func TestListByEmail_InvalidEmail(t *testing.T) {
	svc := service.SubscriptionService{
		Store:     memstore.New(),
		GitHub:    ghfake.Client{Exists: true},
		Email:     log.Sender{},
		PublicURL: "http://localhost:8080",
	}

	_, err := svc.ListByEmail(context.Background(), "not-an-email")
	if err == nil || err != domain.ErrInvalidEmail {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}
}

func TestSubscribe_InvalidEmail(t *testing.T) {
	svc := service.SubscriptionService{
		Store:     memstore.New(),
		GitHub:    ghfake.Client{Exists: true},
		Email:     &captureSender{},
		PublicURL: "http://localhost:8080",
	}
	err := svc.Subscribe(context.Background(), "not-an-email", "golang/go")
	if err != domain.ErrInvalidEmail {
		t.Fatalf("want ErrInvalidEmail, got %v", err)
	}
}

func TestConfirm_Success_And_ListActive(t *testing.T) {
	st := memstore.New()
	sender := &captureSender{}
	svc := service.SubscriptionService{
		Store:     st,
		GitHub:    ghfake.Client{Exists: true},
		Email:     sender,
		PublicURL: "http://localhost:8080",
	}
	if err := svc.Subscribe(context.Background(), "a@b.com", "golang/go"); err != nil {
		t.Fatal(err)
	}
	tok := confirmTokenFromURL(sender.LastConfirmURL)
	if tok == "" {
		t.Fatal("empty confirm token from URL")
	}
	if err := svc.Confirm(context.Background(), tok); err != nil {
		t.Fatal(err)
	}
	subs, err := svc.ListByEmail(context.Background(), "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || !subs[0].Confirmed || subs[0].Repo != "golang/go" {
		t.Fatalf("subs = %+v", subs)
	}
}

func TestUnsubscribe_Success_RemovesFromActiveList(t *testing.T) {
	st := memstore.New()
	sender := &captureSender{}
	svc := service.SubscriptionService{
		Store:     st,
		GitHub:    ghfake.Client{Exists: true},
		Email:     sender,
		PublicURL: "http://localhost:8080",
	}
	if err := svc.Subscribe(context.Background(), "a@b.com", "golang/go"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Confirm(context.Background(), confirmTokenFromURL(sender.LastConfirmURL)); err != nil {
		t.Fatal(err)
	}
	_, unsubTok := st.TokensFor("a@b.com", domain.Repo{Owner: "golang", Name: "go"})
	if unsubTok == "" {
		t.Fatal("empty unsub token")
	}
	if err := svc.Unsubscribe(context.Background(), unsubTok); err != nil {
		t.Fatal(err)
	}
	subs, err := svc.ListByEmail(context.Background(), "a@b.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 0 {
		t.Fatalf("want no active subs, got %+v", subs)
	}
}

func TestSubscribe_AgainAfterUnsubscribe_Succeeds(t *testing.T) {
	st := memstore.New()
	sender := &captureSender{}
	svc := service.SubscriptionService{
		Store:     st,
		GitHub:    ghfake.Client{Exists: true},
		Email:     sender,
		PublicURL: "http://localhost:8080",
	}
	repo := domain.Repo{Owner: "golang", Name: "go"}

	if err := svc.Subscribe(context.Background(), "a@b.com", "golang/go"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Confirm(context.Background(), confirmTokenFromURL(sender.LastConfirmURL)); err != nil {
		t.Fatal(err)
	}
	_, unsubTok := st.TokensFor("a@b.com", repo)
	if err := svc.Unsubscribe(context.Background(), unsubTok); err != nil {
		t.Fatal(err)
	}
	if err := svc.Subscribe(context.Background(), "a@b.com", "golang/go"); err != nil {
		t.Fatalf("second subscribe after unsubscribe: %v", err)
	}
}