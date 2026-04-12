package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"email-subscription-service/internal/domain"
)

type HTTPClientConfig struct {
	// Token is optional; if provided, sent as Authorization: Bearer <token>.
	Token string

	// BaseURL defaults to https://api.github.com.
	BaseURL string

	// UserAgent defaults to email-subscription-service/1.0.
	UserAgent string

	// MaxAttempts defaults to 5.
	MaxAttempts int

	// CacheTTL enables a small in-memory cache for RepoExists results when > 0.
	CacheTTL time.Duration

	// HTTP is the underlying transport. If nil, a default client with a 30s timeout is used.
	HTTP *http.Client

	// Sleep can be overridden in tests. If nil, time.Sleep is used (via timer).
	Sleep func(ctx context.Context, d time.Duration) error
}

type HTTPClient struct {
	http      *http.Client
	sleep     func(ctx context.Context, d time.Duration) error
	baseURL   string
	userAgent string
	token     string
	max       int

	cacheTTL time.Duration
	cacheMu  sync.Mutex
	cache    map[string]cacheEntry
}

type cacheEntry struct {
	exists  bool
	expires time.Time
}

func NewHTTPClient(cfg HTTPClientConfig) *HTTPClient {
	httpClient := cfg.HTTP
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	sleepFn := cfg.Sleep
	if sleepFn == nil {
		sleepFn = sleepCtx
	}
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	ua := cfg.UserAgent
	if ua == "" {
		ua = "email-subscription-service/1.0"
	}
	max := cfg.MaxAttempts
	if max <= 0 {
		max = 5
	}

	c := &HTTPClient{
		http:      httpClient,
		sleep:     sleepFn,
		baseURL:   baseURL,
		userAgent: ua,
		token:     strings.TrimSpace(cfg.Token),
		max:       max,
		cacheTTL:  cfg.CacheTTL,
	}
	if c.cacheTTL > 0 {
		c.cache = make(map[string]cacheEntry)
	}
	return c
}

var _ Client = (*HTTPClient)(nil)

func (c *HTTPClient) RepoExists(ctx context.Context, repo domain.Repo) (bool, error) {
	key := repo.FullName()
	if c.cacheTTL > 0 {
		if _, exists, hit := c.cacheGet(key); hit {
			return exists, nil
		}
	}

	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, repo.Owner, repo.Name)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	c.applyHeaders(req)

	_, status, err := c.doWithRetry(ctx, req)
	if err != nil {
		return false, err
	}
	if status == http.StatusNotFound {
		if c.cacheTTL > 0 {
			c.cachePut(key, false)
		}
		return false, nil
	}
	if status == http.StatusOK {
		if c.cacheTTL > 0 {
			c.cachePut(key, true)
		}
		return true, nil
	}
	return false, fmt.Errorf("github: unexpected status %d", status)
}

type latestReleaseResp struct {
	TagName string `json:"tag_name"`
}

func (c *HTTPClient) LatestReleaseTag(ctx context.Context, repo domain.Repo) (tag string, ok bool, err error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.baseURL, repo.Owner, repo.Name)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	c.applyHeaders(req)

	b, status, err := c.doWithRetry(ctx, req)
	if err != nil {
		return "", false, err
	}
	if status == http.StatusNotFound {
		return "", false, nil
	}
	if status != http.StatusOK {
		return "", false, fmt.Errorf("github: unexpected status %d", status)
	}
	var lr latestReleaseResp
	if err := json.Unmarshal(b, &lr); err != nil {
		return "", false, err
	}
	if strings.TrimSpace(lr.TagName) == "" {
		return "", false, nil
	}
	return lr.TagName, true, nil
}

func (c *HTTPClient) applyHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}

func (c *HTTPClient) doWithRetry(ctx context.Context, req *http.Request) (body []byte, status int, err error) {
	var lastStatus int
	for attempt := 0; attempt < c.max; attempt++ {
		resp, err := c.http.Do(req.Clone(ctx))
		if err != nil {
			// transient transport error: backoff and retry
			if err := c.sleep(ctx, backoffWithJitter(attempt)); err != nil {
				return nil, 0, err
			}
			continue
		}

		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		lastStatus = resp.StatusCode

		switch resp.StatusCode {
		case http.StatusOK, http.StatusNotFound:
			return b, resp.StatusCode, nil
		case http.StatusTooManyRequests:
			wait := retryAfter(resp)
			if wait <= 0 {
				wait = backoffWithJitter(attempt)
			}
			if err := c.sleep(ctx, wait); err != nil {
				return nil, resp.StatusCode, err
			}
			continue
		default:
			if resp.StatusCode >= 500 && resp.StatusCode <= 599 {
				if err := c.sleep(ctx, backoffWithJitter(attempt)); err != nil {
					return nil, resp.StatusCode, err
				}
				continue
			}
			// non-retryable client error
			return b, resp.StatusCode, fmt.Errorf("github: status %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
		}
	}
	return nil, lastStatus, errors.New("github: max retries exceeded")
}

func retryAfter(resp *http.Response) time.Duration {
	if s := strings.TrimSpace(resp.Header.Get("Retry-After")); s != "" {
		// GitHub commonly returns integer seconds.
		if sec, err := strconv.Atoi(s); err == nil {
			if sec < 0 {
				return 0
			}
			return time.Duration(sec) * time.Second
		}
		// Could parse HTTP-date here if needed.
	}
	if rs := strings.TrimSpace(resp.Header.Get("X-RateLimit-Reset")); rs != "" {
		if unix, err := strconv.ParseInt(rs, 10, 64); err == nil {
			reset := time.Unix(unix, 0)
			if d := time.Until(reset); d > 0 {
				return d
			}
		}
	}
	return 0
}

func backoffWithJitter(attempt int) time.Duration {
	// 1s, 2s, 4s... capped at 64s, plus up to 250ms jitter.
	pow := attempt
	if pow < 0 {
		pow = 0
	}
	if pow > 6 {
		pow = 6
	}
	base := time.Duration(1<<pow) * time.Second
	j := time.Duration(rand.Intn(250)) * time.Millisecond
	return base + j
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *HTTPClient) cacheGet(key string) (ok bool, exists bool, hit bool) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if c.cache == nil {
		return false, false, false
	}
	ent, ok := c.cache[key]
	if !ok {
		return false, false, false
	}
	if time.Now().After(ent.expires) {
		delete(c.cache, key)
		return false, false, false
	}
	return true, ent.exists, true
}

func (c *HTTPClient) cachePut(key string, exists bool) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	if c.cache == nil {
		return
	}
	c.cache[key] = cacheEntry{
		exists:  exists,
		expires: time.Now().Add(c.cacheTTL),
	}
}
