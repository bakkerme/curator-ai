package impl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/sources/reddit"
)

type Fetcher struct {
	client *http.Client
}

func NewFetcher(timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{Timeout: timeout},
	}
}

func (f *Fetcher) Fetch(ctx context.Context, config reddit.Config) ([]reddit.Item, error) {
	if len(config.Subreddits) == 0 {
		return nil, fmt.Errorf("no subreddits configured")
	}

	sort := config.Sort
	if sort == "" {
		sort = "hot"
	}

	limit := config.Limit
	if limit <= 0 {
		limit = 25
	}

	subreddits := strings.Join(config.Subreddits, "+")
	endpoint := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json", url.PathEscape(subreddits), url.PathEscape(sort))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	query := req.URL.Query()
	query.Set("limit", fmt.Sprintf("%d", limit))
	if config.TimeFilter != "" {
		query.Set("t", config.TimeFilter)
	}
	req.URL.RawQuery = query.Encode()

	if config.UserAgent != "" {
		req.Header.Set("User-Agent", config.UserAgent)
	} else {
		req.Header.Set("User-Agent", "curator-ai/0.1")
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("reddit fetch failed: %s", resp.Status)
	}

	var payload listingResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode reddit response: %w", err)
	}

	items := make([]reddit.Item, 0, len(payload.Data.Children))
	for _, child := range payload.Data.Children {
		if config.MinScore > 0 && child.Data.Score < config.MinScore {
			continue
		}
		item := reddit.Item{
			ID:        child.Data.ID,
			Title:     child.Data.Title,
			URL:       child.Data.URL,
			Content:   child.Data.SelfText,
			Author:    child.Data.Author,
			Score:     child.Data.Score,
			CreatedAt: time.Unix(int64(child.Data.CreatedUTC), 0).UTC(),
		}
		items = append(items, item)
	}

	return items, nil
}

type listingResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				ID         string  `json:"id"`
				Title      string  `json:"title"`
				URL        string  `json:"url"`
				SelfText   string  `json:"selftext"`
				Author     string  `json:"author"`
				Score      int     `json:"score"`
				CreatedUTC float64 `json:"created_utc"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}
