package whodb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakeCli writes an executable shell script standing in for the whodb CLI.
// It appends one line to countFile per invocation so tests can assert on
// exec counts (cache behavior).
func fakeCli(t *testing.T, body string) (command, countFile string) {
	t.Helper()
	dir := t.TempDir()
	countFile = filepath.Join(dir, "count")
	command = filepath.Join(dir, "whodb")
	script := fmt.Sprintf("#!/bin/sh\necho x >> %q\n%s\n", countFile, body)
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return command, countFile
}

func execCount(t *testing.T, countFile string) int {
	t.Helper()
	data, err := os.ReadFile(countFile)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return len(data) / 2 // "x\n" per invocation
}

func TestCliCredentialsMissingBinary(t *testing.T) {
	credentials := &cliCredentials{command: filepath.Join(t.TempDir(), "no-such-binary")}
	_, err := credentials.Token(context.Background())
	if !errors.Is(err, ErrCliCredentials) {
		t.Errorf("missing binary must map to ErrCliCredentials, got %v", err)
	}
}

func TestCliCredentialsInvalidJSON(t *testing.T) {
	command, _ := fakeCli(t, `echo "not json"`)
	credentials := &cliCredentials{command: command}
	_, err := credentials.Token(context.Background())
	if !errors.Is(err, ErrCliCredentials) {
		t.Errorf("invalid JSON must map to ErrCliCredentials, got %v", err)
	}
}

func TestCliCredentialsNonZeroExit(t *testing.T) {
	command, _ := fakeCli(t, `echo "run: whodb login" >&2; exit 1`)
	credentials := &cliCredentials{command: command}
	_, err := credentials.Token(context.Background())
	if !errors.Is(err, ErrCliCredentials) {
		t.Errorf("CLI failure must map to ErrCliCredentials, got %v", err)
	}
}

func TestCliCredentialsCachesUntilNearExpiry(t *testing.T) {
	expiry := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	command, countFile := fakeCli(t, fmt.Sprintf(
		`echo '{"access_token":"tok-1","expires_at":"%s"}'`, expiry))
	credentials := &cliCredentials{command: command}
	for range 3 {
		token, err := credentials.Token(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if token != "tok-1" {
			t.Fatalf("token: got %q", token)
		}
	}
	if got := execCount(t, countFile); got != 1 {
		t.Errorf("a fresh token must be served from cache: %d execs", got)
	}
}

func TestCliCredentialsReExecsNearExpiry(t *testing.T) {
	expiry := time.Now().Add(30 * time.Second).Format(time.RFC3339) // inside the 60s skew
	command, countFile := fakeCli(t, fmt.Sprintf(
		`echo '{"access_token":"tok-1","expires_at":"%s"}'`, expiry))
	credentials := &cliCredentials{command: command}
	for range 2 {
		if _, err := credentials.Token(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if got := execCount(t, countFile); got != 2 {
		t.Errorf("a near-expiry token must re-exec every call: %d execs", got)
	}
}

func TestCliCredentialsRefreshDropsCache(t *testing.T) {
	expiry := time.Now().Add(1 * time.Hour).Format(time.RFC3339)
	command, countFile := fakeCli(t, fmt.Sprintf(
		`echo '{"access_token":"tok-1","expires_at":"%s"}'`, expiry))
	credentials := &cliCredentials{command: command}
	if _, err := credentials.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	credentials.Refresh()
	if _, err := credentials.Token(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := execCount(t, countFile); got != 2 {
		t.Errorf("Refresh must force a re-exec: %d execs", got)
	}
}
