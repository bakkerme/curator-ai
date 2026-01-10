package config

import (
	"testing"
	"time"
)

func TestParseDurationExtended_DaysWeeksAndFallback(t *testing.T) {
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"1w", 7 * 24 * time.Hour},
		{"1w2d3h", (7*24 + 2*24 + 3) * time.Hour},
		{"1.5d", 36 * time.Hour},
		{"-2w", -14 * 24 * time.Hour},
		{"90m", 90 * time.Minute}, // Go fallback
	}

	for _, tc := range cases {
		got, err := parseDurationExtended(tc.in)
		if err != nil {
			t.Fatalf("parseDurationExtended(%q) unexpected error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseDurationExtended(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestParseDurationExtended_Invalid(t *testing.T) {
	bad := []string{"", "   ", "3x", "2d3x", "-"}
	for _, in := range bad {
		if _, err := parseDurationExtended(in); err == nil {
			t.Fatalf("parseDurationExtended(%q) expected error, got nil", in)
		}
	}
}
