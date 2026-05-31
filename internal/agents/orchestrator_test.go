package agents

import (
	"context"
	"testing"

	"ecomteam/internal/llm"
)

func TestOrchestratorRunsAllStages(t *testing.T) {
	o := NewOrchestrator(llm.NewMock())

	seen := map[string]int{} // agent -> max percent observed
	progress := func(agent string, percent int, _ string) {
		if percent > seen[agent] {
			seen[agent] = percent
		}
	}

	d, err := o.Run(context.Background(), "ครีมกันแดด", "th", progress)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every stage must have reported 100%.
	for _, name := range o.Stages() {
		if seen[name] != 100 {
			t.Fatalf("agent %q did not reach 100%% (got %d)", name, seen[name])
		}
	}

	// Listing fields populated by each stage.
	l := d.Listing
	if len(l.SellingPoints) == 0 {
		t.Error("expected selling points from BENEFIT")
	}
	if l.Headline == "" {
		t.Error("expected headline from PROMO")
	}
	if l.Layout == "" || l.ColorTone == "" {
		t.Error("expected layout/color from DESIGN")
	}
	if l.ImagePrompt == "" {
		t.Error("expected image prompt from PROMPT")
	}
	if len(d.ImagePNG) == 0 {
		t.Error("expected image bytes from STUDIO")
	}
	if l.QCStatus != "passed" {
		t.Errorf("expected QC passed, got %q", l.QCStatus)
	}
}

func TestOrchestratorStagesOrder(t *testing.T) {
	o := NewOrchestrator(llm.NewMock())
	want := []string{NameBenefit, NamePromo, NameDesign, NamePrompt, NameStudio, NameQC}
	got := o.Stages()
	if len(got) != len(want) {
		t.Fatalf("expected %d stages, got %d", len(want), len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stage %d: want %q got %q", i, want[i], got[i])
		}
	}
}
