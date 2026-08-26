package sdkgen

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*.tmpl
var templates embed.FS

// Language is a supported typed-codegen output language.
type Language string

// Supported codegen languages.
const (
	LanguageTypeScript Language = "ts"
	LanguagePython     Language = "python"
)

// ParseLanguage validates a --language flag value.
func ParseLanguage(value string) (Language, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "ts", "typescript":
		return LanguageTypeScript, nil
	case "python", "py":
		return LanguagePython, nil
	default:
		return "", fmt.Errorf("unsupported language %q — supported: ts, python", value)
	}
}

// templateEntity is the per-entity view passed to templates.
type templateEntity struct {
	APIName     string
	Ident       string
	DisplayName string
	Description string
	PrimaryKey  string
	Properties  []templateProperty
	Links       []templateLink
}

type templateProperty struct {
	APIName  string
	Required bool
	TSType   string
	PyType   string
}

type templateLink struct {
	APIName       string
	TargetAPIName string
	Cardinality   string
	MethodName    string // TS: camelCase link accessor
	PyMethodName  string // Python: snake_case link accessor
}

type templateData struct {
	Hash     string
	Entities []templateEntity
}

// Generate renders the typed client for one language into outDir. It returns
// the paths of the files written.
func Generate(model *Model, language Language, outDir string) ([]string, error) {
	if err := validateModel(model); err != nil {
		return nil, err
	}
	data := templateData{Hash: model.Hash()}
	for _, entity := range model.Entities {
		view := templateEntity{
			APIName:     entity.APIName,
			Ident:       identifier(entity.APIName),
			DisplayName: entity.DisplayName,
			Description: strings.ReplaceAll(entity.Description, "*/", ""),
			PrimaryKey:  entity.PrimaryKey,
		}
		for _, property := range entity.Properties {
			view.Properties = append(view.Properties, templateProperty{
				APIName:  property.APIName,
				Required: property.Required,
				TSType:   tsType(property.DataType),
				PyType:   pyType(property.DataType),
			})
		}
		for _, link := range entity.Links {
			view.Links = append(view.Links, templateLink{
				APIName:       link.APIName,
				TargetAPIName: link.TargetAPIName,
				Cardinality:   link.Cardinality,
				MethodName:    linkMethodName(link.APIName),
				PyMethodName:  pyLinkMethodName(link.APIName),
			})
		}
		data.Entities = append(data.Entities, view)
	}

	var templateName, outFile string
	switch language {
	case LanguageTypeScript:
		templateName, outFile = "typescript.ts.tmpl", "whodb.generated.ts"
	case LanguagePython:
		templateName, outFile = "python.py.tmpl", "whodb_generated.py"
	default:
		return nil, fmt.Errorf("unsupported language %q", language)
	}

	parsed, err := template.ParseFS(templates, "templates/"+templateName)
	if err != nil {
		return nil, fmt.Errorf("parse template: %w", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	outPath := filepath.Join(outDir, outFile)
	file, err := os.Create(outPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	if err := parsed.ExecuteTemplate(file, templateName, data); err != nil {
		return nil, fmt.Errorf("render %s: %w", templateName, err)
	}
	return []string{outPath}, nil
}

// linkMethodName renders a link apiName as a camelCase TS method name.
func linkMethodName(apiName string) string {
	ident := identifier(apiName)
	return strings.ToLower(ident[:1]) + ident[1:]
}

// pyLinkMethodName renders a link apiName as a snake_case Python method name.
func pyLinkMethodName(apiName string) string {
	var builder strings.Builder
	for i, r := range apiName {
		switch {
		case r >= 'A' && r <= 'Z':
			if i > 0 {
				builder.WriteByte('_')
			}
			builder.WriteRune(r - 'A' + 'a')
		case r == '-' || r == ' ':
			builder.WriteByte('_')
		default:
			builder.WriteRune(r)
		}
	}
	return builder.String()
}
