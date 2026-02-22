package source

import (
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/bakkerme/curator-ai/internal/config"
)

func extractValue(doc *goquery.Document, selector, attr string) string {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return ""
	}
	sel := doc.Find(selector).First()
	if sel.Length() == 0 {
		return ""
	}
	if strings.TrimSpace(attr) != "" {
		if v, ok := sel.Attr(attr); ok {
			return strings.TrimSpace(v)
		}
		return ""
	}
	return strings.TrimSpace(sel.Text())
}

func resolveURL(baseURL, href string) string {
	href = strings.TrimSpace(href)
	if href == "" {
		return ""
	}
	u, err := url.Parse(href)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		return u.String()
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	return base.ResolveReference(u).String()
}

func parseLookbackWindow(value string) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false, nil
	}
	d, err := parseExtendedDuration(value)
	if err != nil {
		return time.Time{}, false, err
	}
	return time.Now().UTC().Add(-d), true, nil
}

func parseExtendedDuration(value string) (time.Duration, error) {
	return config.ParseDurationExtended(value)
}

func parseTimeFlexible(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	layouts := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02", time.RFC1123, time.RFC822}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}
