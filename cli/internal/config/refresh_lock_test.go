package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func TestRefreshLockCoordinatesAccountsAndCancellation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	configDirOnce = sync.Once{}
	t.Cleanup(func() { configDirOnce = sync.Once{} })
	cfg := DefaultConfig()
	entered, release, done := make(chan struct{}), make(chan struct{}), make(chan error, 1)
	go func() {
		done <- cfg.WithPlatformRefreshLock(context.Background(), "host", "account", func() error { close(entered); <-release; return nil })
	}()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := cfg.WithPlatformRefreshLock(ctx, "host", "other-account", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	err := cfg.WithPlatformRefreshLock(ctx, "host", "account", func() error { t.Error("overlapping refresh"); return nil })
	close(release)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if err := cfg.WithPlatformRefreshLock(context.Background(), "host", "account", func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshLockAcrossProcesses(t *testing.T) {
	if os.Getenv("WHODB_TEST_REFRESH_LOCK_CHILD") == "1" {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()
		err := DefaultConfig().WithPlatformRefreshLock(ctx, "process-host", "process-account", func() error { return nil })
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("child acquired parent lock: %v", err)
		}
		return
	}
	t.Setenv("HOME", t.TempDir())
	configDirOnce = sync.Once{}
	t.Cleanup(func() { configDirOnce = sync.Once{} })
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	err = DefaultConfig().WithPlatformRefreshLock(context.Background(), "process-host", "process-account", func() error {
		child := exec.Command(executable, "-test.run=^TestRefreshLockAcrossProcesses$")
		child.Env = append(os.Environ(), "WHODB_TEST_REFRESH_LOCK_CHILD=1")
		output, err := child.CombinedOutput()
		if err != nil {
			return fmt.Errorf("child: %w: %s", err, output)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
