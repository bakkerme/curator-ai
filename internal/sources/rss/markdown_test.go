package rss

import (
	"strings"
	"testing"
)

// Basic tests verify the rss.ConvertHTMLToMarkdown wrapper delegates correctly.
// The full converter test suite lives in internal/htmlutil/markdown_test.go.

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
