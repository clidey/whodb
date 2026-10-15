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

package audit

import "testing"

func TestClassifyDefaults(t *testing.T) {
	cases := map[string]Class{
		"http.request":               ClassTelemetry,
		"source.list_objects":        ClassTelemetry,
		"graphql.query.SourceRows":   ClassTelemetry,
		"graphql.subscription.Tick":  ClassTelemetry,
		"source.read_rows":           ClassAccess,
		"source.export_rows_ndjson":  ClassAccess,
		"graphql.mutation.Login":     ClassCompliance,
		"login.source":               ClassCompliance,
		"source.delete_row":          ClassCompliance,
		"something.never.registered": ClassCompliance,
		"":                           ClassCompliance,
	}
	for action, want := range cases {
		if got := Classify(action); got != want {
			t.Errorf("Classify(%q) = %q, want %q", action, got, want)
		}
	}
}

func TestRegisterClassificationPrecedence(t *testing.T) {
	t.Cleanup(func() {
		classMu.Lock()
		delete(classExact, "graphql.query.TestAccessField")
		delete(classPrefixes, "test.prefix.")
		delete(classPrefixes, "test.prefix.long.")
		classMu.Unlock()
	})

	RegisterClassification(
		map[string]Class{"graphql.query.TestAccessField": ClassAccess},
		map[string]Class{"test.prefix.": ClassTelemetry, "test.prefix.long.": ClassAccess},
	)

	cases := map[string]Class{
		"graphql.query.TestAccessField": ClassAccess,
		"graphql.query.OtherField":      ClassTelemetry,
		"test.prefix.short":             ClassTelemetry,
		"test.prefix.long.action":       ClassAccess,
	}
	for action, want := range cases {
		if got := Classify(action); got != want {
			t.Errorf("Classify(%q) = %q, want %q", action, got, want)
		}
	}

	RegisterClassification(map[string]Class{"graphql.query.TestAccessField": ClassTelemetry}, nil)
	if got := Classify("graphql.query.TestAccessField"); got != ClassTelemetry {
		t.Errorf("later registration should override, got %q", got)
	}
}
