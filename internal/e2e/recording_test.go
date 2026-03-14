//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bakkerme/curator-ai/internal/llm/recording"
)

// TestRecordingLLMClient verifies the full record/replay lifecycle by running
// the curator binary twice:
//  1. Record mode: against a mock OpenAI-compatible server, producing a tape.
//  2. Replay mode: using the recorded tape, no real LLM calls.
//
// Both runs must succeed and the replay run must not make any real LLM calls.
func TestRecordingLLMClient(t *testing.T) {
	if os.Getenv("CURATOR_E2E") == "" {
		t.Skip("set CURATOR_E2E=1 to enable e2e tests")
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}

	// ---- mock OpenAI server ------------------------------------------------
	var llmCallCount atomic.Int64
	mockLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/chat/completions") {
			http.NotFound(w, r)
			return
		}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()

		var req struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.Unmarshal(body, &req)

		n := llmCallCount.Add(1)

		// Determine response based on system prompt content.
		systemPrompt := ""
		for _, m := range req.Messages {
			if m.Role == "system" {
				systemPrompt = m.Content
				break
			}
		}

		var content string
		if strings.Contains(systemPrompt, "quality") || strings.Contains(systemPrompt, "score") || strings.Contains(systemPrompt, "relevance") {
			content = fmt.Sprintf(`{"score": 0.95, "reason": "high quality test content (call %d)"}`, n)
		} else {
			content = fmt.Sprintf("This is a mock summary of the test content. (call %d)", n)
		}

		resp := map[string]interface{}{
			"id":      fmt.Sprintf("mock-%d", n),
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": content,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]interface{}{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(mockLLM.Close)

	// ---- mock SMTP server (accepts and discards) ---------------------------
	smtpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen smtp: %v", err)
	}
	t.Cleanup(func() { smtpListener.Close() })
	smtpAddr := smtpListener.Addr().(*net.TCPAddr)

	go func() {
		for {
			conn, err := smtpListener.Accept()
			if err != nil {
				return
			}
			go handleSMTP(conn)
		}
	}()

	// ---- fixture files -----------------------------------------------------
	tmpDir := t.TempDir()

	testContentPath := filepath.Join(tmpDir, "test_content.md")
	if err := os.WriteFile(testContentPath, []byte(testContentMD), 0o600); err != nil {
		t.Fatalf("write test content: %v", err)
	}

	flowYAML := strings.ReplaceAll(recordingFlowYAML, "__TEST_FILE_PATH__", testContentPath)
	flowYAML = strings.ReplaceAll(flowYAML, "__SMTP_PORT__", fmt.Sprintf("%d", smtpAddr.Port))
	flowFile := filepath.Join(tmpDir, "flow.yml")
	if err := os.WriteFile(flowFile, []byte(flowYAML), 0o600); err != nil {
		t.Fatalf("write flow file: %v", err)
	}

	tapePath := filepath.Join(tmpDir, "recorded.tape.json")

	// ---- step 1: record mode -----------------------------------------------
	t.Run("record", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "go", "run", "./cmd/curator", "-config", flowFile, "-run-once")
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(),
			"OPENAI_API_KEY=test-key",
			"OPENAI_BASE_URL="+mockLLM.URL+"/v1",
			"CURATOR_LLM_RECORD="+tapePath,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("curator record run failed: %v\n%s", err, out)
		}
		t.Logf("record output:\n%s", out)

		if _, err := os.Stat(tapePath); err != nil {
			t.Fatalf("tape file not created: %v", err)
		}

		tape, err := recording.LoadTape(tapePath)
		if err != nil {
			t.Fatalf("load tape: %v", err)
		}
		if len(tape.Interactions) == 0 {
			t.Fatal("tape has no interactions")
		}
		t.Logf("tape has %d interactions", len(tape.Interactions))

		for i, interaction := range tape.Interactions {
			if interaction.Key == "" {
				t.Errorf("interaction %d has empty key", i)
			}
			if interaction.Response.Content == "" && interaction.Error == "" {
				t.Errorf("interaction %d has empty response and no error", i)
			}
		}
	})

	// ---- step 2: replay mode -----------------------------------------------
	t.Run("replay", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		llmCallCount.Store(0)

		cmd := exec.CommandContext(ctx, "go", "run", "./cmd/curator", "-config", flowFile, "-run-once")
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(),
			"CURATOR_LLM_REPLAY="+tapePath,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("curator replay run failed: %v\n%s", err, out)
		}
		t.Logf("replay output:\n%s", out)

		if n := llmCallCount.Load(); n != 0 {
			t.Errorf("expected 0 real LLM calls during replay, got %d", n)
		}
	})
}

// handleSMTP implements a minimal SMTP conversation that accepts any message.
func handleSMTP(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	write := func(s string) {
		conn.Write([]byte(s + "\r\n"))
	}

	write("220 localhost mock SMTP")

	buf := make([]byte, 4096)
	inData := false
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}
		lines := strings.Split(string(buf[:n]), "\r\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if inData {
				if line == "." {
					inData = false
					write("250 OK")
				}
				continue
			}
			upper := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
				write("250-localhost")
				write("250 OK")
			case strings.HasPrefix(upper, "MAIL FROM"):
				write("250 OK")
			case strings.HasPrefix(upper, "RCPT TO"):
				write("250 OK")
			case strings.HasPrefix(upper, "DATA"):
				inData = true
				write("354 Start mail input")
			case strings.HasPrefix(upper, "QUIT"):
				write("221 Bye")
				return
			default:
				write("250 OK")
			}
		}
	}
}

const testContentMD = `# Advances in Large Language Model Alignment

Recent research in AI alignment has focused on several key areas:

## RLHF Improvements
Researchers have developed new techniques for Reinforcement Learning from Human
Feedback that reduce the amount of human annotation required while maintaining
alignment quality. These approaches use synthetic preference data generated by
the model itself, combined with constitutional AI principles.

## Evaluation Frameworks
New benchmarks have been proposed to measure model alignment across multiple
dimensions including helpfulness, harmlessness, and honesty. These frameworks
move beyond simple accuracy metrics to capture nuanced aspects of model behavior.

## Key Findings
- Constitutional AI reduces reliance on human feedback by 40%
- Multi-objective alignment improves safety without sacrificing capability
- Automated red-teaming discovers novel failure modes missed by human testers
`

const recordingFlowYAML = `workflow:
  name: "Recording LLM E2E"
  trigger:
    - cron:
        schedule: "* * * * *"
  sources:
    - testfile:
        path: "__TEST_FILE_PATH__"
  quality:
    - llm:
        name: "relevance_check"
        system_template: "You are a quality evaluator. Assess the relevance and quality of content. Respond with a JSON object containing a score (0-1) and reason."
        prompt_template: "Evaluate the following content for quality and relevance:\n\nTitle: {{.Title}}\nContent: {{.Content}}"
        threshold: 0.5
  post_summary:
    - llm:
        name: "content_summary"
        context: "post"
        system_template: "You are a summarization assistant. Provide a concise summary of the given content."
        prompt_template: "Summarize the following content:\n\nTitle: {{.Title}}\nContent: {{.Content}}"
  output:
    - email:
        to: "dev@example.com"
        from: "curator@example.com"
        subject: "Recording E2E Test"
        smtp_host: "127.0.0.1"
        smtp_port: __SMTP_PORT__
        smtp_tls_mode: "disabled"
        template: |-
          <html><body>
          {{ range .Blocks -}}
            <h2>{{ .Title }}</h2>
            {{ if .Summary }}<p>{{ .Summary.Summary }}</p>{{ end }}
          {{- end }}
          </body></html>
`
