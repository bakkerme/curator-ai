package arxiv

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

type arxivChunkingConfig struct {
	mode             string
	fallbackMaxChars int
	minSectionChars  int
}

var (
	sectionNumberPattern = regexp.MustCompile(`^\d+(\.\d+)*\s+\S+`)
	knownSectionPattern  = regexp.MustCompile(`^(abstract|introduction|related work|method|methods|methodology|results|discussion|conclusion|limitations|references|appendix)\b`)
)

func defaultArxivChunkingConfig(cfg *config.ArxivChunkingConfig) arxivChunkingConfig {
	if cfg == nil {
		return arxivChunkingConfig{
			mode:             "section",
			fallbackMaxChars: 4000,
			minSectionChars:  400,
		}
	}
	chunking := arxivChunkingConfig{
		mode:             strings.TrimSpace(cfg.Mode),
		fallbackMaxChars: cfg.FallbackMaxChars,
		minSectionChars:  cfg.MinSectionChars,
	}
	if chunking.mode == "" {
		chunking.mode = "section"
	}
	if chunking.fallbackMaxChars <= 0 {
		chunking.fallbackMaxChars = 4000
	}
	if chunking.minSectionChars <= 0 {
		chunking.minSectionChars = 400
	}
	return chunking
}

func chunkArxivContent(content string, abstract string, includeAbstractInChunks bool, cfg arxivChunkingConfig) []core.ContentChunk {
	content = strings.TrimSpace(content)
	abstract = strings.TrimSpace(abstract)

	if content == "" && abstract != "" {
		// When no full content is available, emit the abstract exactly once.
		return []core.ContentChunk{{Content: buildChunkText("Abstract", abstract)}}
	}

	var chunks []core.ContentChunk
	if includeAbstractInChunks && abstract != "" {
		chunks = append(chunks, core.ContentChunk{Content: buildChunkText("Abstract", abstract)})
	}

	var contentChunks []core.ContentChunk
	if strings.EqualFold(cfg.mode, "size") {
		contentChunks = chunkBySize(content, cfg.fallbackMaxChars)
	} else {
		sections, headingsFound := splitSections(content)
		if !headingsFound {
			contentChunks = chunkBySize(content, cfg.fallbackMaxChars)
		} else {
			merged := mergeSmallSections(sections, cfg.minSectionChars)
			contentChunks = make([]core.ContentChunk, 0, len(merged))
			for _, section := range merged {
				text := buildChunkText(section.title, section.content)
				contentChunks = append(contentChunks, core.ContentChunk{Content: text})
			}
		}
	}

	return append(chunks, contentChunks...)
}

type section struct {
	title   string
	content string
}

func splitSections(content string) ([]section, bool) {
	lines := strings.Split(content, "\n")
	var sections []section
	current := section{}
	var buffer []string
	headingsFound := false

	flush := func() {
		if len(buffer) == 0 {
			return
		}
		current.content = strings.TrimSpace(strings.Join(buffer, "\n"))
		if strings.TrimSpace(current.content) != "" {
			sections = append(sections, current)
		}
		buffer = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if isSectionHeading(trimmed) {
			headingsFound = true
			flush()
			current = section{title: normalizeHeading(trimmed)}
			continue
		}
		buffer = append(buffer, line)
	}
	flush()
	return sections, headingsFound
}

func isSectionHeading(line string) bool {
	if line == "" {
		return false
	}
	if strings.HasPrefix(line, "#") {
		heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
		return heading != ""
	}
	if sectionNumberPattern.MatchString(line) {
		return true
	}
	if knownSectionPattern.MatchString(strings.ToLower(line)) {
		if len(line) > 64 || strings.ContainsAny(line, ".:") {
			return false
		}
		return true
	}
	return false
}

func normalizeHeading(line string) string {
	if strings.HasPrefix(line, "#") {
		return strings.TrimSpace(strings.TrimLeft(line, "#"))
	}
	return strings.TrimSpace(line)
}

func mergeSmallSections(sections []section, minChars int) []section {
	if minChars <= 0 || len(sections) == 0 {
		return sections
	}
	merged := make([]section, 0, len(sections))
	var current *section
	for _, sec := range sections {
		if current == nil {
			secCopy := sec
			current = &secCopy
			continue
		}
		// Use rune count so thresholds are based on characters, not UTF-8 bytes.
		if utf8.RuneCountInString(current.content) < minChars {
			current.content = strings.TrimSpace(current.content + "\n\n" + sec.title + "\n" + sec.content)
			continue
		}
		merged = append(merged, *current)
		secCopy := sec
		current = &secCopy
	}
	if current != nil {
		merged = append(merged, *current)
	}
	return merged
}

func chunkBySize(content string, maxChars int) []core.ContentChunk {
	if maxChars <= 0 {
		maxChars = 4000
	}
	runes := []rune(content)
	chunks := make([]core.ContentChunk, 0, (len(runes)+maxChars-1)/maxChars)
	for start := 0; start < len(runes); start += maxChars {
		end := start + maxChars
		if end > len(runes) {
			end = len(runes)
		}
		segment := string(runes[start:end])
		text := buildChunkText("", segment)
		chunks = append(chunks, core.ContentChunk{Content: text})
	}
	return chunks
}

func buildChunkText(sectionTitle string, content string) string {
	var builder strings.Builder
	if strings.TrimSpace(sectionTitle) != "" {
		if strings.EqualFold(sectionTitle, "Abstract") {
			builder.WriteString(strings.TrimSpace(content))
			return builder.String()
		}
		builder.WriteString("Section: ")
		builder.WriteString(strings.TrimSpace(sectionTitle))
		builder.WriteString("\n")
	}
	builder.WriteString(strings.TrimSpace(content))
	return strings.TrimSpace(builder.String())
}
