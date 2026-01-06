package mock

import (
	"context"

	"github.com/bakkerme/curator-ai/internal/sources/reddit"
)

type Fetcher struct {
	Items []reddit.Item
	Err   error
}

func (f *Fetcher) Fetch(ctx context.Context, config reddit.Config) ([]reddit.Item, error) {
	_ = ctx
	_ = config
	if f.Err != nil {
		return nil, f.Err
	}
	return f.Items, nil
}
