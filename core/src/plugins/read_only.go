package plugins

import (
	"fmt"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/sourcecatalog"
	"github.com/clidey/whodb/core/src/sqlguard"
)

// ReadOnlyConfig validates protected SQL and returns a request-local configuration.
// Unsupported sources and grammar never fall back to ordinary execution.
func ReadOnlyConfig(config *engine.PluginConfig, query string) (*engine.PluginConfig, error) {
	cls := sqlguard.Classify(query)
	if cls.Mutating {
		return nil, fmt.Errorf("read-only query rejected: %s", cls.Reason)
	}
	spec, ok := sourcecatalog.Find(config.Credentials.Type)
	if !ok || !spec.Traits.Query.SupportsReadOnlyExecution {
		return nil, fmt.Errorf("protected read-only execution is not supported for %s", config.Credentials.Type)
	}
	if cls.MultiStatement && !spec.Traits.Query.SupportsMultiStatement {
		return nil, engine.ErrMultiStatementUnsupported
	}
	protected := *config
	protected.ReadOnly = true
	protected.MultiStatement = cls.MultiStatement
	return &protected, nil
}
