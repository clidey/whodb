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

package clickhouse

import "testing"

func TestImplicitClickHouseDefault(t *testing.T) {
	cases := map[string]string{
		"Int8":                         "0",
		"UInt64":                       "0",
		"Float32":                      "0",
		"Decimal(10, 2)":               "0",
		"Bool":                         "false",
		"String":                       "''",
		"FixedString(16)":              "''",
		"UUID":                         "'00000000-0000-0000-0000-000000000000'",
		"Date":                         "'1970-01-01'",
		"DateTime64(3)":                "'1970-01-01 00:00:00'",
		"Array(String)":                "[]",
		"Map(String, Int32)":           "{}",
		"Tuple(String, Int32)":         "()",
		"Enum8('active' = 1, 'x' = 2)": "'active'",
	}
	for typeName, want := range cases {
		got, ok := implicitClickHouseDefault(typeName)
		if !ok || got != want {
			t.Errorf("implicitClickHouseDefault(%q) = %q, %v; want %q, true", typeName, got, ok, want)
		}
	}
	if _, ok := implicitClickHouseDefault("AggregateFunction(sum, UInt64)"); ok {
		t.Errorf("expected no implicit default for AggregateFunction")
	}
}
