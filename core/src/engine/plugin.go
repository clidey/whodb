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

type GraphUnitRelationshipType string

const (
	GraphUnitRelationshipType_OneToOne   = "OneToOne"
	GraphUnitRelationshipType_OneToMany  = "OneToMany"
	GraphUnitRelationshipType_ManyToOne  = "ManyToOne"
	GraphUnitRelationshipType_ManyToMany = "ManyToMany"
	GraphUnitRelationshipType_Unknown    = "Unknown"
)

type GraphUnitRelationship struct {
	Name             string
	RelationshipType GraphUnitRelationshipType
}

type GraphUnit struct {
	Unit      StorageUnit
	Relations []GraphUnitRelationship
}

type PluginFunctions interface {
	GetDatabases() ([]string, error)
	IsAvailable(config *PluginConfig) bool
	GetSchema(config *PluginConfig) ([]string, error)
	GetStorageUnits(config *PluginConfig, schema string) ([]StorageUnit, error)
	UpdateStorageUnit(config *PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error)
	GetRows(config *PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*GetRowsResult, error)
	GetGraph(config *PluginConfig, schema string) ([]GraphUnit, error)
	RawExecute(config *PluginConfig, query string) (*GetRowsResult, error)
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
