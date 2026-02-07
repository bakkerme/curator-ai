package source

import (
	"testing"

	"github.com/bakkerme/curator-ai/internal/config"
	"github.com/bakkerme/curator-ai/internal/core"
)

func TestSummaryPlanFromConfig_DefaultsToFullWhenOmitted(t *testing.T) {
	plan := summaryPlanFromConfig(nil)
	if plan == nil {
		t.Fatalf("expected non-nil summary plan")
	}
	if plan.Mode != core.SummaryModeFull {
		t.Fatalf("expected mode=%q, got %q", core.SummaryModeFull, plan.Mode)
	}
}

func TestSummaryPlanFromConfig_DefaultsToFullWhenModeEmpty(t *testing.T) {
	plan := summaryPlanFromConfig(&config.SummaryPlanConfig{})
	if plan == nil {
		t.Fatalf("expected non-nil summary plan")
	}
	if plan.Mode != core.SummaryModeFull {
		t.Fatalf("expected mode=%q, got %q", core.SummaryModeFull, plan.Mode)
	}
}
