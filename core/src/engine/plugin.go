package engine

type Credentials struct {
	Hostname string
	Username string
	Password string
	Port     int
}

type PluginConfig struct {
	Credentials *Credentials
}

type PluginFunctions interface {
	GetSchema(config *PluginConfig) []string
	GetStorageUnits(config *PluginConfig, schema string) ([]string, error)
	GetRows(config *PluginConfig, schema string, storageUnit string) []string
	GetColumns(config *PluginConfig, schema string, storageUnit string, row string) map[string][]string
	GetConstraints(config *PluginConfig) map[string]string
	RawExecute(config *PluginConfig, sql string) error
}

type Plugin struct {
	PluginFunctions
	Type DatabaseType
}
