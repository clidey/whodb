package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAuthCmd_Registered(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	var auth bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "auth" {
			auth = true
			var printToken bool
			for _, sub := range cmd.Commands() {
				if sub.Use == "print-token" {
					printToken = true
				}
			}
			if !printToken {
				t.Error("Expected 'print-token' subcommand under 'auth'")
			}
		}
	}
	if !auth {
		t.Error("Expected 'auth' subcommand")
	}
}

func TestPrintToken_NotLoggedIn(t *testing.T) {
	cleanup := setupTestEnv(t)
	defer cleanup()

	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"auth", "print-token"})
	defer func() {
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		rootCmd.SetArgs(nil)
	}()

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error when not logged in")
	}
	if !strings.Contains(err.Error(), "whodb login") {
		t.Errorf("error should instruct to run whodb login, got: %v", err)
	}
}

func TestJWTExpiry(t *testing.T) {
	exp := time.Now().Add(5 * time.Minute).Unix()
	payload, _ := json.Marshal(map[string]int64{"exp": exp})
	token := "eyJhbGciOiJSUzI1NiJ9." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"

	got := jwtExpiry(token)
	if got.Unix() != exp {
		t.Errorf("expected exp %d, got %d", exp, got.Unix())
	}

	if !jwtExpiry("not-a-jwt").IsZero() {
		t.Error("malformed token should yield zero time")
	}
	if !jwtExpiry("a.!!!.c").IsZero() {
		t.Error("bad base64 payload should yield zero time")
	}
}
