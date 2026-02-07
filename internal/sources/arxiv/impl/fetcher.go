package impl

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bakkerme/curator-ai/internal/retry"
	"github.com/bakkerme/curator-ai/internal/sources/arxiv"
)

const defaultBaseURL = "https://export.arxiv.org/api/query"

// Fetcher implements the arXiv API client for Atom-based responses.
type Fetcher struct {
	client    *http.Client
	baseURL   string
	userAgent string
}

// NewFetcher constructs an arXiv API client with timeout and user agent controls.
func NewFetcher(timeout time.Duration, userAgent string, baseURL string) *Fetcher {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = defaultBaseURL
	}
	if strings.TrimSpace(userAgent) == "" {
		userAgent = "curator-ai/0.1"
	}
	return &Fetcher{
		client:    &http.Client{Timeout: timeout},
		baseURL:   baseURL,
		userAgent: userAgent,
	}
}

// Search queries arXiv and returns normalized papers based on the provided options.
func (f *Fetcher) Search(ctx context.Context, options arxiv.SearchOptions) ([]arxiv.Paper, error) {
	query, err := buildSearchQuery(options)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(f.baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse arxiv base url: %w", err)
	}
	values := u.Query()
	values.Set("search_query", query)
	if options.MaxResults > 0 {
		values.Set("max_results", fmt.Sprintf("%d", options.MaxResults))
	}
	if strings.TrimSpace(options.SortBy) != "" {
		values.Set("sortBy", strings.TrimSpace(options.SortBy))
	}
	if strings.TrimSpace(options.SortOrder) != "" {
		values.Set("sortOrder", strings.TrimSpace(options.SortOrder))
	}
	u.RawQuery = values.Encode()

	var payload []byte
	err = retry.Do(ctx, retry.Config{Attempts: 3, BaseDelay: 200 * time.Millisecond}, func() error {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", f.userAgent)
		resp, err := f.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			bodySnippet := readBodySnippet(resp.Body, 2048)
			statusErr := fmt.Errorf("arxiv api status %d: %s", resp.StatusCode, bodySnippet)
			if shouldRetryStatus(resp.StatusCode) {
				return statusErr
			}
			return retry.Permanent(statusErr)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		payload = body
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("arxiv api request failed: %w", err)
	}

	entries, err := parseFeed(payload)
	if err != nil {
		return nil, err
	}

	papers := make([]arxiv.Paper, 0, len(entries))
	for _, entry := range entries {
		paper := entry.toPaper()
		papers = append(papers, paper)
	}
	return papers, nil
}

func buildSearchQuery(options arxiv.SearchOptions) (string, error) {
	var clauses []string
	if strings.TrimSpace(options.Query) != "" {
		clauses = append(clauses, fmt.Sprintf("all:%q", strings.TrimSpace(options.Query)))
	}
	if len(options.Categories) > 0 {
		parts := make([]string, 0, len(options.Categories))
		for _, cat := range options.Categories {
			cat = strings.TrimSpace(cat)
			if cat == "" {
				continue
			}
			parts = append(parts, fmt.Sprintf("cat:%s", cat))
		}
		if len(parts) > 0 {
			clauses = append(clauses, "("+strings.Join(parts, " OR ")+")")
		}
	}
	if dateClause := buildDateClause(options.DateFrom, options.DateTo); dateClause != "" {
		clauses = append(clauses, dateClause)
	}
	if len(clauses) == 0 {
		return "", fmt.Errorf("arxiv search requires query or categories")
	}
	return strings.Join(clauses, " AND "), nil
}

func buildDateClause(dateFrom string, dateTo string) string {
	from, fromOK := formatDateRange(dateFrom, false)
	to, toOK := formatDateRange(dateTo, true)
	if !fromOK && !toOK {
		return ""
	}
	if !fromOK {
		from = "*"
	}
	if !toOK {
		to = "*"
	}
	return fmt.Sprintf("submittedDate:[%s TO %s]", from, to)
}

func formatDateRange(input string, endOfDay bool) (string, bool) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", false
	}
	layouts := []string{"2006-01-02", "20060102"}
	var parsed time.Time
	var err error
	for _, layout := range layouts {
		parsed, err = time.Parse(layout, input)
		if err == nil {
			break
		}
	}
	if err != nil {
		return "", false
	}
	if endOfDay {
		parsed = parsed.Add(23*time.Hour + 59*time.Minute)
	}
	return parsed.UTC().Format("200601021504"), true
}

type feed struct {
	Entries []entry `xml:"entry"`
}

type entry struct {
	ID         string     `xml:"id"`
	Title      string     `xml:"title"`
	Summary    string     `xml:"summary"`
	Updated    string     `xml:"updated"`
	Published  string     `xml:"published"`
	Authors    []author   `xml:"author"`
	Categories []category `xml:"category"`
	Links      []link     `xml:"link"`
}

type author struct {
	Name string `xml:"name"`
}

type category struct {
	Term string `xml:"term,attr"`
}

type link struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

func parseFeed(payload []byte) ([]entry, error) {
	var parsed feed
	if err := xml.Unmarshal(payload, &parsed); err != nil {
		return nil, fmt.Errorf("parse arxiv feed: %w", err)
	}
	return parsed.Entries, nil
}

func (e entry) toPaper() arxiv.Paper {
	title := strings.TrimSpace(e.Title)
	abstract := strings.TrimSpace(e.Summary)
	rawID := strings.TrimSpace(e.ID)
	absURL := normalizeAbsURL(rawID)
	id := normalizeArxivID(rawID)

	publishedAt := parseTime(e.Published)
	updatedAt := parseTime(e.Updated)

	authors := make([]string, 0, len(e.Authors))
	for _, a := range e.Authors {
		name := strings.TrimSpace(a.Name)
		if name != "" {
			authors = append(authors, name)
		}
	}
	categories := make([]string, 0, len(e.Categories))
	for _, c := range e.Categories {
		term := strings.TrimSpace(c.Term)
		if term != "" {
			categories = append(categories, term)
		}
	}

	pdfURL := findPDFURL(e.Links)
	if pdfURL == "" && absURL != "" {
		pdfURL = strings.Replace(absURL, "/abs/", "/pdf/", 1) + ".pdf"
	}

	htmlURL := ""
	if id != "" {
		htmlURL = "https://arxiv.org/html/" + id
	}

	return arxiv.Paper{
		ID:          id,
		Title:       title,
		Abstract:    abstract,
		Authors:     authors,
		Categories:  categories,
		PublishedAt: publishedAt,
		UpdatedAt:   updatedAt,
		AbsURL:      absURL,
		PDFURL:      pdfURL,
		HTMLURL:     htmlURL,
	}
}

func parseTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func normalizeAbsURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "/abs/") {
		return raw
	}
	return strings.TrimSuffix(raw, "/") + "/abs"
}

func normalizeArxivID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// IDs are commonly provided as full URLs (`.../abs/<id>`) but may also be bare
	// IDs. Legacy IDs have an archive prefix (`hep-th/9901001v2`) that must be
	// preserved for stable dedupe keys.
	id := raw
	hadAbsPrefix := false
	if _, after, ok := strings.Cut(raw, "/abs/"); ok {
		id = after
		hadAbsPrefix = true
	}

	id = strings.Trim(id, "/")
	if idx := strings.IndexAny(id, "?#"); idx >= 0 {
		id = id[:idx]
	}
	if id == "" {
		return ""
	}

	if strings.Contains(id, "/") {
		parts := strings.Split(id, "/")
		if hadAbsPrefix || strings.Count(id, "/") == 1 {
			// Preserve legacy archive-prefixed IDs (for example, `hep-th/9901001v2`).
			id = strings.TrimSpace(parts[len(parts)-2]) + "/" + strings.TrimSpace(parts[len(parts)-1])
		} else {
			id = strings.TrimSpace(parts[len(parts)-1])
		}
	}

	return stripArxivVersionSuffix(id)
}

func stripArxivVersionSuffix(id string) string {
	versionIndex := strings.LastIndexAny(id, "vV")
	if versionIndex <= 0 || versionIndex == len(id)-1 {
		return id
	}
	for _, r := range id[versionIndex+1:] {
		if r < '0' || r > '9' {
			return id
		}
	}
	return id[:versionIndex]
}

func findPDFURL(links []link) string {
	for _, l := range links {
		if strings.EqualFold(l.Type, "application/pdf") && strings.TrimSpace(l.Href) != "" {
			return strings.TrimSpace(l.Href)
		}
	}
	return ""
}

func shouldRetryStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func readBodySnippet(body io.Reader, limit int64) string {
	if limit <= 0 {
		limit = 2048
	}
	payload, err := io.ReadAll(io.LimitReader(body, limit))
	if err != nil {
		return "failed to read response body"
	}
	snippet := strings.TrimSpace(string(payload))
	if snippet == "" {
		return "empty response body"
	}
	return snippet
}
