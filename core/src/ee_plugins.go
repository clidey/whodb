//go:build ee

package src

import (
	"github.com/clidey/whodb/core/src/env"
)

func init() {
	// Set enterprise edition flag
	env.IsEnterpriseEdition = true

	// Note: The actual EE initialization is registered by the EE module itself
	// via RegisterEEInitializer when it's imported by the main application
}
