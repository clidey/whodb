package datadir

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGetCreatesDirectoryWithExpectedBase(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	// Prefer an explicit base directory when supported by the platform.
	// - Linux/other: uses XDG_DATA_HOME if set.
	// - Windows: uses APPDATA if set.
	// - macOS: always uses ~/Library/Application Support (so HOME override is used).
	tempBase := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tempBase)
	t.Setenv("APPDATA", tempBase)

	opts := Options{AppName: "whodb-test"}
	dir, err := Get(opts)
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatalf("expected %q to be a directory", dir)
	}

	switch runtime.GOOS {
	case "darwin":
		expected := filepath.Join(tempHome, "Library", "Application Support", "whodb-test")
		if dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	case "windows":
		expected := filepath.Join(tempBase, "whodb-test")
		if dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	default:
		expected := filepath.Join(tempBase, "whodb-test")
		if dir != expected {
			t.Fatalf("expected %q, got %q", expected, dir)
		}
	}
}

func TestGetAppliesSuffixes(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("USERPROFILE", tempHome)

	dir, err := Get(Options{
		AppName:           "whodb-test",
		EnterpriseEdition: true,
		Development:       true,
	})
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}

	if !strings.HasSuffix(dir, "whodb-test-ee-dev") {
		t.Fatalf("expected suffix whodb-test-ee-dev, got %q", dir)
	}
}
