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
