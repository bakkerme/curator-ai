package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
)

var defaultRenderer = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
)

// Render converts markdown into HTML using Curator's shared renderer settings.
func Render(input string) (string, error) {
	if input == "" {
		return "", nil
	}

	var buf bytes.Buffer
	if err := defaultRenderer.Convert([]byte(input), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}
