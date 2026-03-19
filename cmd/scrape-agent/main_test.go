package main

import "testing"

func TestBuildPrompt(t *testing.T) {
	template := "site: %s"
	got := buildPrompt(template, "https://example.com")
	want := "site: https://example.com"
	if got != want {
		t.Fatalf("buildPrompt() = %q, want %q", got, want)
	}
}

func TestBuildCommandArgs(t *testing.T) {
	command := " codex   exec  --json "
	got := buildCommandArgs(command)
	want := []string{"codex", "exec", "--json"}
	if len(got) != len(want) {
		t.Fatalf("buildCommandArgs() length = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildCommandArgs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
