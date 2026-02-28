package rss

import (
	"github.com/bakkerme/curator-ai/internal/sources"
)

func ConvertHTMLToMarkdown(html string) (string, error) {
	return sources.ConvertHTMLToMarkdown(html)
}
