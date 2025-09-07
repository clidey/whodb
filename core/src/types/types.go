package types

type DatabaseCredentials struct {
	Alias    string            `json:"alias"`
	Hostname string            `json:"host"`
	Username string            `json:"user"`
	Password string            `json:"password"`
	Database string            `json:"database"`
	Port     string            `json:"port"`
	Config   map[string]string `json:"config"`
	Extra    map[string]any

	IsProfile bool
	Type      string
	CustomId  string
	Source    string
}
