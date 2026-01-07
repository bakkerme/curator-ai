package smtp

import "testing"

func TestIsLocalDevSMTPHost(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"mailpit", true},
		{"smtp.example.com", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := isLocalDevSMTPHost(tc.host); got != tc.want {
			t.Fatalf("isLocalDevSMTPHost(%q)=%v want %v", tc.host, got, tc.want)
		}
	}
}

