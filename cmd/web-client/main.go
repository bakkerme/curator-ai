package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bakkerme/curator-ai/internal/web"
	"github.com/gorilla/websocket"
)

const (
	defaultBaseURL = "http://localhost:8080"
)

var (
	pollInterval           = time.Second
	stdout       io.Writer = os.Stdout
)

// apiClient wraps the Curator web API calls used by the test client.
type apiClient struct {
	baseURL    string
	wsBaseURL  string
	httpClient *http.Client
}

// newAPIClient builds an API client and derives the websocket base URL when it is not provided.
func newAPIClient(baseURL, wsBaseURL string) (*apiClient, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	if wsBaseURL == "" {
		derived, err := deriveWebsocketBaseURL(baseURL)
		if err != nil {
			return nil, err
		}
		wsBaseURL = derived
	}
	return &apiClient{
		baseURL:   baseURL,
		wsBaseURL: strings.TrimRight(strings.TrimSpace(wsBaseURL), "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}, nil
}

// listDocs fetches all registered documents from the web API.
func (c *apiClient) listDocs(ctx context.Context) ([]web.DocStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/docs", nil)
	if err != nil {
		return nil, fmt.Errorf("build list docs request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list docs request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, decodeAPIError(resp)
	}

	var payload web.ListDocsResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode list docs response: %w", err)
	}
	return payload.Docs, nil
}

// triggerRun starts a run for the requested document.
func (c *apiClient) triggerRun(ctx context.Context, docID string) (web.TriggerRunResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/docs/"+url.PathEscape(docID)+"/run", nil)
	if err != nil {
		return web.TriggerRunResponse{}, fmt.Errorf("build trigger run request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return web.TriggerRunResponse{}, fmt.Errorf("trigger run request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		return web.TriggerRunResponse{}, decodeAPIError(resp)
	}

	var payload web.TriggerRunResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return web.TriggerRunResponse{}, fmt.Errorf("decode trigger run response: %w", err)
	}
	return payload, nil
}

// waitForRunCompletion polls the doc status until the selected run reaches a terminal state.
func (c *apiClient) waitForRunCompletion(ctx context.Context, docID, runID string) (web.RunInfo, error) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		info, done, err := c.lookupRun(ctx, docID, runID)
		if err != nil {
			return web.RunInfo{}, err
		}
		if done {
			return info, nil
		}

		select {
		case <-ctx.Done():
			return web.RunInfo{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

// lookupRun finds the current status for a specific run by polling the docs endpoint.
func (c *apiClient) lookupRun(ctx context.Context, docID, runID string) (web.RunInfo, bool, error) {
	docs, err := c.listDocs(ctx)
	if err != nil {
		return web.RunInfo{}, false, err
	}
	for _, doc := range docs {
		if doc.ID != docID || doc.LastRun == nil || doc.LastRun.ID != runID {
			continue
		}
		switch doc.LastRun.Status {
		case web.RunStatusCompleted, web.RunStatusFailed, web.RunStatusCancelled:
			return *doc.LastRun, true, nil
		default:
			return *doc.LastRun, false, nil
		}
	}
	return web.RunInfo{}, false, nil
}

// streamLogs connects to the run log websocket and writes each log line to the provided writer.
func (c *apiClient) streamLogs(ctx context.Context, runID string, out io.Writer) error {
	wsURL := c.wsBaseURL + "/api/runs/" + url.PathEscape(runID) + "/logs"
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, wsURL, nil)
	if err != nil {
		return fmt.Errorf("connect websocket: %w", err)
	}
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()

	for {
		var entry web.LogEntry
		if err := conn.ReadJSON(&entry); err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) {
				return nil
			}
			return fmt.Errorf("read websocket log entry: %w", err)
		}
		fmt.Fprintf(out, "%s %-5s %s\n", entry.Time.Format(time.RFC3339), entry.Level, entry.Message)
	}
}

func main() {
	command, args := splitCommandAndArgs(os.Args[1:])

	fs := flag.NewFlagSet("web-client", flag.ExitOnError)
	var (
		baseURL   string
		wsBaseURL string
		docID     string
		timeout   time.Duration
		jsonOut   bool
	)

	fs.StringVar(&baseURL, "base-url", defaultBaseURL, "base HTTP URL for the Curator web API")
	fs.StringVar(&wsBaseURL, "ws-base-url", "", "base websocket URL for the Curator web API; derived from -base-url when empty")
	fs.StringVar(&docID, "doc", "", "document ID to run; if empty the client lists docs and requires exactly one document")
	fs.DurationVar(&timeout, "timeout", 0, "optional overall timeout for the end-to-end run")
	fs.BoolVar(&jsonOut, "json", false, "print docs as JSON for the list command")
	fs.Parse(args)

	client, err := newAPIClient(baseURL, wsBaseURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	switch command {
	case "list":
		if err := runList(ctx, client, jsonOut); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	case "run":
		if err := runEndToEnd(ctx, client, docID); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q (supported: list, run)\n", command)
		os.Exit(1)
	}
}

// splitCommandAndArgs allows both:
//
//	web-client run -doc X
//	web-client -doc X run
//
// by treating the first non-flag token ("run" or "list") as the command.
func splitCommandAndArgs(raw []string) (string, []string) {
	command := "run"
	for i, token := range raw {
		if token == "run" || token == "list" {
			command = token
			out := make([]string, 0, len(raw)-1)
			out = append(out, raw[:i]...)
			out = append(out, raw[i+1:]...)
			return command, out
		}
	}
	return command, raw
}

// runList prints the available docs in either table or JSON form.
func runList(ctx context.Context, client *apiClient, jsonOut bool) error {
	docs, err := client.listDocs(ctx)
	if err != nil {
		return err
	}
	if jsonOut {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(web.ListDocsResponse{Docs: docs})
	}
	printDocs(stdout, docs)
	return nil
}

// runEndToEnd lists docs, chooses a target flow, starts a run, streams logs, and waits for completion.
func runEndToEnd(ctx context.Context, client *apiClient, requestedDocID string) error {
	docs, err := client.listDocs(ctx)
	if err != nil {
		return err
	}
	if len(docs) == 0 {
		return fmt.Errorf("no docs registered at %s", client.baseURL)
	}

	printDocs(stdout, docs)

	selectedDocID, err := selectDocID(docs, requestedDocID)
	if err != nil {
		return err
	}

	resp, err := client.triggerRun(ctx, selectedDocID)
	if err != nil {
		return err
	}

	fmt.Fprintf(stdout, "\nStarted run %s for doc %s at %s\n\n", resp.RunID, resp.FlowID, resp.StartedAt.Format(time.RFC3339))

	streamCtx, cancelStream := context.WithCancel(ctx)
	defer cancelStream()

	var wg sync.WaitGroup
	var streamErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		streamErr = client.streamLogs(streamCtx, resp.RunID, stdout)
	}()

	runInfo, waitErr := client.waitForRunCompletion(ctx, resp.FlowID, resp.RunID)
	cancelStream()
	wg.Wait()

	if streamErr != nil && !isExpectedStreamShutdown(streamErr, waitErr) {
		return streamErr
	}
	if waitErr != nil {
		return waitErr
	}

	if runInfo.CompletedAt != nil {
		fmt.Fprintf(stdout, "\nRun %s finished with status=%s at %s\n", runInfo.ID, runInfo.Status, runInfo.CompletedAt.Format(time.RFC3339))
	} else {
		fmt.Fprintf(stdout, "\nRun %s finished with status=%s\n", runInfo.ID, runInfo.Status)
	}
	if runInfo.Error != "" {
		fmt.Fprintf(stdout, "Run error: %s\n", runInfo.Error)
	}
	if runInfo.Status == web.RunStatusFailed || runInfo.Status == web.RunStatusCancelled {
		return fmt.Errorf("run %s finished with status %s", runInfo.ID, runInfo.Status)
	}
	return nil
}

// isExpectedStreamShutdown recognizes the websocket goroutine exiting because the run already finished.
func isExpectedStreamShutdown(streamErr, waitErr error) bool {
	if streamErr == nil {
		return true
	}
	if waitErr != nil {
		return false
	}
	if errors.Is(streamErr, context.Canceled) || errors.Is(streamErr, context.DeadlineExceeded) {
		return true
	}
	msg := strings.ToLower(streamErr.Error())
	return strings.Contains(msg, "operation was canceled") || strings.Contains(msg, "use of closed network connection")
}

// selectDocID validates or infers the target doc ID for a run.
func selectDocID(docs []web.DocStatus, requestedDocID string) (string, error) {
	if requestedDocID != "" {
		var matches []web.DocStatus
		for _, doc := range docs {
			if matchesRequestedDoc(doc, requestedDocID) {
				matches = append(matches, doc)
			}
		}
		switch len(matches) {
		case 1:
			return matches[0].ID, nil
		case 0:
			return "", fmt.Errorf("doc %q was not found", requestedDocID)
		default:
			return "", fmt.Errorf("doc %q matched multiple docs; use the exact flow id", requestedDocID)
		}
	}
	if len(docs) == 1 {
		return docs[0].ID, nil
	}
	return "", fmt.Errorf("multiple docs registered; rerun with -doc <id>")
}

// matchesRequestedDoc accepts the flow ID and a few human-friendly aliases.
func matchesRequestedDoc(doc web.DocStatus, requested string) bool {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return false
	}
	if doc.ID == requested || doc.SourcePath == requested {
		return true
	}

	base := filepath.Base(doc.SourcePath)
	if base == requested {
		return true
	}

	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return name == requested
}

// printDocs renders a small table that is easy to read in a terminal.
func printDocs(out io.Writer, docs []web.DocStatus) {
	fmt.Fprintln(out, "Available docs:")
	for _, doc := range docs {
		status := "never-run"
		if doc.LastRun != nil {
			status = string(doc.LastRun.Status)
		}
		active := ""
		if doc.ActiveRunID != "" {
			active = " active_run=" + doc.ActiveRunID
		}
		fmt.Fprintf(out, "- %s\t%s\tstatus=%s%s\t%s\n", doc.ID, doc.Name, status, active, doc.SourcePath)
	}
}

// decodeAPIError converts a non-2xx API response into a readable error.
func decodeAPIError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr web.APIResponse
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error != "" {
		return fmt.Errorf("api error (%d): %s", resp.StatusCode, apiErr.Error)
	}
	if len(body) == 0 {
		return fmt.Errorf("api error (%d)", resp.StatusCode)
	}
	return fmt.Errorf("api error (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
}

// deriveWebsocketBaseURL maps an http(s) API URL onto ws(s).
func deriveWebsocketBaseURL(baseURL string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	switch parsed.Scheme {
	case "http":
		parsed.Scheme = "ws"
	case "https":
		parsed.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("unsupported base URL scheme %q", parsed.Scheme)
	}
	parsed.Path = ""
	parsed.RawPath = ""
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return strings.TrimRight(parsed.String(), "/"), nil
}
