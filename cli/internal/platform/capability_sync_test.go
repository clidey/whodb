package platform

import (
	"reflect"
	"strings"
	"testing"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/parser"
)

func TestAutomationFunctionModelAndSelectionsAgree(t *testing.T) {
	typ := reflect.TypeFor[Function]()
	modelFields := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		modelFields[strings.Split(typ.Field(i).Tag.Get("json"), ",")[0]] = true
	}
	for name := range functionSelectionFields {
		if !modelFields[name] {
			t.Errorf("selected field %s is discarded by Function model", name)
		}
	}
	for name := range modelFields {
		if _, ok := functionSelectionFields[name]; !ok {
			t.Errorf("Function field %s cannot be selected", name)
		}
	}
}

func TestAutomationReadModelsPreserveSelectedMetadata(t *testing.T) {
	for _, test := range []struct {
		model     reflect.Type
		selection string
	}{
		{reflect.TypeFor[Dataset](), datasetFields},
		{reflect.TypeFor[Transform](), transformFields},
	} {
		document, err := parser.ParseQuery(&ast.Source{Input: "query { result { " + test.selection + " } }"})
		if err != nil {
			t.Fatal(err)
		}
		fields := map[string]bool{}
		for i := range test.model.NumField() {
			fields[strings.Split(test.model.Field(i).Tag.Get("json"), ",")[0]] = true
		}
		root := document.Operations[0].SelectionSet[0].(*ast.Field)
		for _, selection := range root.SelectionSet {
			field := selection.(*ast.Field)
			name := field.Name
			if field.Alias != "" {
				name = field.Alias
			}
			if !fields[name] {
				t.Errorf("%s silently discards selected %s", test.model.Name(), name)
			}
		}
	}
}

func TestFunctionReadsRespectHostFieldManifest(t *testing.T) {
	client := &Client{manifest: &PlatformManifest{Types: []PlatformManifestType{{Name: "OntologyFunction", Fields: []PlatformManifestField{{Name: "id"}, {Name: "name"}}}}}}
	for _, selection := range [][]string{nil, {"id", "executionMode", "edgeRunnerId"}, {"currentRunStatus"}} {
		document := operationFunctionDetailForFields(client.functionReadFields(selection))
		for _, unsupported := range []string{"executionMode", "liveBehavior", "edgeRunnerId", "currentRunStatus"} {
			if strings.Contains(document, unsupported) {
				t.Errorf("older host queried unsupported field %s", unsupported)
			}
		}
	}
}
