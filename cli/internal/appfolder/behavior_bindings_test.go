package appfolder

import "testing"

func TestBehaviorBindingsRequireExplicitNamedCommands(t *testing.T) {
	if err := checkBehaviorBindings(map[string]BehaviorBinding{"release": {Ontology: "stock", Action: "release"}}); err != nil {
		t.Fatal(err)
	}
	for _, binding := range []BehaviorBinding{{}, {Ontology: "stock"}, {Ontology: "stock", Action: "create"}, {Ontology: "stock", Action: "update"}, {Ontology: "stock", Action: "delete"}} {
		if err := checkBehaviorBindings(map[string]BehaviorBinding{"release": binding}); err == nil {
			t.Fatalf("accepted invalid binding: %#v", binding)
		}
	}
}
