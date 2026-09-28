package security

import (
	"testing"

	"github.com/clidey/whodb/core/src/env"
)

func TestValidateOutboundURL_BlocksInternal(t *testing.T) {
	blocked := []string{
		"http://169.254.169.254/latest/meta-data/",   // cloud metadata
		"http://metadata.google.internal/",           // GCP metadata (resolves link-local)
		"http://127.0.0.1:8080/",                     // loopback
		"http://localhost/admin",                     // loopback by name
		"http://10.0.0.5/",                           // RFC1918
		"http://192.168.1.1/",                        // RFC1918
		"http://[::1]/",                              // IPv6 loopback
		"http://[64:ff9b::a9fe:a9fe]/",               // NAT64 well-known prefix
		"http://[64:ff9b::7f00:1]/",                  // NAT64-embedded loopback
		"http://[64:ff9b:1::a9fe:a9fe]/",             // NAT64 local-use prefix
		"http://[2002:a9fe:a9fe::1]/",                // 6to4
		"http://[2002:7f00:1::]/",                    // 6to4-embedded loopback
		"http://[2001:0:b00:a:2442:2c40:f4ff:ffe1]/", // Teredo
		"ftp://example.com/",                         // disallowed scheme
		"file:///etc/passwd",                         // disallowed scheme
		"http://0.0.0.0/",                            // unspecified
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
		"https://[2606:4700:4700::1111]/",
	}
	for _, u := range allowed {
		if err := ValidateOutboundURL(u); err != nil {
			t.Errorf("expected %q to be allowed, got %v", u, err)
		}
	}
}

func TestValidateOutboundURL_BlocksConfiguredCIDR(t *testing.T) {
	previous := env.AIEndpointBlockedCIDRs
	env.AIEndpointBlockedCIDRs = "2001:4860:64::/96"
	t.Cleanup(func() {
		env.AIEndpointBlockedCIDRs = previous
	})

	if err := ValidateOutboundURL("http://[2001:4860:64::a4d:2]/"); err == nil {
		t.Fatal("expected deployment-specific NAT64 prefix to be blocked")
	}
}

func TestValidateOutboundURL_FailsClosedForInvalidConfiguredCIDR(t *testing.T) {
	previous := env.AIEndpointBlockedCIDRs
	env.AIEndpointBlockedCIDRs = "not-a-prefix"
	t.Cleanup(func() {
		env.AIEndpointBlockedCIDRs = previous
	})

	if err := ValidateOutboundURL("https://8.8.8.8/"); err == nil {
		t.Fatal("expected invalid blocked CIDR configuration to reject the request")
	}
}
