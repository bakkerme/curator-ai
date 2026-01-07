package impl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/retry"
	"github.com/bakkerme/curator-ai/internal/sources/jina"
)

const defaultBaseURL = "https://r.jina.ai/"
const tokenBudget = "15000" // 10k tokens

type Reader struct {
	client      *http.Client
	apiKey      string
	baseURL     string
	userAgent   string
	maxBodySize int64
}

func NewReader(timeout time.Duration, userAgent, baseURL, apiKey string) *Reader {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if userAgent == "" {
		userAgent = "curator-ai/0.1"
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	return &Reader{
		client:      &http.Client{Timeout: timeout},
		apiKey:      strings.TrimSpace(apiKey),
		baseURL:     baseURL,
		userAgent:   userAgent,
		maxBodySize: 10 << 20, // 10 MiB
	}
}

func (r *Reader) Read(ctx context.Context, url string, options jina.ReadOptions) (string, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return "", fmt.Errorf("jina: url is required")
	}
	if r.apiKey == "" {
		return "", fmt.Errorf("jina: missing api key (set JINA_API_KEY)")
	}

	retainImages := strings.TrimSpace(options.RetainImages)
	if retainImages == "" {
		retainImages = "none"
	}

	var (
		lastStatus int
		respBody   []byte
	)
	err := retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		payload, err := json.Marshal(map[string]string{"url": url})
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.baseURL, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+r.apiKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Retain-Images", retainImages)
		req.Header.Set("User-Agent", r.userAgent)
		req.Header.Set("X-Token-Budget", tokenBudget) // 10k tokens

		resp, err := r.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		limited := io.LimitReader(resp.Body, r.maxBodySize+1)
		body, err := io.ReadAll(limited)
		if err != nil {
			return err
		}
		if int64(len(body)) > r.maxBodySize {
			return fmt.Errorf("jina: response too large")
		}

		lastStatus = resp.StatusCode
		respBody = body

		if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= http.StatusInternalServerError {
			return fmt.Errorf("jina transient error: %s", resp.Status)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("jina request failed: %s", resp.Status)
		}
		return nil
	})
	if err != nil {
		msg := strings.TrimSpace(string(respBody))
		if msg != "" {
			msg = ": " + msg
		}
		if lastStatus != 0 {
			return "", fmt.Errorf("jina: status %d%s: %w", lastStatus, msg, err)
		}
		return "", fmt.Errorf("jina: %w", err)
	}

	return strings.TrimSpace(string(respBody)), nil
}
