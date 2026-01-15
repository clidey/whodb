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

func TestMapElastiCacheStatus(t *testing.T) {
	testCases := []struct {
		status   string
		expected providers.ConnectionStatus
	}{
		{"available", providers.ConnectionStatusAvailable},
		{"Available", providers.ConnectionStatusAvailable},
		{"AVAILABLE", providers.ConnectionStatusAvailable},
		{"creating", providers.ConnectionStatusStarting},
		{"modifying", providers.ConnectionStatusStarting},
		{"snapshotting", providers.ConnectionStatusStarting},
		{"rebooting cluster nodes", providers.ConnectionStatusStarting},
		{"deleted", providers.ConnectionStatusDeleting},
		{"deleting", providers.ConnectionStatusDeleting},
		{"create-failed", providers.ConnectionStatusFailed},
		{"restore-failed", providers.ConnectionStatusFailed},
		{"unknown-status", providers.ConnectionStatusUnknown},
		{"", providers.ConnectionStatusUnknown},
	}

	for _, tc := range testCases {
		result := mapElastiCacheStatus(tc.status)
		if result != tc.expected {
			t.Errorf("mapElastiCacheStatus(%s): expected %s, got %s", tc.status, tc.expected, result)
		}
	}
}

func TestMapElastiCacheStatus_CaseInsensitive(t *testing.T) {
	statuses := []string{"available", "Available", "AVAILABLE", "AvAiLaBlE"}
	for _, s := range statuses {
		result := mapElastiCacheStatus(s)
		if result != providers.ConnectionStatusAvailable {
			t.Errorf("mapElastiCacheStatus(%s): expected Available, got %s", s, result)
		}
	}
}
