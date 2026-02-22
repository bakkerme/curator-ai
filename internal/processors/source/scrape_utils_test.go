package source

import (
	"testing"
	"time"
)

func TestParseExtendedDuration_ComplexFormats(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
	}{
		{in: "1w2d", want: (7*24 + 2*24) * time.Hour},
		{in: "1.5d", want: time.Duration(36) * time.Hour},
		{in: "72h", want: 72 * time.Hour},
	}

	for _, tc := range tests {
		got, err := parseExtendedDuration(tc.in)
		if err != nil {
			t.Fatalf("parseExtendedDuration(%q) error: %v", tc.in, err)
		}
		if got != tc.want {
			t.Fatalf("parseExtendedDuration(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
}
