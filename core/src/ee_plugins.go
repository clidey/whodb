//go:build ee

package src

import (
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	ee "github.com/clidey/whodb/ee/core/src"
)

func init() {
	// Set enterprise edition flag
	env.IsEnterpriseEdition = true
	
	// Register EE plugins when building with -tags ee
	SetEEInitializer(func(e *engine.Engine) {
		// Use the EE init function which handles both plugins and ports
		ee.InitEE(e)
	})
}