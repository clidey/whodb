// Package sdkgen generates typed per-project SDK clients (entity classes and
// a typed root) from a project's ontology, layered over the dynamic
// @clidey/whodb-sdk / whodb SDK facades. See ee SDK_DESIGN.md §3.2.
package sdkgen

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// Model is the language-neutral description of a project's ontology used by
// the code generators. Its canonical JSON is hashed so generated code can
// detect drift from the live ontology at runtime.
type Model struct {
	Entities []Entity `json:"entities"`
}

// Entity is one ontology object type.
type Entity struct {
	APIName     string     `json:"apiName"`
	DisplayName string     `json:"displayName"`
	Description string     `json:"description"`
	PrimaryKey  string     `json:"primaryKey"`
	Properties  []Property `json:"properties"`
	Links       []Link     `json:"links"`
}

// Property is one typed field of an entity.
type Property struct {
	APIName     string `json:"apiName"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	DataType    string `json:"dataType"`
	Required    bool   `json:"required"`
	IsPK        bool   `json:"isPk"`
}

// Link is one relationship to another entity.
type Link struct {
	APIName       string `json:"apiName"`
	TargetAPIName string `json:"targetApiName"`
	Cardinality   string `json:"cardinality"`
}

// BuildModel converts platform ontology metadata into the generation model,
// sorted deterministically so the hash is stable.
func BuildModel(ontologies []platformapi.Ontology) *Model {
	model := &Model{}
	for _, ontology := range ontologies {
		entity := Entity{
			APIName:     ontology.APIName,
			DisplayName: ontology.DisplayName,
			Description: ontology.Description,
			PrimaryKey:  ontology.PrimaryKey,
		}
		for _, property := range ontology.Properties {
			entity.Properties = append(entity.Properties, Property{
				APIName:     property.APIName,
				DisplayName: property.DisplayName,
				Description: property.Description,
				DataType:    property.DataType,
				Required:    property.IsRequired,
				IsPK:        property.APIName == ontology.PrimaryKey,
			})
		}
		for _, link := range ontology.Links {
			entity.Links = append(entity.Links, Link{
				APIName:       link.APIName,
				TargetAPIName: link.TargetOntologyAPIName,
				Cardinality:   link.Cardinality,
			})
		}
		sort.Slice(entity.Properties, func(i, j int) bool { return entity.Properties[i].APIName < entity.Properties[j].APIName })
		sort.Slice(entity.Links, func(i, j int) bool { return entity.Links[i].APIName < entity.Links[j].APIName })
		model.Entities = append(model.Entities, entity)
	}
	sort.Slice(model.Entities, func(i, j int) bool { return model.Entities[i].APIName < model.Entities[j].APIName })
	return model
}

// Hash returns the SHA-256 of the model's canonical JSON. Generated files
// embed it; the SDK compares it against live metadata to warn on drift.
func (m *Model) Hash() string {
	encoded, err := json.Marshal(m)
	if err != nil {
		return ""
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:])
}

// tsType maps an ontology data type to its TypeScript representation.
func tsType(dataType string) string {
	switch dataType {
	case "Integer", "Long", "Double", "Float":
		return "number"
	case "Boolean":
		return "boolean"
	case "Date", "Timestamp":
		return "Date"
	case "Array":
		return "unknown[]"
	case "Struct":
		return "Record<string, unknown>"
	default: // String, UUID, unknown
		return "string"
	}
}

// pyType maps an ontology data type to its Python type-hint representation.
func pyType(dataType string) string {
	switch dataType {
	case "Integer", "Long":
		return "int"
	case "Double", "Float":
		return "float"
	case "Boolean":
		return "bool"
	case "Date", "Timestamp":
		return "datetime"
	case "Array":
		return "list"
	case "Struct":
		return "dict"
	default:
		return "str"
	}
}

// identifier converts an ontology apiName into a safe exported identifier for
// class names (e.g. "user_account" -> "UserAccount").
func identifier(apiName string) string {
	var builder strings.Builder
	upperNext := true
	for _, r := range apiName {
		switch {
		case r == '_' || r == '-' || r == ' ':
			upperNext = true
		case upperNext:
			builder.WriteString(strings.ToUpper(string(r)))
			upperNext = false
		default:
			builder.WriteRune(r)
		}
	}
	name := builder.String()
	if name == "" {
		return "Entity"
	}
	if name[0] >= '0' && name[0] <= '9' {
		name = "E" + name
	}
	return name
}

// validateModel rejects models the generators cannot represent.
func validateModel(model *Model) error {
	if len(model.Entities) == 0 {
		return fmt.Errorf("project has no ontology entities — nothing to generate")
	}
	seen := map[string]string{}
	for _, entity := range model.Entities {
		ident := identifier(entity.APIName)
		if existing, ok := seen[ident]; ok {
			return fmt.Errorf("entities %q and %q both map to identifier %s — rename one", existing, entity.APIName, ident)
		}
		seen[ident] = entity.APIName
	}
	return nil
}
