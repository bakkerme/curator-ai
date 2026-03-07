package arxiv

import (
	"strings"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

// ChunkingPreview captures how one chunking mode behaved for a given paper body.
// This is intended for local debugging so chunk boundaries can be inspected without
// running the full source processor.
type ChunkingPreview struct {
	Mode          string
	HeadingsFound bool
	UsedFallback  bool
	Chunks        []core.ContentChunk
}

// PreviewChunking runs the current arXiv chunking logic and returns the emitted
// chunks plus a small amount of metadata describing whether section detection
// succeeded or fell back to size-based chunking.
func PreviewChunking(content string, abstract string, includeAbstractInChunks bool, cfg *config.ArxivChunkingConfig) ChunkingPreview {
	resolved := defaultArxivChunkingConfig(cfg)
	content = strings.TrimSpace(content)

	preview := ChunkingPreview{
		Mode:   resolved.mode,
		Chunks: chunkArxivContent(content, abstract, includeAbstractInChunks, resolved),
	}

	// Size mode skips section detection entirely, so there is no fallback state.
	if strings.EqualFold(resolved.mode, "size") {
		return preview
	}

	_, headingsFound := splitSections(content)
	preview.HeadingsFound = headingsFound
	preview.UsedFallback = content != "" && !headingsFound
	return preview
}
