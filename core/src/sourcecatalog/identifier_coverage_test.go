package sourcecatalog_test

import (
	"maps"
	"testing"

	"github.com/clidey/whodb/core/src/dbcatalog"
	"github.com/clidey/whodb/core/src/source"
	coresourcecatalog "github.com/clidey/whodb/core/src/sourcecatalog"
)

func TestImportCapableSourcesUseCoveredIdentifierStrategy(t *testing.T) {
	coveredPluginTypes := map[string]string{
		"Postgres":    "live postgres-family hostile identifier test",
		"CockroachDB": "live postgres-family hostile identifier test",
		"YugabyteDB":  "live postgres-family hostile identifier test",
		"MySQL":       "live mysql-family hostile identifier test",
		"MariaDB":     "live mysql-family hostile identifier test",
		"TiDB":        "live mysql-family hostile identifier test",
		"Sqlite3":     "live SQLite hostile identifier test",
		"DuckDB":      "live DuckDB hostile identifier test",
		"ClickHouse":  "live ClickHouse hostile identifier test",
	}

	for _, entry := range dbcatalog.All() {
		spec, ok := coresourcecatalog.BuildTypeSpec(coresourcecatalog.DatabaseEntry{
			ID:             string(entry.ID),
			Label:          entry.Label,
			Connector:      string(entry.PluginType),
			Extra:          maps.Clone(entry.Extra),
			Fields:         coresourcecatalog.FieldVisibility(entry.Fields),
			RequiredFields: coresourcecatalog.FieldRequirements(entry.RequiredFields),
			IsAWSManaged:   entry.IsAWSManaged,
			SSLModes:       sourceSSLModes(entry.SSLModes),
		})
		if !ok || !supportsImport(spec) {
			continue
		}
		if coveredPluginTypes[string(entry.PluginType)] == "" {
			t.Errorf("import-capable source %q uses unclassified plugin type %q", entry.ID, entry.PluginType)
		}
	}
}

func supportsImport(spec source.TypeSpec) bool {
	for _, objectType := range spec.Contract.ObjectTypes {
		if objectType.SupportsAction(source.ActionImportData) {
			return true
		}
	}
	return false
}
