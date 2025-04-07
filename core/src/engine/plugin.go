package engine

import "github.com/clidey/whodb/core/graph/model"

type Credentials struct {
	Id          *string
	Type        string
	Hostname    string
	Username    string
	Password    string
	Database    string
	Advanced    []Record
	AccessToken *string
	IsProfile   bool
}

type ExternalModel struct {
	Type  string
	Token string
}

type PluginConfig struct {
	Credentials   *Credentials
	ExternalModel *ExternalModel
}

type Record struct {
	Key   string
	Value string
	Extra map[string]string
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
	Columns       []Column
	Rows          [][]string
	DisableUpdate bool
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

type ChatMessage struct {
	Type   string
	Result *GetRowsResult
	Text   string
}

type PluginFunctions interface {
	GetDatabases(config *PluginConfig) ([]string, error)
	IsAvailable(config *PluginConfig) bool
	GetAllSchemas(config *PluginConfig) ([]string, error)
	GetStorageUnits(config *PluginConfig, schema string) ([]StorageUnit, error)
	AddStorageUnit(config *PluginConfig, schema string, storageUnit string, fields map[string]string) (bool, error)
	UpdateStorageUnit(config *PluginConfig, schema string, storageUnit string, values map[string]string, updatedColumns []string) (bool, error)
	AddRow(config *PluginConfig, schema string, storageUnit string, values []Record) (bool, error)
	DeleteRow(config *PluginConfig, schema string, storageUnit string, values map[string]string) (bool, error)
	GetRows(config *PluginConfig, schema string, storageUnit string, where *model.WhereCondition, pageSize int, pageOffset int) (*GetRowsResult, error)
	GetGraph(config *PluginConfig, schema string) ([]GraphUnit, error)
	RawExecute(config *PluginConfig, query string) (*GetRowsResult, error)
	Chat(config *PluginConfig, schema string, model string, previousConversation string, query string) ([]*ChatMessage, error)
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
