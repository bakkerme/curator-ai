package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/web"
	"github.com/gorilla/websocket"
)

func TestDeriveWebsocketBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "http", in: "http://localhost:8080", want: "ws://localhost:8080"},
		{name: "https", in: "https://example.com/api", want: "wss://example.com"},
		{name: "ws passthrough", in: "ws://localhost:8080", want: "ws://localhost:8080"},
		{name: "bad scheme", in: "ftp://example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := deriveWebsocketBaseURL(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("deriveWebsocketBaseURL(%q) error = %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("deriveWebsocketBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestRunEndToEnd(t *testing.T) {
	t.Parallel()

	runID := "run-123"
	docID := "doc-1"
	completedAt := time.Date(2026, 4, 1, 10, 11, 12, 0, time.UTC)
	logEntries := []web.LogEntry{
		{Time: completedAt.Add(-2 * time.Second), Level: "INFO", Message: "run started", RunID: runID, FlowID: docID},
		{Time: completedAt.Add(-1 * time.Second), Level: "INFO", Message: "run completed", RunID: runID, FlowID: docID},
	}

	var pollCount int
	var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/docs":
			pollCount++
			resp := web.ListDocsResponse{
				Docs: []web.DocStatus{{
					ID:         docID,
					Name:       "Test Flow",
					SourcePath: "/tmp/test.yaml",
					LastRun: &web.RunInfo{
						ID:        runID,
						FlowID:    docID,
						StartedAt: completedAt.Add(-5 * time.Second),
						Status:    web.RunStatusRunning,
					},
					IsRunning: true,
				}},
			}
			if pollCount >= 2 {
				resp.Docs[0].LastRun = &web.RunInfo{
					ID:          runID,
					FlowID:      docID,
					StartedAt:   completedAt.Add(-5 * time.Second),
					CompletedAt: &completedAt,
					Status:      web.RunStatusCompleted,
				}
				resp.Docs[0].IsRunning = false
			}
			_ = json.NewEncoder(w).Encode(resp)
		case r.Method == http.MethodPost && r.URL.Path == "/api/docs/doc-1/run":
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(web.TriggerRunResponse{
				RunID:     runID,
				FlowID:    docID,
				StartedAt: completedAt.Add(-5 * time.Second),
			})
		case r.URL.Path == "/api/runs/run-123/logs":
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Fatalf("upgrade failed: %v", err)
			}
			defer conn.Close()
			for _, entry := range logEntries {
				if err := conn.WriteJSON(entry); err != nil {
					return
				}
			}
			time.Sleep(50 * time.Millisecond)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client, err := newAPIClient(server.URL, strings.Replace(server.URL, "http://", "ws://", 1))
	if err != nil {
		t.Fatalf("newAPIClient() error = %v", err)
	}

	oldPollInterval := pollInterval
	pollInterval = 10 * time.Millisecond
	defer func() { pollInterval = oldPollInterval }()

	var output bytes.Buffer
	origStdout := stdout
	stdout = &output
	defer func() { stdout = origStdout }()

	if err := runEndToEnd(context.Background(), client, docID); err != nil {
		t.Fatalf("runEndToEnd() error = %v", err)
	}

	out := output.String()
	if !strings.Contains(out, "Available docs:") {
		t.Fatalf("expected docs output, got %q", out)
	}
	if !strings.Contains(out, "Started run run-123") {
		t.Fatalf("expected started output, got %q", out)
	}
	if !strings.Contains(out, "status=completed") {
		t.Fatalf("expected final status output, got %q", out)
	}
}

func TestSelectDocID_AcceptsSourceFilename(t *testing.T) {
	t.Parallel()

	docs := []web.DocStatus{
		{
			ID:         "bot-defence-and-cybersecurity-research",
			Name:       "Bot Defence and Cybersecurity Research",
			SourcePath: "curator-docs/minimal-rss.yaml",
		},
	}

	got, err := selectDocID(docs, "minimal-rss")
	if err != nil {
		t.Fatalf("selectDocID() error = %v", err)
	}
	if got != docs[0].ID {
		t.Fatalf("selectDocID() = %q, want %q", got, docs[0].ID)
	}
}

func TestSplitCommandAndArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		in          []string
		wantCommand string
		wantArgs    []string
	}{
		{
			name:        "subcommand first",
			in:          []string{"run", "-doc", "minimal-rss"},
			wantCommand: "run",
			wantArgs:    []string{"-doc", "minimal-rss"},
		},
		{
			name:        "flags first",
			in:          []string{"-doc", "minimal-rss", "run"},
			wantCommand: "run",
			wantArgs:    []string{"-doc", "minimal-rss"},
		},
		{
			name:        "list command",
			in:          []string{"list", "-json"},
			wantCommand: "list",
			wantArgs:    []string{"-json"},
		},
		{
			name:        "default run",
			in:          []string{"-doc", "minimal-rss"},
			wantCommand: "run",
			wantArgs:    []string{"-doc", "minimal-rss"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotCommand, gotArgs := splitCommandAndArgs(tt.in)
			if gotCommand != tt.wantCommand {
				t.Fatalf("splitCommandAndArgs(%v) command = %q, want %q", tt.in, gotCommand, tt.wantCommand)
			}
			if strings.Join(gotArgs, "|") != strings.Join(tt.wantArgs, "|") {
				t.Fatalf("splitCommandAndArgs(%v) args = %v, want %v", tt.in, gotArgs, tt.wantArgs)
			}
		})
	}
}
