package reader

import "context"

// Reader fetches a URL and returns its content as markdown or plain text.
type Reader interface {
	Read(ctx context.Context, url string) (string, error)
}
