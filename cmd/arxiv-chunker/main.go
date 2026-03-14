package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/sources/arxiv"
	doclingimpl "github.com/bakkerme/curator-ai/internal/sources/docling/impl"
)

func main() {
	env, err := config.LoadEnv()
	if err != nil {
		log.Panicf("failed to load environment: %v", err)
	}

	url := flag.String("url", "", "arXiv PDF URL to fetch and chunk")
	mode := flag.String("mode", "all", "chunking mode to preview: all, section, or size")
	abstract := flag.String("abstract", "", "optional abstract text to prepend when include-abstract is enabled")
	includeAbstract := flag.Bool("include-abstract", false, "include the supplied abstract as the first chunk")
	fallbackMaxChars := flag.Int("fallback-max-chars", 4000, "max characters for size chunking and section fallback")
	minSectionChars := flag.Int("min-section-chars", 400, "merge section chunks smaller than this threshold")
	flag.Parse()

	if strings.TrimSpace(*url) == "" {
		log.Fatal("-url is required")
	}

	reader := doclingimpl.NewReader(env.Docling.HTTPTimeout, env.Docling.BaseURL)
	content, err := reader.Read(context.Background(), strings.TrimSpace(*url))
	if err != nil {
		log.Fatalf("failed to fetch PDF content: %v", err)
	}

	selectedModes, err := resolveModes(*mode)
	if err != nil {
		log.Fatal(err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Fetched %d characters from %s\n", len([]rune(content)), strings.TrimSpace(*url))
	for _, selectedMode := range selectedModes {
		cfg := &config.ArxivChunkingConfig{
			Mode:             selectedMode,
			FallbackMaxChars: *fallbackMaxChars,
			MinSectionChars:  *minSectionChars,
		}
		preview := arxiv.PreviewChunking(content, *abstract, *includeAbstract, cfg)
		printPreview(os.Stdout, preview)
	}
}

// resolveModes expands the CLI mode selector into the specific chunking modes to run.
func resolveModes(raw string) ([]string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "all":
		return []string{"section", "size"}, nil
	case "section":
		return []string{"section"}, nil
	case "size":
		return []string{"size"}, nil
	default:
		return nil, fmt.Errorf("invalid -mode %q (expected all, section, or size)", raw)
	}
}

// printPreview writes a stable, human-readable representation of each chunk so the
// current heuristics can be compared against real papers.
func printPreview(out *os.File, preview arxiv.ChunkingPreview) {
	_, _ = fmt.Fprintf(out, "\n=== MODE: %s ===\n", preview.Mode)
	if preview.Mode == "section" {
		_, _ = fmt.Fprintf(out, "headings_found=%t fallback_to_size=%t\n", preview.HeadingsFound, preview.UsedFallback)
	}
	_, _ = fmt.Fprintf(out, "chunk_count=%d\n", len(preview.Chunks))

	for i, chunk := range preview.Chunks {
		length := len([]rune(chunk.Content))
		_, _ = fmt.Fprintf(out, "\n--- CHUNK %d (%d chars) ---\n", i+1, length)
		_, _ = fmt.Fprintln(out, chunk.Content)
	}
}
