package integration_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"
	"fmt"
	"os"
)

func TestSubscribeFlow_SendsEmailToMailhog(t *testing.T) {

	if os.Getenv("INTEGRATION_MAILHOG") != "1" {
		t.Skip("set INTEGRATION_MAILHOG=1 with docker compose + mailhog + SMTP, app on :8080")
	}
	
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Subscribe
	form := url.Values{}
	email := fmt.Sprintf("it+%d@example.com", time.Now().UnixNano())
	form.Set("email", email)
	form.Set("repo", "cli/cli") // repo must exist and have releases

	resp, err := httpClient.PostForm("http://localhost:8080/api/subscribe", form)
	if err != nil {
		t.Fatalf("subscribe request failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("subscribe status=%d", resp.StatusCode)
	}

	// Poll MailHog until message arrives (up to ~5s)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		total, err := mailhogTotal()
		if err != nil {
			t.Fatalf("mailhogTotal: %v", err)
		}
		if total > 0 {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Fatalf("expected at least 1 email in MailHog, got 0")
}

func mailhogTotal() (int, error) {
	resp, err := http.Get("http://localhost:8025/api/v2/messages")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var payload struct {
		Total int `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}
	return payload.Total, nil
}