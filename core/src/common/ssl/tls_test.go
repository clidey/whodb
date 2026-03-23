/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ssl

import (
	"crypto/tls"
	"testing"
)

// Test helper functions - moved from tls.go as they're only used in tests

// buildInsecureTLSConfig creates a simple TLS config with verification disabled.
// This is a test helper, not exported from the package.
func buildInsecureTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}
}

// Sample self-signed CA certificate for testing (not for production use)
const testCACertPEM = `-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHBfpvJHIMdMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBExDzANBgNVBAMM
BnRlc3RjYTBcMA0GCSqGSIb3DQEBAQUAA0sAMEgCQQC7o96WdWGkfJFq8bBM+Ufk
PiP8XMmB4etXjHv0NBzfnOuLJdOLHl5dj8JHKlrFqEYbzLApQZWaPKDqxJd/NHXP
AgMBAAGjUzBRMB0GA1UdDgQWBBQExample0123456789012345678901MB8GA1Ud
IwQYMBaAFBQExample0123456789012345678901MA8GA1UdEwEB/wQFMAMBAf8w
DQYJKoZIhvcNAQELBQADQQBExample
-----END CERTIFICATE-----`

// Test client certificate (not for production use)
const testClientCertPEM = `-----BEGIN CERTIFICATE-----
MIIBjTCB9wIJAKHBfpvJHIMeMA0GCSqGSIb3DQEBCwUAMBExDzANBgNVBAMMBnRl
c3RjYTAeFw0yNDAxMDEwMDAwMDBaFw0yNTAxMDEwMDAwMDBaMBMxETAPBgNVBAMM
CHRlc3R1c2VyMFwwDQYJKoZIhvcNAQEBBQADSwAwSAJBALuj3pZ1YaR8kWrxsEz5
R+Q+I/xcyYHh61eMe/Q0HN+c64sl04seXl2PwkcqWsWoRhvMsClBlZo8oOrEl380
dc8CAwEAAaNTMFEwHQYDVR0OBBYEFBQExample0123456789012345678901MB8G
A1UdIwQYMBaAFBQExample0123456789012345678901MA8GA1UdEwEB/wQFMAMB
Af8wDQYJKoZIhvcNAQELBQADQQBExample
-----END CERTIFICATE-----`

// Test client key (not for production use)
const testClientKeyPEM = `-----BEGIN RSA PRIVATE KEY-----
MIIBOgIBAAJBALuj3pZ1YaR8kWrxsEz5R+Q+I/xcyYHh61eMe/Q0HN+c64sl04se
Xl2PwkcqWsWoRhvMsClBlZo8oOrEl380dc8CAwEAAQJAExample01234567890123
456789012345678901234567890123456789012345678901234567890123456789
0123456789012345678901234567890IYwJiIYwJiIYwJiIYwJiIYwIhAExample01
234567890123456789012345678901AiEAExample01234567890123456789012345
678901ECIBExample01234567890123456789012345678901AiEAExample01234567
890123456789012345678901ECIBExample012345678901234567890123456789
-----END RSA PRIVATE KEY-----`

func TestCertificateInputIsEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input *CertificateInput
		want  bool
	}{
		{"nil input", nil, true},
		{"empty struct", &CertificateInput{}, true},
		{"empty string", &CertificateInput{Content: ""}, true},
		{"has content", &CertificateInput{Content: "cert content"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.IsEmpty()
			if got != tt.want {
				t.Errorf("CertificateInput.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCertificateInputLoad(t *testing.T) {
	tests := []struct {
		name    string
		input   *CertificateInput
		wantNil bool
		wantErr bool
	}{
		{"nil input", nil, true, false},
		{"empty input", &CertificateInput{}, true, false},
		{"loads from content", &CertificateInput{Content: testCACertPEM}, false, false},
		{"loads plain text content", &CertificateInput{Content: "test content"}, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.input.Load()

			if tt.wantErr {
				if err == nil {
					t.Errorf("CertificateInput.Load() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("CertificateInput.Load() unexpected error: %v", err)
				return
			}

			if tt.wantNil && data != nil {
				t.Errorf("CertificateInput.Load() = %v, want nil", data)
			}
			if !tt.wantNil && data == nil {
				t.Errorf("CertificateInput.Load() = nil, want non-nil")
			}
		})
	}
}

func TestCertificateInputLoadReturnsExactContent(t *testing.T) {
	directContent := "direct content"

	input := &CertificateInput{
		Content: directContent,
	}

	data, err := input.Load()
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if string(data) != directContent {
		t.Errorf("Load() = %s, want %s", string(data), directContent)
	}
}

func TestBuildTLSConfigNilDisabled(t *testing.T) {
	tests := []struct {
		name   string
		config *SSLConfig
	}{
		{"nil config", nil},
		{"disabled mode", &SSLConfig{Mode: SSLModeDisabled}},
		{"empty mode", &SSLConfig{Mode: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := BuildTLSConfig(tt.config, "localhost")
			if err != nil {
				t.Errorf("BuildTLSConfig() error = %v", err)
			}
			if tlsConfig != nil {
				t.Errorf("BuildTLSConfig() = %v, want nil", tlsConfig)
			}
		})
	}
}

func TestBuildTLSConfigInsecureModes(t *testing.T) {
	modes := []SSLMode{SSLModeInsecure, SSLModePreferred, SSLModeRequired}

	for _, mode := range modes {
		t.Run(string(mode), func(t *testing.T) {
			config := &SSLConfig{Mode: mode}
			tlsConfig, err := BuildTLSConfig(config, "localhost")

			if err != nil {
				t.Errorf("BuildTLSConfig() error = %v", err)
				return
			}
			if tlsConfig == nil {
				t.Errorf("BuildTLSConfig() = nil, want non-nil")
				return
			}
			if !tlsConfig.InsecureSkipVerify {
				t.Errorf("TLSConfig.InsecureSkipVerify = false, want true for mode %s", mode)
			}
		})
	}
}

func TestBuildTLSConfigEnabledMode(t *testing.T) {
	config := &SSLConfig{Mode: SSLModeEnabled}
	tlsConfig, err := BuildTLSConfig(config, "localhost")

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}
	if tlsConfig == nil {
		t.Fatalf("BuildTLSConfig() = nil, want non-nil")
	}
	if tlsConfig.InsecureSkipVerify {
		t.Errorf("TLSConfig.InsecureSkipVerify = true, want false for enabled mode")
	}
}

func TestBuildTLSConfigVerifyIdentityServerName(t *testing.T) {
	tests := []struct {
		name           string
		configServer   string
		hostname       string
		wantServerName string
	}{
		{"uses config server name", "config.example.com", "host.example.com", "config.example.com"},
		{"uses hostname when config empty", "", "host.example.com", "host.example.com"},
		{"empty both", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SSLConfig{
				Mode:       SSLModeVerifyIdentity,
				ServerName: tt.configServer,
			}
			tlsConfig, err := BuildTLSConfig(config, tt.hostname)

			if err != nil {
				t.Fatalf("BuildTLSConfig() error = %v", err)
			}
			if tlsConfig == nil {
				t.Fatalf("BuildTLSConfig() = nil, want non-nil")
			}
			if tlsConfig.ServerName != tt.wantServerName {
				t.Errorf("TLSConfig.ServerName = %s, want %s", tlsConfig.ServerName, tt.wantServerName)
			}
		})
	}
}

func TestBuildTLSConfigVerifyCANoServerName(t *testing.T) {
	config := &SSLConfig{Mode: SSLModeVerifyCA}
	tlsConfig, err := BuildTLSConfig(config, "localhost")

	if err != nil {
		t.Fatalf("BuildTLSConfig() error = %v", err)
	}
	if tlsConfig == nil {
		t.Fatalf("BuildTLSConfig() = nil, want non-nil")
	}
	// verify-ca should not set ServerName (that's only for verify-identity)
	if tlsConfig.ServerName != "" {
		t.Errorf("TLSConfig.ServerName = %s, want empty for verify-ca mode", tlsConfig.ServerName)
	}
}

func TestBuildInsecureTLSConfig(t *testing.T) {
	tlsConfig := buildInsecureTLSConfig()

	if tlsConfig == nil {
		t.Fatalf("buildInsecureTLSConfig() = nil, want non-nil")
	}
	if !tlsConfig.InsecureSkipVerify {
		t.Errorf("TLSConfig.InsecureSkipVerify = false, want true")
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("TLSConfig.MinVersion = %d, want %d", tlsConfig.MinVersion, tls.VersionTLS12)
	}
}

func TestBuildTLSConfigWithCACertContent(t *testing.T) {
	// Note: We can't test with a real valid CA cert easily without generating one,
	// but we can test that invalid PEM is rejected
	config := &SSLConfig{
		Mode: SSLModeVerifyCA,
		CACert: CertificateInput{
			Content: "not a valid PEM certificate",
		},
	}

	_, err := BuildTLSConfig(config, "localhost")
	if err == nil {
		t.Errorf("BuildTLSConfig() with invalid CA cert should return error")
	}
}

// Note: Path-based loading is no longer supported to prevent path traversal attacks.
// Frontend reads files client-side and sends Content directly.

func TestBuildTLSConfigClientCertsRequireBoth(t *testing.T) {
	tests := []struct {
		name       string
		clientCert CertificateInput
		clientKey  CertificateInput
		wantCerts  bool
	}{
		{"neither provided", CertificateInput{}, CertificateInput{}, false},
		{"only cert provided", CertificateInput{Content: "cert"}, CertificateInput{}, false},
		{"only key provided", CertificateInput{}, CertificateInput{Content: "key"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &SSLConfig{
				Mode:       SSLModeEnabled,
				ClientCert: tt.clientCert,
				ClientKey:  tt.clientKey,
			}
			tlsConfig, err := BuildTLSConfig(config, "localhost")

			if err != nil {
				t.Fatalf("BuildTLSConfig() error = %v", err)
			}
			if tlsConfig == nil {
				t.Fatalf("BuildTLSConfig() = nil, want non-nil")
			}

			hasCerts := len(tlsConfig.Certificates) > 0
			if hasCerts != tt.wantCerts {
				t.Errorf("TLSConfig has certificates = %v, want %v", hasCerts, tt.wantCerts)
			}
		})
	}
}
