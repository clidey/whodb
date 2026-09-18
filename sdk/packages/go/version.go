package whodb

import (
	"runtime/debug"
	"strings"
	"sync"
)

// SDKVersion is stamped by the release tooling (sync-versions.mjs) and serves
// as the fallback User-Agent version for in-repo builds. Consumer builds
// report the module version from Go build info instead — the in-repo
// manifests stay at 0.0.0 and release git tags carry the real version.
const SDKVersion = "0.0.0"

const modulePath = "github.com/clidey/whodb/sdk/packages/go"

// resolvedVersion returns the SDK version for the User-Agent header: the
// module version recorded in the consumer's build info when available,
// falling back to the stamped constant for in-module and dev builds.
var resolvedVersion = sync.OnceValue(func() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return SDKVersion
	}
	for _, dep := range info.Deps {
		if dep.Path != modulePath {
			continue
		}
		version := dep.Version
		if dep.Replace != nil {
			version = dep.Replace.Version
		}
		if version != "" && version != "(devel)" {
			return strings.TrimPrefix(version, "v")
		}
	}
	return SDKVersion
})
