package mock

import (
	"context"

	"github.com/bakkerme/curator-ai/internal/sources/rss"
)

type Fetcher struct {
	ItemsByFeed map[string][]rss.Item
	ErrByFeed   map[string]error
}

func (f *Fetcher) Fetch(ctx context.Context, feedURL string, options rss.FetchOptions) ([]rss.Item, error) {
	_ = ctx
	if f.ErrByFeed != nil {
		if err, ok := f.ErrByFeed[feedURL]; ok {
			return nil, err
		}
	}
	items := f.ItemsByFeed[feedURL]
	if options.Limit > 0 && len(items) > options.Limit {
		return items[:options.Limit], nil
	}
	return items, nil
}
