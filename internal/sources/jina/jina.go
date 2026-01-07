package jina

import "context"

// ReadOptions controls Jina Reader behavior.
type ReadOptions struct {
	// RetainImages controls the "X-Retain-Images" header. Defaults to "none".
	RetainImages string
}

// Reader fetches a URL via Jina Reader and returns markdown.
type Reader interface {
	Read(ctx context.Context, url string, options ReadOptions) (string, error)
}

