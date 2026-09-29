package mongodb

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/clidey/whodb/core/src/engine"
	_ "github.com/clidey/whodb/core/src/sources/database"
)

func TestValidateURIOptionsAllowsSupportedOptions(t *testing.T) {
	for option := range allowedMongoURIOptions {
		t.Run(option, func(t *testing.T) {
			if err := validateURIOptions("mongodb://localhost/?" + option + "=value"); err != nil {
				t.Fatalf("expected supported option, got %v", err)
			}
		})
	}

	for _, uri := range []string{
		"mongodb://localhost/?AUTHSOURCE=admin",
		"mongodb://localhost/?%61uthSource=admin",
		"mongodb://localhost/?tls=true;retryWrites=false",
	} {
		if err := validateURIOptions(uri); err != nil {
			t.Fatalf("expected normalized supported options, got %v", err)
		}
	}
}

func TestDBRejectsUnsupportedURIOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-certificate.pem")
	if err := os.WriteFile(path, []byte("file must not be read"), 0600); err != nil {
		t.Fatal(err)
	}
	file := url.QueryEscape(path)
	for _, params := range []string{
		"?tlsCAFile=" + file,
		"?sslCertificateAuthorityFile=" + file,
		"?tlsCertificateKeyFile=" + file,
		"?sslClientCertificateKeyFile=" + file,
		"?tlsCertificateFile=" + file + "&tlsPrivateKeyFile=" + file,
		"?TLSCAFILE=" + file,
		"?%74lsCAFile=" + file,
		"?tls=true;tlsCAFile=" + file,
		"?tlsCAFile=" + file + "&tlsCAFile=" + file,
		"?futureDriverOption=value",
		"?bad%zz=value",
	} {
		t.Run(params, func(t *testing.T) {
			for _, profile := range []bool{false, true} {
				config := engine.NewPluginConfig(&engine.Credentials{
					Hostname: "127.0.0.1", IsProfile: profile,
					Advanced: []engine.Record{{Key: "URL Params", Value: params}},
				})
				_, err := DB(config)
				if err == nil || (!strings.Contains(err.Error(), "is not supported") && !strings.Contains(err.Error(), "invalid MongoDB URL parameter name")) {
					t.Fatalf("profile=%v: expected unsupported option rejection before driver processing, got %v", profile, err)
				}
			}
		})
	}
}

func TestDBRejectsFileOptionsInjectedThroughOtherFields(t *testing.T) {
	path := url.QueryEscape(filepath.Join(t.TempDir(), "missing.pem"))
	for name, credentials := range map[string]*engine.Credentials{
		"database":          {Hostname: "127.0.0.1", Database: "?tlsCAFile=" + path},
		"database fragment": {Hostname: "127.0.0.1", Database: "safe#?tlsCAFile=" + path},
		"hostname":          {Hostname: "127.0.0.1/?tlsCAFile=" + path + "&appName=ignored"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := DB(engine.NewPluginConfig(credentials))
			if err == nil || !strings.Contains(err.Error(), "is not supported") {
				t.Fatalf("expected rejection before driver file access, got %v", err)
			}
		})
	}
}

func TestDBRetainsProfileSSLPaths(t *testing.T) {
	path := filepath.Join(t.TempDir(), "not-a-certificate.pem")
	if err := os.WriteFile(path, []byte("invalid certificate"), 0600); err != nil {
		t.Fatal(err)
	}
	config := engine.NewPluginConfig(&engine.Credentials{
		Hostname: "127.0.0.1", IsProfile: true,
		Advanced: []engine.Record{
			{Key: "SSL Mode", Value: "enabled"},
			{Key: "SSL CA Path", Value: path},
		},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	config.Context = ctx
	_, err := DB(config)
	if err == nil || !strings.Contains(err.Error(), "certificate") {
		t.Fatalf("expected profile CA content to be validated, got %v", err)
	}
}
