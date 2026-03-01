package reddit

import (
	"log/slog"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/enetx/g"
	"github.com/enetx/surf"
)

// newSurfHTTPClient builds the shared Reddit HTTP client stack used by both the
// go-reddit path and the direct public JSON fetcher. It returns a standard
// library-compatible client so existing call sites and retry logic stay intact.
func newSurfHTTPClient(logger *slog.Logger, timeout time.Duration, counter *atomic.Uint64, proxyURL *url.URL) *http.Client {
	builder := surf.NewClient().
		Builder().
		Impersonate().
		RandomOS().
		Firefox()

	if proxyURL != nil {
		builder = builder.Proxy(g.String(proxyURL.String()))
	}

	stdClient := builder.
		Build().
		Unwrap().
		Std()
	stdClient.Timeout = timeout
	stdClient.Transport = &observabilityRoundTripper{
		base:    stdClient.Transport,
		logger:  logger,
		counter: counter,
	}
	return stdClient
}
