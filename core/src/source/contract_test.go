package source

import (
	"testing"
)

//go:fix inline
func ptr[T any](v T) *T { return new(v) }

func specWithGraph(graphScopeKind *ObjectKind, objectTypes []ObjectType, rootActions []Action) TypeSpec {
	return TypeSpec{
		ID:    "test-source",
		Label: "Test",
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser, SurfaceGraph},
			RootActions:       rootActions,
			BrowsePath:        []ObjectKind{"database", "table"},
			DefaultObjectKind: "table",
			GraphScopeKind:    graphScopeKind,
			ObjectTypes:       objectTypes,
		},
	}
}

func TestValidateGraphSupported_NilRef_NoGraphScopeKind_RequiresRootAction(t *testing.T) {
	spec := specWithGraph(nil, nil, []Action{ActionBrowse})
	err := ValidateGraphSupported(spec, nil)
	if err == nil {
		t.Fatal("expected error when root has no ActionViewGraph and GraphScopeKind is nil")
	}
}

func TestValidateGraphSupported_NilRef_NoGraphScopeKind_PassesWithRootAction(t *testing.T) {
	spec := specWithGraph(nil, nil, []Action{ActionBrowse, ActionViewGraph})
	err := ValidateGraphSupported(spec, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGraphSupported_NilRef_WithGraphScopeKind_AlwaysPasses(t *testing.T) {
	spec := specWithGraph(new(ObjectKind("database")), nil, []Action{ActionBrowse})
	err := ValidateGraphSupported(spec, nil)
	if err != nil {
		t.Fatalf("unexpected error when GraphScopeKind is set and ref is nil: %v", err)
	}
}

func TestValidateGraphSupported_RefMatchesGraphScopeKind_PassesWithoutAction(t *testing.T) {
	dbKind := ObjectKind("database")
	spec := specWithGraph(new(dbKind), []ObjectType{
		{Kind: dbKind, Actions: []Action{ActionBrowse}},
	}, []Action{ActionBrowse})

	ref := &ObjectRef{Kind: dbKind, Locator: "test_db"}
	err := ValidateGraphSupported(spec, ref)
	if err != nil {
		t.Fatalf("expected pass when ref.Kind matches GraphScopeKind, got: %v", err)
	}
}

func TestValidateGraphSupported_RefDoesNotMatchGraphScopeKind_RequiresAction(t *testing.T) {
	dbKind := ObjectKind("database")
	tableKind := ObjectKind("table")
	spec := specWithGraph(new(dbKind), []ObjectType{
		{Kind: dbKind, Actions: []Action{ActionBrowse}},
		{Kind: tableKind, Actions: []Action{ActionInspect, ActionViewRows}},
	}, []Action{ActionBrowse})

	ref := &ObjectRef{Kind: tableKind, Locator: "users"}
	err := ValidateGraphSupported(spec, ref)
	if err == nil {
		t.Fatal("expected error when ref.Kind != GraphScopeKind and table lacks ActionViewGraph")
	}
}

func TestValidateGraphSupported_RefDoesNotMatchGraphScopeKind_PassesWithAction(t *testing.T) {
	dbKind := ObjectKind("database")
	tableKind := ObjectKind("table")
	spec := specWithGraph(new(dbKind), []ObjectType{
		{Kind: dbKind, Actions: []Action{ActionBrowse}},
		{Kind: tableKind, Actions: []Action{ActionInspect, ActionViewGraph}},
	}, []Action{ActionBrowse})

	ref := &ObjectRef{Kind: tableKind, Locator: "users"}
	err := ValidateGraphSupported(spec, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateGraphSupported_NoSurfaceGraph_AlwaysFails(t *testing.T) {
	spec := TypeSpec{
		ID:    "no-graph",
		Label: "NoGraph",
		Contract: Contract{
			Surfaces:    []Surface{SurfaceBrowser},
			RootActions: []Action{ActionBrowse, ActionViewGraph},
		},
	}
	err := ValidateGraphSupported(spec, nil)
	if err == nil {
		t.Fatal("expected error when SurfaceGraph is not declared")
	}
}

func TestValidateBrowseSupported_NilParent_PassesWithRootAction(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			RootActions: []Action{ActionBrowse},
		},
	}
	err := ValidateBrowseSupported(spec, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateBrowseSupported_NilParent_FailsWithoutRootAction(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			RootActions: []Action{ActionExecute},
		},
	}
	err := ValidateBrowseSupported(spec, nil)
	if err == nil {
		t.Fatal("expected error when root lacks ActionBrowse")
	}
}

func TestValidateBrowseSupported_WithParent_PassesWhenObjectHasAction(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			ObjectTypes: []ObjectType{
				{Kind: "database", Actions: []Action{ActionBrowse, ActionCreateChild}},
			},
		},
	}
	ref := &ObjectRef{Kind: "database", Locator: "mydb"}
	err := ValidateBrowseSupported(spec, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateBrowseSupported_WithParent_FailsWhenObjectLacksAction(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			ObjectTypes: []ObjectType{
				{Kind: "table", Actions: []Action{ActionInspect, ActionViewRows}},
			},
		},
	}
	ref := &ObjectRef{Kind: "table", Locator: "users"}
	err := ValidateBrowseSupported(spec, ref)
	if err == nil {
		t.Fatal("expected error when table lacks ActionBrowse")
	}
}

func TestValidateCreateChildSupported_NilParent_PassesWithRootAction(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			RootActions: []Action{ActionBrowse, ActionCreateChild},
		},
	}
	err := ValidateCreateChildSupported(spec, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCreateChildSupported_NilParent_FailsWithoutRootAction(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			RootActions: []Action{ActionBrowse},
		},
	}
	err := ValidateCreateChildSupported(spec, nil)
	if err == nil {
		t.Fatal("expected error when root lacks ActionCreateChild")
	}
}

func TestValidateCreateChildSupported_WithParent_PassesWhenObjectHasAction(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			ObjectTypes: []ObjectType{
				{Kind: "schema", Actions: []Action{ActionBrowse, ActionCreateChild}},
			},
		},
	}
	ref := &ObjectRef{Kind: "schema", Locator: "public"}
	err := ValidateCreateChildSupported(spec, ref)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCreateChildSupported_UnknownObjectKind_Fails(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			ObjectTypes: []ObjectType{
				{Kind: "table", Actions: []Action{ActionInspect}},
			},
		},
	}
	ref := &ObjectRef{Kind: "nonexistent", Locator: "x"}
	err := ValidateCreateChildSupported(spec, ref)
	if err == nil {
		t.Fatal("expected error for unknown object kind")
	}
}

func TestNormalizeContract_AddsViewGraphToGraphScopeKind(t *testing.T) {
	dbKind := ObjectKind("database")
	contract := Contract{
		Surfaces:       []Surface{SurfaceGraph},
		GraphScopeKind: &dbKind,
		ObjectTypes: []ObjectType{
			{Kind: dbKind, Actions: []Action{ActionBrowse}},
		},
	}

	normalized := NormalizeContract(contract)
	dbType, ok := normalized.ObjectTypeForKind(dbKind)
	if !ok {
		t.Fatal("database object type not found after normalization")
	}
	if !dbType.SupportsAction(ActionViewGraph) {
		t.Fatal("NormalizeContract should add ActionViewGraph to GraphScopeKind object type")
	}
}

func TestNormalizeContract_AddsViewGraphToRoot_WhenNoGraphScopeKind(t *testing.T) {
	contract := Contract{
		Surfaces:    []Surface{SurfaceGraph},
		RootActions: []Action{ActionBrowse},
	}

	normalized := NormalizeContract(contract)
	if !normalized.SupportsRootAction(ActionViewGraph) {
		t.Fatal("NormalizeContract should add ActionViewGraph to RootActions when GraphScopeKind is nil")
	}
}

func TestNormalizeContract_DoesNotAddViewGraph_WhenNoSurfaceGraph(t *testing.T) {
	contract := Contract{
		Surfaces:    []Surface{SurfaceBrowser},
		RootActions: []Action{ActionBrowse},
	}

	normalized := NormalizeContract(contract)
	if normalized.SupportsRootAction(ActionViewGraph) {
		t.Fatal("NormalizeContract should not add ActionViewGraph when SurfaceGraph is absent")
	}
}

func TestValidateObjectActionSupported_UnknownKind_ReturnsError(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			ObjectTypes: []ObjectType{
				{Kind: "table", Actions: []Action{ActionInspect}},
			},
		},
	}
	err := ValidateObjectActionSupported(spec, "unknown_kind", ActionInspect)
	if err == nil {
		t.Fatal("expected error for unknown object kind")
	}
}

func TestValidateObjectActionSupported_KnownKind_MissingAction_ReturnsError(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			ObjectTypes: []ObjectType{
				{Kind: "table", Actions: []Action{ActionInspect, ActionViewRows}},
			},
		},
	}
	err := ValidateObjectActionSupported(spec, "table", ActionDeleteData)
	if err == nil {
		t.Fatal("expected error when action is not declared")
	}
}

func TestValidateObjectActionSupported_KnownKind_HasAction_Passes(t *testing.T) {
	spec := TypeSpec{
		ID:    "test",
		Label: "Test",
		Contract: Contract{
			ObjectTypes: []ObjectType{
				{Kind: "table", Actions: []Action{ActionInspect, ActionViewRows, ActionDeleteData}},
			},
		},
	}
	err := ValidateObjectActionSupported(spec, "table", ActionDeleteData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── ValidateExecutionContract ──────────────────────────────────────────────

func TestValidateExecutionContract_ScriptsWithoutQuerySurface_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: true}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceBrowser},
			RootActions: []Action{ActionBrowse},
		},
	}
	err := ValidateExecutionContract(spec)
	if err == nil {
		t.Fatal("expected error when SupportsScripts=true but no SurfaceQuery")
	}
}

func TestValidateExecutionContract_StreamingWithoutQuerySurface_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsStreaming: true}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceBrowser},
			RootActions: []Action{ActionBrowse},
		},
	}
	err := ValidateExecutionContract(spec)
	if err == nil {
		t.Fatal("expected error when SupportsStreaming=true but no SurfaceQuery")
	}
}

func TestValidateExecutionContract_MultiStatementWithoutScripts_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsMultiStatement: true, SupportsScripts: false}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceQuery},
			RootActions: []Action{ActionBrowse, ActionExecute},
		},
	}
	err := ValidateExecutionContract(spec)
	if err == nil {
		t.Fatal("expected error when SupportsMultiStatement=true but SupportsScripts=false")
	}
}

func TestValidateExecutionContract_ScriptsWithoutRootExecute_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: true}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceQuery},
			RootActions: []Action{ActionBrowse},
		},
	}
	err := ValidateExecutionContract(spec)
	if err == nil {
		t.Fatal("expected error when SupportsScripts=true but root lacks ActionExecute")
	}
}

func TestValidateExecutionContract_ValidConfig_Passes(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: true, SupportsStreaming: true, SupportsMultiStatement: true, ExplainMode: QueryExplainModeNone}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceQuery},
			RootActions: []Action{ActionBrowse, ActionExecute},
		},
	}
	err := ValidateExecutionContract(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── ValidateScriptExecutionSupported ───────────────────────────────────────

func TestValidateScriptExecutionSupported_NoQuerySurface_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: true}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceBrowser},
			RootActions: []Action{ActionBrowse, ActionExecute},
		},
	}
	err := ValidateScriptExecutionSupported(spec)
	if err == nil {
		t.Fatal("expected error when SurfaceQuery is not declared")
	}
}

func TestValidateScriptExecutionSupported_NoRootExecute_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: true}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceQuery},
			RootActions: []Action{ActionBrowse},
		},
	}
	err := ValidateScriptExecutionSupported(spec)
	if err == nil {
		t.Fatal("expected error when root lacks ActionExecute")
	}
}

func TestValidateScriptExecutionSupported_ScriptsDisabled_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: false}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceQuery},
			RootActions: []Action{ActionBrowse, ActionExecute},
		},
	}
	err := ValidateScriptExecutionSupported(spec)
	if err == nil {
		t.Fatal("expected error when SupportsScripts=false")
	}
}

func TestValidateScriptExecutionSupported_ValidConfig_Passes(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: true}},
		Contract: Contract{
			Surfaces:    []Surface{SurfaceQuery},
			RootActions: []Action{ActionBrowse, ActionExecute},
		},
	}
	err := ValidateScriptExecutionSupported(spec)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ─── ValidateContract (integration) ─────────────────────────────────────────

func TestValidateContract_BrowserSurfaceWithoutBrowsePath_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser},
			RootActions:       []Action{ActionBrowse},
			BrowsePath:        []ObjectKind{},
			DefaultObjectKind: "table",
			ObjectTypes:       []ObjectType{{Kind: "table", Actions: []Action{ActionInspect}}},
		},
	}
	err := ValidateContract(spec)
	if err == nil {
		t.Fatal("expected error when browser surface has empty browse path")
	}
}

func TestValidateContract_BrowsePathKindNotInObjectTypes_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser},
			RootActions:       []Action{ActionBrowse},
			BrowsePath:        []ObjectKind{"database", "table"},
			DefaultObjectKind: "table",
			ObjectTypes:       []ObjectType{{Kind: "table", Actions: []Action{ActionInspect}}},
		},
	}
	err := ValidateContract(spec)
	if err == nil {
		t.Fatal("expected error when browse path kind 'database' not in object types")
	}
}

func TestValidateContract_BrowseParentWithoutBrowseAction_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser},
			RootActions:       []Action{ActionBrowse},
			BrowsePath:        []ObjectKind{"database", "table"},
			DefaultObjectKind: "table",
			ObjectTypes: []ObjectType{
				{Kind: "database", Actions: []Action{ActionCreateChild}},
				{Kind: "table", Actions: []Action{ActionInspect, ActionViewRows}},
			},
		},
	}
	err := ValidateContract(spec)
	if err == nil {
		t.Fatal("expected error when browse parent 'database' lacks ActionBrowse")
	}
}

func TestValidateContract_GraphScopeKindNotInObjectTypes_Fails(t *testing.T) {
	missingKind := ObjectKind("schema")
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser, SurfaceGraph},
			RootActions:       []Action{ActionBrowse},
			BrowsePath:        []ObjectKind{"table"},
			DefaultObjectKind: "table",
			GraphScopeKind:    &missingKind,
			ObjectTypes:       []ObjectType{{Kind: "table", Actions: []Action{ActionInspect, ActionViewRows}}},
		},
	}
	err := ValidateContract(spec)
	if err == nil {
		t.Fatal("expected error when GraphScopeKind 'schema' is not in object types")
	}
}

func TestValidateContract_ValidRelationalDatabase_Passes(t *testing.T) {
	schemaKind := ObjectKind("schema")
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Traits: TypeTraits{Query: QueryTraits{SupportsScripts: true, SupportsStreaming: true, SupportsMultiStatement: true, ExplainMode: QueryExplainModeNone}},
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser, SurfaceQuery, SurfaceGraph},
			RootActions:       []Action{ActionBrowse, ActionExecute},
			BrowsePath:        []ObjectKind{"schema", "table"},
			DefaultObjectKind: "table",
			GraphScopeKind:    &schemaKind,
			ObjectTypes: []ObjectType{
				{Kind: "schema", Actions: []Action{ActionBrowse, ActionCreateChild, ActionViewGraph}, Views: []View{ViewGraph}},
				{Kind: "table", Actions: []Action{ActionInspect, ActionViewRows, ActionInsertData, ActionUpdateData, ActionDeleteData}},
			},
		},
	}
	err := ValidateContract(spec)
	if err != nil {
		t.Fatalf("unexpected error for valid relational DB spec: %v", err)
	}
}

func TestValidateContract_ValidFlatSource_Passes(t *testing.T) {
	spec := TypeSpec{
		ID: "redis", Label: "Redis",
		Traits: TypeTraits{Query: QueryTraits{ExplainMode: QueryExplainModeNone}},
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser},
			RootActions:       []Action{ActionBrowse, ActionCreateChild},
			BrowsePath:        []ObjectKind{"key"},
			DefaultObjectKind: "key",
			ObjectTypes: []ObjectType{
				{Kind: "key", Actions: []Action{ActionInspect, ActionViewRows, ActionInsertData, ActionUpdateData, ActionDeleteData}},
			},
		},
	}
	err := ValidateContract(spec)
	if err != nil {
		t.Fatalf("unexpected error for valid flat source spec: %v", err)
	}
}

func TestValidateContract_BrowserSurfaceWithoutRootBrowse_Fails(t *testing.T) {
	spec := TypeSpec{
		ID: "test", Label: "Test",
		Contract: Contract{
			Surfaces:          []Surface{SurfaceBrowser},
			RootActions:       []Action{ActionExecute},
			BrowsePath:        []ObjectKind{"table"},
			DefaultObjectKind: "table",
			ObjectTypes:       []ObjectType{{Kind: "table", Actions: []Action{ActionInspect}}},
		},
	}
	err := ValidateContract(spec)
	if err == nil {
		t.Fatal("expected error when browser surface is declared but root lacks ActionBrowse")
	}
}
