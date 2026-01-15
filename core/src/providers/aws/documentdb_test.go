/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package aws

import (
	"testing"

	"github.com/clidey/whodb/core/src/providers"
)

func TestMapDocDBStatus(t *testing.T) {
	testCases := []struct {
		status   string
		expected providers.ConnectionStatus
	}{
		{"available", providers.ConnectionStatusAvailable},
		{"Available", providers.ConnectionStatusAvailable},
		{"AVAILABLE", providers.ConnectionStatusAvailable},
		{"creating", providers.ConnectionStatusStarting},
		{"modifying", providers.ConnectionStatusStarting},
		{"upgrading", providers.ConnectionStatusStarting},
		{"migrating", providers.ConnectionStatusStarting},
		{"preparing-data-migration", providers.ConnectionStatusStarting},
		{"stopped", providers.ConnectionStatusStopped},
		{"stopping", providers.ConnectionStatusStopped},
		{"starting", providers.ConnectionStatusStopped},
		{"deleting", providers.ConnectionStatusDeleting},
		{"failed", providers.ConnectionStatusFailed},
		{"inaccessible-encryption-credentials", providers.ConnectionStatusFailed},
		{"unknown-status", providers.ConnectionStatusUnknown},
		{"", providers.ConnectionStatusUnknown},
	}

	for _, tc := range testCases {
		status := tc.status
		result := mapDocDBStatus(&status)
		if result != tc.expected {
			t.Errorf("mapDocDBStatus(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}

	// Test nil status
	result := mapDocDBStatus(nil)
	if result != providers.ConnectionStatusUnknown {
		t.Errorf("mapDocDBStatus(nil): expected %s, got %s", providers.ConnectionStatusUnknown, result)
	}
}

func TestMapDocDBStatus_CaseInsensitive(t *testing.T) {
	statuses := []string{"available", "Available", "AVAILABLE", "AvAiLaBlE"}
	for _, s := range statuses {
		result := mapDocDBStatus(&s)
		if result != providers.ConnectionStatusAvailable {
			t.Errorf("mapDocDBStatus(%s): expected Available, got %s", s, result)
		}
	}
}
