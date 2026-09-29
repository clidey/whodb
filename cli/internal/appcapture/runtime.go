/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package appcapture

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const playwrightVersion = "1.62.1"

// RuntimeDir returns the user-owned directory for WhoDB's optional browser runtime.
func RuntimeDir() (string, error) {
	if path := strings.TrimSpace(os.Getenv("WHODB_CAPTURE_RUNTIME_DIR")); path != "" {
		return filepath.Abs(path)
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("find user cache: %w", err)
	}
	return filepath.Join(cache, "whodb", "app-capture"), nil
}

// SetupRuntime installs the optional Playwright package and Chromium in the user cache.
// It is only called by an explicit CLI setup command or --install flag.
func SetupRuntime(ctx context.Context) (string, error) {
	if _, err := exec.LookPath("node"); err != nil {
		return "", errors.New("Node.js is required; install Node.js, then run whodb apps setup-capture")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		return "", errors.New("npm is required; install Node.js with npm, then run whodb apps setup-capture")
	}
	dir, err := RuntimeDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create capture runtime: %w", err)
	}
	packageFile := filepath.Join(dir, "package.json")
	if _, err := os.Stat(packageFile); errors.Is(err, os.ErrNotExist) {
		if err := os.WriteFile(packageFile, []byte("{\"private\":true,\"name\":\"whodb-app-capture-runtime\",\"version\":\"1.0.0\"}\n"), 0o644); err != nil {
			return "", fmt.Errorf("create capture package: %w", err)
		}
	} else if err != nil {
		return "", err
	}
	packageName := "@playwright/test@" + playwrightVersion
	install := exec.CommandContext(ctx, "npm", "install", "--prefix", dir, "--no-audit", "--no-fund", "--save-exact", packageName)
	if output, err := install.CombinedOutput(); err != nil {
		return "", fmt.Errorf("install %s: %w: %s", packageName, err, strings.TrimSpace(string(output)))
	}
	browserCLI := filepath.Join(dir, "node_modules", "playwright", "cli.js")
	browser := exec.CommandContext(ctx, "node", browserCLI, "install", "chromium")
	browser.Env = append(os.Environ(), "PLAYWRIGHT_BROWSERS_PATH="+filepath.Join(dir, "browsers"))
	if output, err := browser.CombinedOutput(); err != nil {
		return "", fmt.Errorf("install Chromium: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return packageFile, nil
}

func findPlaywrightPackage() (string, error) {
	if path := strings.TrimSpace(os.Getenv("WHODB_CAPTURE_PLAYWRIGHT_PACKAGE")); path != "" {
		if hasPlaywrightPackage(path) {
			return path, nil
		}
		return "", fmt.Errorf("@playwright/test is not installed beside %s; run whodb apps setup-capture", path)
	}
	if dir, err := RuntimeDir(); err == nil {
		path := filepath.Join(dir, "package.json")
		if hasPlaywrightPackage(path) {
			return path, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for _, path := range []string{
		filepath.Join(cwd, "frontend", "package.json"),
		filepath.Join(cwd, "..", "frontend", "package.json"),
		filepath.Join(cwd, "..", "whodb3", "frontend", "package.json"),
	} {
		if hasPlaywrightPackage(path) {
			return path, nil
		}
	}
	return "", errors.New("app capture needs Playwright and Chromium; run whodb apps setup-capture, or use whodb apps screenshot --install")
}

func hasPlaywrightPackage(packageFile string) bool {
	if _, err := os.Stat(packageFile); err != nil {
		return false
	}
	packageDir := filepath.Dir(packageFile)
	if _, err := os.Stat(filepath.Join(packageDir, "node_modules", "@playwright", "test", "package.json")); err == nil {
		return true
	}
	return false
}

func managedBrowserPath(packageFile string) string {
	dir, err := RuntimeDir()
	if err != nil || filepath.Clean(filepath.Dir(packageFile)) != filepath.Clean(dir) {
		return ""
	}
	return filepath.Join(dir, "browsers")
}
