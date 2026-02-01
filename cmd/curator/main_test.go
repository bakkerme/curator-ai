package main

import (
	"path/filepath"
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
)

func TestDefaultFlowID_WorkflowNameSet(t *testing.T) {
	t.Parallel()

	docPath := filepath.Join("some", "path", "rss.yaml")
	doc := &config.CuratorDocument{
		Workflow: config.Workflow{
			Name: "My RSS Flow",
		},
	}

	if got, want := defaultFlowID(docPath, doc), "my-rss-flow"; got != want {
		t.Fatalf("defaultFlowID() = %q, want %q", got, want)
	}
}

func TestDefaultFlowID_WorkflowNameWhitespaceFallsBackToFilename(t *testing.T) {
	t.Parallel()

	docPath := filepath.Join("some", "path", "rss.yaml")
	doc := &config.CuratorDocument{
		Workflow: config.Workflow{
			Name: "   \n\t  ",
		},
	}

	if got, want := defaultFlowID(docPath, doc), "rss"; got != want {
		t.Fatalf("defaultFlowID() = %q, want %q", got, want)
	}
}

func TestDefaultFlowID_FallsBackToFilenameWhenDocIsNil(t *testing.T) {
	t.Parallel()

	docPath := filepath.Join("some", "path", "reddit-locallama.yaml")

	if got, want := defaultFlowID(docPath, nil), "reddit-locallama"; got != want {
		t.Fatalf("defaultFlowID() = %q, want %q", got, want)
	}
}

func TestDefaultFlowID_StripsOnlyFinalExtension(t *testing.T) {
	t.Parallel()

	docPath := filepath.Join("some", "path", "flow.backup.yaml")
	doc := &config.CuratorDocument{
		Workflow: config.Workflow{
			Name: "",
		},
	}

	// filepath.Ext returns ".yaml", so the base becomes "flow.backup" (not "flow").
	if got, want := defaultFlowID(docPath, doc), "flow-backup"; got != want {
		t.Fatalf("defaultFlowID() = %q, want %q", got, want)
	}
}

func TestDefaultFlowID_HandlesUppercaseExtensions(t *testing.T) {
	t.Parallel()

	docPath := filepath.Join("some", "path", "RSS.YAML")
	doc := &config.CuratorDocument{
		Workflow: config.Workflow{
			Name: "",
		},
	}

	if got, want := defaultFlowID(docPath, doc), "rss"; got != want {
		t.Fatalf("defaultFlowID() = %q, want %q", got, want)
	}
}

func TestSlugify_EmptyString(t *testing.T) {
	t.Parallel()

	if got, want := slugify(""), "flow"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}

func TestSlugify_OnlySpecialCharacters(t *testing.T) {
	t.Parallel()

	if got, want := slugify("!@#$%^&*()"), "flow"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}

func TestSlugify_MixedCaseNumbersAndSpecialCharacters(t *testing.T) {
	t.Parallel()

	if got, want := slugify("Hello, World! 2025"), "hello-world-2025"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}

func TestSlugify_LeadingAndTrailingSeparatorsTrimmed(t *testing.T) {
	t.Parallel()

	if got, want := slugify("---Hello---"), "hello"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}

func TestSlugify_UnicodeLettersArePreserved(t *testing.T) {
	t.Parallel()

	// unicode.IsLetter treats these as letters, so they should remain.
	if got, want := slugify("Café Münchner Kindl"), "café-münchner-kindl"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}

func TestSlugify_UnicodeEmDashBecomesSeparator(t *testing.T) {
	t.Parallel()

	// Em dash isn't a letter/digit, so it becomes a separator.
	if got, want := slugify("Hello—World"), "hello-world"; got != want {
		t.Fatalf("slugify() = %q, want %q", got, want)
	}
}
