/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 */

package mcp

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSetupToolModulesAndFilters(t *testing.T) {
	for _, tc := range []struct {
		name               string
		platform, database bool
		enabled, disabled  []string
		want               bool
	}{
		{name: "platform", platform: true, want: true},
		{name: "database", database: true, want: true},
		{name: "both", platform: true, database: true, want: true},
		{name: "explicit", platform: true, enabled: []string{"whodb_mcp_setup"}, want: true},
		{name: "disabled", platform: true, disabled: []string{"whodb_mcp_setup"}},
		{name: "excluded", platform: true, enabled: []string{"whodb_platform_hosts"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := scopeTestClient(t, &ServerOptions{PlatformEnabled: tc.platform, DatabaseEnabled: tc.database, EnabledTools: tc.enabled, DisabledTools: tc.disabled, SetupHandler: func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
				return json.RawMessage(`{}`), nil
			}})
			result, err := client.ListTools(context.Background(), nil)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, tool := range result.Tools {
				found = found || tool.Name == "whodb_mcp_setup"
			}
			if found != tc.want {
				t.Fatalf("setup tool present=%v, want %v", found, tc.want)
			}
		})
	}
}
