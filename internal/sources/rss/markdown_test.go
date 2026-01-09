package rss

import "testing"

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
