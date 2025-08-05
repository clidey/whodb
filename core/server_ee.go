//go:build ee

package main

import (
	// Import EE package to register EE plugins
	_ "github.com/clidey/whodb/ee/core/src/plugins"
)