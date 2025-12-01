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

package version

import (
	"fmt"
	"runtime"
	"strings"
)

// Build-time variables injected via ldflags
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

type Info struct {
	Version   string
	Commit    string
	BuildDate string
	GoVersion string
	Platform  string
}

func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

func (i Info) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("whodb-cli %s\n", i.Version))
	b.WriteString(fmt.Sprintf("  Commit:     %s\n", i.Commit))
	b.WriteString(fmt.Sprintf("  Built:      %s\n", i.BuildDate))
	b.WriteString(fmt.Sprintf("  Go version: %s\n", i.GoVersion))
	b.WriteString(fmt.Sprintf("  Platform:   %s", i.Platform))
	return b.String()
}

func Short() string {
	return fmt.Sprintf("whodb-cli %s (%s)", Version, Commit)
}
