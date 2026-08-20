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

package analytics

import (
	"fmt"
	"testing"
)

type fakeCodedError struct{ code string }

func (e fakeCodedError) Error() string { return "not found: something" }
func (e fakeCodedError) Code() string  { return e.code }

func TestErrorCodePrefersCodedErrorOverStringMatch(t *testing.T) {
	// Message contains "not found" (which would otherwise match not_found),
	// but a wrapped CodedError's explicit code must win.
	err := fmt.Errorf("wrap: %w", fakeCodedError{code: "quota_exceeded"})
	if got := ErrorCode(err); got != "quota_exceeded" {
		t.Fatalf("ErrorCode = %q, want quota_exceeded", got)
	}
}

func TestErrorCodeIgnoresCodedErrorWithEmptyCode(t *testing.T) {
	err := fakeCodedError{code: ""}
	if got := ErrorCode(err); got != "not_found" {
		t.Fatalf("ErrorCode = %q, want not_found (fallback to string match)", got)
	}
}
