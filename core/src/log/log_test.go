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

package log

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestToLogrusLevel(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected logrus.Level
	}{
		{name: "info uppercase", input: "INFO", expected: logrus.InfoLevel},
		{name: "warn", input: "warn", expected: logrus.WarnLevel},
		{name: "warning", input: "warning", expected: logrus.WarnLevel},
		{name: "error mixed case", input: "Error", expected: logrus.ErrorLevel},
		{name: "debug", input: "debug", expected: logrus.DebugLevel},
		{name: "empty defaults to info", input: "", expected: logrus.InfoLevel},
		{name: "none silences output", input: "none", expected: logrus.PanicLevel},
		{name: "off silences output", input: "OFF", expected: logrus.PanicLevel},
		{name: "disabled silences output", input: "Disabled", expected: logrus.PanicLevel},
		{name: "invalid defaults to info", input: "garbage", expected: logrus.InfoLevel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if level := toLogrusLevel(tc.input); level != tc.expected {
				t.Fatalf("toLogrusLevel(%q) = %v, expected %v", tc.input, level, tc.expected)
			}
		})
	}
}
