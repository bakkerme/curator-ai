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

func TestParseTLSMode(t *testing.T) {
	cases := []struct {
		mode    string
		want    TLSMode
		wantErr bool
	}{
		{"", TLSModeAuto, false},
		{"auto", TLSModeAuto, false},
		{"disabled", TLSModeDisabled, false},
		{"off", TLSModeDisabled, false},
		{"starttls", TLSModeStartTLS, false},
		{"start_tls", TLSModeStartTLS, false},
		{"implicit", TLSModeImplicit, false},
		{"smtptls", TLSModeImplicit, false},
		{"smtp_tls", TLSModeImplicit, false},
		{"unknown", "", true},
	}

	for _, tc := range cases {
		got, err := parseTLSMode(tc.mode)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("parseTLSMode(%q) expected error", tc.mode)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parseTLSMode(%q) unexpected error: %v", tc.mode, err)
		}
		if got != tc.want {
			t.Fatalf("parseTLSMode(%q)=%v want %v", tc.mode, got, tc.want)
		}
	}
}

func TestResolveTLSMode(t *testing.T) {
	cases := []struct {
		name string
		port int
		mode string
		want TLSMode
	}{
		{"auto-implicit", 465, "", TLSModeImplicit},
		{"auto-starttls", 587, "", TLSModeStartTLS},
		{"explicit-starttls", 465, "starttls", TLSModeStartTLS},
		{"explicit-implicit", 587, "implicit", TLSModeImplicit},
		{"explicit-disabled", 25, "disabled", TLSModeDisabled},
	}

	for _, tc := range cases {
		sender := &Sender{
			port:    tc.port,
			tlsMode: tc.mode,
		}
		got, err := sender.resolveTLSMode()
		if err != nil {
			t.Fatalf("resolveTLSMode(%s) unexpected error: %v", tc.name, err)
		}
		if got != tc.want {
			t.Fatalf("resolveTLSMode(%s)=%v want %v", tc.name, got, tc.want)
		}
	}
}
