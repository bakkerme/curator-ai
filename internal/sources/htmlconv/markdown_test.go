package htmlconv

import (
	"strings"
	"testing"
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

func TestConvertHTMLToMarkdown_LargeHTMLInput(t *testing.T) {
	// Keep this reasonably sized so unit tests stay fast, but large enough to
	// exercise the parser/renderer paths.
	var b strings.Builder
	for i := 0; i < 10000; i++ {
		b.WriteString("<p>hello</p>")
	}

	md, err := ConvertHTMLToMarkdown(b.String())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(md, "hello") {
		t.Fatalf("expected output to contain %q, got %q", "hello", md)
	}
}
