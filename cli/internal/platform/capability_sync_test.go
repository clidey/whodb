package platform

import (
	"reflect"
	"strings"
	"testing"
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
