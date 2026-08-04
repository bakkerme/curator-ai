package rss

import "github.com/bakkerme/curator-ai/internal/htmlutil"

// ConvertHTMLToMarkdown converts an HTML string to GitHub Flavored Markdown.
// Delegates to htmlutil.ConvertHTMLToMarkdown; kept here for backward compatibility.
func ConvertHTMLToMarkdown(html string) (string, error) {
	return htmlutil.ConvertHTMLToMarkdown(html)
}
