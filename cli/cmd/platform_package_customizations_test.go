/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 */

package cmd

import "testing"

func TestPackageCustomizationModeNormalization(t *testing.T) {
	if got := normalizePlatformPackageCustomizationMode(" Merge-JSON "); got != "merge_json" {
		t.Fatalf("normalized mode = %q", got)
	}
}
