package engine

type Credentials struct {
	Hostname string
	Username string
	Password string
	Database string
}

type PluginConfig struct {
	Credentials *Credentials
}

type Record struct {
	Key   string
	Value string
}

type StorageUnit struct {
	Name       string
	Attributes []Record
}

type Column struct {
	Type string
	Name string
}

type GetRowsResult struct {
	Columns []Column
	Rows    [][]string
}

type PluginFunctions interface {
	GetStorageUnits(config *PluginConfig) ([]StorageUnit, error)
	GetRows(config *PluginConfig, storageUnit string, pageSize int, pageOffset int) (*GetRowsResult, error)
	GetColumns(config *PluginConfig, storageUnit string, row string) (map[string][]string, error)
	GetConstraints(config *PluginConfig) map[string]string
	RawExecute(config *PluginConfig, sql string) error
}

type Plugin struct {
	PluginFunctions
	Type DatabaseType
}

func NewPluginConfig(credentials *Credentials) *PluginConfig {
	return &PluginConfig{
		Credentials: credentials,
	}
}
