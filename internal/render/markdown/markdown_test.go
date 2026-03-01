package markdown

import (
	"strings"
	"testing"
)

func TestRenderParagraph(t *testing.T) {
	rendered, err := Render("Hello, world.")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(rendered, "<p>Hello, world.</p>") {
		t.Fatalf("expected paragraph HTML, got %q", rendered)
	}
}

func TestRenderGFMTable(t *testing.T) {
	rendered, err := Render("| A | B |\n| --- | --- |\n| 1 | 2 |\n")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(rendered, "<table>") {
		t.Fatalf("expected table HTML, got %q", rendered)
	}
}

func TestRenderEmptyString(t *testing.T) {
	rendered, err := Render("")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if rendered != "" {
		t.Fatalf("expected empty output, got %q", rendered)
	}
}

func TestRenderOmitsRawHTML(t *testing.T) {
	rendered, err := Render("<strong>unsafe</strong>")
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if strings.Contains(rendered, "<strong>unsafe</strong>") {
		t.Fatalf("expected raw HTML to remain disabled, got %q", rendered)
	}
}
