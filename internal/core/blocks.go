package core

import (
	"net/http"
	"time"
)

// PostBlock contains the data and metadata of a Post, including everything
// needed to represent and operate on the Post as it flows through the pipeline
type PostBlock struct {
	FlowID      string         `json:"flow_id" yaml:"flow_id"`
	ID          string         `json:"id" yaml:"id"`
	URL         string         `json:"url" yaml:"url"`
	Title       string         `json:"title" yaml:"title"`
	Content     string         `json:"content" yaml:"content"`
	Author      string         `json:"author" yaml:"author"`
	CreatedAt   time.Time      `json:"created_at" yaml:"created_at"`
	Comments    []CommentBlock `json:"comments,omitempty" yaml:"comments,omitempty"`
	WebBlocks   []WebBlock     `json:"web_blocks,omitempty" yaml:"web_blocks,omitempty"`
	ImageBlocks []ImageBlock   `json:"image_blocks,omitempty" yaml:"image_blocks,omitempty"`
	Chunks      []ContentChunk `json:"chunks,omitempty" yaml:"chunks,omitempty"`
	SummaryPlan *SummaryPlan   `json:"summary_plan,omitempty" yaml:"summary_plan,omitempty"`
	Summary     *SummaryResult `json:"summary,omitempty" yaml:"summary,omitempty"`
	Quality     *QualityResult `json:"quality,omitempty" yaml:"quality,omitempty"`
	ProcessedAt time.Time      `json:"processed_at" yaml:"processed_at"`
	Errors      []ProcessError `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// SummaryMode describes how summarization processors should interpret a PostBlock.
type SummaryMode string

const (
	SummaryModeFull      SummaryMode = "full"
	SummaryModePerChunk  SummaryMode = "per_chunk"
	SummaryModeMapReduce SummaryMode = "map_reduce"
)

// ContentChunk contains raw chunk content plus any chunk-level summary.
// Chunk summaries are written by summary processors, not sources.
type ContentChunk struct {
	Content string `json:"content" yaml:"content"`
	Summary string `json:"summary,omitempty" yaml:"summary,omitempty"`
}

// SummaryPlan is an intent signal for summary processors describing how to handle the PostBlock.
// Summary processors may ignore this if they do not support the requested mode.
type SummaryPlan struct {
	Mode          SummaryMode `json:"mode" yaml:"mode"`
	MaxChunkChars int         `json:"max_chunk_chars,omitempty" yaml:"max_chunk_chars,omitempty"`
	ChunkLimit    int         `json:"chunk_limit,omitempty" yaml:"chunk_limit,omitempty"`
}

// CommentBlock contains data and metadata representing a Comment attached to a Post
type CommentBlock struct {
	ID            string         `json:"id" yaml:"id"`
	Author        string         `json:"author" yaml:"author"`
	Content       string         `json:"content" yaml:"content"`
	CreatedAt     time.Time      `json:"created_at" yaml:"created_at"`
	Comments      []string       `json:"comments,omitempty" yaml:"comments,omitempty"`
	URLs          []WebBlock     `json:"urls,omitempty" yaml:"urls,omitempty"`
	Images        []ImageBlock   `json:"images,omitempty" yaml:"images,omitempty"`
	WasSummarised bool           `json:"was_summarised" yaml:"was_summarised"`
	Summary       string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	Quality       *QualityResult `json:"quality,omitempty" yaml:"quality,omitempty"`
}

// WebBlock contains the data and metadata of a website
// This can represent a URL attached to a post and can optionally be used
// to scrape data from that page for further processing
type WebBlock struct {
	URL           string         `json:"url" yaml:"url"`
	WasFetched    bool           `json:"was_fetched" yaml:"was_fetched"`
	Page          string         `json:"page,omitempty" yaml:"page,omitempty"`
	Request       *http.Request  `json:"request,omitempty" yaml:"request,omitempty"`
	Summary       string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	WasSummarised bool           `json:"was_summarised" yaml:"was_summarised"`
	Quality       *QualityResult `json:"quality,omitempty" yaml:"quality,omitempty"`
}

// ImageBlock contains the data and metadata of an Image, including a URL source
// An Image might start life as a URL parsed from another Block that matches
// existing patterns for a URL that contains an image
type ImageBlock struct {
	URL           string         `json:"url" yaml:"url"`
	ImageData     []byte         `json:"image_data,omitempty" yaml:"image_data,omitempty"`
	WasFetched    bool           `json:"was_fetched" yaml:"was_fetched"`
	Summary       string         `json:"summary,omitempty" yaml:"summary,omitempty"`
	WasSummarised bool           `json:"was_summarised" yaml:"was_summarised"`
	Quality       *QualityResult `json:"quality,omitempty" yaml:"quality,omitempty"`
}

// QualityResult represents the output of quality assessment processors
type QualityResult struct {
	ProcessorName string            `json:"processor_name" yaml:"processor_name"`
	Result        string            `json:"result" yaml:"result"` // "pass", "drop"
	Score         float64           `json:"score,omitempty" yaml:"score,omitempty"`
	Reason        string            `json:"reason,omitempty" yaml:"reason,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	ProcessedAt   time.Time         `json:"processed_at" yaml:"processed_at"`
}

// SummaryResult represents the output of summarization processors
type SummaryResult struct {
	ProcessorName string            `json:"processor_name" yaml:"processor_name"`
	Summary       string            `json:"summary" yaml:"summary"`
	HTML          string            `json:"html,omitempty" yaml:"html,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	ProcessedAt   time.Time         `json:"processed_at" yaml:"processed_at"`
}

// ProcessError tracks errors that occur during processing
type ProcessError struct {
	ProcessorName string    `json:"processor_name" yaml:"processor_name"`
	Stage         string    `json:"stage" yaml:"stage"` // "trigger", "source", "quality", "summary", "output"
	Error         string    `json:"error" yaml:"error"`
	OccurredAt    time.Time `json:"occurred_at" yaml:"occurred_at"`
}

// RunSummary represents the aggregate summary across all posts in a run
type RunSummary struct {
	ProcessorName string            `json:"processor_name" yaml:"processor_name"`
	Summary       string            `json:"summary" yaml:"summary"`
	HTML          string            `json:"html,omitempty" yaml:"html,omitempty"`
	PostCount     int               `json:"post_count" yaml:"post_count"`
	Metadata      map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	ProcessedAt   time.Time         `json:"processed_at" yaml:"processed_at"`
}
