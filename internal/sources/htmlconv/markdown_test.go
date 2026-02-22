package htmlconv

import (
	"strings"
	"testing"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
)

func TestConvertHTMLToMarkdown_Strong(t *testing.T) {
	md, err := ConvertHTMLToMarkdown(`<p><strong>Bold Text</strong></p>`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if md != "**Bold Text**" {
		t.Fatalf("expected '**Bold Text**', got %q", md)
	}
}

func TestConvertHTMLToMarkdown_PlainTextPassThrough(t *testing.T) {
	in := "already markdown-ish *text*"
	md, err := ConvertHTMLToMarkdown(in)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if md != in {
		t.Fatalf("expected pass-through %q, got %q", in, md)
	}
}

func TestConvertHTMLToMarkdown_EmptyString(t *testing.T) {
	md, err := ConvertHTMLToMarkdown("")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if md != "" {
		t.Fatalf("expected empty string, got %q", md)
	}
}

func TestConvertHTMLToMarkdown_InvalidHTML_Graceful(t *testing.T) {
	md, err := ConvertHTMLToMarkdown(`<p><strong>Bold Text`)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(md, "Bold Text") {
		t.Fatalf("expected output to contain %q, got %q", "Bold Text", md)
	}
}

func TestConvertHTMLToMarkdown_ConverterErrorPropagates(t *testing.T) {
	prev := newHTMLToMarkdownConverter
	t.Cleanup(func() { newHTMLToMarkdownConverter = prev })

	newHTMLToMarkdownConverter = func() *converter.Converter {
		return converter.NewConverter()
	}

	_, err := ConvertHTMLToMarkdown("<p>hi</p>")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestConvertHTMLToMarkdown_ConverterMisconfigurationErrors(t *testing.T) {
	prev := newHTMLToMarkdownConverter
	t.Cleanup(func() { newHTMLToMarkdownConverter = prev })

	newHTMLToMarkdownConverter = func() *converter.Converter {
		return converter.NewConverter(
			converter.WithEscapeMode("smart"),
			converter.WithPlugins(commonmark.NewCommonmarkPlugin()),
		)
	}

	_, err := ConvertHTMLToMarkdown("<p>hi</p>")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
