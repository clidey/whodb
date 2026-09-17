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

package platform

import "testing"

func TestSessionScopeFromEnvironment(t *testing.T) {
	t.Setenv(SessionHostEnv, " http://localhost:4000 ")
	t.Setenv(SessionOrgEnv, " acme ")
	t.Setenv(SessionProjectEnv, " analysis ")

	scope := SessionScopeFromEnvironment()
	if scope.Host != "http://localhost:4000" || scope.Org != "acme" || scope.Project != "analysis" {
		t.Fatalf("SessionScopeFromEnvironment() = %#v", scope)
	}
	if !scope.HasWorkspace() {
		t.Fatal("expected complete workspace override")
	}
	if err := scope.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSessionScopeRejectsPartialWorkspace(t *testing.T) {
	for _, scope := range []SessionScope{{Org: "acme"}, {Project: "analysis"}} {
		if err := scope.Validate(); err == nil {
			t.Fatalf("Validate(%#v) succeeded, want error", scope)
		}
	}
}
