/*
 * Copyright 2025 Clidey, Inc.
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
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/clidey/whodb/cli/pkg/version"
)

const issueURL = "https://github.com/clidey/whodb/issues/new?template=bug_report.md"

func Handler() {
	if r := recover(); r != nil {
		printCrashReport(r)
		os.Exit(1)
	}
}

func printCrashReport(err any) {
	// Get stack trace
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	stackTrace := string(buf[:n])

	// Get version info
	v := version.Get()

	// Get command line
	cmdLine := strings.Join(os.Args, " ")

	// Print user-friendly message
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "============================================================")
	fmt.Fprintln(os.Stderr, "  WhoDB CLI crashed unexpectedly!")
	fmt.Fprintln(os.Stderr, "============================================================")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Please report this issue at:")
	fmt.Fprintln(os.Stderr, "  "+issueURL)
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Copy and paste the following into the issue:")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "------------------------------------------------------------")
	fmt.Fprintln(os.Stderr, "")

	// Print pre-filled bug report in markdown format
	fmt.Fprintf(os.Stderr, `**Describe the bug**
WhoDB CLI crashed with an unexpected error.

**Error**
%v

**To Reproduce**
Command that caused the crash:
`+"```"+`
%s
`+"```"+`

**Expected behavior**
The command should complete without crashing.

**Desktop (please complete the following information):**
- OS: %s
- Architecture: %s
- WhoDB CLI Version: %s
- Commit: %s
- Built: %s
- Go Version: %s

**Stack Trace**
`+"```"+`
%s
`+"```"+`

**Additional context**
[Add any additional context here, such as what you were trying to do]
`,
		err,
		cmdLine,
		runtime.GOOS,
		runtime.GOARCH,
		v.Version,
		v.Commit,
		v.BuildDate,
		v.GoVersion,
		stackTrace,
	)

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "------------------------------------------------------------")
}
