package source

import (
	"strings"
	"testing"
)

func TestChunkArxivContent_SplitsBySectionAndIncludesAbstract(t *testing.T) {
	content := strings.Join([]string{
		"# Introduction",
		"Intro content.",
		"",
		"## Methods",
		"Method content.",
	}, "\n")

	chunks := chunkArxivContent(content, "Abstract text.", true, arxivChunkingConfig{
		mode:             "section",
		fallbackMaxChars: 1000,
		minSectionChars:  1,
	})

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if !strings.Contains(chunks[0].Content, "Abstract text.") {
		t.Fatalf("expected abstract context in first chunk")
	}
	if !strings.Contains(chunks[0].Content, "Section: Introduction") {
		t.Fatalf("expected section title in first chunk")
	}
}

func TestChunkArxivContent_FallbacksToSizeChunking(t *testing.T) {
	content := "no headings here"
	chunks := chunkArxivContent(content, "", false, arxivChunkingConfig{
		mode:             "section",
		fallbackMaxChars: 5,
		minSectionChars:  1,
	})

	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
}

func TestChunkArxivContent_EmptyContentAbstractPresent_NoDupWhenIncludeAbstractTrue(t *testing.T) {
	chunks := chunkArxivContent("", "Abstract text.", true, arxivChunkingConfig{
		mode:             "section",
		fallbackMaxChars: 1000,
		minSectionChars:  1,
	})

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "Abstract text." {
		t.Fatalf("expected abstract once, got %q", chunks[0].Content)
	}
}

func TestChunkArxivContent_EmptyContentAbstractPresent_IncludeAbstractFalseStillReturnsAbstract(t *testing.T) {
	chunks := chunkArxivContent("", "Abstract text.", false, arxivChunkingConfig{
		mode:             "section",
		fallbackMaxChars: 1000,
		minSectionChars:  1,
	})

	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0].Content != "Abstract text." {
		t.Fatalf("expected abstract content, got %q", chunks[0].Content)
	}
}
