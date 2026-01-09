package rss

import (
	"strings"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
)

func ConvertHTMLToMarkdown(html string) (string, error) {
	if html == "" {
		return "", nil
	}

	// Fast path: if there's no tag-ish content, avoid converting (and potentially escaping) plain text.
	if !strings.Contains(html, "<") {
		return html, nil
	}

	conv := converter.NewConverter(
		converter.WithEscapeMode("smart"),
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
		),
	)
	md, err := conv.ConvertString(html)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(md), nil
}
