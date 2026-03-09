//go:build ee

package src

import (
	"os"

	"github.com/clidey/whodb/core/src/env"
)

func init() {
	// Set enterprise edition flag
	env.IsEnterpriseEdition = true
	os.Setenv("WHODB_DISABLE_UPDATE_CHECK", "true")

	// Note: The actual EE initialization is registered by the EE module itself
	// via RegisterEEInitializer when it's imported by the main application
}
