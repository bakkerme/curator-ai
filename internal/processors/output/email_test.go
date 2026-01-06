package output

import (
	"strings"
	"testing"

	"github.com/bakkerme/curator-ai/internal/core"
)

func TestRenderEmailTemplate_RendersSummaryHTMLUnescaped(t *testing.T) {
	body, err := renderEmailTemplate(
		`{{.RunSummary.HTML}}{{range .Blocks}}{{.Title}}|{{.Summary.HTML}}{{end}}`,
		[]*core.PostBlock{
			{
				Title: `<b>Title</b>`,
				Summary: &core.SummaryResult{
					HTML: `<p>Post summary</p>`,
				},
			},
		},
		&core.RunSummary{
			HTML: `<p>Run summary</p>`,
		},
	)
	if err != nil {
		t.Fatalf("renderEmailTemplate failed: %v", err)
	}

	if strings.Contains(body, "&lt;p&gt;Run summary&lt;/p&gt;") || !strings.Contains(body, "<p>Run summary</p>") {
		t.Fatalf("expected run summary HTML to be unescaped, got %q", body)
	}
	if strings.Contains(body, "&lt;p&gt;Post summary&lt;/p&gt;") || !strings.Contains(body, "<p>Post summary</p>") {
		t.Fatalf("expected post summary HTML to be unescaped, got %q", body)
	}
	if !strings.Contains(body, "&lt;b&gt;Title&lt;/b&gt;") {
		t.Fatalf("expected other fields to remain escaped, got %q", body)
	}
}

