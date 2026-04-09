package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"sync"
	"strconv"

	"email-subscription-service/internal/domain"
	"email-subscription-service/internal/integrations/github"
)

func TestHTTPClient_RepoExists_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/golang/go" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := github.NewHTTPClient(github.HTTPClientConfig{
		BaseURL: srv.URL,
		Sleep:   func(ctx context.Context, d time.Duration) error { return nil },
	})

	ok, err := c.RepoExists(context.Background(), domain.Repo{Owner: "golang", Name: "go"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Fatalf("expected exists=true")
	}
}

func TestHTTPClient_RepoExists_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c := github.NewHTTPClient(github.HTTPClientConfig{
		BaseURL: srv.URL,
		Sleep:   func(ctx context.Context, d time.Duration) error { return nil },
	})

	ok, err := c.RepoExists(context.Background(), domain.Repo{Owner: "missing", Name: "repo"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected exists=false")
	}
}

func TestHTTPClient_RepoExists_429_RetryAfterThen200(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := github.NewHTTPClient(github.HTTPClientConfig{
		BaseURL:     srv.URL,
		MaxAttempts: 3,
		Sleep:       func(ctx context.Context, d time.Duration) error { return nil },
	})

	ok, err := c.RepoExists(context.Background(), domain.Repo{Owner: "golang", Name: "go"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Fatalf("expected exists=true")
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestHTTPClient_LatestReleaseTag_200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/golang/go/releases/latest" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3"}`))
	}))
	t.Cleanup(srv.Close)

	c := github.NewHTTPClient(github.HTTPClientConfig{
		BaseURL: srv.URL,
		Sleep:   func(ctx context.Context, d time.Duration) error { return nil },
	})

	tag, ok, err := c.LatestReleaseTag(context.Background(), domain.Repo{Owner: "golang", Name: "go"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok || tag != "v1.2.3" {
		t.Fatalf("tag=%q ok=%v", tag, ok)
	}
}

func TestHTTPClient_LatestReleaseTag_404_NoRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	c := github.NewHTTPClient(github.HTTPClientConfig{
		BaseURL: srv.URL,
		Sleep:   func(ctx context.Context, d time.Duration) error { return nil },
	})

	tag, ok, err := c.LatestReleaseTag(context.Background(), domain.Repo{Owner: "golang", Name: "go"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok || tag != "" {
		t.Fatalf("expected ok=false, empty tag; got tag=%q ok=%v", tag, ok)
	}
}

func TestHTTPClient_429_RetryAfter_Respected(t *testing.T) {
	var (
		mu       sync.Mutex
		slept    []time.Duration
		calls    int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	c := github.NewHTTPClient(github.HTTPClientConfig{
		BaseURL:     srv.URL,
		MaxAttempts: 3,
		Sleep: func(ctx context.Context, d time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			slept = append(slept, d)
			return nil
		},
	})
	ok, err := c.RepoExists(context.Background(), domain.Repo{Owner: "golang", Name: "go"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Fatalf("expected exists=true")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(slept) == 0 {
		t.Fatalf("expected Sleep to be called")
	}
	if slept[0] != 2*time.Second {
		t.Fatalf("expected sleep=2s, got %v", slept[0])
	}
}

func TestHTTPClient_429_RateLimitReset_Fallback(t *testing.T) {
	var (
		mu    sync.Mutex
		slept []time.Duration
		calls int
	)

	reset := time.Now().Add(2 * time.Second).Unix()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			// No Retry-After header, so client must use X-RateLimit-Reset.
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset, 10))
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	c := github.NewHTTPClient(github.HTTPClientConfig{
		BaseURL:     srv.URL,
		MaxAttempts: 3,
		Sleep: func(ctx context.Context, d time.Duration) error {
			mu.Lock()
			defer mu.Unlock()
			slept = append(slept, d)
			return nil
		},
	})

	ok, err := c.RepoExists(context.Background(), domain.Repo{Owner: "golang", Name: "go"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok {
		t.Fatalf("expected exists=true")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(slept) == 0 {
		t.Fatalf("expected Sleep to be called")
	}
	if slept[0] <= 0 {
		t.Fatalf("expected sleep > 0, got %v", slept[0])
	}
}

