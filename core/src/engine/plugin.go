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
)

type GraphUnitRelationship struct {
	Name             string
	RelationshipType GraphUnitRelationshipType
}

type GraphUnit struct {
	Name       string
	References []GraphUnitRelationship
	Dependents []GraphUnitRelationship
}

type PluginFunctions interface {
	GetStorageUnits(config *PluginConfig) ([]StorageUnit, error)
	GetRows(config *PluginConfig, storageUnit string, where string, pageSize int, pageOffset int) (*GetRowsResult, error)
	GetGraph(config *PluginConfig) ([]GraphUnit, error)
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
