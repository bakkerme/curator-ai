package openai

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/bakkerme/curator-ai/internal/config"

	"github.com/openai/openai-go/option"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func openAIMiddleware(cfg config.OpenAIOTelEnvConfig) option.Middleware {
	return func(req *http.Request, next option.MiddlewareNext) (*http.Response, error) {
		span := trace.SpanFromContext(req.Context())
		if cfg.CaptureBodies && span.IsRecording() && req.Body != nil {
			req.Body = newCaptureReadCloser(req.Body, cfg.MaxBodyBytes, func(body []byte, truncated bool) {
				bodyStr := bytesToString(body)
				span.SetAttributes(
					attribute.String("input.mime_type", "application/json"),
					attribute.String("input.value", bodyStr),
					attribute.Bool("input.truncated", truncated),
					attribute.String("openai.request.body", bodyStr),
					attribute.Bool("openai.request.body.truncated", truncated),
				)
				span.AddEvent("openai.request.body", trace.WithAttributes(
					attribute.String("http.method", req.Method),
					attribute.String("http.url", req.URL.String()),
					attribute.String("body", bodyStr),
					attribute.Bool("truncated", truncated),
				))
			})
		}

		res, err := next(req)
		if err != nil {
			return res, err
		}
		if res == nil {
			return res, nil
		}

		if span.IsRecording() {
			span.AddEvent("openai.response.meta", trace.WithAttributes(
				attribute.Int("http.status_code", res.StatusCode),
			))
		}

		if cfg.CaptureBodies && span.IsRecording() && res.Body != nil {
			res.Body = newCaptureReadCloser(res.Body, cfg.MaxBodyBytes, func(body []byte, truncated bool) {
				bodyStr := bytesToString(body)
				span.SetAttributes(
					attribute.String("output.mime_type", "application/json"),
					attribute.String("output.value", bodyStr),
					attribute.Bool("output.truncated", truncated),
					attribute.String("openai.response.body", bodyStr),
					attribute.Bool("openai.response.body.truncated", truncated),
				)
				span.AddEvent("openai.response.body", trace.WithAttributes(
					attribute.Int("http.status_code", res.StatusCode),
					attribute.String("body", bodyStr),
					attribute.Bool("truncated", truncated),
				))
			})
		}

		return res, nil
	}
}

type captureReadCloser struct {
	rc          io.ReadCloser
	maxBytes    int
	buf         bytes.Buffer
	truncated   bool
	onCloseOnce sync.Once
	onClose     func([]byte, bool)
}

func newCaptureReadCloser(rc io.ReadCloser, maxBytes int, onClose func([]byte, bool)) io.ReadCloser {
	if rc == nil {
		return rc
	}
	return &captureReadCloser{rc: rc, maxBytes: maxBytes, onClose: onClose}
}

func (c *captureReadCloser) Read(p []byte) (int, error) {
	n, err := c.rc.Read(p)
	if n > 0 && c.maxBytes != 0 {
		remaining := c.maxBytes - c.buf.Len()
		if c.maxBytes < 0 {
			remaining = n
		}
		if remaining > 0 {
			if remaining >= n {
				_, _ = c.buf.Write(p[:n])
			} else {
				_, _ = c.buf.Write(p[:remaining])
				c.truncated = true
			}
		} else {
			c.truncated = true
		}
	}
	return n, err
}

func (c *captureReadCloser) Close() error {
	c.onCloseOnce.Do(func() {
		if c.onClose != nil {
			c.onClose(c.buf.Bytes(), c.truncated)
		}
	})
	return c.rc.Close()
}

func bytesToString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	// Ensure this is always valid UTF-8 for attribute transport; invalid bytes are replaced.
	return strings.ToValidUTF8(string(b), "\uFFFD")
}
