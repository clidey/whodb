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

package crash

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestPrintCrashReport(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	// Call printCrashReport
	printCrashReport("test panic error")

	// Restore stderr
	w.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains expected elements
	expectedStrings := []string{
		"WhoDB CLI crashed unexpectedly",
		"test panic error",
		"github.com/clidey/whodb/issues",
		"Stack Trace",
		"Describe the bug",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Crash report should contain %q", expected)
		}
	}
}

func TestPrintCrashReport_ContainsVersionInfo(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printCrashReport("version test error")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain system info sections
	if !strings.Contains(output, "OS:") {
		t.Error("Crash report should contain OS info")
	}
	if !strings.Contains(output, "Architecture:") {
		t.Error("Crash report should contain architecture info")
	}
	if !strings.Contains(output, "Go Version:") {
		t.Error("Crash report should contain Go version")
	}
}

func TestPrintCrashReport_ContainsCommandLine(t *testing.T) {
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	printCrashReport("command line test")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should contain "To Reproduce" section
	if !strings.Contains(output, "To Reproduce") {
		t.Error("Crash report should contain 'To Reproduce' section")
	}
}

func TestHandler_NoPanic(t *testing.T) {
	// Handler should do nothing when there's no panic
	func() {
		defer Handler()
		// No panic here
	}()
	// If we get here, the test passed (Handler didn't cause issues)
}

// Note: Testing Handler() with an actual panic is tricky because it calls os.Exit(1).
// In production tests, you would use a subprocess test pattern.
