/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 */

package mcp

import (
	"errors"
	"testing"
)

func TestPlatformErrorFieldsDoesNotMisreadGenerateAsRateLimited(t *testing.T) {
	code, retryable, _ := platformErrorFields(errors.New("this WhoDB host does not support platform write GenerateApp yet"))
	if code != string(PlatformErrorValidation) {
		t.Fatalf("expected %q, got %q", PlatformErrorValidation, code)
	}
	if retryable {
		t.Fatal("expected not retryable for an unsupported operation")
	}
}

func TestPlatformErrorFieldsDetectsRealRateLimit(t *testing.T) {
	code, retryable, _ := platformErrorFields(errors.New("429: rate limit exceeded, please slow down"))
	if code != string(PlatformErrorRateLimited) {
		t.Fatalf("expected %q, got %q", PlatformErrorRateLimited, code)
	}
	if !retryable {
		t.Fatal("expected retryable for a real rate limit")
	}
}

func TestPlatformErrorFieldsDetectsUnsupportedOperation(t *testing.T) {
	code, _, _ := platformErrorFields(errors.New("operation unsupported on this host"))
	if code != string(PlatformErrorValidation) {
		t.Fatalf("expected %q, got %q", PlatformErrorValidation, code)
	}
}
