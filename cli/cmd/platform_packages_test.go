/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import "testing"

func TestParsePackageBindings(t *testing.T) {
	bindings, err := parsePackageBindings([]string{" source:erp = source-123 ", "secret:api=secret-456"})
	if err != nil {
		t.Fatalf("parsePackageBindings() error = %v", err)
	}
	if len(bindings) != 2 || bindings[0]["key"] != "source:erp" || bindings[0]["targetId"] != "source-123" {
		t.Fatalf("parsePackageBindings() = %#v", bindings)
	}
}

func TestParsePackageBindingsRejectsMissingTarget(t *testing.T) {
	if _, err := parsePackageBindings([]string{"source:erp="}); err == nil {
		t.Fatal("parsePackageBindings() expected an error")
	}
}
