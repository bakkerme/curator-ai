package output

import (
	"errors"
	"html/template"
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

func TestRenderEmailTemplate_RendersMarkdownViaToHTML(t *testing.T) {
	body, err := renderEmailTemplate(
		`{{toHTML .RunSummary.Summary}}{{range .Blocks}}{{.Title}}|{{toHTML .Summary.Summary}}{{end}}`,
		[]*core.PostBlock{
			{
				Title: `<b>Title</b>`,
				Summary: &core.SummaryResult{
					Summary: "* Post summary",
				},
			},
		},
		&core.RunSummary{
			Summary: "# Run summary",
		},
	)
	if err != nil {
		t.Fatalf("renderEmailTemplate failed: %v", err)
	}

	if !strings.Contains(body, "<h1") || !strings.Contains(body, "Run summary</h1>") {
		t.Fatalf("expected rendered run summary heading, got %q", body)
	}
	if !strings.Contains(body, "<ul>") || !strings.Contains(body, "<li>Post summary</li>") {
		t.Fatalf("expected rendered post summary list, got %q", body)
	}
	if !strings.Contains(body, "&lt;b&gt;Title&lt;/b&gt;") {
		t.Fatalf("expected non-helper fields to remain escaped, got %q", body)
	}
}

func TestRenderEmailTemplate_ToHTMLSupportsEmptyInput(t *testing.T) {
	body, err := renderEmailTemplate(`before{{toHTML .RunSummary.Summary}}after`, nil, &core.RunSummary{})
	if err != nil {
		t.Fatalf("renderEmailTemplate failed: %v", err)
	}

	if body != "beforeafter" {
		t.Fatalf("expected empty markdown to render empty output, got %q", body)
	}
}

func TestParseEmailTemplate_RejectsUnknownFunctions(t *testing.T) {
	if _, err := parseEmailTemplate(`{{unknown .RunSummary.Summary}}`); err == nil {
		t.Fatal("expected parseEmailTemplate to reject unknown functions")
	}
}

func TestExecuteEmailTemplate_ToHTMLPropagatesErrors(t *testing.T) {
	tmpl := template.Must(template.New("email").Funcs(template.FuncMap{
		"toHTML": func(string) (template.HTML, error) {
			return "", errors.New("boom")
		},
	}).Parse(`{{toHTML .RunSummary.Summary}}`))

	if _, err := executeEmailTemplate(tmpl, nil, &core.RunSummary{Summary: "x"}); err == nil {
		t.Fatal("expected executeEmailTemplate to return template function errors")
	}
}
