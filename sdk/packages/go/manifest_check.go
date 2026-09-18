package whodb

import (
	"log"
	"sync"

	"github.com/clidey/whodb/sdk/packages/go/gen"
)

var (
	warnedMu sync.Mutex
	warned   = map[string]bool{}
)

// warnIfFlagged emits the once-per-process deprecation/behavior-change
// warning for an operation (SDK versioning policy §2.3). Actually-removed
// operations surface as ErrVersion from interpretServerError.
func warnIfFlagged(operation string) {
	policy, ok := gen.EmbeddedManifest[operation]
	if !ok {
		return
	}
	warnedMu.Lock()
	defer warnedMu.Unlock()
	if warned[operation] {
		return
	}
	switch {
	case policy.Deprecated:
		warned[operation] = true
		suffix := ""
		if policy.SunsetAt != "" {
			suffix = " and will be removed after " + policy.SunsetAt
		}
		log.Printf("[whodb] %s is deprecated%s — upgrade the Go SDK module before then. %s", operation, suffix, policy.Note)
	case policy.BehaviorChanged:
		warned[operation] = true
		log.Printf("[whodb] %s's behavior changed in this platform release — results may differ from previous SDK versions. %s", operation, policy.Note)
	}
}
