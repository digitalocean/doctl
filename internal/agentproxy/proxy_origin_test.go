package agentproxy

import "testing"

func TestAllowedProxyOrigin(t *testing.T) {
	cases := []struct {
		origin string
		want   bool
	}{
		{"", true},
		{"http://localhost", true},
		{"http://localhost:3000", true},
		{"http://127.0.0.1", true},
		{"http://127.0.0.1:8080", true},
		{"http://[" + "::1" + "]", true},
		{"https://example.com", false},
		{"http://evil.example", false},
		{"://bad", false},
	}
	for _, tc := range cases {
		if got := allowedProxyOrigin(tc.origin); got != tc.want {
			t.Errorf("allowedProxyOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}
