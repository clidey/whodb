package security

import "testing"

func TestValidateOutboundURL_BlocksInternal(t *testing.T) {
	blocked := []string{
		"http://169.254.169.254/latest/meta-data/", // cloud metadata
		"http://metadata.google.internal/",         // GCP metadata (resolves link-local)
		"http://127.0.0.1:8080/",                   // loopback
		"http://localhost/admin",                   // loopback by name
		"http://10.0.0.5/",                         // RFC1918
		"http://192.168.1.1/",                      // RFC1918
		"http://[::1]/",                            // IPv6 loopback
		"ftp://example.com/",                       // disallowed scheme
		"file:///etc/passwd",                       // disallowed scheme
		"http://0.0.0.0/",                          // unspecified
	}
	for _, u := range blocked {
		if err := ValidateOutboundURL(u); err == nil {
			t.Errorf("expected %q to be blocked, but it was allowed", u)
		}
	}
}

func TestValidateOutboundURL_AllowsPublic(t *testing.T) {
	allowed := []string{
		"https://api.openai.com/v1",
		"https://api.anthropic.com/v1",
		"https://8.8.8.8/",
	}
	for _, u := range allowed {
		if err := ValidateOutboundURL(u); err != nil {
			t.Errorf("expected %q to be allowed, got %v", u, err)
		}
	}
}
