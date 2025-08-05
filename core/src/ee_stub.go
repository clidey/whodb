//go:build !ee

package src

import "github.com/clidey/whodb/core/src/engine"

// Stub for CE builds - this file is used during code generation
// when EE modules are not available
func init() {
	// CE version - no EE initialization
	SetEEInitializer(func(e *engine.Engine) {
		// No-op for CE
	})
}
