//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMailpitE2E(t *testing.T) {
	if os.Getenv("CURATOR_E2E") == "" {
		t.Skip("set CURATOR_E2E=1 to enable e2e tests")
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	composeFile := getenv("MAILPIT_COMPOSE_FILE", filepath.Join(repoRoot, "docker-compose.yml"))
	apiBase := strings.TrimRight(getenv("MAILPIT_API_BASE", "http://localhost:8025"), "/")

	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not installed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := dockerCompose(ctx, repoRoot, composeFile, "up", "-d"); err != nil {
		t.Fatalf("docker compose up: %v", err)
	}
	if os.Getenv("MAILPIT_KEEP_RUNNING") == "" {
		t.Cleanup(func() {
			_ = dockerCompose(context.Background(), repoRoot, composeFile, "down")
		})
	}

	waitForHTTP200(t, ctx, apiBase+"/api/v1/messages")
	_ = httpDo(ctx, http.MethodDelete, apiBase+"/api/v1/messages", nil)

	rssServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feed.xml" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
		_, _ = io.WriteString(w, rssFixtureXML)
	}))
	t.Cleanup(rssServer.Close)

	runID := fmt.Sprintf("%d-%d", time.Now().Unix(), rand.IntN(1_000_000))
	flowYAML := strings.ReplaceAll(flowFixtureYAML, "__FEED_URL__", rssServer.URL+"/feed.xml")
	flowYAML = strings.ReplaceAll(flowYAML, "__RUN_ID__", runID)

	flowFile := filepath.Join(t.TempDir(), "flow.yml")
	if err := os.WriteFile(flowFile, []byte(flowYAML), 0o600); err != nil {
		t.Fatalf("write flow file: %v", err)
	}

	curatorEnv := append(os.Environ(),
		"SMTP_HOST=localhost",
		"SMTP_PORT=1025",
		"SMTP_USER=user@curator.ai",
		"SMTP_PASSWORD=123asdf123",
		"SMTP_USE_TLS=false",
	)

	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/curator", "-config", flowFile, "-run-once")
	cmd.Dir = repoRoot
	cmd.Env = curatorEnv
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("curator run failed: %v\n%s", err, out)
	}

	msgID := waitForMailpitMessageID(t, ctx, apiBase, runID)
	raw := mustHTTPGet(t, ctx, apiBase+"/api/v1/message/"+msgID)

	var msg mailpitMessage
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("parse message json: %v\n%s", err, raw)
	}

	if !strings.Contains(msg.Subject, "Curator Mailpit E2E") || !strings.Contains(msg.Subject, runID) {
		t.Fatalf("unexpected subject: %q", msg.Subject)
	}
	body := firstNonEmpty(msg.HTML, msg.Text, msg.Body)
	if !strings.Contains(body, "Curator Mailpit E2E Item") {
		t.Fatalf("expected RSS item title not found in message body")
	}
}

const rssFixtureXML = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Curator E2E Feed</title>
    <link>http://localhost/</link>
    <description>Local RSS feed for Curator Mailpit e2e.</description>
    <item>
      <title>Curator Mailpit E2E Item</title>
      <link>http://localhost/item-1</link>
      <guid>curator-mailpit-e2e-item-1</guid>
      <pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
      <description><![CDATA[This is a deterministic RSS item for local testing.]]></description>
    </item>
  </channel>
</rss>`

const flowFixtureYAML = `workflow:
  name: "Mailpit E2E"
  trigger:
    - cron:
        schedule: "* * * * *"
  sources:
    - rss:
        feeds:
          - "__FEED_URL__"
        limit: 1
        include_content: true
  output:
    - email:
        to: "dev@example.com"
        from: "curator@example.com"
        subject: "Curator Mailpit E2E __RUN_ID__"
        smtp_host: "localhost"
        smtp_port: 1025
        use_tls: false
        template: |-
          <html>
            <body>
              <h1>Curator Mailpit E2E __RUN_ID__</h1>
              <ul>
                {{ range .Blocks -}}
                  <li>{{ .Title }}</li>
                {{- end }}
              </ul>
            </body>
          </html>
`

type mailpitMessagesResponse struct {
	Messages []mailpitMessageSummary `json:"messages"`
}

type mailpitMessageSummary struct {
	ID      string `json:"ID"`
	Subject string `json:"Subject"`
}

type mailpitMessage struct {
	Subject string `json:"Subject"`
	HTML    string `json:"HTML"`
	Text    string `json:"Text"`
	Body    string `json:"Body"`
}

func waitForMailpitMessageID(t *testing.T, ctx context.Context, apiBase string, runID string) string {
	t.Helper()

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		raw := mustHTTPGet(t, ctx, apiBase+"/api/v1/messages")
		var res mailpitMessagesResponse
		_ = json.Unmarshal(raw, &res)
		for _, m := range res.Messages {
			if strings.Contains(m.Subject, runID) && m.ID != "" {
				return m.ID
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for mailpit message with run id %q", runID)
	return ""
}

func dockerCompose(ctx context.Context, repoRoot string, composeFile string, args ...string) error {
	all := append([]string{"compose", "-f", composeFile}, args...)
	cmd := exec.CommandContext(ctx, "docker", all...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %w\n%s", cmd.Args, err, out)
	}
	return nil
}

func waitForHTTP200(t *testing.T, ctx context.Context, url string) {
	t.Helper()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		resp, err := http.DefaultClient.Do(req)
		if err == nil && resp != nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", url)
}

func mustHTTPGet(t *testing.T, ctx context.Context, url string) []byte {
	t.Helper()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		t.Fatalf("GET %s: status=%d body=%s", url, resp.StatusCode, body)
	}
	return body
}

func httpDo(ctx context.Context, method string, url string, body []byte) error {
	var r io.Reader
	if body != nil {
		r = bytes.NewReader(body)
	}
	req, _ := http.NewRequestWithContext(ctx, method, url, r)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s: status=%d", method, url, resp.StatusCode)
	}
	return nil
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	return "", errors.New("go.mod not found in parent directories")
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
